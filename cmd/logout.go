package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.AuthStore.Delete(); err != nil {
				return err
			}
			app.Tracker.Capture("cli_logout", nil)
			app.Tracker.Reset()
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}
