package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSynthesizeCmd(app *App) *cobra.Command {
	var productID string

	cmd := &cobra.Command{
		Use:   "synthesize",
		Short: "Trigger context synthesis for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.Synthesize(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_product_synthesized", map[string]interface{}{"product_id": productID})
			fmt.Fprintf(cmd.OutOrStdout(), "Context synthesis %s: %s\n", resp.Status, productID)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	return cmd
}
