package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/telemetry"
)

// loginTracker is the subset of telemetry.Client we need from runLogin.
type loginTracker interface {
	Identify(userID string, props map[string]any)
	TrackLoginState(state telemetry.LoginState)
}

type callbackResult struct {
	token string
}

func newLoginCmd(app *App) *cobra.Command {
	var noBrowser bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Oriyn via browser login",
		Long: "Starts a local callback server and opens the browser for Clerk " +
			"sign-in. Use --no-browser on headless machines or remote shells — " +
			"the URL will be printed for you to open manually.\n\n" +
			"For non-interactive CI/agent environments, set ORIYN_ACCESS_TOKEN " +
			"to a Clerk JWT instead of running `oriyn login`.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), app.WebBase, app.APIBase, app.AuthStore, app.Tracker, noBrowser, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the login URL instead of trying to open a browser")
	return cmd
}

func runLogin(ctx context.Context, webBase, apiBase string, authStore *auth.Store, tracker loginTracker, noBrowser bool, w io.Writer) error {
	trackState(tracker, telemetry.LoginStateStarted)

	stateParam := uuid.NewString()
	callbackCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", makeCallbackHandler(stateParam, callbackCh))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		trackState(tracker, telemetry.LoginStateFailed)
		return fmt.Errorf("binding local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Shutdown(context.Background()) }()

	loginURL := fmt.Sprintf("%s/auth/cli/login?port=%d&state=%s", webBase, port, stateParam)

	if noBrowser {
		fmt.Fprintf(w, "Open this URL to log in:\n\n  %s\n\n", loginURL)
	} else if err := browser.OpenURL(loginURL); err != nil {
		fmt.Fprintf(w, "Could not open a browser. Open this URL manually:\n\n  %s\n\n", loginURL)
	} else {
		fmt.Fprintln(w, "Opening browser to log in...")
		trackState(tracker, telemetry.LoginStateBrowserOpened)
	}
	trackState(tracker, telemetry.LoginStateAwaitingCallback)
	fmt.Fprintln(w, "Waiting for authentication...")

	select {
	case cb := <-callbackCh:
		trackState(tracker, telemetry.LoginStateCallbackReceived)

		if cb.token == "" {
			trackState(tracker, telemetry.LoginStateFailed)
			return fmt.Errorf("login callback returned no token — your CLI is likely out of date for this server. Re-run install.sh to update")
		}

		expiresAt, err := jwtExpirySeconds(cb.token)
		if err != nil {
			// Fall back to a 24h assumption if parsing fails — matches the
			// recommended `cli` template lifetime. The API rejects on real
			// expiry, which forces a re-login anyway.
			expiresAt = time.Now().Unix() + 24*60*60
		}

		creds := &auth.Credentials{
			AccessToken: cb.token,
			ExpiresAt:   expiresAt,
		}

		// Verify the token reaches the API before persisting. Saving first
		// then probing risks caching a useless token (the exact failure mode
		// of the v0.4.0 → v0.5.0 callback-shape skew). A network error is
		// not authoritative — we still save and let the user retry later.
		me, fetchErr := fetchMe(ctx, apiBase, creds.AccessToken)
		var rejected *authRejectedError
		if errors.As(fetchErr, &rejected) {
			trackState(tracker, telemetry.LoginStateFailed)
			return fmt.Errorf("API rejected the token (status %d) — your CLI is likely out of date for this server. Re-run install.sh to update", rejected.statusCode)
		}

		if err := authStore.Save(creds); err != nil {
			trackState(tracker, telemetry.LoginStateFailed)
			return err
		}

		if me != nil {
			trackState(tracker, telemetry.LoginStateProfileFetched)
			if me.userID != "" && tracker != nil {
				tracker.Identify(me.userID, nil)
			}
			if me.email != "" {
				fmt.Fprintf(w, "Logged in as %s\n", me.email)
			} else {
				fmt.Fprintln(w, "Logged in successfully.")
			}
		} else {
			fmt.Fprintln(w, "Logged in (could not reach the API to verify profile — token saved).")
		}

		trackState(tracker, telemetry.LoginStateSucceeded)
		return nil
	case <-time.After(120 * time.Second):
		trackState(tracker, telemetry.LoginStateTimedOut)
		return fmt.Errorf("login timed out after 120 seconds — please try again")
	case <-ctx.Done():
		trackState(tracker, telemetry.LoginStateCanceled)
		return ctx.Err()
	}
}

func trackState(t loginTracker, state telemetry.LoginState) {
	if t != nil {
		t.TrackLoginState(state)
	}
}

type meInfo struct {
	userID string
	email  string
}

// authRejectedError signals that the API explicitly rejected the token
// (401/403). It is distinct from a network/transport error so callers can
// refuse to persist a token the server has already disowned, while still
// being lenient when /v1/me is simply unreachable.
type authRejectedError struct {
	statusCode int
}

func (e *authRejectedError) Error() string {
	return fmt.Sprintf("API rejected token (status %d)", e.statusCode)
}

func fetchMe(ctx context.Context, apiBase, token string) (*meInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/v1/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &authRejectedError{statusCode: resp.StatusCode}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var data struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &meInfo{userID: data.UserID, email: data.Email}, nil
}

// jwtExpirySeconds extracts the `exp` claim from a JWT without verifying the
// signature. We don't need to verify here — the API verifies on every request
// and rejects forged or tampered tokens. The CLI just needs `exp` to know
// when to prompt for re-login.
func jwtExpirySeconds(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, fmt.Errorf("decode payload: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, fmt.Errorf("parse payload: %w", err)
	}
	if claims.Exp == 0 {
		return 0, fmt.Errorf("no exp claim")
	}
	return claims.Exp, nil
}

func makeCallbackHandler(expectedState string, ch chan<- callbackResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if q.Get("state") != expectedState {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h1>Error</h1><p>State mismatch — possible CSRF. Please try again.</p></body></html>")
			return
		}

		token := q.Get("token")
		// Render success only when a token is actually present. An empty
		// token here means the web app sent a callback shape this binary
		// doesn't understand (typical version-skew failure). Rendering
		// "successful" anyway is the bug that masked the v0.4.0 → v0.5.0
		// migration; refuse the page and signal the failure to runLogin.
		if token == "" {
			ch <- callbackResult{token: ""}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
				`<h1>Authentication failed</h1>`+
				`<p>The login callback did not include a token. Your CLI is likely out of date for this server.</p>`+
				`<p>Re-run the installer and try again:</p>`+
				`<pre>curl -fsSL https://raw.githubusercontent.com/oriyn-ai/cli/main/install.sh | bash</pre>`+
				`</body></html>`)
			return
		}

		ch <- callbackResult{token: token}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
			`<h1>Authentication successful</h1>`+
			`<p>You may close this page and return to your terminal.</p>`+
			`</body></html>`)
	}
}
