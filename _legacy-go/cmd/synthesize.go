package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newSynthesizeCmd(app *App) *cobra.Command {
	var productID string
	var wait bool
	var timeout, pollInterval time.Duration

	cmd := &cobra.Command{
		Use:   "synthesize",
		Short: "Trigger product-context synthesis (optionally wait until ready)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.Synthesize(cmd.Context(), productID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Context synthesis %s: %s\n", resp.Status, productID)

			if !wait {
				return nil
			}
			return waitForStatus(cmd.Context(), w, pollInterval, timeout, func() (string, error) {
				p, err := app.API.GetProduct(cmd.Context(), productID)
				if err != nil {
					return "", err
				}
				return p.ContextStatus, nil
			})
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&wait, "wait", false, "Block until context_status is ready or failed")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Maximum time to wait")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 3*time.Second, "How often to poll when waiting")
	return cmd
}
