package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTimelineCmd(app *App) *cobra.Command {
	var productID, userID, outputPath string
	var limit int
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Cross-provider timeline (events + replays + revenue) for one resolved user",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetUserTimeline(cmd.Context(), productID, userID, limit)
			if err != nil {
				return err
			}
			app.Tracker.TrackOutputCount("timeline", len(resp.Items))
			w := cmd.OutOrStdout()

			if outputPath != "" {
				data, err := json.Marshal(resp)
				if err != nil {
					return fmt.Errorf("serializing timeline: %w", err)
				}
				//nolint:gosec // G306: user-supplied --output path; standard 0o644 file perms.
				if err := os.WriteFile(outputPath, data, 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", outputPath, err)
				}
				fmt.Fprintf(w, "Wrote %d timeline items to %s\n", len(resp.Items), outputPath)
				return nil
			}

			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}
			if len(resp.Items) == 0 {
				fmt.Fprintln(w, "No events for this user.")
				return nil
			}
			for _, item := range resp.Items {
				fmt.Fprintf(w, "%s [%s/%s] %s\n", item.Timestamp, item.Provider, item.Kind, item.EventName)
				if item.SessionSummary != nil && *item.SessionSummary != "" {
					fmt.Fprintf(w, "  %s\n", truncate(*item.SessionSummary, 160))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&userID, "user-id", "", "The resolved user ID (UUID in the identity layer)")
	_ = cmd.MarkFlagRequired("user-id")
	cmd.Flags().IntVar(&limit, "limit", 60, "Maximum events to return (1-500)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Write the full JSON response to this file instead of stdout")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON to stdout")
	return cmd
}
