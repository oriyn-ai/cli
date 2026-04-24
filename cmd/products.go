package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/apiclient"
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
	cmd.AddCommand(newProductsContextCmd(app))
	cmd.AddCommand(newProductsScrapeCmd(app))

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
	if agentMode(cmd, jsonOutput) {
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
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, product)
			}

			printProductDetail(w, product)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func printProductDetail(w io.Writer, product *apiclient.ProductDetail) {
	fmt.Fprintf(w, "ID:                %s\n", product.ID)
	fmt.Fprintf(w, "Name:              %s\n", product.Name)
	fmt.Fprintf(w, "Context status:    %s\n", product.ContextStatus)
	fmt.Fprintf(w, "Enrichment status: %s\n", product.EnrichmentStatus)
	fmt.Fprintf(w, "Created:           %s\n", product.CreatedAt)

	if product.Context != nil {
		ctx := product.Context
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Company:           %s\n", ctx.Company)
		fmt.Fprintf(w, "Product summary:   %s\n", ctx.ProductSummary)
		fmt.Fprintf(w, "Target users:      %s\n", ctx.TargetUsers)
		fmt.Fprintf(w, "Value proposition: %s\n", ctx.ValueProposition)
		if len(ctx.CoreFeatures) > 0 {
			fmt.Fprintln(w, "Core features:")
			for _, f := range ctx.CoreFeatures {
				fmt.Fprintf(w, "  - %s\n", f)
			}
		}
		if len(ctx.UseCases) > 0 {
			fmt.Fprintln(w, "Use cases:")
			for _, u := range ctx.UseCases {
				fmt.Fprintf(w, "  - %s\n", u)
			}
		}
	}
}

func newProductsContextCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Inspect and edit synthesized product context",
	}
	cmd.AddCommand(newContextShowCmd(app))
	cmd.AddCommand(newContextEditCmd(app))
	cmd.AddCommand(newContextHistoryCmd(app))
	cmd.AddCommand(newContextVersionCmd(app))
	return cmd
}

func newContextShowCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current synthesized context for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			product, err := app.API.GetProduct(cmd.Context(), productID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, product.Context)
			}
			if product.Context == nil {
				fmt.Fprintln(w, "No context synthesized yet — run `oriyn synthesize --product-id "+productID+"`.")
				return nil
			}
			printProductDetail(w, product)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newContextEditCmd(app *App) *cobra.Command {
	var productID, field, value, jsonBody string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Patch one or more context fields (scalar via --field/--value, or a full JSON body via --json-body)",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildUpdateContextRequest(field, value, jsonBody)
			if err != nil {
				return err
			}
			updated, err := app.API.UpdateContext(cmd.Context(), productID, body)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, updated)
			}
			fmt.Fprintln(w, "Context updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&field, "field", "", "Field name (company, product_summary, target_users, value_proposition)")
	cmd.Flags().StringVar(&value, "value", "", "Field value (paired with --field)")
	cmd.Flags().StringVar(&jsonBody, "json-body", "", "Full JSON patch body (overrides --field/--value; read from stdin if '-')")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func buildUpdateContextRequest(field, value, jsonBody string) (apiclient.UpdateContextRequest, error) {
	var body apiclient.UpdateContextRequest
	if jsonBody != "" {
		data := []byte(jsonBody)
		if jsonBody == "-" {
			stdin, err := io.ReadAll(readStdin())
			if err != nil {
				return body, fmt.Errorf("reading --json-body from stdin: %w", err)
			}
			data = stdin
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return body, fmt.Errorf("parsing --json-body: %w", err)
		}
		return body, nil
	}
	if field == "" || value == "" {
		return body, fmt.Errorf("provide --field and --value, or --json-body")
	}
	switch field {
	case "company":
		body.Company = &value
	case "product_summary":
		body.ProductSummary = &value
	case "target_users":
		body.TargetUsers = &value
	case "value_proposition":
		body.ValueProposition = &value
	default:
		return body, fmt.Errorf("unsupported field %q — use --json-body for list fields (core_features, use_cases)", field)
	}
	return body, nil
}

func newContextHistoryCmd(app *App) *cobra.Command {
	var productID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "history",
		Short: "List context versions for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.ListContextVersions(cmd.Context(), productID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}
			if len(resp.Versions) == 0 {
				fmt.Fprintln(w, "No context versions yet.")
				return nil
			}
			fmt.Fprintf(w, "%-38s %-5s %-10s CREATED\n", "ID", "V", "SOURCE")
			for _, v := range resp.Versions {
				fmt.Fprintf(w, "%-38s %-5d %-10s %s\n", v.ID, v.Version, v.Source, v.CreatedAt)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newContextVersionCmd(app *App) *cobra.Command {
	var productID, versionID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Get a specific context version",
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := app.API.GetContextVersion(cmd.Context(), productID, versionID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, v)
			}
			fmt.Fprintf(w, "Version %d (%s) — %s\n\n", v.Version, v.Source, v.CreatedAt)
			fmt.Fprintf(w, "Company:           %s\n", v.Context.Company)
			fmt.Fprintf(w, "Product summary:   %s\n", v.Context.ProductSummary)
			fmt.Fprintf(w, "Target users:      %s\n", v.Context.TargetUsers)
			fmt.Fprintf(w, "Value proposition: %s\n", v.Context.ValueProposition)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&versionID, "version-id", "", "The context version ID")
	_ = cmd.MarkFlagRequired("version-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newProductsScrapeCmd(app *App) *cobra.Command {
	var productID, sourceID string
	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Kick off a Firecrawl scrape for a product source",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.ScrapeSource(cmd.Context(), productID, sourceID)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_source_scraped", map[string]interface{}{"product_id": productID, "source_id": sourceID})
			fmt.Fprintf(cmd.OutOrStdout(), "Scrape %s: %s\n", resp.Status, sourceID)
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "The product source row ID")
	_ = cmd.MarkFlagRequired("source-id")
	return cmd
}

func readStdin() io.Reader { return stdinReader }
