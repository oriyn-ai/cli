package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	// defaultSkillURL is the single source of truth for the Oriyn agent skill.
	// Editing happens in the marketing app's public/ dir; every install pulls
	// from here. No copy is embedded in this binary — the architectural decision
	// logged in /decisions/agent-skill-and-discover-2026-04-23.md requires one
	// editable file on Earth, not two.
	defaultSkillURL    = "https://oriyn.ai/skill.md"
	defaultSkillTarget = ".claude/skills/oriyn"
	skillFetchTimeout  = 15 * time.Second
)

func newSkillCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the Oriyn agent skill",
	}
	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillUpdateCmd())
	cmd.AddCommand(newSkillPrintCmd())
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var target, sourceURL string
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Fetch the Oriyn skill from oriyn.ai and install it into your agent",
		Long: "Downloads https://oriyn.ai/skill.md (the single source of truth) and " +
			"writes it to $HOME/.claude/skills/oriyn/SKILL.md. Use --url to install " +
			"from a different URL or a local file (for skill development). Use --path " +
			"to change the install directory. Network required — no copy ships in the " +
			"oriyn binary.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := resolveSkillTarget(target)
			if err != nil {
				return err
			}
			return installSkill(cmd.Context(), cmd.OutOrStdout(), dest, sourceURL, force)
		},
	}
	cmd.Flags().StringVar(&target, "path", "", "Destination directory (default: $HOME/.claude/skills/oriyn)")
	cmd.Flags().StringVar(&sourceURL, "url", "", "Skill source (URL or file path, default: "+defaultSkillURL+")")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing install")
	return cmd
}

func newSkillUpdateCmd() *cobra.Command {
	var target, sourceURL string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Re-fetch the skill from oriyn.ai and overwrite the installed copy",
		Long: "Equivalent to `oriyn skill install --force`. Use when the remote skill " +
			"has been updated and you want the latest version locally. Idempotent — " +
			"safe to re-run.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := resolveSkillTarget(target)
			if err != nil {
				return err
			}
			return installSkill(cmd.Context(), cmd.OutOrStdout(), dest, sourceURL, true)
		},
	}
	cmd.Flags().StringVar(&target, "path", "", "Destination directory (default: $HOME/.claude/skills/oriyn)")
	cmd.Flags().StringVar(&sourceURL, "url", "", "Skill source (URL or file path, default: "+defaultSkillURL+")")
	return cmd
}

func newSkillPrintCmd() *cobra.Command {
	var sourceURL string
	cmd := &cobra.Command{
		Use:   "print",
		Short: "Fetch the current skill and print it to stdout (for piping)",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := fetchSkill(cmd.Context(), sourceURL)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	cmd.Flags().StringVar(&sourceURL, "url", "", "Skill source (URL or file path, default: "+defaultSkillURL+")")
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

// fetchSkill returns the skill bytes from the given source — an HTTP(S) URL
// or a local file path. Empty source falls back to defaultSkillURL.
func fetchSkill(ctx context.Context, source string) ([]byte, error) {
	if source == "" {
		source = defaultSkillURL
	}

	parsed, parseErr := url.Parse(source)
	if parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return fetchSkillHTTP(ctx, source)
	}

	// Treat anything else as a local file path — covers `file://` URLs,
	// relative paths, and absolute paths.
	path := source
	if parsed != nil && parsed.Scheme == "file" {
		path = parsed.Path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading skill from %s: %w", path, err)
	}
	return data, nil
}

func fetchSkillHTTP(ctx context.Context, u string) ([]byte, error) {
	client := &http.Client{Timeout: skillFetchTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", u, err)
	}
	req.Header.Set("Accept", "text/markdown, text/plain, */*")
	req.Header.Set("User-Agent", "oriyn-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"fetching %s: %w\n\nThe Oriyn skill is fetched on install — there is no "+
				"bundled copy. Check your network, or use --url to install from a "+
				"local file",
			u, err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", u, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", u, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("fetching %s: empty response", u)
	}
	return data, nil
}

// installSkill fetches the remote skill and writes it to destDir/SKILL.md.
// When force is false and SKILL.md already exists, returns an error naming
// the file so the caller can re-run with --force.
func installSkill(ctx context.Context, out io.Writer, destDir, sourceURL string, force bool) error {
	data, err := fetchSkill(ctx, sourceURL)
	if err != nil {
		return err
	}

	//nolint:gosec // G301: skill install dir lives inside the user's project; standard 0o755 perms.
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", destDir, err)
	}

	target := filepath.Join(destDir, "SKILL.md")
	if !force {
		if _, statErr := os.Stat(target); statErr == nil {
			return fmt.Errorf("refusing to overwrite %s — re-run with --force or use `oriyn skill update`", target)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", target, statErr)
		}
	}

	//nolint:gosec // G306: skill file lives inside the user's project; standard 0o644 perms.
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}

	source := sourceURL
	if source == "" {
		source = defaultSkillURL
	}
	fmt.Fprintf(out, "Installed Oriyn skill (%d bytes) → %s\n", len(data), target)
	fmt.Fprintf(out, "  source: %s\n", source)
	if strings.HasPrefix(source, "http") {
		fmt.Fprintln(out, "Re-run `oriyn skill update` to pull the latest version.")
	}
	fmt.Fprintln(out, "Restart your agent (Claude Code / Codex) to pick up the skill.")
	return nil
}
