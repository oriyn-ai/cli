package telemetry

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CurrentSchemaVersion bumps any time the captured-fields schema gains a
// new dimension. A bump triggers re-disclosure on the next interactive run.
const CurrentSchemaVersion = 1

// Config is the on-disk telemetry state at ~/.config/oriyn/telemetry.json.
//
// `Enabled` follows the Vercel pattern: an explicit nil means "user has
// not yet been disclosed to" (default-on after disclosure); a non-nil
// pointer means the user (or a migration) has made an explicit choice.
type Config struct {
	Enabled       *bool      `json:"enabled,omitempty"`
	DeviceID      string     `json:"device_id,omitempty"`
	SessionID     string     `json:"session_id,omitempty"`
	SessionStart  *time.Time `json:"session_started_at,omitempty"`
	SessionSeen   *time.Time `json:"session_last_seen_at,omitempty"`
	DecidedAt     *time.Time `json:"decided_at,omitempty"`
	SchemaVersion int        `json:"schema_version,omitempty"`
}

// IsEnabled reports the persisted on/off state, treating an unset
// Enabled pointer as on (matches the Vercel default).
func (c *Config) IsEnabled() bool {
	if c == nil {
		return false
	}
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// HasDecided reports whether disclosure has already been shown.
func (c *Config) HasDecided() bool {
	return c != nil && c.DecidedAt != nil
}

// SchemaIsCurrent reports whether the saved schema_version matches the
// version the binary was built against. Mismatches force a re-disclosure.
func (c *Config) SchemaIsCurrent() bool {
	return c != nil && c.SchemaVersion == CurrentSchemaVersion
}

// ConfigDir returns the directory holding the telemetry config and any
// migrated legacy files.
func ConfigDir() string {
	if dir := os.Getenv("ORIYN_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "oriyn")
}

func configPath() string {
	return filepath.Join(ConfigDir(), "telemetry.json")
}

// LoadConfig reads telemetry.json. A missing file returns an empty Config
// (not an error). Legacy single-purpose files (anonymous-id, user-id,
// telemetry-disabled) are folded into the result and persisted on first
// successful load — see migrateLegacy.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(configPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// fall through with empty config so migrateLegacy can populate it
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	migrated := migrateLegacy(cfg)
	if migrated {
		_ = SaveConfig(cfg)
	}
	return cfg, nil
}

// SaveConfig writes telemetry.json with 0600 perms.
func SaveConfig(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o600)
}

// migrateLegacy folds the pre-rebuild files into the new Config struct.
// Returns true if any change was applied (caller persists).
func migrateLegacy(cfg *Config) bool {
	dir := ConfigDir()
	changed := false

	// Old sentinel: presence meant "telemetry disabled."
	if _, err := os.Stat(filepath.Join(dir, "telemetry-disabled")); err == nil {
		if cfg.Enabled == nil {
			off := false
			cfg.Enabled = &off
			now := time.Now().UTC()
			cfg.DecidedAt = &now
			changed = true
		}
		_ = os.Remove(filepath.Join(dir, "telemetry-disabled"))
	}

	// Old anonymous device ID file → DeviceID.
	if cfg.DeviceID == "" {
		if data, err := os.ReadFile(filepath.Join(dir, "anonymous-id")); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				if _, err := uuid.Parse(id); err == nil {
					cfg.DeviceID = id
					changed = true
				}
			}
			_ = os.Remove(filepath.Join(dir, "anonymous-id"))
		}
	}

	// Old user-id sidecar — not part of the new Config (we re-derive on
	// login), but remove it so stale state doesn't linger.
	_ = os.Remove(filepath.Join(dir, "user-id"))

	return changed
}
