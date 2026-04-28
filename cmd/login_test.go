package cmd

import (
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

// teeWriter is a writer the test goroutine can both read from and let runLogin
// write into. It buffers output and signals new data on a channel so the URL
// watcher doesn't have to poll on a fixed interval.
type teeWriter struct {
	mu  sync.Mutex
	buf strings.Builder
	ch  chan struct{}
}

func newTeeWriter() *teeWriter { return &teeWriter{ch: make(chan struct{}, 1)} }

func (t *teeWriter) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	n, err := t.buf.Write(b)
	select {
	case t.ch <- struct{}{}:
	default:
	}
	return n, err
}

func (t *teeWriter) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func waitForLoginURL(t *testing.T, w *teeWriter) (state, port string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		for _, line := range strings.Split(w.String(), "\n") {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "port=") || !strings.Contains(line, "state=") {
				continue
			}
			u, err := url.Parse(line)
			if err != nil {
				continue
			}
			return u.Query().Get("state"), u.Query().Get("port")
		}
		select {
		case <-w.ch:
		case <-deadline:
			t.Fatalf("login URL never appeared in output: %q", w.String())
			return "", ""
		}
	}
}

// driveLogin runs runLogin with noBrowser=true, watches its output for the
// printed login URL, then fires the callback with callbackQuery+state. The
// fake API server answers /v1/me with apiStatus.
func driveLogin(t *testing.T, callbackQuery url.Values, apiStatus int) (out string, saved bool, err error) {
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

	tw := newTeeWriter()

	go func() {
		state, port := waitForLoginURL(t, tw)
		q := url.Values{}
		for k, vs := range callbackQuery {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		q.Set("state", state)
		resp, getErr := http.Get("http://127.0.0.1:" + port + "/callback?" + q.Encode())
		if getErr == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// webBase is unused on the noBrowser=true path beyond being printed in the
	// URL — runLogin still embeds it, so any value works.
	err = runLogin(ctx, "http://web.invalid", api.URL, store, nullTracker{}, true, tw)
	out = tw.String()

	if _, kerr := kr.Get("oriyn-cli", "credentials"); kerr == nil {
		saved = true
	}
	return out, saved, err
}

// Empty token in callback (the v0.4.0 → v0.5.0 skew shape) must NOT save creds
// and must NOT report success.
func TestRunLogin_EmptyTokenInCallbackFailsLoudly(t *testing.T) {
	q := url.Values{} // no token, no access_token, nothing
	out, saved, err := driveLogin(t, q, http.StatusOK)
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

	out, saved, err := driveLogin(t, q, http.StatusUnauthorized)
	if err == nil {
		t.Fatalf("expected runLogin to fail when API returned 401; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("expected error to mention API rejection, got: %v", err)
	}
	if saved {
		t.Fatal("creds were saved despite API rejection")
	}
}

// Happy path: valid token + 200 from /v1/me saves creds and prints the email.
func TestRunLogin_HappyPathSavesAndGreets(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour).Unix()
	jwt := makeFakeJWT(t, exp)
	q := url.Values{}
	q.Set("token", jwt)

	out, saved, err := driveLogin(t, q, http.StatusOK)
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
