package telemetry

import (
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// EnvDecision summarizes what the environment alone says about telemetry,
// before consulting the on-disk config. Used for both first-run disclosure
// gating and per-invocation runtime gating.
type EnvDecision struct {
	ExplicitlyDisabled bool   // ORIYN_TELEMETRY=0|false|off, DO_NOT_TRACK=1
	LogMode            bool   // ORIYN_TELEMETRY=log: print payload, do not send
	IsCI               bool   // any common CI env var present
	CIVendor           string // best-effort vendor name; "" if not in CI
	IsTTY              bool   // stdout is a terminal
}

// ReadEnv inspects process env once.
func ReadEnv() EnvDecision {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ORIYN_TELEMETRY")))
	dnt := strings.ToLower(strings.TrimSpace(os.Getenv("DO_NOT_TRACK")))

	d := EnvDecision{
		LogMode:            v == "log" || v == "debug",
		ExplicitlyDisabled: v == "0" || v == "false" || v == "off" || v == "disabled" || dnt == "1" || dnt == "true",
		IsTTY:              isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stderr.Fd()),
	}

	d.CIVendor, d.IsCI = detectCI()
	return d
}

// detectCI returns (vendor, true) when running in a recognized CI system.
// The vendor string is captured as a property when telemetry is enabled in
// CI (rare — see CIAutoSkip), and is "" when not in CI.
func detectCI() (string, bool) {
	// Vendor-specific markers take precedence so the vendor name is useful.
	for _, c := range []struct {
		envVar string
		vendor string
	}{
		{"GITHUB_ACTIONS", "github-actions"},
		{"GITLAB_CI", "gitlab-ci"},
		{"CIRCLECI", "circleci"},
		{"BUILDKITE", "buildkite"},
		{"TF_BUILD", "azure-pipelines"},
		{"TEAMCITY_VERSION", "teamcity"},
		{"JENKINS_URL", "jenkins"},
		{"BITBUCKET_BUILD_NUMBER", "bitbucket-pipelines"},
		{"DRONE", "drone"},
		{"VERCEL", "vercel"},
		{"NETLIFY", "netlify"},
	} {
		if os.Getenv(c.envVar) != "" {
			return c.vendor, true
		}
	}
	if os.Getenv("CI") != "" {
		return "unknown", true
	}
	return "", false
}

// CIAutoSkip reports whether the current invocation should skip telemetry
// because it's running under CI without an explicit override. Power users
// who want CI capture can set ORIYN_TELEMETRY=1.
func (d EnvDecision) CIAutoSkip() bool {
	if !d.IsCI {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ORIYN_TELEMETRY")))
	return v != "1" && v != "true" && v != "on" && v != "enabled"
}
