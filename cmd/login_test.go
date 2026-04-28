package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/telemetry"
)

type fakeKeyring struct {
	mu    sync.Mutex
	store map[string]string
}

func newFakeKeyring() *fakeKeyring { return &fakeKeyring{store: map[string]string{}} }

func (f *fakeKeyring) key(service, user string) string { return service + "/" + user }
func (f *fakeKeyring) Get(service, user string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.store[f.key(service, user)]
	if !ok {
		return "", auth.ErrNotLoggedIn
	}
	return v, nil
}
func (f *fakeKeyring) Set(service, user, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[f.key(service, user)] = password
	return nil
}
func (f *fakeKeyring) Delete(service, user string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, f.key(service, user))
	return nil
}

type nullTracker struct{}

func (nullTracker) Identify(string, map[string]any)            {}
func (nullTracker) TrackLoginState(state telemetry.LoginState) {}

func makeFakeJWT(t *testing.T, exp int64) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"exp": exp})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// Drives runLogin against a fake web app that hits the local callback with
// the given query, and a fake API that responds to /v1/me with apiStatus.
// Returns the runLogin error, captured stdout, and whether creds were saved.
func driveLogin(t *testing.T, callbackQuery url.Values, apiStatus int) (loginErr error, out string, saved bool) {
	t.Helper()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(apiStatus)
		if apiStatus < 400 {
			_, _ = w.Write([]byte(`{"user_id":"u_test","email":"shipit@oriyn.ai"}`))
		}
	}))
	defer api.Close()

	kr := newFakeKeyring()
	store := auth.NewStoreWithKeyring(kr)

	// Fake "browser" that hits the local callback as soon as runLogin opens it.
	// runLogin builds the loginURL with `?port=...&state=...`, but we don't see
	// that URL — so we drive it by intercepting `browser.OpenURL` via a fake
	// web base that's a server that, on GET, forwards the inbound `port` and
	// `state` to 127.0.0.1:<port>/callback?<callbackQuery>+state.
	fakeWeb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		port := r.URL.Query().Get("port")
		state := r.URL.Query().Get("state")
		q := url.Values{}
		for k, vs := range callbackQuery {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		q.Set("state", state)
		// Hit the local callback directly; runLogin's success/failure flows
		// kick in regardless of what the browser sees.
		go func() {
			_, _ = http.Get("http://127.0.0.1:" + port + "/callback?" + q.Encode())
		}()
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeWeb.Close()

	// Patch browser.OpenURL via the noBrowser=false path: runLogin will call
	// browser.OpenURL with `<webBase>/auth/cli/login?port=X&state=Y`. We can't
	// intercept the package-level function here, so instead we use noBrowser
	// mode and fire the request manually from a goroutine.
	go func() {
		// give runLogin a moment to bind + register the handler
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			// poll the printed URL: not directly observable, so just kick the
			// fake web app on the assumption runLogin already printed/awaiting
			time.Sleep(20 * time.Millisecond)
			break
		}
	}()

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We need the loginURL that runLogin would open. Since we're using
	// noBrowser=true, runLogin prints it. We watch the buffer for the URL,
	// extract port+state, and drive the callback.
	pr, pw := newPipeWriter()
	go func() {
		state, port := waitForLoginURL(t, pr)
		q := url.Values{}
		for k, vs := range callbackQuery {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		q.Set("state", state)
		_, _ = http.Get("http://127.0.0.1:" + port + "/callback?" + q.Encode())
	}()

	loginErr = runLogin(ctx, fakeWeb.URL, api.URL, store, nullTracker{}, true, pw)
	pw.Close()
	out = buf.String() + drainPipe(pr)

	if _, err := kr.Get("oriyn-cli", "credentials"); err == nil {
		saved = true
	}
	return loginErr, out, saved
}

// helpers — a teed buffer so the goroutine can read what runLogin writes.

type pipeWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	ch     chan struct{}
	closed bool
}

func newPipeWriter() (*pipeWriter, *pipeWriter) {
	p := &pipeWriter{ch: make(chan struct{}, 1)}
	return p, p
}

func (p *pipeWriter) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n, err := p.buf.Write(b)
	select {
	case p.ch <- struct{}{}:
	default:
	}
	return n, err
}

func (p *pipeWriter) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	close(p.ch)
	return nil
}

func waitForLoginURL(t *testing.T, p *pipeWriter) (state, port string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		s := p.buf.String()
		p.mu.Unlock()
		if i := strings.Index(s, "http://"); i >= 0 || strings.Contains(s, "://") {
			// Find any URL with port= and state=
			for _, line := range strings.Split(s, "\n") {
				if strings.Contains(line, "port=") && strings.Contains(line, "state=") {
					line = strings.TrimSpace(line)
					u, err := url.Parse(strings.TrimSpace(line))
					if err != nil {
						continue
					}
					return u.Query().Get("state"), u.Query().Get("port")
				}
			}
		}
		select {
		case <-p.ch:
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("login URL never appeared in output: %q", p.buf.String())
	return "", ""
}

func drainPipe(p *pipeWriter) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.String()
}

// --- the actual regressions ---

// Empty token in callback (the v0.4.0 → v0.5.0 skew shape) must NOT save creds
// and must NOT report success.
func TestRunLogin_EmptyTokenInCallbackFailsLoudly(t *testing.T) {
	q := url.Values{} // no token, no access_token, nothing
	err, out, saved := driveLogin(t, q, http.StatusOK)
	if err == nil {
		t.Fatalf("expected runLogin to fail with empty token; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "no token") {
		t.Fatalf("expected error to mention missing token, got: %v", err)
	}
	if saved {
		t.Fatalf("creds were saved despite empty token — this is the silent-success regression")
	}
}

// API rejecting the token (401/403) must NOT save creds and must surface a
// clear "out of date" hint, not a generic auth error.
func TestRunLogin_APIRejectsTokenDoesNotSave(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour).Unix()
	jwt := makeFakeJWT(t, exp)
	q := url.Values{}
	q.Set("token", jwt)

	err, out, saved := driveLogin(t, q, http.StatusUnauthorized)
	if err == nil {
		t.Fatalf("expected runLogin to fail when API returned 401; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("expected error to mention API rejection, got: %v", err)
	}
	if saved {
		t.Fatalf("creds were saved despite API rejection")
	}
}

// Happy path: valid token + 200 from /v1/me saves creds and prints the email.
func TestRunLogin_HappyPathSavesAndGreets(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour).Unix()
	jwt := makeFakeJWT(t, exp)
	q := url.Values{}
	q.Set("token", jwt)

	err, out, saved := driveLogin(t, q, http.StatusOK)
	if err != nil {
		t.Fatalf("expected runLogin to succeed; got %v. out=%q", err, out)
	}
	if !saved {
		t.Fatal("creds were not saved on happy path")
	}
	if !strings.Contains(out, "shipit@oriyn.ai") {
		t.Fatalf("expected email in output, got %q", out)
	}
}
