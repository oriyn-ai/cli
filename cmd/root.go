package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
	// commandTop / commandSub are captured at PreRunE so the
	// completion event uses the same identifiers the started event
	// did, even on the error path where cmd is no longer in scope.
	commandTop string
	commandSub string
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
			top, sub := splitCommandPath(cmd)
			app.commandTop = top
			app.commandSub = sub
			dispatchTrackCommand(app.Tracker, top, sub)
			cmdLabel := top
			if sub != "" {
				cmdLabel = top + " " + sub
			}
			trackFlagsAndOptions(app.Tracker, cmd, cmdLabel)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if app.Tracker != nil {
				app.Tracker.TrackCommandComplete(
					app.commandTop,
					app.commandSub,
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
		newUninstallCmd(app),
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
		// PreRunE may not have fired (e.g. unknown subcommand fails
		// during cobra's argument parsing). Fall back to "root" so
		// dashboards still group these.
		top := app.commandTop
		sub := app.commandSub
		if top == "" {
			top = "root"
		}
		if app.Tracker != nil {
			app.Tracker.TrackCommandComplete(
				top,
				sub,
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

// trackFlagsAndOptions walks the cobra flag set and records every
// flag that was explicitly set on this invocation. Mirrors Vercel's
// pattern of auto-capturing all flags by name (never by value): the
// flag name is part of the public API surface (visible in --help)
// and so safe to ship; the value would freely contain user paths,
// IDs, or text input and is never captured here.
//
// String/int/duration options are recorded via TrackOption with an
// empty value — the option name alone is the signal.
func trackFlagsAndOptions(t *telemetry.Client, cmd *cobra.Command, cmdName string) {
	if t == nil || !t.Enabled() {
		return
	}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name == "" {
			return
		}
		switch f.Value.Type() {
		case "bool":
			t.TrackFlag(cmdName, f.Name)
		default:
			t.TrackOption(cmdName, f.Name, "")
		}
	})
}

// splitCommandPath extracts the top-level command name and the
// space-joined subcommand path from a cobra command. For
// `oriyn products list` it returns ("products", "list"). For a
// top-level command without subcommands it returns (name, "").
// For the bare root command it returns ("root", "").
func splitCommandPath(cmd *cobra.Command) (top, sub string) {
	if cmd == nil {
		return "root", ""
	}
	parts := []string{}
	for c := cmd; c != nil && c.Name() != ""; c = c.Parent() {
		if c.Parent() == nil {
			break
		}
		parts = append([]string{c.Name()}, parts...)
	}
	switch len(parts) {
	case 0:
		return "root", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], strings.Join(parts[1:], " ")
	}
}

// dispatchTrackCommand routes a cobra invocation to the typed
// TrackCliCommand{X} method on the telemetry client. The switch is
// the allowlist: adding a new top-level cobra command without adding
// a case here means it lands in the default branch and is captured
// as TrackCliCommandRoot, which the test suite asserts is unreachable
// for known commands. Keep alphabetical for grep-ability.
func dispatchTrackCommand(t *telemetry.Client, top, sub string) {
	if t == nil {
		return
	}
	switch top {
	case "doctor":
		t.TrackCliCommandDoctor(sub)
	case "enrich":
		t.TrackCliCommandEnrich(sub)
	case "experiment":
		t.TrackCliCommandExperiment(sub)
	case "hypotheses":
		t.TrackCliCommandHypotheses(sub)
	case "init":
		t.TrackCliCommandInit(sub)
	case "knowledge":
		t.TrackCliCommandKnowledge(sub)
	case "login":
		t.TrackCliCommandLogin(sub)
	case "logout":
		t.TrackCliCommandLogout(sub)
	case "personas":
		t.TrackCliCommandPersonas(sub)
	case "products":
		t.TrackCliCommandProducts(sub)
	case "replay":
		t.TrackCliCommandReplay(sub)
	case "skill":
		t.TrackCliCommandSkill(sub)
	case "synthesize":
		t.TrackCliCommandSynthesize(sub)
	case "telemetry":
		t.TrackCliCommandTelemetry(sub)
	case "timeline":
		t.TrackCliCommandTimeline(sub)
	case "uninstall":
		t.TrackCliCommandUninstall(sub)
	case "whoami":
		t.TrackCliCommandWhoami(sub)
	default:
		t.TrackCliCommandRoot(top)
	}
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
