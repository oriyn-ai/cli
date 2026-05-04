package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/oauth"
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

// teeWriter buffers runLogin's output and signals new data on a channel so
// the test goroutine can extract the authorize URL the moment it appears.
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

func waitForAuthorizeURL(t *testing.T, w *teeWriter) (state, redirectURI string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		for _, line := range strings.Split(w.String(), "\n") {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "code_challenge=") || !strings.Contains(line, "state=") {
				continue
			}
			u, err := url.Parse(line)
			if err != nil {
				continue
			}
			return u.Query().Get("state"), u.Query().Get("redirect_uri")
		}
		select {
		case <-w.ch:
		case <-deadline:
			t.Fatalf("authorize URL never appeared in output: %q", w.String())
			return "", ""
		}
	}
}

type loginEnv struct {
	store          *auth.Store
	keyring        *fakeKeyring
	cfg            oauth.Config
	tokenHits      int32
	userinfoHits   int32
	tokenResponder func(w http.ResponseWriter, r *http.Request)
	userInfoStatus int
}

func newLoginEnv(t *testing.T) *loginEnv {
	t.Helper()
	env := &loginEnv{
		userInfoStatus: http.StatusOK,
	}

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&env.tokenHits, 1)
		if env.tokenResponder != nil {
			env.tokenResponder(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test_access","refresh_token":"test_refresh","expires_in":3600,"token_type":"Bearer","scope":"openid email"}`))
	}))
	t.Cleanup(tokenSrv.Close)

	userInfoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&env.userinfoHits, 1)
		w.WriteHeader(env.userInfoStatus)
		if env.userInfoStatus < 400 {
			_, _ = w.Write([]byte(`{"sub":"user_test","email":"shipit@oriyn.ai"}`))
		}
	}))
	t.Cleanup(userInfoSrv.Close)

	env.cfg = oauth.Config{
		ClientID:     "test_client",
		AuthorizeURL: "https://auth.invalid/authorize",
		TokenURL:     tokenSrv.URL,
		UserInfoURL:  userInfoSrv.URL,
		Scopes:       []string{"openid", "email"},
	}
	env.keyring = newFakeKeyring()
	env.store = auth.NewStoreWithKeyring(env.keyring, env.cfg)
	return env
}

func driveLogin(t *testing.T, env *loginEnv, callbackQuery url.Values) (out string, err error) {
	t.Helper()

	tw := newTeeWriter()
	go func() {
		state, redirectURI := waitForAuthorizeURL(t, tw)
		q := url.Values{}
		for k, vs := range callbackQuery {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		// Default to a non-empty state matching the one runLogin printed,
		// unless the test is explicitly overriding it.
		if q.Get("state") == "" {
			q.Set("state", state)
		}
		resp, getErr := http.Get(redirectURI + "?" + q.Encode())
		if getErr == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = runLogin(ctx, env.cfg, env.store, nullTracker{}, true, tw)
	out = tw.String()
	return out, err
}

// Happy path: code + matching state → exchanged for tokens, persisted, email greeted.
func TestRunLogin_HappyPath(t *testing.T) {
	env := newLoginEnv(t)

	q := url.Values{}
	q.Set("code", "test_authcode")
	out, err := driveLogin(t, env, q)
	if err != nil {
		t.Fatalf("expected runLogin to succeed; got %v. out=%q", err, out)
	}

	if _, getErr := env.keyring.Get("oriyn-cli", "credentials"); getErr != nil {
		t.Fatal("creds were not saved on happy path")
	}
	if !strings.Contains(out, "shipit@oriyn.ai") {
		t.Fatalf("expected email in output, got %q", out)
	}
	if got := atomic.LoadInt32(&env.tokenHits); got != 1 {
		t.Fatalf("expected 1 token endpoint hit, got %d", got)
	}
}

// OAuth-style error in the callback (?error=access_denied) must NOT save creds.
func TestRunLogin_OAuthErrorInCallback(t *testing.T) {
	env := newLoginEnv(t)

	q := url.Values{}
	q.Set("error", "access_denied")
	q.Set("error_description", "user declined")
	out, err := driveLogin(t, env, q)
	if err == nil {
		t.Fatalf("expected runLogin to fail; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "user declined") && !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("expected error to surface OAuth description, got: %v", err)
	}
	if _, getErr := env.keyring.Get("oriyn-cli", "credentials"); getErr == nil {
		t.Fatal("creds were saved despite OAuth error")
	}
}

// Missing code in the callback (state matches but code absent) is a hard fail.
func TestRunLogin_MissingCodeInCallback(t *testing.T) {
	env := newLoginEnv(t)

	q := url.Values{} // no code, no error
	out, err := driveLogin(t, env, q)
	if err == nil {
		t.Fatalf("expected runLogin to fail without code; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "code missing") {
		t.Fatalf("expected error to mention missing code, got: %v", err)
	}
	if _, getErr := env.keyring.Get("oriyn-cli", "credentials"); getErr == nil {
		t.Fatal("creds were saved despite missing code")
	}
	if got := atomic.LoadInt32(&env.tokenHits); got != 0 {
		t.Fatalf("expected no token exchange when code missing, got %d hits", got)
	}
}

// Token endpoint rejects the code → no creds saved, clear error.
func TestRunLogin_TokenEndpointRejects(t *testing.T) {
	env := newLoginEnv(t)
	env.tokenResponder = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"code expired"}`))
	}

	q := url.Values{}
	q.Set("code", "expired_code")
	out, err := driveLogin(t, env, q)
	if err == nil {
		t.Fatalf("expected runLogin to fail when token endpoint rejects; got success. out=%q", out)
	}
	if !strings.Contains(err.Error(), "exchanging code") {
		t.Fatalf("expected error to mention code exchange, got: %v", err)
	}
	if _, getErr := env.keyring.Get("oriyn-cli", "credentials"); getErr == nil {
		t.Fatal("creds were saved despite token-endpoint rejection")
	}
}

// /userinfo failing must NOT block login — the access token is valid.
func TestRunLogin_UserinfoFailureIsNonFatal(t *testing.T) {
	env := newLoginEnv(t)
	env.userInfoStatus = http.StatusServiceUnavailable

	q := url.Values{}
	q.Set("code", "test_authcode")
	out, err := driveLogin(t, env, q)
	if err != nil {
		t.Fatalf("expected runLogin to succeed despite userinfo error; got %v. out=%q", err, out)
	}
	if _, getErr := env.keyring.Get("oriyn-cli", "credentials"); getErr != nil {
		t.Fatal("creds were not saved when userinfo failed")
	}
	if !strings.Contains(out, "Logged in successfully") {
		t.Fatalf("expected fallback success message, got %q", out)
	}
}
