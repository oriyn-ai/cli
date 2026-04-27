package telemetry

import (
	"time"

	"github.com/google/uuid"
)

// Vercel uses 30 minutes inactive / 24 hours absolute as session bounds.
// Same numbers — same product reasoning (interactive sessions cluster
// under 30 min; nobody's CLI invocation legitimately spans a full day).
const (
	sessionInactivityTimeout = 30 * time.Minute
	sessionMaxLifetime       = 24 * time.Hour
)

// EnsureDeviceID returns the persisted device ID, creating one if missing.
// Caller is responsible for SaveConfig if a new ID was minted.
func EnsureDeviceID(cfg *Config) (id string, minted bool) {
	if cfg.DeviceID != "" {
		return cfg.DeviceID, false
	}
	cfg.DeviceID = uuid.NewString()
	return cfg.DeviceID, true
}

// EnsureSession returns the active session ID. A new session is minted
// when the prior session is too old (inactivity timeout exceeded or max
// lifetime reached). Caller persists if rotated.
func EnsureSession(cfg *Config, now time.Time) (id string, rotated bool) {
	if cfg.SessionID != "" && cfg.SessionStart != nil && cfg.SessionSeen != nil {
		fresh := now.Sub(*cfg.SessionSeen) <= sessionInactivityTimeout
		young := now.Sub(*cfg.SessionStart) <= sessionMaxLifetime
		if fresh && young {
			n := now
			cfg.SessionSeen = &n
			return cfg.SessionID, false
		}
	}
	cfg.SessionID = uuid.NewString()
	n := now
	cfg.SessionStart = &n
	cfg.SessionSeen = &n
	return cfg.SessionID, true
}

// NewInvocationID returns a UUID unique to this CLI run. Not persisted.
func NewInvocationID() string {
	return uuid.NewString()
}
