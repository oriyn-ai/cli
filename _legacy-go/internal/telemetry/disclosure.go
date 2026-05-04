package telemetry

import (
	"fmt"
	"io"
	"time"
)

// TelemetryURL is the public transparency page enumerating every field
// captured. Every disclosure points here.
const TelemetryURL = "https://oriyn.ai/telemetry"

// CheckDisclosure prints the first-run notice if appropriate, persists
// the decision, and returns whether it printed. Mirrors Vercel's
// checkTelemetryStatus contract: silent if env opt-out, silent if
// already decided, silent in non-interactive contexts.
//
// w is typically os.Stderr. Pass nil to suppress output (useful in
// tests or when CLI output is in machine-readable mode).
func CheckDisclosure(cfg *Config, env EnvDecision, w io.Writer) (printed bool) {
	if cfg.HasDecided() && cfg.SchemaIsCurrent() {
		return false
	}
	// Env opt-out implies the user knows what telemetry is. Don't pester.
	if env.ExplicitlyDisabled {
		off := false
		now := time.Now().UTC()
		cfg.Enabled = &off
		cfg.DecidedAt = &now
		cfg.SchemaVersion = CurrentSchemaVersion
		_ = SaveConfig(cfg)
		return false
	}
	// Non-interactive contexts get silent default-off — printing a notice
	// to a redirected stderr is noise nobody asked for, and CI runs would
	// skew funnels anyway.
	if !env.IsTTY || env.IsCI {
		// Don't write a decision here — the user may run interactively
		// later and deserves the disclosure then. Just avoid printing.
		return false
	}

	if w != nil {
		printNotice(cfg, w)
	}
	on := true
	now := time.Now().UTC()
	cfg.Enabled = &on
	cfg.DecidedAt = &now
	cfg.SchemaVersion = CurrentSchemaVersion
	_ = SaveConfig(cfg)
	return true
}

func printNotice(cfg *Config, w io.Writer) {
	if cfg.HasDecided() && !cfg.SchemaIsCurrent() {
		fmt.Fprintln(w, "Note: oriyn's telemetry schema has changed. New fields are documented at:")
		fmt.Fprintln(w, "  "+TelemetryURL)
		fmt.Fprintln(w, "Disable with: oriyn telemetry disable")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w, "Attention: oriyn collects anonymous usage data to improve the CLI.")
	fmt.Fprintln(w, "This information shapes the CLI roadmap and prioritizes features.")
	fmt.Fprintln(w, "Learn more, including how to opt out, at:")
	fmt.Fprintln(w, "  "+TelemetryURL)
	fmt.Fprintln(w)
}
