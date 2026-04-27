package cmd

import (
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
			app.Tracker.TrackOutputCount("personas", len(resp.Data))

			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}

			fmt.Fprintf(w, "Enrichment status: %s\n\n", resp.EnrichmentStatus)

			if len(resp.Data) == 0 {
				fmt.Fprintln(w, "No personas found.")
				return nil
			}

			for _, persona := range resp.Data {
				fmt.Fprintf(w, "%s (~%d%% of users) [%s]\n", persona.Name, persona.SizeEstimate, persona.ID)
				fmt.Fprintf(w, "  %s\n", persona.Description)
				for _, t := range persona.BehavioralTraits {
					fmt.Fprintf(w, "  - %s\n", t)
				}
				fmt.Fprintln(w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")

	cmd.AddCommand(newPersonaProfileCmd(app))
	cmd.AddCommand(newPersonaCitationsCmd(app))
	return cmd
}

func newPersonaProfileCmd(app *App) *cobra.Command {
	var productID, personaID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Show the Supermemory-derived profile for a persona",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetPersonaProfile(cmd.Context(), productID, personaID)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}
			if len(resp.StaticFacts) > 0 {
				fmt.Fprintln(w, "Static facts (long-term identity):")
				for _, f := range resp.StaticFacts {
					fmt.Fprintf(w, "  - %s\n", f)
				}
			}
			if len(resp.DynamicFacts) > 0 {
				fmt.Fprintln(w, "Dynamic facts (recent simulation context):")
				for _, f := range resp.DynamicFacts {
					fmt.Fprintf(w, "  - %s\n", f)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&personaID, "persona-id", "", "The persona ID")
	_ = cmd.MarkFlagRequired("persona-id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newPersonaCitationsCmd(app *App) *cobra.Command {
	var productID, personaID string
	var traitIndex int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "citations",
		Short: "List evidence sessions behind a persona's behavioral trait",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.GetPersonaCitations(cmd.Context(), productID, personaID, traitIndex)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, resp)
			}
			if len(resp.Citations) == 0 {
				fmt.Fprintln(w, "No citations found for this trait.")
				return nil
			}
			for _, c := range resp.Citations {
				fmt.Fprintf(w, "Session %s (frustration %.2f, %dms)\n", c.ExternalSessionID, c.FrustrationScore, c.DurationMS)
				if c.SessionSummary != "" {
					fmt.Fprintf(w, "  %s\n", c.SessionSummary)
				}
				if c.ReplayURL != nil {
					fmt.Fprintf(w, "  Replay: %s\n", *c.ReplayURL)
				}
				fmt.Fprintln(w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&productID, "product-id", "", "The product ID")
	_ = cmd.MarkFlagRequired("product-id")
	cmd.Flags().StringVar(&personaID, "persona-id", "", "The persona ID")
	_ = cmd.MarkFlagRequired("persona-id")
	cmd.Flags().IntVar(&traitIndex, "trait-index", 0, "Zero-based index into the persona's behavioral_traits array")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}
