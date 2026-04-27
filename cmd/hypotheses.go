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
			app.Tracker.TrackOutputCount("hypotheses", len(resp.Data))

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

	cmd.AddCommand(newHypothesesRefreshCmd(app, &productID, &jsonOutput))
	cmd.AddCommand(newBottlenecksCmd(app, &productID, &jsonOutput))
	return cmd
}

func newHypothesesRefreshCmd(app *App, productID *string, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Trigger a fresh pattern-mining pass (202 async; re-poll `hypotheses` shortly after)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.RefreshHypotheses(cmd.Context(), *productID)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if agentMode(cmd, *jsonOutput) {
				return printJSON(w, resp)
			}
			fmt.Fprintf(w, "Pattern mining started: %s\n", resp.Status)
			fmt.Fprintln(w, "Re-run `oriyn hypotheses --product-id="+*productID+"` in a few seconds to see fresh patterns.")
			return nil
		},
	}
}

func newBottlenecksCmd(app *App, productID *string, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "bottlenecks",
		Short: "List the slowest transitions between events — where users stall",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetBottlenecks(cmd.Context(), *productID)
			if err != nil {
				return err
			}
			app.Tracker.TrackOutputCount("bottlenecks", len(resp.Data))

			w := cmd.OutOrStdout()
			if agentMode(cmd, *jsonOutput) {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No bottlenecks yet — run `oriyn hypotheses refresh --product-id="+*productID+"`.")
				return nil
			}

			for _, b := range resp.Data {
				fmt.Fprintln(w, strings.Join(b.RenderedSequence, " → "))
				fmt.Fprintf(w, "  Avg transit: %s\n", formatDuration(b.AvgDurationSeconds))
				fmt.Fprintf(w, "  Users:       %d\n", b.UserCount)
				fmt.Fprintf(w, "  Traversals:  %d\n", b.Traversals)
				fmt.Fprintln(w)
			}
			return nil
		},
	}
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%.1fh", seconds/3600)
	}
	return fmt.Sprintf("%.1fd", seconds/86400)
}
