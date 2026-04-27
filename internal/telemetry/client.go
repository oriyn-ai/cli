// Package telemetry implements oriyn's CLI usage telemetry.
//
// Architecture follows the Vercel CLI pattern (see docs at
// https://oriyn.ai/telemetry):
//
//   - Default-on after a one-time first-run disclosure.
//   - Honors ORIYN_TELEMETRY=0|false|off and DO_NOT_TRACK=1.
//   - CI runs auto-skip unless ORIYN_TELEMETRY=1 forces capture.
//   - Async best-effort flush on shutdown; never blocks exit code.
//   - Allowlist event surface: arbitrary maps are not exported. Add a
//     typed method to Client for any new dimension you want to capture,
//     and document it on the transparency page in the same change.
package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/posthog/posthog-go"
)

// Public PostHog client key — equivalent to a publishable key. Safe to
// ship in OSS binaries; PostHog's project security model treats this as
// a write-only ingestion identifier.
//
//nolint:gosec // G101: not a credential.
const posthogProjectKey = "phc_RpuEAMMomACJxc7hG4mRMKURklt2BXtfzzwYQYlzr0W"

// posthogEndpoint is the reverse proxy that fronts PostHog ingestion,
// matching the convention used by the web app and api. Keeping CLI
// traffic on the same host avoids surprising users who allowlist
// e.oriyn.ai but block i.posthog.com.
const posthogEndpoint = "https://e.oriyn.ai"

// flushDeadline bounds shutdown — telemetry must never make the user
// wait. 2s matches Vercel's subprocess-kill timeout.
const flushDeadline = 2 * time.Second

// Client owns the lifecycle of a single CLI invocation's telemetry.
// The zero value is a no-op; obtain a real one via NewClient.
type Client struct {
	store    *Store
	posthog  posthog.Client
	identity Identity
	version  string
	mode     mode
}

// Identity holds the IDs attached to every event. UserID is empty until
// after a successful login.
type Identity struct {
	DeviceID     string
	SessionID    string
	InvocationID string
	UserID       string
}

type mode int

const (
	modeOff     mode = iota // not initialized; Capture is a no-op
	modeLive                // send to PostHog
	modeLog                 // print events to writer; do not send
	modeDevOnly             // dev/test build; never sends
)

// Options configures a Client.
type Options struct {
	Version string

	// LogWriter receives event JSON when ORIYN_TELEMETRY=log. Defaults
	// to os.Stderr when nil.
	LogWriter io.Writer
}

// NewClientForPreview returns a Client forced into log mode regardless
// of env or saved config. Used by `oriyn telemetry preview` so users
// can inspect what would be sent even when telemetry is currently off.
func NewClientForPreview(opts Options) *Client {
	c := newClientCommon(opts)
	if c.posthog != nil {
		_ = c.posthog.Close()
		c.posthog = nil
	}
	c.mode = modeLog
	if c.identity.DeviceID == "" {
		c.identity = Identity{
			DeviceID:     "preview-" + NewInvocationID(),
			SessionID:    NewInvocationID(),
			InvocationID: NewInvocationID(),
		}
	}
	return c
}

// NewClient initializes telemetry for the current CLI invocation.
// It loads the on-disk config (running migrations from legacy files
// if needed), checks env opt-outs, and prepares — but does not yet
// call — the PostHog client. The first call to Capture starts the
// flush goroutine implicitly via posthog-go's batcher.
//
// The disclosure notice is *not* printed by NewClient. Call
// CheckDisclosure separately from your command's PreRun so the
// notice text reaches stderr only when intended.
func NewClient(opts Options) *Client {
	return newClientCommon(opts)
}

func newClientCommon(opts Options) *Client {
	c := &Client{version: opts.Version}

	cfg, err := LoadConfig()
	if err != nil || cfg == nil {
		cfg = &Config{}
	}

	env := ReadEnv()

	switch {
	case opts.Version == "" || opts.Version == "dev":
		c.mode = modeDevOnly
	case env.ExplicitlyDisabled:
		c.mode = modeOff
	case env.CIAutoSkip():
		c.mode = modeOff
	case cfg.Enabled != nil && !*cfg.Enabled:
		c.mode = modeOff
	case env.LogMode:
		c.mode = modeLog
	default:
		c.mode = modeLive
	}

	logOut := opts.LogWriter
	if logOut == nil {
		logOut = os.Stderr
	}
	c.store = newStore(logOut)

	// Only modeLive persists identity to disk. modeLog (env-driven
	// preview) gets transient IDs so log-mode runs never mutate config.
	// modeOff and modeDevOnly stay completely identity-free — running
	// `oriyn telemetry status` after `disable` must not silently mint
	// a fresh device ID just because the tracker was constructed.
	switch c.mode {
	case modeLive:
		_, mintedDevice := EnsureDeviceID(cfg)
		_, rotatedSession := EnsureSession(cfg, time.Now().UTC())
		if mintedDevice || rotatedSession {
			_ = SaveConfig(cfg)
		}
		c.identity = Identity{
			DeviceID:     cfg.DeviceID,
			SessionID:    cfg.SessionID,
			InvocationID: NewInvocationID(),
		}
	case modeLog:
		c.identity = Identity{
			DeviceID:     "preview-" + NewInvocationID(),
			SessionID:    NewInvocationID(),
			InvocationID: NewInvocationID(),
		}
	}

	if c.mode == modeLive {
		ph, err := posthog.NewWithConfig(posthogProjectKey, posthog.Config{
			Endpoint: posthogEndpoint,
			Interval: 5 * time.Second,
			Verbose:  false,
		})
		if err == nil {
			c.posthog = ph
		} else {
			c.mode = modeOff
		}
	}

	return c
}

