package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/apiclient"
)

func newKnowledgeCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Search the Supermemory-backed knowledge graph for a product",
	}
	cmd.AddCommand(newKnowledgeSearchCmd(app))
	return cmd
}

func newKnowledgeSearchCmd(app *App) *cobra.Command {
	var productID, query string
	var limit int
	var threshold float64
	var rerank, jsonOutput bool

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Semantic search across a product's knowledge graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := apiclient.KnowledgeSearchRequest{
				Query:     query,
				Limit:     limit,
				Threshold: threshold,
				Rerank:    rerank,
			}
			resp, err := app.API.SearchKnowledge(cmd.Context(), productID, body)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}
			if len(resp.Results) == 0 {
				fmt.Fprintln(w, "No results.")
				return nil
			}
			for i, r := range resp.Results {
				fmt.Fprintf(w, "[%d] score=%.2f\n", i+1, r.Score)
				fmt.Fprintf(w, "    %s\n", truncate(r.Content, 200))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&query, "query", "", "The search query")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results (1-50)")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.5, "Minimum similarity score (0.0-1.0)")
	cmd.Flags().BoolVar(&rerank, "rerank", false, "Apply Supermemory's reranker")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}
