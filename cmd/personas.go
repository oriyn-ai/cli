package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newPersonasCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "personas",
		Short: "View behavioral personas for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetPersonas(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_personas_viewed", map[string]interface{}{"product_id": productID})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No personas found.")
				return nil
			}

			for _, persona := range resp.Data {
				fmt.Fprintf(w, "%s (~%d%% of users)\n", persona.Name, persona.SizeEstimate)
				fmt.Fprintf(w, "  %s\n", persona.Description)
				var traits []string
				if err := json.Unmarshal(persona.BehavioralTraits, &traits); err == nil {
					for _, t := range traits {
						fmt.Fprintf(w, "  - %s\n", t)
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
