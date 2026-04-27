package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
)

// PostHog write-only client key — safe to ship in OSS binaries; equivalent
// to a public publishable key. See posthog.com/docs/api#authentication.
//
//nolint:gosec // G101: not a credential — public PostHog client key.
const posthogAPIKey = "phc_RpuEAMMomACJxc7hG4mRMKURklt2BXtfzzwYQYlzr0W"

type Tracker struct {
	client     posthog.Client
	distinctID string
	version    string
}

func NewTracker(version string) *Tracker {
	disabled := version == "dev" || isEnvDisabled() || sentinelFileExists()

	t := &Tracker{version: version}

	if !disabled {
		client, err := posthog.NewWithConfig(posthogAPIKey, posthog.Config{})
		if err == nil {
			t.client = client
		}
	}

	if id := loadUserID(); id != "" {
		t.distinctID = id
	} else {
		t.distinctID = loadOrCreateAnonymousID()
	}

	return t
}

func (t *Tracker) Capture(event string, props map[string]interface{}) {
	if t.client == nil || t.distinctID == "" {
		return
	}
	p := posthog.NewProperties()
	p.Set("$lib", "oriyn-cli")
	p.Set("cli_version", t.version)
	p.Set("$os", runtime.GOOS)
	for k, v := range props {
		p.Set(k, v)
	}
	_ = t.client.Enqueue(posthog.Capture{
		DistinctId: t.distinctID,
		Event:      event,
		Properties: p,
	})
}

func (t *Tracker) Close() {
	if t.client != nil {
		_ = t.client.Close()
	}
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "oriyn")
}

func StoreUserID(id string) {
	dir := configDir()
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "user-id"), []byte(id), 0o600)
}

func ClearUserID() {
	_ = os.Remove(filepath.Join(configDir(), "user-id"))
}

func GetUserID() string {
	return loadUserID()
}

func loadUserID() string {
	data, err := os.ReadFile(filepath.Join(configDir(), "user-id"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func loadOrCreateAnonymousID() string {
	path := filepath.Join(configDir(), "anonymous-id")
	data, err := os.ReadFile(path)
	if err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}
	id := uuid.NewString()
	_ = os.MkdirAll(configDir(), 0o700)
	_ = os.WriteFile(path, []byte(id), 0o600)
	return id
}

func isEnvDisabled() bool {
	v := strings.ToLower(os.Getenv("ORIYN_TELEMETRY"))
	return v == "0" || v == "false" || v == "off"
}

func sentinelFileExists() bool {
	_, err := os.Stat(filepath.Join(configDir(), "telemetry-disabled"))
	return err == nil
}

func Manage(disable, enable, status bool, version string) {
	flagPath := filepath.Join(configDir(), "telemetry-disabled")

	switch {
	case disable:
		_ = os.MkdirAll(configDir(), 0o700)
		_ = os.WriteFile(flagPath, []byte(""), 0o600)
		fmt.Println("Telemetry disabled.")
	case enable:
		_ = os.Remove(flagPath)
		fmt.Println("Telemetry enabled.")
	case status:
		disabled := version == "dev" || isEnvDisabled() || sentinelFileExists()
		if disabled {
			fmt.Println("Telemetry: disabled")
		} else {
			fmt.Println("Telemetry: enabled")
		}
	}
}
