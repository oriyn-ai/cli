package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDirectionCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "direction",
		Short: "View prescriptive product direction",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetDirection(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_direction_viewed", map[string]interface{}{"product_id": productID})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No direction data found.")
				return nil
			}

			for _, direction := range resp.Data {
				fmt.Fprintln(w, "Recommendations:")
				for _, rec := range direction.Recommendations {
					fmt.Fprintf(w, "  [%s] %s\n", strings.ToUpper(rec.Priority), rec.Title)
					fmt.Fprintf(w, "    %s\n", rec.Rationale)
				}
				var sources []string
				if err := json.Unmarshal(direction.DerivedFrom, &sources); err == nil && len(sources) > 0 {
					fmt.Fprintln(w)
					fmt.Fprintln(w, "Derived from:")
					for _, label := range sources {
						fmt.Fprintf(w, "  - %s\n", label)
					}
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
