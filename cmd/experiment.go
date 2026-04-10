package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/apiclient"
)

func newExperimentCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiment",
		Short: "Run hypothesis experiments against product personas",
	}
	cmd.AddCommand(newExperimentRunCmd(app))
	cmd.AddCommand(newExperimentListCmd(app))
	cmd.AddCommand(newExperimentGetCmd(app))
	return cmd
}

func newExperimentRunCmd(app *App) *cobra.Command {
	var product, hypothesis string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a new experiment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmd.OutOrStdout()

			created, err := app.API.CreateExperiment(ctx, product, hypothesis)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_experiment_created", map[string]interface{}{"product_id": product})

			if !jsonOutput {
				fmt.Fprintf(w, "Experiment started (%s)\n", created.ExperimentID)
				fmt.Fprintln(w, "Polling for results...")
			}

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					exp, err := app.API.GetExperiment(ctx, product, created.ExperimentID)
					if err != nil {
						return err
					}
					switch exp.Status {
					case "processing":
						if !jsonOutput {
							fmt.Fprint(w, ".")
						}
					case "failed":
						return fmt.Errorf("Experiment failed")
					case "complete":
						if jsonOutput {
							return printJSON(w, exp)
						}
						fmt.Fprintln(w)
						printResults(w, exp)
						return nil
					default:
						return fmt.Errorf("Unexpected experiment status: %s", exp.Status)
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
	}
	cmd.Flags().StringVar(&product, "product", "", "The product ID to run the experiment against")
	_ = cmd.MarkFlagRequired("product")
	cmd.Flags().StringVar(&hypothesis, "hypothesis", "", "The hypothesis to test")
	_ = cmd.MarkFlagRequired("hypothesis")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON (for agent/programmatic consumption)")
	return cmd
}

func newExperimentListCmd(app *App) *cobra.Command {
	var product string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List experiments for a product",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := app.API.ListExperiments(cmd.Context(), product)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_experiment_listed", map[string]interface{}{"product_id": product})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, items)
			}

			if len(items) == 0 {
				fmt.Fprintln(w, "No experiments found.")
				return nil
			}

			fmt.Fprintf(w, "%-38s %-12s %-10s %-30s HYPOTHESIS\n", "ID", "STATUS", "VERDICT", "RUN BY")
			for _, item := range items {
				verdict := "-"
				if item.Verdict != nil {
					verdict = *item.Verdict
				}
				hypothesis := truncate(item.Hypothesis, 50)
				fmt.Fprintf(w, "%-38s %-12s %-10s %-30s %s\n",
					item.ID, item.Status, verdict, item.CreatedByEmail, hypothesis)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&product, "product", "", "The product ID")
	_ = cmd.MarkFlagRequired("product")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func newExperimentGetCmd(app *App) *cobra.Command {
	var product, experiment string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific experiment's results",
		RunE: func(cmd *cobra.Command, args []string) error {
			exp, err := app.API.GetExperiment(cmd.Context(), product, experiment)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_experiment_viewed", map[string]interface{}{
				"product_id":    product,
				"experiment_id": experiment,
			})

			w := cmd.OutOrStdout()
			if jsonOutput {
				return printJSON(w, exp)
			}

			fmt.Fprintf(w, "Hypothesis: %s\n", exp.Hypothesis)
			fmt.Fprintf(w, "Status:     %s\n", exp.Status)
			fmt.Fprintf(w, "Run by:     %s\n", exp.CreatedByEmail)
			fmt.Fprintln(w)
			printResults(w, exp)
			return nil
		},
	}
	cmd.Flags().StringVar(&product, "product", "", "The product ID")
	_ = cmd.MarkFlagRequired("product")
	cmd.Flags().StringVar(&experiment, "experiment", "", "The experiment ID")
	_ = cmd.MarkFlagRequired("experiment")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

func printResults(w interface{ Write([]byte) (int, error) }, exp *apiclient.ExperimentResponse) {
	if exp.Summary == nil {
		fmt.Fprintln(w, "Experiment complete but no summary available.")
		return
	}
	s := exp.Summary
	fmt.Fprintf(w, "Verdict:     %s\n", colorVerdict(s.Verdict))
	fmt.Fprintf(w, "Convergence: %.0f%%\n", s.Convergence*100)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Summary:")
	fmt.Fprintln(w, s.Summary)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Persona Breakdown:")
	for _, item := range s.PersonaBreakdown {
		fmt.Fprintf(w, "  %s (%s, %.0f%% adoption): %s\n", item.Persona, item.Response, item.AdoptionRate*100, item.Reasoning)
	}
}

func colorVerdict(v string) string {
	switch v {
	case "ship":
		return color.GreenString(v)
	case "revise":
		return color.YellowString(v)
	case "reject":
		return color.RedString(v)
	default:
		return v
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
