package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newHypothesesCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "hypotheses",
		Short: "List testable hypotheses mined from cross-provider user behavior",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetHypotheses(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_hypotheses_viewed", map[string]interface{}{"product_id": productID})

			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No hypotheses yet — ingest more user behavior.")
				return nil
			}

			for _, h := range resp.Data {
				fmt.Fprintln(w, strings.Join(h.RenderedSequence, " → "))
				fmt.Fprintf(w, "  Users:        %d (%.1f%% of product)\n", h.UserCount, h.SignificancePct)
				fmt.Fprintf(w, "  Occurrences:  %d\n", h.Frequency)
				fmt.Fprintln(w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}
