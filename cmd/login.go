package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/telemetry"
)

// loginTracker is the subset of telemetry.Client we need from runLogin.
// Defined as an interface so tests can pass a stub.
type loginTracker interface {
	Identify(userID string, props map[string]any)
	TrackLoginState(state telemetry.LoginState)
}

type callbackResult struct {
	accessToken  string
	refreshToken string
	expiresIn    int64
}

func newLoginCmd(app *App) *cobra.Command {
	var noBrowser bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Oriyn via browser login",
		Long: "Starts a local callback server and opens the browser for Supabase " +
			"sign-in. Use --no-browser on headless machines or remote shells — " +
			"the URL will be printed for you to open manually.\n\n" +
			"For non-interactive CI/agent environments, set ORIYN_ACCESS_TOKEN " +
			"instead of running `oriyn login`.",
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
		creds := &auth.Credentials{
			AccessToken:  cb.accessToken,
			RefreshToken: cb.refreshToken,
			ExpiresAt:    time.Now().Unix() + cb.expiresIn,
		}
		if err := authStore.Save(creds); err != nil {
			trackState(tracker, telemetry.LoginStateFailed)
			return err
		}

		me, err := fetchMe(ctx, apiBase, creds.AccessToken)
		if err == nil {
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
			fmt.Fprintln(w, "Logged in successfully.")
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

func makeCallbackHandler(expectedState string, ch chan<- callbackResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if q.Get("state") != expectedState {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h1>Error</h1><p>State mismatch — possible CSRF. Please try again.</p></body></html>")
			return
		}

		var expiresIn int64
		if v := q.Get("expires_in"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				expiresIn = parsed
			}
		}

		ch <- callbackResult{
			accessToken:  q.Get("access_token"),
			refreshToken: q.Get("refresh_token"),
			expiresIn:    expiresIn,
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
			`<h1>Authentication successful</h1>`+
			`<p>You may close this page and return to your terminal.</p>`+
			`</body></html>`)
	}
}
