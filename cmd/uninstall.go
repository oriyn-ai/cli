package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/telemetry"
)

// uninstallTargets is the set of paths an `oriyn` install can leave on disk.
// Computed from env (XDG_*, ORIYN_CONFIG_DIR, HOME) so the layout matches
// what install.sh produced. Each field is independent — uninstall walks them
// in order and tolerates missing entries.
type uninstallTargets struct {
	Binary    string // actual binary, e.g. ~/.local/share/oriyn/oriyn
	ShareDir  string // parent dir of Binary, removed when empty
	Symlink   string // ~/.local/bin/oriyn — pointer install.sh creates
	ConfigDir string // ~/.config/oriyn — telemetry.json, anonymous-id, etc.
	SkillDir  string // ~/.claude/skills/oriyn — agent skill drop site
}

type uninstallOptions struct {
	KeepConfig bool
	KeepSkill  bool
	KeepBinary bool
	DryRun     bool
	Assume     bool // -y / --yes
	SkillPath  string
}

func newUninstallCmd(app *App) *cobra.Command {
	opts := &uninstallOptions{}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Oriyn from this machine (credentials, config, skill, binary)",
		Long: "Reverses what `install.sh` and the CLI itself put on disk:\n" +
			"  • Clears the API token from the OS keychain (same as `oriyn logout`).\n" +
			"  • Removes the agent skill at ~/.claude/skills/oriyn (unless --keep-skill).\n" +
			"  • Removes the config dir at ~/.config/oriyn (unless --keep-config).\n" +
			"  • Removes the binary and its $PATH symlink (unless --keep-binary).\n\n" +
			"Re-run with --dry-run first to see exactly what will go.",
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolveUninstallTargets(opts.SkillPath)
			if err != nil {
				return err
			}
			return runUninstall(cmd.OutOrStdout(), cmd.InOrStdin(), opts, targets, app)
		},
	}
	cmd.Flags().BoolVar(&opts.KeepConfig, "keep-config", false, "Leave ~/.config/oriyn in place")
	cmd.Flags().BoolVar(&opts.KeepSkill, "keep-skill", false, "Leave the installed agent skill in place")
	cmd.Flags().BoolVar(&opts.KeepBinary, "keep-binary", false, "Leave the binary and PATH symlink in place")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Print what would be removed and exit")
	cmd.Flags().BoolVarP(&opts.Assume, "yes", "y", false, "Skip the confirmation prompt")
	cmd.Flags().StringVar(&opts.SkillPath, "skill-path", "", "Skill directory to remove (default: $HOME/.claude/skills/oriyn)")
	return cmd
}

func resolveUninstallTargets(skillPath string) (*uninstallTargets, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home dir: %w", err)
	}

	shareDir := envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	binDir := envOr("XDG_BIN_HOME", filepath.Join(home, ".local", "bin"))
	configDir := telemetry.ConfigDir()

	skill := skillPath
	if skill == "" {
		skill = filepath.Join(home, defaultSkillTarget)
	}

	return &uninstallTargets{
		Binary:    filepath.Join(shareDir, "oriyn", "oriyn"),
		ShareDir:  filepath.Join(shareDir, "oriyn"),
		Symlink:   filepath.Join(binDir, "oriyn"),
		ConfigDir: configDir,
		SkillDir:  skill,
	}, nil
}

// runUninstall is the testable core. It prints the plan, optionally confirms,
// and then removes each target. Errors on individual targets are reported but
// do not abort the rest — partial uninstalls are better than none.
func runUninstall(out io.Writer, in io.Reader, opts *uninstallOptions, t *uninstallTargets, app *App) error {
	plan := buildUninstallPlan(opts, t)

	fmt.Fprintln(out, "This will remove:")
	for _, p := range plan {
		fmt.Fprintf(out, "  %s  %s\n", p.label, p.path)
	}
	if len(plan) == 0 {
		fmt.Fprintln(out, "  (nothing — every target was skipped via --keep-*)")
	}
	fmt.Fprintln(out)

	if opts.DryRun {
		fmt.Fprintln(out, "Dry run — nothing was removed. Re-run without --dry-run to apply.")
		return nil
	}

	if !opts.Assume {
		if !isInteractiveFn(in) {
			return errors.New("uninstall requires confirmation; pass --yes to run non-interactively")
		}
		fmt.Fprint(out, "Continue? [y/N] ")
		reader := bufio.NewReader(in)
		line, _ := reader.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans != "y" && ans != "yes" {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	// Logout first so the keychain entry is cleared even if a later step
	// (binary removal) fails. Auth Delete already swallows missing entries.
	if app != nil && app.AuthStore != nil {
		_ = app.AuthStore.Delete()
		if app.Tracker != nil {
			app.Tracker.Reset()
		}
		fmt.Fprintln(out, "✓ cleared keychain entry (oriyn-cli)")
	}

	for _, p := range plan {
		if err := removePath(p.path); err != nil {
			fmt.Fprintf(out, "! %s  %s — %v\n", p.label, p.path, err)
			continue
		}
		fmt.Fprintf(out, "✓ %s  %s\n", p.label, p.path)
	}

	// If the share dir is now empty (the binary was the only thing in it),
	// drop the empty parent too. Best-effort; ignore non-empty / missing.
	if !opts.KeepBinary {
		_ = os.Remove(t.ShareDir)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Uninstall complete. Thanks for trying Oriyn.")
	return nil
}

type planItem struct {
	label string // short tag printed before the path
	path  string
}

func buildUninstallPlan(opts *uninstallOptions, t *uninstallTargets) []planItem {
	var plan []planItem
	if !opts.KeepSkill {
		plan = append(plan, planItem{"skill   ", t.SkillDir})
	}
	if !opts.KeepConfig {
		plan = append(plan, planItem{"config  ", t.ConfigDir})
	}
	if !opts.KeepBinary {
		plan = append(plan, planItem{"symlink ", t.Symlink})
		plan = append(plan, planItem{"binary  ", t.Binary})
	}
	return plan
}

// removePath removes a file, symlink, or directory tree. Missing paths are
// not an error — the goal is to leave the system in the post-uninstall state,
// not to report on what was already gone.
func removePath(path string) error {
	if path == "" {
		return nil
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// isInteractiveFn is overridable so tests can simulate a TTY without
// allocating a real pty. Production code uses the os.File char-device check.
var isInteractiveFn = func(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
