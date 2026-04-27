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

	// commandStartedAt is set by PersistentPreRunE so the error and
	// success paths can both compute a duration without a per-command
	// timer. cobra's PostRunE only fires on success, so we can't rely
	// on a defer there.
	commandStartedAt time.Time
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

			app.commandStartedAt = time.Now()
			app.Tracker.TrackCommand(commandPath(cmd))
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if app.Tracker != nil {
				app.Tracker.TrackCommandComplete(
					commandPath(cmd),
					time.Since(app.commandStartedAt),
					nil,
				)
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
		// CalledAs is empty on errors at the root (e.g. unknown
		// subcommand); fall back to "root" so dashboards still group.
		cmdPath := rootCmd.CalledAs()
		if cmdPath == "" {
			cmdPath = "root"
		}
		if app.Tracker != nil {
			app.Tracker.TrackCommandComplete(
				cmdPath,
				time.Since(app.commandStartedAt),
				err,
			)
			app.Tracker.Close()
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

// commandPath returns the dot-joined path of the cobra command for
// telemetry, e.g. "products list" rather than just "list". Stable
// across releases, since each verb is renamed at most once.
func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return "root"
	}
	parts := []string{}
	for c := cmd; c != nil && c.Name() != ""; c = c.Parent() {
		// Skip the root binary name to keep the value short.
		if c.Parent() == nil {
			break
		}
		parts = append([]string{c.Name()}, parts...)
	}
	if len(parts) == 0 {
		return "root"
	}
	return strings.Join(parts, " ")
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
