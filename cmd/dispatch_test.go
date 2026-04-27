package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/telemetry"
)

// scrubCIEnv clears every env var that could push the telemetry
// client into CI auto-skip mode, so dispatch tests are deterministic
// regardless of whether they run locally or in GitHub Actions.
func scrubCIEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DO_NOT_TRACK",
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "BUILDKITE",
		"TF_BUILD", "TEAMCITY_VERSION", "JENKINS_URL",
		"BITBUCKET_BUILD_NUMBER", "DRONE", "VERCEL", "NETLIFY",
	} {
		t.Setenv(k, "")
	}
}

// TestDispatchTrackCommand_AllCobraCommandsHaveCases is the safety net
// for the typed-allowlist hierarchy: every top-level cobra command
// registered in Execute() must have a matching case in
// dispatchTrackCommand. A new command without a case falls through to
// TrackCliCommandRoot, which this test detects.
func TestDispatchTrackCommand_AllCobraCommandsHaveCases(t *testing.T) {
	scrubCIEnv(t)
	t.Setenv("ORIYN_TELEMETRY", "log")
	t.Setenv("ORIYN_CONFIG_DIR", t.TempDir())

	// The full set of top-level command names registered in Execute().
	// Keep in sync with the AddCommand block.
	knownCommands := []string{
		"init", "login", "logout", "whoami", "doctor", "skill",
		"products", "personas", "hypotheses", "knowledge",
		"timeline", "replay", "synthesize", "enrich",
		"experiment", "telemetry",
	}

	for _, top := range knownCommands {
		t.Run(top, func(t *testing.T) {
			var buf bytes.Buffer
			tracker := telemetry.NewClient(telemetry.Options{
				Version:   "1.0.0",
				LogWriter: &buf,
			})
			defer tracker.Close()

			dispatchTrackCommand(tracker, top, "")

			out := buf.String()
			if !strings.Contains(out, `"command":"`+top+`"`) {
				t.Errorf("dispatch for %q produced wrong command property: %q", top, out)
			}
			// The fallback path tags the command as "root" — if we
			// see that, we hit the default branch instead of a typed
			// case, which means a missing method.
			if strings.Contains(out, `"command":"root"`) {
				t.Errorf("dispatch for %q fell through to TrackCliCommandRoot — missing typed case", top)
			}
		})
	}
}

func TestDispatchTrackCommand_UnknownFallsBackToRoot(t *testing.T) {
	scrubCIEnv(t)
	t.Setenv("ORIYN_TELEMETRY", "log")
	t.Setenv("ORIYN_CONFIG_DIR", t.TempDir())

	var buf bytes.Buffer
	tracker := telemetry.NewClient(telemetry.Options{
		Version:   "1.0.0",
		LogWriter: &buf,
	})
	defer tracker.Close()

	dispatchTrackCommand(tracker, "nonexistent", "")

	if !strings.Contains(buf.String(), `"command":"root"`) {
		t.Errorf("unknown command should route to root, got %q", buf.String())
	}
}

func TestSplitCommandPath(t *testing.T) {
	root := &cobra.Command{Use: "oriyn"}
	products := &cobra.Command{Use: "products"}
	productsList := &cobra.Command{Use: "list"}
	products.AddCommand(productsList)
	root.AddCommand(products)

	whoami := &cobra.Command{Use: "whoami"}
	root.AddCommand(whoami)

	cases := []struct {
		cmd     *cobra.Command
		wantTop string
		wantSub string
	}{
		{productsList, "products", "list"},
		{whoami, "whoami", ""},
		{root, "root", ""},
		{nil, "root", ""},
	}
	for _, c := range cases {
		name := "nil"
		if c.cmd != nil {
			name = c.cmd.Name()
			if name == "" {
				name = "root"
			}
		}
		t.Run(name, func(t *testing.T) {
			top, sub := splitCommandPath(c.cmd)
			if top != c.wantTop || sub != c.wantSub {
				t.Errorf("got (%q, %q), want (%q, %q)", top, sub, c.wantTop, c.wantSub)
			}
		})
	}
}
