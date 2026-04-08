package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEnrichCmd(app *App) *cobra.Command {
	var productID string

	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "Trigger behavioral enrichment for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.Enrich(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_product_enriched", map[string]interface{}{"product_id": productID})
			fmt.Fprintf(cmd.OutOrStdout(), "Enrichment %s: %s\n", resp.Status, productID)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	return cmd
}
