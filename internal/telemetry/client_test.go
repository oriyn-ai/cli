package telemetry

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewClient_DevBuildIsOff(t *testing.T) {
	withConfigDir(t)
	envScrub(t)

	c := NewClient(Options{Version: "dev"})
	defer c.Close()
	if c.Enabled() {
		t.Error("dev build should be off")
	}
}

func TestNewClient_ExplicitDisableWins(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "0")

	c := NewClient(Options{Version: "1.0.0"})
	defer c.Close()
	if c.Enabled() {
		t.Error("ORIYN_TELEMETRY=0 must turn telemetry off")
	}
}

func TestNewClient_LogModeWritesAndDoesNotSend(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "log")

	var buf bytes.Buffer
	c := NewClient(Options{Version: "1.0.0", LogWriter: &buf})
	defer c.Close()

	if c.Mode() != "log" {
		t.Errorf("Mode = %q, want log", c.Mode())
	}
	c.TrackCliCommandProducts("list")

	if !strings.Contains(buf.String(), "[telemetry]") {
		t.Errorf("log output missing prefix: %q", buf.String())
	}
	if !strings.Contains(buf.String(), `"event":"`+EventCommandStarted+`"`) {
		t.Errorf("log output missing event name: %q", buf.String())
	}
	if !strings.Contains(buf.String(), `"command":"products"`) {
		t.Errorf("log output missing command property: %q", buf.String())
	}
	if !strings.Contains(buf.String(), `"subcommand":"list"`) {
		t.Errorf("log output missing subcommand property: %q", buf.String())
	}
}

func TestNewClient_OffDoesNotEmit(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "0")

	var buf bytes.Buffer
	c := NewClient(Options{Version: "1.0.0", LogWriter: &buf})
	defer c.Close()

	c.TrackCliCommandWhoami("")
	if buf.Len() != 0 {
		t.Errorf("off mode wrote: %q", buf.String())
	}
}

func TestNewClient_PreviewForcesLogModeEvenWhenDisabled(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "0")

	var buf bytes.Buffer
	c := NewClientForPreview(Options{Version: "1.0.0", LogWriter: &buf})
	defer c.Close()

	if c.Mode() != "log" {
		t.Errorf("preview Mode = %q, want log", c.Mode())
	}
	c.TrackPreview()
	if !strings.Contains(buf.String(), `"event":"cli_preview"`) {
		t.Errorf("preview should print payload, got %q", buf.String())
	}
}

func TestClient_IdentifyPopulatesUserID(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "log")

	var buf bytes.Buffer
	c := NewClient(Options{Version: "1.0.0", LogWriter: &buf})
	defer c.Close()

	c.Identify("user-uuid", map[string]any{"plan": "pro"})
	if got := c.IdentitySnapshot().UserID; got != "user-uuid" {
		t.Errorf("UserID = %q, want user-uuid", got)
	}
	if !strings.Contains(buf.String(), `"type":"identify"`) {
		t.Errorf("identify event missing from log: %q", buf.String())
	}
}

func TestClient_ResetClearsUser(t *testing.T) {
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "log")

	c := NewClient(Options{Version: "1.0.0", LogWriter: &bytes.Buffer{}})
	defer c.Close()

	c.Identify("user-uuid", nil)
	c.Reset()
	if got := c.IdentitySnapshot().UserID; got != "" {
		t.Errorf("after Reset, UserID = %q, want empty", got)
	}
}
