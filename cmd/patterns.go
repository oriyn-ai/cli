package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newPatternsCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "View behavioral patterns for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetPatterns(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_patterns_viewed", map[string]interface{}{"product_id": productID})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No patterns found.")
				return nil
			}

			for _, pattern := range resp.Data {
				fmt.Fprintln(w, pattern.Title)
				fmt.Fprintf(w, "  %s\n", pattern.Description)
				fmt.Fprintf(w, "  Frequency:    %s\n", pattern.Frequency)
				fmt.Fprintf(w, "  Significance: %s\n", pattern.Significance)
				var steps []string
				if err := json.Unmarshal(pattern.RawSequence, &steps); err == nil && len(steps) > 0 {
					fmt.Fprintf(w, "  Sequence:     %s\n", strings.Join(steps, " -> "))
				}
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
