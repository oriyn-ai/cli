package cmd

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	embeddedSkillRoot  = "skills/oriyn"
	defaultSkillTarget = ".claude/skills/oriyn"
)

// skillFS is set from main() via cmd.Execute — see root.go.
var skillFS embed.FS

func newSkillCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the Oriyn agent skill (install into Claude Code / Codex)",
	}
	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillPrintCmd())
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var target string
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Copy the embedded Oriyn skill files into ~/.claude/skills/oriyn (or --path)",
		Long: "The Oriyn skill teaches coding agents when to run experiments and " +
			"how to read the verdicts. This command lays the skill files down " +
			"onto disk — SKILL.md plus references/ — at the given path (default: " +
			"$HOME/.claude/skills/oriyn). The files are embedded in the oriyn binary, " +
			"so no network access is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := resolveSkillTarget(target)
			if err != nil {
				return err
			}
			return installEmbeddedSkill(cmd.OutOrStdout(), dest, force)
		},
	}
	cmd.Flags().StringVar(&target, "path", "", "Destination directory (default: $HOME/.claude/skills/oriyn)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files at the target")
	return cmd
}

func newSkillPrintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print",
		Short: "Print the SKILL.md content to stdout (for piping into other tools)",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := skillFS.ReadFile(embeddedSkillRoot + "/SKILL.md")
			if err != nil {
				return fmt.Errorf("reading embedded SKILL.md: %w", err)
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	return cmd
}

func resolveSkillTarget(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, defaultSkillTarget), nil
}

// installEmbeddedSkill unpacks the embedded skill tree into destDir. When
// force is false and files already exist, it returns an error identifying
// the first conflict so the agent can decide whether to re-run with --force.
func installEmbeddedSkill(out io.Writer, destDir string, force bool) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", destDir, err)
	}

	written := 0
	err := fs.WalkDir(skillFS, embeddedSkillRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(path, embeddedSkillRoot)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		if _, err := os.Stat(target); err == nil && !force {
			return fmt.Errorf("refusing to overwrite %s — re-run with --force", target)
		}

		data, err := skillFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
		written++
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Installed Oriyn skill (%d files) → %s\n", written, destDir)
	fmt.Fprintln(out, "Restart your agent (Claude Code / Codex) to pick up the skill.")
	return nil
}
