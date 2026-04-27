package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/telemetry"
)

func newTelemetryCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage anonymous usage telemetry",
		Long: "oriyn collects anonymous usage data to improve the CLI. " +
			"This command lets you see what's collected and turn it on or off.\n\n" +
			"Full schema: " + telemetry.TelemetryURL,
	}
	cmd.AddCommand(
		newTelemetryStatusCmd(version),
		newTelemetryEnableCmd(),
		newTelemetryDisableCmd(),
		newTelemetryPreviewCmd(version),
	)
	return cmd
}

func newTelemetryStatusCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether telemetry is enabled and which IDs are in use",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := telemetry.LoadConfig()
			if err != nil {
				return err
			}
			env := telemetry.ReadEnv()

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "mode:           "+resolveMode(cfg, env, version))
			fmt.Fprintln(out, "device_id:      "+orDash(cfg.DeviceID))
			fmt.Fprintln(out, "session_id:     "+orDash(cfg.SessionID))
			fmt.Fprintln(out, "schema_version: "+fmt.Sprint(cfg.SchemaVersion))
			fmt.Fprintln(out, "decided_at:     "+orDashTime(cfg.DecidedAt))
			if env.ExplicitlyDisabled {
				fmt.Fprintln(out, "note:           disabled by environment variable")
			} else if env.CIAutoSkip() {
				fmt.Fprintln(out, "note:           CI detected — auto-skipped (set ORIYN_TELEMETRY=1 to override)")
			}
			fmt.Fprintln(out, "details:        "+telemetry.TelemetryURL)
			return nil
		},
	}
}

func newTelemetryEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable telemetry collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := telemetry.LoadConfig()
			if err != nil {
				return err
			}
			on := true
			now := time.Now().UTC()
			cfg.Enabled = &on
			cfg.DecidedAt = &now
			cfg.SchemaVersion = telemetry.CurrentSchemaVersion
			if err := telemetry.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Telemetry enabled.")
			return nil
		},
	}
}

func newTelemetryDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable telemetry collection and rotate the local device ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := telemetry.LoadConfig()
			if err != nil {
				return err
			}
			off := false
			now := time.Now().UTC()
			cfg.Enabled = &off
			cfg.DecidedAt = &now
			cfg.SchemaVersion = telemetry.CurrentSchemaVersion
			// Rotate IDs so even if the user re-enables later, future
			// events aren't linkable to past anonymous activity.
			cfg.DeviceID = ""
			cfg.SessionID = ""
			cfg.SessionStart = nil
			cfg.SessionSeen = nil
			if err := telemetry.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Telemetry disabled. Local device ID cleared.")
			return nil
		},
	}
}

func newTelemetryPreviewCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "preview",
		Short: "Print the next telemetry event payload without sending it",
		Long: "Runs in log-mode: identifies, captures a sample event, and prints " +
			"the JSON payload to stderr instead of sending. Equivalent to running " +
			"any oriyn command with ORIYN_TELEMETRY=log.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := telemetry.NewClientForPreview(telemetry.Options{
				Version:   version,
				LogWriter: cmd.ErrOrStderr(),
			})
			defer client.Close()
			client.Capture("cli_preview", map[string]any{
				"command": "telemetry preview",
				"success": true,
			})
			return nil
		},
	}
}

func resolveMode(cfg *telemetry.Config, env telemetry.EnvDecision, version string) string {
	if version == "" || version == "dev" {
		return "off (dev build)"
	}
	if env.ExplicitlyDisabled {
		return "off (env)"
	}
	if env.CIAutoSkip() {
		return "off (ci)"
	}
	if env.LogMode {
		return "log"
	}
	if cfg.Enabled != nil && !*cfg.Enabled {
		return "off"
	}
	if cfg.Enabled != nil && *cfg.Enabled {
		return "on"
	}
	return "on (default)"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func orDashTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}
