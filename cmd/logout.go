package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/telemetry"
)

func newLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.AuthStore.Delete(); err != nil {
				return err
			}
			telemetry.ClearUserID()
			app.Tracker.Capture("cli_logout", nil)
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}
