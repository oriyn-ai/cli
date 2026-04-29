package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/oauth"
	"github.com/oriyn-ai/cli/internal/telemetry"
)

// loginTracker is the subset of telemetry.Client we need from runLogin.
type loginTracker interface {
	Identify(userID string, props map[string]any)
	TrackLoginState(state telemetry.LoginState)
}

type callbackResult struct {
	code  string
	state string
	err   string
}

func newLoginCmd(app *App) *cobra.Command {
	var noBrowser bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Oriyn via browser login",
		Long: "Starts a local callback server and opens the browser for Clerk " +
			"sign-in over OAuth 2.0 + PKCE. Use --no-browser on headless " +
			"machines or remote shells — the URL will be printed for you to " +
			"open manually.\n\n" +
			"For non-interactive CI/agent environments, set ORIYN_ACCESS_TOKEN " +
			"to a Clerk OAuth access token instead of running `oriyn login`.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), app.OAuth, app.AuthStore, app.Tracker, noBrowser, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the login URL instead of trying to open a browser")
	return cmd
}

func runLogin(ctx context.Context, oauthCfg oauth.Config, authStore *auth.Store, tracker loginTracker, noBrowser bool, w io.Writer) error {
	trackState(tracker, telemetry.LoginStateStarted)

	pkce, err := oauth.GeneratePKCE()
	if err != nil {
		trackState(tracker, telemetry.LoginStateFailed)
		return err
	}
	state, err := oauth.GenerateState()
	if err != nil {
		trackState(tracker, telemetry.LoginStateFailed)
		return err
	}

	callbackCh := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", makeCallbackHandler(state, callbackCh))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		trackState(tracker, telemetry.LoginStateFailed)
		return fmt.Errorf("binding local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Shutdown(context.Background()) }()

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	authorizeURL := oauthCfg.BuildAuthorizeURL(state, pkce.Challenge, redirectURI)

	if noBrowser {
		fmt.Fprintf(w, "Open this URL to log in:\n\n  %s\n\n", authorizeURL)
	} else if err := browser.OpenURL(authorizeURL); err != nil {
		fmt.Fprintf(w, "Could not open a browser. Open this URL manually:\n\n  %s\n\n", authorizeURL)
	} else {
		fmt.Fprintln(w, "Opening browser to log in...")
		trackState(tracker, telemetry.LoginStateBrowserOpened)
	}
	trackState(tracker, telemetry.LoginStateAwaitingCallback)
	fmt.Fprintln(w, "Waiting for authentication...")

	select {
	case cb := <-callbackCh:
		trackState(tracker, telemetry.LoginStateCallbackReceived)
		if cb.err != "" {
			trackState(tracker, telemetry.LoginStateFailed)
			return fmt.Errorf("authorization failed: %s", cb.err)
		}
		if cb.code == "" {
			trackState(tracker, telemetry.LoginStateFailed)
			return fmt.Errorf("authorization code missing from callback — sign-in did not complete")
		}

		tr, err := oauthCfg.ExchangeCode(ctx, nil, cb.code, pkce.Verifier, redirectURI)
		if err != nil {
			trackState(tracker, telemetry.LoginStateFailed)
			return fmt.Errorf("exchanging code: %w", err)
		}

		creds := &auth.Credentials{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Unix() + tr.ExpiresIn,
		}
		if err := authStore.Save(creds); err != nil {
			trackState(tracker, telemetry.LoginStateFailed)
			return err
		}

		// /userinfo failure is non-fatal — the access token is valid; we
		// just don't have a friendly email to print. Better to ship the
		// happy path than block on a side-channel call.
		ui, err := oauthCfg.FetchUserInfo(ctx, nil, creds.AccessToken)
		if err == nil && ui != nil {
			trackState(tracker, telemetry.LoginStateProfileFetched)
			if ui.Sub != "" && tracker != nil {
				tracker.Identify(ui.Sub, nil)
			}
			if ui.Email != "" {
				fmt.Fprintf(w, "Logged in as %s\n", ui.Email)
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

func makeCallbackHandler(expectedState string, ch chan<- callbackResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if errCode := q.Get("error"); errCode != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = errCode
			}
			ch <- callbackResult{err: desc}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
				`<h1>Authentication failed</h1>`+
				`<p>%s</p>`+
				`</body></html>`, htmlEscape(desc))
			return
		}

		if q.Get("state") != expectedState {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h1>Error</h1><p>State mismatch — possible CSRF. Please try again.</p></body></html>")
			return
		}

		code := q.Get("code")
		if code == "" {
			ch <- callbackResult{}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
				`<h1>Authentication failed</h1>`+
				`<p>The authorization server didn't return a code. Please try again.</p>`+
				`</body></html>`)
			return
		}

		ch <- callbackResult{code: code, state: q.Get("state")}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:4rem">`+
			`<h1>Authentication successful</h1>`+
			`<p>You may close this page and return to your terminal.</p>`+
			`</body></html>`)
	}
}

// htmlEscape is a tiny sanitizer for the inline error description. We
// don't need full HTML escaping coverage — just enough that an attacker-
// supplied error_description can't inject markup into the success page.
func htmlEscape(s string) string {
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		switch c {
		case '<':
			out = append(out, []byte("&lt;")...)
		case '>':
			out = append(out, []byte("&gt;")...)
		case '&':
			out = append(out, []byte("&amp;")...)
		case '"':
			out = append(out, []byte("&quot;")...)
		case '\'':
			out = append(out, []byte("&#39;")...)
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
