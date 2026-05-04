package telemetry

import (
	"testing"
	"time"
)

func TestEnsureDeviceID_MintsOnce(t *testing.T) {
	cfg := &Config{}
	id1, minted := EnsureDeviceID(cfg)
	if !minted || id1 == "" {
		t.Fatal("first call should mint")
	}
	id2, minted := EnsureDeviceID(cfg)
	if minted || id2 != id1 {
		t.Errorf("second call should be a no-op; got minted=%v id=%q", minted, id2)
	}
}

func TestEnsureSession_RotatesAfterInactivity(t *testing.T) {
	cfg := &Config{}
	t0 := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	id1, rotated := EnsureSession(cfg, t0)
	if !rotated || id1 == "" {
		t.Fatal("first call should mint a session")
	}

	id2, rotated := EnsureSession(cfg, t0.Add(15*time.Minute))
	if rotated || id2 != id1 {
		t.Error("within inactivity timeout, should reuse")
	}

	// SessionSeen got bumped to t0+15min by the previous call. Push
	// past the 30-minute inactivity window from that point.
	id3, rotated := EnsureSession(cfg, t0.Add(50*time.Minute))
	if !rotated || id3 == id1 {
		t.Error("past inactivity timeout, should rotate")
	}
}

func TestEnsureSession_RotatesAfterMaxLifetime(t *testing.T) {
	cfg := &Config{}
	t0 := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	id1, _ := EnsureSession(cfg, t0)
	// Stay within inactivity timeout but blow past 24h max lifetime.
	id2, rotated := EnsureSession(cfg, t0.Add(25*time.Hour))
	if !rotated || id2 == id1 {
		t.Error("past max lifetime, session should rotate")
	}
}
