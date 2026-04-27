package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
)

func newEnrichCmd(app *App) *cobra.Command {
	var productID string
	var wait bool
	var timeout, pollInterval time.Duration

	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "Trigger behavioral enrichment for a product (optionally wait until ready)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.Enrich(cmd.Context(), productID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Enrichment %s: %s\n", resp.Status, productID)

			if !wait {
				return nil
			}
			return waitForStatus(cmd.Context(), w, pollInterval, timeout, func() (string, error) {
				p, err := app.API.GetProduct(cmd.Context(), productID)
				if err != nil {
					return "", err
				}
				return p.EnrichmentStatus, nil
			})
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&wait, "wait", false, "Block until enrichment_status is ready or failed")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum time to wait")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 5*time.Second, "How often to poll when waiting")
	return cmd
}

// waitForStatus polls the given status function until it returns a terminal
// status or the timeout/context expires. Shared by `synthesize --wait` and
// `enrich --wait` so the termination semantics stay consistent.
func waitForStatus(
	ctx context.Context,
	w io.Writer,
	pollInterval, timeout time.Duration,
	getStatus func() (string, error),
) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("did not reach terminal status within %s", timeout)
			}
			status, err := getStatus()
			if err != nil {
				return err
			}
			switch status {
			case "ready":
				fmt.Fprintln(w, "ready")
				return nil
			case "failed":
				return fmt.Errorf("operation failed (status: failed)")
			default:
				// processing / idle — keep polling silently; printing dots
				// here would dirty JSON-adjacent outputs on agents that
				// wrap these commands.
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
