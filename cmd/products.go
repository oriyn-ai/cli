package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newProductsCmd(app *App) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "products",
		Short: "List products or get product details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductsList(cmd, app, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")

	cmd.AddCommand(newProductsListCmd(app))
	cmd.AddCommand(newProductsLsCmd(app))
	cmd.AddCommand(newProductsGetCmd(app))

	return cmd
}

func newProductsListCmd(app *App) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all products",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductsList(cmd, app, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newProductsLsCmd(app *App) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all products (alias for list)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductsList(cmd, app, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func runProductsList(cmd *cobra.Command, app *App, jsonOutput bool) error {
	products, err := app.API.ListProducts(cmd.Context())
	if err != nil {
		return err
	}
	app.Tracker.Capture("cli_products_listed", nil)

	w := cmd.OutOrStdout()
	if jsonOutput {
		return printJSON(w, products)
	}

	if len(products) == 0 {
		fmt.Fprintln(w, "No products found.")
		return nil
	}

	fmt.Fprintf(w, "%-38s %-30s STATUS\n", "ID", "NAME")
	for _, p := range products {
		fmt.Fprintf(w, "%-38s %-30s %s\n", p.ID, p.Name, p.ContextStatus)
	}
	return nil
}

func newProductsGetCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get details for a specific product",
		RunE: func(cmd *cobra.Command, args []string) error {
			product, err := app.API.GetProduct(cmd.Context(), productID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_product_viewed", map[string]interface{}{"product_id": productID})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, product)
			}

			desc := "-"
			if product.Description != nil {
				desc = *product.Description
			}

			fmt.Fprintf(w, "ID:                %s\n", product.ID)
			fmt.Fprintf(w, "Name:              %s\n", product.Name)
			fmt.Fprintf(w, "Description:       %s\n", desc)
			if len(product.URLs) > 0 {
				fmt.Fprintf(w, "URLs:              %s\n", joinStrings(product.URLs, ", "))
			}
			fmt.Fprintf(w, "Context status:    %s\n", product.ContextStatus)
			fmt.Fprintf(w, "Enrichment status: %s\n", product.EnrichmentStatus)
			fmt.Fprintf(w, "Created:           %s\n", product.CreatedAt)

			if len(product.Context) > 0 && string(product.Context) != "null" {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "Context:")
				var pretty json.RawMessage
				if err := json.Unmarshal(product.Context, &pretty); err == nil {
					formatted, err := json.MarshalIndent(pretty, "", "  ")
					if err == nil {
						fmt.Fprintln(w, string(formatted))
					} else {
						fmt.Fprintln(w, string(product.Context))
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func printJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
