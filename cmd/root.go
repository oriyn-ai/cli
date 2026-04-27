package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/apiclient"
	"github.com/oriyn-ai/cli/internal/auth"
	"github.com/oriyn-ai/cli/internal/telemetry"
)

const sentryDSN = "https://7a9c0f680579c791f90ecee37a16375f@o4510953905651712.ingest.us.sentry.io/4511156841283584"

type App struct {
	AuthStore *auth.Store
	API       *apiclient.Client
	Tracker   *telemetry.Client
	APIBase   string
	WebBase   string
}

// Execute runs the root command and returns a process exit code.
func Execute(version, commit string) int {
	app := &App{}

	rootCmd := &cobra.Command{
		Use:     "oriyn",
		Short:   "Oriyn CLI — predict how users will respond to a change before shipping it",
		Version: fmt.Sprintf("%s (%s)", version, commit),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			env := "production"
			if version == "dev" {
				env = "development"
			}
			cfg, _ := telemetry.LoadConfig()
			envDecision := telemetry.ReadEnv()

			// Sentry is bundled under the same consent switch as
			// PostHog: one telemetry off means both stay quiet.
			sentryEnabled := version != "" && version != "dev" &&
				!envDecision.ExplicitlyDisabled &&
				!envDecision.CIAutoSkip() &&
				(cfg.Enabled == nil || *cfg.Enabled)

			if sentryEnabled {
				if err := sentry.Init(sentry.ClientOptions{
					Dsn:            sentryDSN,
					Release:        version,
					Environment:    env,
					SendDefaultPII: false,
					BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
						return scrubEvent(event)
					},
				}); err != nil {
					_ = err
				}
			}

			// First-run disclosure: prints once, persists decision,
			// silent under env opt-out / CI / non-TTY.
			telemetry.CheckDisclosure(cfg, envDecision, cmd.ErrOrStderr())

			if quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet"); quiet {
				color.NoColor = true
			}
			if v := strings.ToLower(os.Getenv("ORIYN_AGENT")); v == "1" || v == "true" {
				color.NoColor = true
			}

			app.AuthStore = auth.NewStore()
			app.APIBase, _ = cmd.Flags().GetString("api-base")
			app.WebBase, _ = cmd.Flags().GetString("web-base")
			app.API = apiclient.New(app.APIBase, app.AuthStore)
			app.Tracker = telemetry.NewClient(telemetry.Options{Version: version})

			if uid := app.Tracker.IdentitySnapshot().UserID; uid != "" && sentryEnabled {
				sentry.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{ID: uid})
				})
			}

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if app.Tracker != nil {
				app.Tracker.Close()
			}
			// Sentry.Flush is a no-op when the client wasn't init'd,
			// so this is safe to call unconditionally.
			sentry.Flush(2 * time.Second)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("api-base", envOr("ORIYN_API_BASE", "https://api.oriyn.ai"), "Base URL for the Oriyn API")
	rootCmd.PersistentFlags().String("web-base", envOr("ORIYN_WEB_BASE", "https://app.oriyn.ai"), "Base URL for the Oriyn web app")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output; implies --json on commands that support it")

	rootCmd.AddCommand(
		newInitCmd(app),
		newLoginCmd(app),
		newLogoutCmd(app),
		newWhoamiCmd(app),
		newDoctorCmd(app, version, commit),
		newSkillCmd(app),
		newProductsCmd(app),
		newPersonasCmd(app),
		newHypothesesCmd(app),
		newKnowledgeCmd(app),
		newTimelineCmd(app),
		newReplayCmd(app),
		newSynthesizeCmd(app),
		newEnrichCmd(app),
		newExperimentCmd(app),
		newTelemetryCmd(version),
	)

	if err := rootCmd.Execute(); err != nil {
		cmdName := ""
		if rootCmd.CalledAs() != "" {
			cmdName = rootCmd.CalledAs()
		}
		if app.Tracker != nil {
			app.Tracker.Capture("cli_error", map[string]any{
				"command":    cmdName,
				"error_kind": classifyErrorKind(err),
			})
		}
		if isInfraError(err) {
			sentry.CaptureException(err)
			sentry.Flush(2 * time.Second)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		return classifyError(err)
	}
	return 0
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func redactTokens(s string) string {
	result := s
	for {
		idx := strings.Index(result, "Bearer ")
		if idx == -1 {
			break
		}
		start := idx + 7
		end := len(result)
		for i := start; i < len(result); i++ {
			c := result[i]
			if c == ' ' || c == '\t' || c == '\n' || c == '"' || c == '\'' {
				end = i
				break
			}
		}
		if start >= end {
			break
		}
		result = result[:start] + "[REDACTED]" + result[end:]
	}
	return result
}

func scrubEvent(event *sentry.Event) *sentry.Event {
	for i := range event.Exception {
		event.Exception[i].Value = redactTokens(event.Exception[i].Value)
	}
	for i := range event.Breadcrumbs {
		event.Breadcrumbs[i].Message = redactTokens(event.Breadcrumbs[i].Message)
	}
	sensitive := []string{"token", "key", "password", "secret", "authorization", "credential"}
	for k := range event.Extra {
		lower := strings.ToLower(k)
		for _, s := range sensitive {
			if strings.Contains(lower, s) {
				delete(event.Extra, k)
				break
			}
		}
	}
	return event
}

func isInfraError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "failed to access OS keychain") ||
		strings.Contains(msg, "failed to store credentials in OS keychain") ||
		strings.Contains(msg, "failed to parse stored credentials")
}

// classifyErrorKind buckets errors into a small allowlist for telemetry.
// We never send the raw err.Error(): user-facing error strings often
// contain paths, URLs, or argument values that we promised never to ship.
func classifyErrorKind(err error) string {
	if err == nil {
		return "none"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not logged in"):
		return "auth_missing"
	case strings.Contains(msg, "session expired"):
		return "auth_expired"
	case strings.Contains(msg, "keychain"):
		return "keychain"
	case strings.Contains(msg, "timed out"), strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"):
		return "network"
	case strings.Contains(msg, "api returned"):
		return "api_status"
	default:
		return "unknown"
	}
}
