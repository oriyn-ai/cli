package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
)

// newInitCmd is the one-shot onboarding entry — modeled on Firecrawl's
// `init --all --browser`. It runs the install / auth / skill-drop / health-check
// sequence a coding agent (or its human operator) needs to do exactly once.
// Safe to re-run: each step checks whether it's already done.
func newInitCmd(app *App) *cobra.Command {
	var skipLogin, skipSkill, skipDoctor, force, noBrowser bool
	var skillPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "One-shot setup: authenticate, install the agent skill, verify the API",
		Long: "Runs login (if needed), installs the Oriyn skill into ~/.claude/skills/oriyn, " +
			"and then runs doctor to confirm everything works. Safe to re-run — each " +
			"step short-circuits if already complete.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmd.OutOrStdout()

			if !skipLogin {
				if err := ensureLoggedIn(ctx, app, w, noBrowser); err != nil {
					return err
				}
			}

			if !skipSkill {
				dest, err := resolveSkillTarget(skillPath)
				if err != nil {
					return err
				}
				if err := installEmbeddedSkill(w, dest, force); err != nil {
					return err
				}
			}

			if !skipDoctor {
				report := runDoctorChecks(ctx, app, "", "")
				for _, r := range report.Checks {
					marker := "ok"
					if !r.OK {
						marker = "FAIL"
					}
					fmt.Fprintf(w, "[doctor] %-16s %s — %s\n", r.Name, marker, r.Detail)
				}
				if !report.OK {
					return fmt.Errorf("one or more doctor checks failed")
				}
			}

			fmt.Fprintln(w)
			fmt.Fprintln(w, "Oriyn is ready. Try `oriyn products list`.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&skipLogin, "skip-login", false, "Skip the login step (use when ORIYN_ACCESS_TOKEN is set)")
	cmd.Flags().BoolVar(&skipSkill, "skip-skill", false, "Skip installing the agent skill")
	cmd.Flags().BoolVar(&skipDoctor, "skip-doctor", false, "Skip the final health check")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing skill install")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the login URL instead of opening a browser")
	cmd.Flags().StringVar(&skillPath, "skill-path", "", "Destination directory for the skill (default: $HOME/.claude/skills/oriyn)")
	return cmd
}

// ensureLoggedIn short-circuits if GetValidAccessToken succeeds (either via a
// valid keychain entry or an ORIYN_ACCESS_TOKEN env var). Only the "not logged
// in" / "expired" cases trigger an interactive login.
func ensureLoggedIn(ctx context.Context, app *App, w io.Writer, noBrowser bool) error {
	if _, err := app.AuthStore.GetValidAccessToken(ctx); err == nil {
		fmt.Fprintln(w, "[login] already authenticated")
		return nil
	} else if !errors.Is(err, auth.ErrNotLoggedIn) && !errors.Is(err, auth.ErrSessionExpired) {
		return err
	}
	fmt.Fprintln(w, "[login] starting browser OAuth...")
	return runLogin(ctx, app.WebBase, app.APIBase, app.AuthStore, noBrowser, w)
}
