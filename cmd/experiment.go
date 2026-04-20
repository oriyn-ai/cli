package cmd

import (
	"context"
	"fmt"
	"io"
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
	cmd.AddCommand(newExperimentArchiveCmd(app))
	return cmd
}

func newExperimentRunCmd(app *App) *cobra.Command {
	var product, hypothesis string
	var agents int
	var jsonOutput, hypothesisStdin, noWait bool
	var pollInterval, timeout time.Duration

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a new experiment (blocks until complete unless --no-wait)",
		Long: "Create an experiment for the given hypothesis and wait for the " +
			"simulation to finish. Agents typically run this with --json to capture " +
			"the final verdict + persona_breakdown as structured output.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmd.OutOrStdout()

			text, err := readHypothesis(cmd, hypothesis, hypothesisStdin)
			if err != nil {
				return err
			}

			body := apiclient.CreateExperimentRequest{Hypothesis: text}
			if agents > 0 {
				body.AgentCount = &agents
			}

			created, err := app.API.CreateExperiment(ctx, product, body)
			if err != nil {
				return err
			}
			app.Tracker.Capture("cli_experiment_created", map[string]interface{}{
				"product_id":  product,
				"agent_count": agents,
			})

			agent := agentMode(cmd, jsonOutput)

			if noWait {
				if agent {
					return printJSON(w, created)
				}
				fmt.Fprintf(w, "Experiment started: %s\n", created.ExperimentID)
				fmt.Fprintf(w, "Check status: oriyn experiment get --product %s --experiment %s\n", product, created.ExperimentID)
				return nil
			}

			if !agent {
				fmt.Fprintf(w, "Experiment started (%s)\n", created.ExperimentID)
				fmt.Fprintln(w, "Polling for results...")
			}

			exp, err := pollExperiment(ctx, app, w, product, created.ExperimentID, pollInterval, timeout, agent)
			if err != nil {
				return err
			}

			if agent {
				return printJSON(w, exp)
			}
			fmt.Fprintln(w)
			printResults(w, exp)
			return nil
		},
	}
	cmd.Flags().StringVar(&product, "product", "", "The product ID to run the experiment against")
	_ = cmd.MarkFlagRequired("product")
	cmd.Flags().StringVar(&hypothesis, "hypothesis", "", "The hypothesis to test")
	cmd.Flags().BoolVar(&hypothesisStdin, "hypothesis-stdin", false, "Read hypothesis from stdin (for long proposals)")
	cmd.Flags().IntVar(&agents, "agents", 0, "Number of simulation agents (plan-limited; omit for default)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON (for agent/programmatic consumption)")
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Return immediately after creation without polling")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 2*time.Second, "How often to poll when waiting")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum time to wait for experiment completion")
	return cmd
}

// pollExperiment blocks until the experiment reaches a terminal status or the
// timeout / context fires. When not in agent mode, a dot is printed on each
// tick so humans see forward progress.
func pollExperiment(
	ctx context.Context,
	app *App,
	w io.Writer,
	productID, experimentID string,
	pollInterval, timeout time.Duration,
	agent bool,
) (*apiclient.ExperimentResponse, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("experiment %s did not complete within %s", experimentID, timeout)
			}
			exp, err := app.API.GetExperiment(ctx, productID, experimentID)
			if err != nil {
				return nil, err
			}
			switch exp.Status {
			case "processing", "queued":
				if !agent {
					fmt.Fprint(w, ".")
				}
			case "failed":
				return nil, fmt.Errorf("experiment %s failed", experimentID)
			case "complete":
				return exp, nil
			default:
				return nil, fmt.Errorf("unexpected experiment status: %s", exp.Status)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, items)
			}

			if len(items) == 0 {
				fmt.Fprintln(w, "No experiments found.")
				return nil
			}

			fmt.Fprintf(w, "%-38s %-10s %-10s %-8s %-30s HYPOTHESIS\n", "ID", "STATUS", "VERDICT", "CONV", "RUN BY")
			for _, item := range items {
				verdict := "-"
				if item.Verdict != nil {
					verdict = *item.Verdict
				}
				conv := "-"
				if item.Convergence != nil {
					conv = fmt.Sprintf("%.0f%%", *item.Convergence*100)
				}
				hypothesis := truncate(item.Hypothesis, 50)
				fmt.Fprintf(w, "%-38s %-10s %-10s %-8s %-30s %s\n",
					item.ID, item.Status, verdict, conv, item.CreatedByEmail, hypothesis)
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
			if agentMode(cmd, jsonOutput) {
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

func newExperimentArchiveCmd(app *App) *cobra.Command {
	var product, experiment string
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive a completed experiment",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := app.API.ArchiveExperiment(cmd.Context(), product, experiment)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Experiment %s: %s\n", experiment, resp.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&product, "product", "", "The product ID")
	_ = cmd.MarkFlagRequired("product")
	cmd.Flags().StringVar(&experiment, "experiment", "", "The experiment ID")
	_ = cmd.MarkFlagRequired("experiment")
	return cmd
}

func printResults(w io.Writer, exp *apiclient.ExperimentResponse) {
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
