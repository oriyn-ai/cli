package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newReplayCmd(app *App) *cobra.Command {
	var productID, sessionAssetID, outputPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Fetch raw rrweb events for a stored session asset",
		Long: "By default prints the rrweb event count. Pass --json to stream the " +
			"raw payload to stdout, or --output FILE to write it to disk — the " +
			"latter is recommended for agents since a single session can be megabytes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetSessionReplay(cmd.Context(), productID, sessionAssetID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			if outputPath != "" {
				data, err := json.Marshal(resp)
				if err != nil {
					return fmt.Errorf("serializing replay: %w", err)
				}
				//nolint:gosec // G306: user-supplied --output path; standard 0o644 file perms.
				if err := os.WriteFile(outputPath, data, 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", outputPath, err)
				}
				fmt.Fprintf(w, "Wrote %d rrweb events to %s\n", len(resp.Events), outputPath)
				return nil
			}

			if jsonOutput || agentMode(cmd, false) {
				return printJSON(w, resp)
			}
			fmt.Fprintf(w, "%d rrweb events — pipe to --json or --output FILE for the raw payload.\n", len(resp.Events))
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&sessionAssetID, "session-asset-id", "", "The session asset ID (from citations or timeline)")
	_ = cmd.MarkFlagRequired("session-asset-id")
	cmd.Flags().StringVar(&outputPath, "output", "", "Write the rrweb JSON to this file instead of stdout")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw rrweb JSON events to stdout")
	return cmd
}