// Enabled reports whether events from Capture will be sent or logged.
func (c *Client) Enabled() bool {
	return c != nil && (c.mode == modeLive || c.mode == modeLog)
}

// Identify links the current device to an authenticated Supabase user.
// Call from the login command after credentials are saved.
func (c *Client) Identify(userID string, props map[string]any) {
	if c == nil || !c.Enabled() || userID == "" {
		return
	}
	c.identity.UserID = userID

	p := posthog.NewProperties()
	for k, v := range props {
		p.Set(k, v)
	}

	switch c.mode {
	case modeLive:
		_ = c.posthog.Enqueue(posthog.Identify{
			DistinctId: userID,
			Properties: p,
		})
	case modeLog:
		c.store.log(map[string]any{
			"type":        "identify",
			"distinct_id": userID,
			"properties":  props,
		})
	}
}

// Reset detaches the current invocation from any user identity. Call
// from logout. Future events fall back to the device ID.
func (c *Client) Reset() {
	if c == nil {
		return
	}
	c.identity.UserID = ""
}

// capture emits a single event. Unexported on purpose: every external
// call site must go through a typed Track* method so adding any new
// captured dimension requires editing this file (which is code-review
// gated). Mirrors Vercel's protected track() pattern.
func (c *Client) capture(event string, props map[string]any) {
	if c == nil || !c.Enabled() || event == "" {
		return
	}

	distinct := c.identity.UserID
	if distinct == "" {
		distinct = c.identity.DeviceID
	}
	if distinct == "" {
		return
	}

	enriched := c.baseProperties()
	for k, v := range props {
		enriched[k] = v
	}

	switch c.mode {
	case modeLive:
		p := posthog.NewProperties()
		for k, v := range enriched {
			p.Set(k, v)
		}
		_ = c.posthog.Enqueue(posthog.Capture{
			DistinctId: distinct,
			Event:      event,
			Properties: p,
		})
	case modeLog:
		c.store.log(map[string]any{
			"type":        "capture",
			"distinct_id": distinct,
			"event":       event,
			"properties":  enriched,
		})
	}
}

// Close flushes pending events with a hard deadline. Telemetry
// failures must never affect the CLI's exit code or visible output.
func (c *Client) Close() {
	if c == nil {
		return
	}
	if c.posthog == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		_ = c.posthog.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(flushDeadline):
	}
}

// Identity returns a copy of the IDs in use this invocation. Read by
// the telemetry status subcommand.
func (c *Client) IdentitySnapshot() Identity {
	if c == nil {
		return Identity{}
	}
	return c.identity
}

// Mode reports the runtime mode as a stable string for status display.
func (c *Client) Mode() string {
	if c == nil {
		return "off"
	}
	switch c.mode {
	case modeLive:
		return "on"
	case modeLog:
		return "log"
	case modeDevOnly:
		return "off (dev build)"
	default:
		return "off"
	}
}

// baseProperties are attached to every captured event. Keep this list
// in lockstep with the transparency page.
func (c *Client) baseProperties() map[string]any {
	props := map[string]any{
		"$lib":          "oriyn-cli",
		"cli_version":   c.version,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"device_id":     c.identity.DeviceID,
		"session_id":    c.identity.SessionID,
		"invocation_id": c.identity.InvocationID,
	}
	if agent := os.Getenv("ORIYN_AGENT"); agent != "" {
		// Captured as boolean to avoid the agent string itself being
		// a fingerprintable identifier of the user's setup.
		props["is_agent"] = true
	}
	return props
}

// TrackCommand records the start of a CLI command invocation. Pair
// each call with TrackCommandComplete on the same invocation; the
// pair is joined server-side via invocation_id from baseProperties.
func (c *Client) TrackCommand(name string) {
	if c == nil || !c.Enabled() || name == "" {
		return
	}
	c.capture(EventCommandStarted, map[string]any{
		"command": name,
	})
}

