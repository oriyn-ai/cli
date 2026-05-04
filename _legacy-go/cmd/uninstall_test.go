package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedInstall lays out a fake install on disk that mirrors what install.sh
// produces, then returns the corresponding uninstallTargets so tests can
// assert against the same paths the resolver would compute.
func seedInstall(t *testing.T) (*uninstallTargets, string) {
	t.Helper()
	root := t.TempDir()

	share := filepath.Join(root, "data", "oriyn")
	bin := filepath.Join(root, "bin")
	cfg := filepath.Join(root, "config", "oriyn")
	skill := filepath.Join(root, "claude", "skills", "oriyn")

	for _, d := range []string{share, bin, cfg, skill} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	binary := filepath.Join(share, "oriyn")
	if err := os.WriteFile(binary, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "telemetry.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	symlink := filepath.Join(bin, "oriyn")
	if err := os.Symlink(binary, symlink); err != nil {
		t.Fatal(err)
	}

	return &uninstallTargets{
		Binary:    binary,
		ShareDir:  share,
		Symlink:   symlink,
		ConfigDir: cfg,
		SkillDir:  skill,
	}, root
}

func TestRunUninstall_RemovesEverything(t *testing.T) {
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	err := runUninstall(&out, strings.NewReader(""), &uninstallOptions{Assume: true}, targets, nil)
	if err != nil {
		t.Fatalf("runUninstall: %v", err)
	}

	for _, p := range []string{targets.Binary, targets.Symlink, targets.ConfigDir, targets.SkillDir} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s to be gone, got err=%v", p, err)
		}
	}
	if _, err := os.Stat(targets.ShareDir); !os.IsNotExist(err) {
		t.Errorf("expected empty ShareDir to be cleaned up, got err=%v", err)
	}
	if !strings.Contains(out.String(), "Uninstall complete") {
		t.Errorf("output should confirm completion: %q", out.String())
	}
}

func TestRunUninstall_KeepFlagsPreserveTargets(t *testing.T) {
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	opts := &uninstallOptions{Assume: true, KeepConfig: true, KeepSkill: true, KeepBinary: true}
	if err := runUninstall(&out, strings.NewReader(""), opts, targets, nil); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}

	for _, p := range []string{targets.Binary, targets.Symlink, targets.ConfigDir, targets.SkillDir} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("expected %s to be preserved: %v", p, err)
		}
	}
	if !strings.Contains(out.String(), "(nothing") {
		t.Errorf("plan should be empty when all keep-* flags set: %q", out.String())
	}
}

func TestRunUninstall_DryRunRemovesNothing(t *testing.T) {
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	if err := runUninstall(&out, strings.NewReader(""), &uninstallOptions{DryRun: true}, targets, nil); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}

	for _, p := range []string{targets.Binary, targets.Symlink, targets.ConfigDir, targets.SkillDir} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("dry-run should leave %s in place: %v", p, err)
		}
	}
	if !strings.Contains(out.String(), "Dry run") {
		t.Errorf("output should explain dry-run: %q", out.String())
	}
}

func TestRunUninstall_NonInteractiveRequiresYes(t *testing.T) {
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	err := runUninstall(&out, strings.NewReader(""), &uninstallOptions{}, targets, nil)
	if err == nil {
		t.Fatal("expected error when neither --yes nor a TTY is available")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error should mention --yes flag: %v", err)
	}
	if _, statErr := os.Lstat(targets.Binary); statErr != nil {
		t.Errorf("binary should be untouched on rejected confirm: %v", statErr)
	}
}

func TestRunUninstall_ToleratesMissingTargets(t *testing.T) {
	dir := t.TempDir()
	targets := &uninstallTargets{
		Binary:    filepath.Join(dir, "ghost-binary"),
		ShareDir:  filepath.Join(dir, "ghost-share"),
		Symlink:   filepath.Join(dir, "ghost-symlink"),
		ConfigDir: filepath.Join(dir, "ghost-config"),
		SkillDir:  filepath.Join(dir, "ghost-skill"),
	}

	var out bytes.Buffer
	if err := runUninstall(&out, strings.NewReader(""), &uninstallOptions{Assume: true}, targets, nil); err != nil {
		t.Fatalf("missing paths should not error: %v", err)
	}
	if !strings.Contains(out.String(), "Uninstall complete") {
		t.Errorf("output should still complete: %q", out.String())
	}
}

func TestResolveUninstallTargets_HonorsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_BIN_HOME", filepath.Join(tmp, "bin"))
	t.Setenv("ORIYN_CONFIG_DIR", filepath.Join(tmp, "cfg"))

	targets, err := resolveUninstallTargets("")
	if err != nil {
		t.Fatalf("resolveUninstallTargets: %v", err)
	}

	wantBinary := filepath.Join(tmp, "data", "oriyn", "oriyn")
	wantSymlink := filepath.Join(tmp, "bin", "oriyn")
	wantConfig := filepath.Join(tmp, "cfg")
	wantSkill := filepath.Join(tmp, ".claude", "skills", "oriyn")

	if targets.Binary != wantBinary {
		t.Errorf("Binary: got %q want %q", targets.Binary, wantBinary)
	}
	if targets.Symlink != wantSymlink {
		t.Errorf("Symlink: got %q want %q", targets.Symlink, wantSymlink)
	}
	if targets.ConfigDir != wantConfig {
		t.Errorf("ConfigDir: got %q want %q", targets.ConfigDir, wantConfig)
	}
	if targets.SkillDir != wantSkill {
		t.Errorf("SkillDir: got %q want %q", targets.SkillDir, wantSkill)
	}
}

func TestResolveUninstallTargets_ExplicitSkillPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("ORIYN_CONFIG_DIR", filepath.Join(tmp, "cfg"))

	targets, err := resolveUninstallTargets("/custom/skill/path")
	if err != nil {
		t.Fatalf("resolveUninstallTargets: %v", err)
	}
	if targets.SkillDir != "/custom/skill/path" {
		t.Errorf("explicit skill path ignored: %q", targets.SkillDir)
	}
}

// withFakeTTY forces isInteractiveFn to true for the duration of t, so
// tests can exercise the confirmation prompt with an in-memory reader.
func withFakeTTY(t *testing.T) {
	t.Helper()
	original := isInteractiveFn
	isInteractiveFn = func(io.Reader) bool { return true }
	t.Cleanup(func() { isInteractiveFn = original })
}

func TestRunUninstall_ConfirmAcceptsYes(t *testing.T) {
	withFakeTTY(t)
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	if err := runUninstall(&out, strings.NewReader("y\n"), &uninstallOptions{}, targets, nil); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	if _, err := os.Lstat(targets.Binary); !os.IsNotExist(err) {
		t.Errorf("binary should be removed after y confirm: err=%v", err)
	}
}

func TestRunUninstall_ConfirmRejectsAbort(t *testing.T) {
	withFakeTTY(t)
	targets, _ := seedInstall(t)

	var out bytes.Buffer
	if err := runUninstall(&out, strings.NewReader("n\n"), &uninstallOptions{}, targets, nil); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	if _, err := os.Lstat(targets.Binary); err != nil {
		t.Errorf("binary should be untouched on rejected confirm: %v", err)
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("expected abort message: %q", out.String())
	}
}
