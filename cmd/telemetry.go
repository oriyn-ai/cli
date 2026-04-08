package cmd

import (
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/telemetry"
)

func newTelemetryCmd(version string) *cobra.Command {
	var disable, enable, status bool

	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage anonymous usage telemetry",
		RunE: func(cmd *cobra.Command, args []string) error {
			telemetry.Manage(disable, enable, status, version)
			return nil
		},
	}
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable telemetry")
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable telemetry")
	cmd.Flags().BoolVar(&status, "status", false, "Show current telemetry status")
	cmd.MarkFlagsMutuallyExclusive("disable", "enable", "status")
	return cmd
}