// TrackCommandComplete records the outcome of a CLI command. Safe to
// call with err == nil for the success path. The raw err.Error() is
// classified into a small enum + fixed-vocabulary fields; nothing
// pulled from err.Error() crosses the wire as a free-form string.
func (c *Client) TrackCommandComplete(name string, duration time.Duration, err error) {
	if c == nil || !c.Enabled() || name == "" {
		return
	}
	info := ClassifyError(err)
	props := map[string]any{
		"command":         name,
		"duration_bucket": string(BucketDuration(duration)),
		"outcome":         string(info.Outcome),
	}
	if info.Status != 0 {
		props["error_status"] = info.Status
	}
	if info.ServerMessage != "" {
		props["error_server_message"] = info.ServerMessage
	}
	if info.Plan != "" {
		props["error_plan"] = info.Plan
	}
	if info.HasCreditsRequired {
		props["error_has_credits_required"] = true
	}
	if info.HasMaxAgentCount {
		props["error_has_max_agent_count"] = true
	}
	if info.RequiredPermission != "" {
		props["error_required_permission"] = info.RequiredPermission
	}
	if info.Role != "" {
		props["error_role"] = info.Role
	}
	if info.HasOrgID {
		props["error_has_org_id"] = true
	}
	c.capture(EventCommandCompleted, props)
}

// TrackLoginState records each phase of the login funnel. Surfaces
// drop-off points: e.g. "browser_opened" → no "callback_received"
// means a browser-side failure; "callback_received" → no
// "profile_fetched" means an oriyn-api failure.
func (c *Client) TrackLoginState(state LoginState) {
	if c == nil || !c.Enabled() || state == "" {
		return
	}
	c.capture(EventLoginStateChanged, map[string]any{
		"login_state": string(state),
	})
}

// TrackOutputCount records the size of a list-shape command's result
// set, bucketed to avoid leaking exact counts (which can be PII for
// small organizations: "this org has 1 product"). Optional — call
// from list commands only.
func (c *Client) TrackOutputCount(command string, count int) {
	if c == nil || !c.Enabled() || command == "" {
		return
	}
	c.capture(EventOutputCount, map[string]any{
		"command":      command,
		"count_bucket": string(BucketCount(count)),
	})
}

// TrackFlag records that a boolean flag was set on a command. Mirrors
// Vercel's trackCliFlag — captures only the flag name, never a value.
// The flag name is part of the CLI's public API surface (visible in
// --help) so it's safe to ship as-is.
func (c *Client) TrackFlag(command, flag string) {
	if c == nil || !c.Enabled() || command == "" || flag == "" {
		return
	}
	c.capture("cli_flag_set", map[string]any{
		"command": command,
		"flag":    flag,
	})
}

// TrackArgumentCount records that a positional or repeatable argument
// was supplied to a command, bucketed to ZERO/ONE/FEW/MANY/HUGE.
// Mirrors Vercel's redactedArgumentsLength: the count itself can be
// fingerprinting for small organizations, so we bucket at the call
// site and never ship the value.
func (c *Client) TrackArgumentCount(command, arg string, count int) {
	if c == nil || !c.Enabled() || command == "" || arg == "" {
		return
	}
	c.capture("cli_argument_count", map[string]any{
		"command":      command,
		"argument":     arg,
		"count_bucket": string(BucketCount(count)),
	})
}

// TrackOption records that a CLI option (a string-valued flag) was
// supplied. The value is never captured unless the caller has
// pre-validated it against an allowlist and passes the canonical form.
// For free-form values, pass an empty string — the option name alone
// is the signal.
func (c *Client) TrackOption(command, option, allowlistedValue string) {
	if c == nil || !c.Enabled() || command == "" || option == "" {
		return
	}
	props := map[string]any{
		"command": command,
		"option":  option,
	}
	if allowlistedValue != "" {
		props["value"] = allowlistedValue
	}
	c.capture("cli_option_set", props)
}

// TrackPreview emits a sample event for the `oriyn telemetry preview`
// subcommand. Used only there; gives users a representative payload
// shape without firing real lifecycle events.
func (c *Client) TrackPreview() {
	if c == nil || !c.Enabled() {
		return
	}
	c.capture("cli_preview", map[string]any{
		"command": "telemetry preview",
	})
}

// Store buffers events for log mode so a single test or `oriyn
// telemetry preview` invocation can read what *would* have been sent.
type Store struct {
	mu    sync.Mutex
	out   io.Writer
	count int
}

func newStore(w io.Writer) *Store {
	return &Store{out: w}
}

func (s *Store) log(event map[string]any) {
	if s == nil || s.out == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	event["_n"] = strconv.Itoa(s.count)
	s.count++

	// Disable HTML escaping so duration_bucket values like "<500ms"
	// render as themselves, not "<500ms" — readable in stderr
	// and easier to grep over in tests.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(event); err != nil {
		return
	}
	// Encoder appends a newline; Fprint preserves it.
	fmt.Fprint(s.out, "[telemetry] "+buf.String())
}
