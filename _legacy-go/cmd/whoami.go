package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
)

func newWhoamiCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			me, err := app.API.GetMe(cmd.Context())
			if err != nil {
				if errors.Is(err, auth.ErrNotLoggedIn) || errors.Is(err, auth.ErrSessionExpired) {
					fmt.Fprintln(cmd.OutOrStdout(), "Not logged in. Run `oriyn login` to authenticate.")
					return nil
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s (%s)\n", me.Email, me.UserID)
			return nil
		},
	}
}
