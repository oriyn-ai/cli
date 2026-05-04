package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ORIYN_CONFIG_DIR", dir)
	return dir
}

func TestLoadConfig_MissingFileReturnsEmpty(t *testing.T) {
	withConfigDir(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Enabled != nil {
		t.Errorf("Enabled should be nil for fresh install, got %v", *cfg.Enabled)
	}
	if cfg.HasDecided() {
		t.Error("HasDecided should be false on fresh install")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	withConfigDir(t)

	on := true
	cfg := &Config{
		Enabled:       &on,
		DeviceID:      "device-uuid",
		SessionID:     "session-uuid",
		SchemaVersion: CurrentSchemaVersion,
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !loaded.IsEnabled() || loaded.DeviceID != "device-uuid" {
		t.Errorf("roundtrip mismatch: %+v", loaded)
	}
}

func TestSaveConfig_Permissions(t *testing.T) {
	dir := withConfigDir(t)

	cfg := &Config{}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "telemetry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file perms = %v, want 0o600", perm)
	}
}

func TestMigrate_LegacyDisabledSentinel(t *testing.T) {
	dir := withConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "telemetry-disabled"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.IsEnabled() {
		t.Error("legacy sentinel should map to disabled")
	}
	if !cfg.HasDecided() {
		t.Error("migration should set DecidedAt")
	}
	if _, err := os.Stat(filepath.Join(dir, "telemetry-disabled")); !os.IsNotExist(err) {
		t.Error("legacy sentinel should be removed after migration")
	}
}

func TestMigrate_LegacyAnonymousID(t *testing.T) {
	dir := withConfigDir(t)
	const legacyID = "11111111-2222-3333-4444-555555555555"
	if err := os.WriteFile(filepath.Join(dir, "anonymous-id"), []byte(legacyID+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DeviceID != legacyID {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, legacyID)
	}
	if _, err := os.Stat(filepath.Join(dir, "anonymous-id")); !os.IsNotExist(err) {
		t.Error("legacy anonymous-id should be removed after migration")
	}
}

func TestMigrate_RejectsNonUUIDLegacy(t *testing.T) {
	dir := withConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "anonymous-id"), []byte("not-a-uuid"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DeviceID != "" {
		t.Errorf("non-UUID legacy ID should be discarded, got %q", cfg.DeviceID)
	}
}
