package telemetry

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckDisclosure_PrintsOncePersistsDecision(t *testing.T) {
	withConfigDir(t)

	cfg, _ := LoadConfig()
	env := EnvDecision{IsTTY: true}

	var buf bytes.Buffer
	if !CheckDisclosure(cfg, env, &buf) {
		t.Fatal("first call should print")
	}
	if !strings.Contains(buf.String(), "anonymous usage data") {
		t.Errorf("disclosure missing key phrase: %q", buf.String())
	}
	if !cfg.HasDecided() || !cfg.IsEnabled() {
		t.Errorf("after disclosure, want decided+enabled; got %+v", cfg)
	}

	// Second call must not print and must not change state.
	reloaded, _ := LoadConfig()
	var buf2 bytes.Buffer
	if CheckDisclosure(reloaded, env, &buf2) {
		t.Error("second call should not print")
	}
	if buf2.Len() != 0 {
		t.Errorf("second call wrote %q", buf2.String())
	}
}

func TestCheckDisclosure_EnvDisabledIsSilentAndPersists(t *testing.T) {
	withConfigDir(t)

	cfg, _ := LoadConfig()
	env := EnvDecision{IsTTY: true, ExplicitlyDisabled: true}

	var buf bytes.Buffer
	if CheckDisclosure(cfg, env, &buf) {
		t.Error("env-disabled should not print")
	}
	if buf.Len() != 0 {
		t.Errorf("expected silent, got %q", buf.String())
	}
	if cfg.IsEnabled() || !cfg.HasDecided() {
		t.Errorf("env-disabled should persist disabled+decided, got %+v", cfg)
	}
}

func TestCheckDisclosure_NonTTYStaysUndecided(t *testing.T) {
	withConfigDir(t)

	cfg, _ := LoadConfig()
	env := EnvDecision{IsTTY: false}

	var buf bytes.Buffer
	if CheckDisclosure(cfg, env, &buf) {
		t.Error("non-TTY should not print")
	}
	// Crucially: do not record a decision yet — the user may run
	// interactively later and deserves the disclosure then.
	if cfg.HasDecided() {
		t.Error("non-TTY first run should NOT mark a decision")
	}
}

func TestCheckDisclosure_CIIsSilent(t *testing.T) {
	withConfigDir(t)

	cfg, _ := LoadConfig()
	env := EnvDecision{IsTTY: true, IsCI: true}

	var buf bytes.Buffer
	if CheckDisclosure(cfg, env, &buf) {
		t.Error("CI run should not print")
	}
	if cfg.HasDecided() {
		t.Error("CI first run should not lock in a decision")
	}
}

func TestCheckDisclosure_SchemaBumpReprompts(t *testing.T) {
	withConfigDir(t)

	cfg, _ := LoadConfig()
	env := EnvDecision{IsTTY: true}
	_ = CheckDisclosure(cfg, env, &bytes.Buffer{})

	// Simulate a build with a bumped schema.
	cfg.SchemaVersion = CurrentSchemaVersion - 1
	_ = SaveConfig(cfg)

	reloaded, _ := LoadConfig()
	var buf bytes.Buffer
	if !CheckDisclosure(reloaded, env, &buf) {
		t.Fatal("schema mismatch should re-prompt")
	}
	if !strings.Contains(buf.String(), "schema has changed") {
		t.Errorf("re-prompt should mention schema change, got %q", buf.String())
	}
}
