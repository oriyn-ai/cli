package telemetry

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oriyn-ai/cli/internal/apiclient"
)

func newLogClient(t *testing.T) (*Client, *bytes.Buffer) {
	t.Helper()
	withConfigDir(t)
	envScrub(t)
	t.Setenv("ORIYN_TELEMETRY", "log")

	var buf bytes.Buffer
	c := NewClient(Options{Version: "1.0.0", LogWriter: &buf})
	t.Cleanup(c.Close)
	return c, &buf
}

func TestTrackCommand_EmitsStartedEvent(t *testing.T) {
	c, buf := newLogClient(t)
	c.TrackCommand("products list")

	out := buf.String()
	if !strings.Contains(out, `"event":"`+EventCommandStarted+`"`) {
		t.Errorf("missing started event: %q", out)
	}
	if !strings.Contains(out, `"command":"products list"`) {
		t.Errorf("missing command property: %q", out)
	}
}

func TestTrackCommandComplete_SuccessShape(t *testing.T) {
	c, buf := newLogClient(t)
	c.TrackCommandComplete("login", 250*time.Millisecond, nil)

	out := buf.String()
	if !strings.Contains(out, `"event":"`+EventCommandCompleted+`"`) {
		t.Errorf("missing completed event: %q", out)
	}
	if !strings.Contains(out, `"outcome":"success"`) {
		t.Errorf("missing success outcome: %q", out)
	}
	if !strings.Contains(out, `"duration_bucket":"<500ms"`) {
		t.Errorf("missing duration bucket: %q", out)
	}
	// Success path must not carry any error_* properties.
	if strings.Contains(out, "error_") {
		t.Errorf("success event leaked error_* property: %q", out)
	}
}

func TestTrackCommandComplete_APIErrorEnrichesPayload(t *testing.T) {
	c, buf := newLogClient(t)

	credits := 50
	apiErr := &apiclient.APIError{
		StatusCode:      402,
		Message:         "credits exhausted",
		Plan:            "starter",
		CreditsRequired: &credits,
	}
	c.TrackCommandComplete("experiments run", time.Second, apiErr)

	out := buf.String()
	if !strings.Contains(out, `"outcome":"api_error"`) {
		t.Errorf("missing api_error outcome: %q", out)
	}
	if !strings.Contains(out, `"error_status":402`) {
		t.Errorf("missing status: %q", out)
	}
	if !strings.Contains(out, `"error_plan":"starter"`) {
		t.Errorf("missing plan: %q", out)
	}
	if !strings.Contains(out, `"error_has_credits_required":true`) {
		t.Errorf("missing credits flag: %q", out)
	}
	if !strings.Contains(out, `"error_server_message":"credits exhausted"`) {
		t.Errorf("missing server message: %q", out)
	}
}

func TestTrackCommandComplete_PermissionErrorPayload(t *testing.T) {
	c, buf := newLogClient(t)

	permErr := &apiclient.PermissionError{
		StatusCode:         403,
		RequiredPermission: "experiments.run",
		Role:               "viewer",
		OrgID:              "org_xyz",
	}
	c.TrackCommandComplete("experiments run", time.Second, permErr)

	out := buf.String()
	if !strings.Contains(out, `"outcome":"permission_error"`) {
		t.Errorf("missing outcome: %q", out)
	}
	if !strings.Contains(out, `"error_required_permission":"experiments.run"`) {
		t.Errorf("missing required permission: %q", out)
	}
	if !strings.Contains(out, `"error_role":"viewer"`) {
		t.Errorf("missing role: %q", out)
	}
	if !strings.Contains(out, `"error_has_org_id":true`) {
		t.Errorf("missing org id flag: %q", out)
	}
	if strings.Contains(out, "org_xyz") {
		t.Errorf("org id leaked into payload: %q", out)
	}
}

func TestTrackCommandComplete_RawErrorNeverShipped(t *testing.T) {
	c, buf := newLogClient(t)
	c.TrackCommandComplete("doctor", 50*time.Millisecond,
		errors.New("dial tcp /Users/shivam/private/path: connection refused"))

	out := buf.String()
	if strings.Contains(out, "/Users/") || strings.Contains(out, "private") {
		t.Errorf("raw error leaked into payload: %q", out)
	}
	if !strings.Contains(out, `"outcome":"network_error"`) {
		t.Errorf("missing classified outcome: %q", out)
	}
}

func TestTrackLoginState_EmitsEvent(t *testing.T) {
	c, buf := newLogClient(t)
	c.TrackLoginState(LoginStateBrowserOpened)

	out := buf.String()
	if !strings.Contains(out, `"event":"`+EventLoginStateChanged+`"`) {
		t.Errorf("missing login state event: %q", out)
	}
	if !strings.Contains(out, `"login_state":"browser_opened"`) {
		t.Errorf("missing login_state property: %q", out)
	}
}

func TestTrackOutputCount_EmitsBucket(t *testing.T) {
	c, buf := newLogClient(t)
	c.TrackOutputCount("products list", 5)

	out := buf.String()
	if !strings.Contains(out, `"count_bucket":"few"`) {
		t.Errorf("missing count bucket: %q", out)
	}
	if strings.Contains(out, `"count":5`) {
		t.Errorf("raw count should not leak: %q", out)
	}
}

func TestAgentEnvVarSetsIsAgentProperty(t *testing.T) {
	c, buf := newLogClient(t)
	t.Setenv("ORIYN_AGENT", "claude-code")
	c.Capture("test", nil)

	out := buf.String()
	if !strings.Contains(out, `"is_agent":true`) {
		t.Errorf("missing is_agent property: %q", out)
	}
	if strings.Contains(out, "claude-code") {
		t.Errorf("agent name leaked into payload: %q", out)
	}
}
