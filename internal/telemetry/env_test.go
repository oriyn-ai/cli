package telemetry

import "testing"

// envScrub clears every known telemetry-relevant env var so tests run
// in a deterministic environment regardless of the developer's shell.
func envScrub(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"ORIYN_TELEMETRY", "DO_NOT_TRACK",
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "BUILDKITE",
		"TF_BUILD", "TEAMCITY_VERSION", "JENKINS_URL",
		"BITBUCKET_BUILD_NUMBER", "DRONE", "VERCEL", "NETLIFY",
	} {
		t.Setenv(k, "")
	}
}

func TestReadEnv_DefaultsAreClean(t *testing.T) {
	envScrub(t)

	d := ReadEnv()
	if d.ExplicitlyDisabled || d.LogMode || d.IsCI {
		t.Errorf("clean env should produce zero-value decision, got %+v", d)
	}
}

func TestReadEnv_OriynDisabled(t *testing.T) {
	envScrub(t)
	for _, v := range []string{"0", "false", "off", "disabled", "FALSE", "Off"} {
		t.Setenv("ORIYN_TELEMETRY", v)
		if d := ReadEnv(); !d.ExplicitlyDisabled {
			t.Errorf("ORIYN_TELEMETRY=%q should disable", v)
		}
	}
}

func TestReadEnv_DoNotTrack(t *testing.T) {
	envScrub(t)
	t.Setenv("DO_NOT_TRACK", "1")
	if d := ReadEnv(); !d.ExplicitlyDisabled {
		t.Error("DO_NOT_TRACK=1 should disable telemetry")
	}
}

func TestReadEnv_LogMode(t *testing.T) {
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "log")
	d := ReadEnv()
	if !d.LogMode || d.ExplicitlyDisabled {
		t.Errorf("log mode misclassified: %+v", d)
	}
}

func TestReadEnv_CIDetection(t *testing.T) {
	cases := []struct {
		envVar string
		vendor string
	}{
		{"GITHUB_ACTIONS", "github-actions"},
		{"GITLAB_CI", "gitlab-ci"},
		{"BUILDKITE", "buildkite"},
		{"CIRCLECI", "circleci"},
	}
	for _, c := range cases {
		t.Run(c.envVar, func(t *testing.T) {
			envScrub(t)
			t.Setenv(c.envVar, "true")
			d := ReadEnv()
			if !d.IsCI || d.CIVendor != c.vendor {
				t.Errorf("%s=true → got IsCI=%v vendor=%q, want true/%q", c.envVar, d.IsCI, d.CIVendor, c.vendor)
			}
		})
	}
}

func TestReadEnv_CIGenericFallback(t *testing.T) {
	envScrub(t)
	t.Setenv("CI", "true")
	d := ReadEnv()
	if !d.IsCI || d.CIVendor != "unknown" {
		t.Errorf("generic CI=true → got %+v, want IsCI/unknown", d)
	}
}

func TestCIAutoSkip_OffByDefault(t *testing.T) {
	envScrub(t)
	t.Setenv("CI", "true")
	if !ReadEnv().CIAutoSkip() {
		t.Error("CI runs should auto-skip telemetry by default")
	}
}

func TestCIAutoSkip_OverrideEnables(t *testing.T) {
	envScrub(t)
	t.Setenv("CI", "true")
	t.Setenv("ORIYN_TELEMETRY", "1")
	if ReadEnv().CIAutoSkip() {
		t.Error("ORIYN_TELEMETRY=1 should override CI auto-skip")
	}
}
