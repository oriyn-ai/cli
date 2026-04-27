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

// Capture emits a single event. Properties must already be safe for
// transmission — the package does not redact at this layer; redaction
// is the responsibility of the call site, which has full context.
func (c *Client) Capture(event string, props map[string]any) {
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
	return map[string]any{
		"$lib":          "oriyn-cli",
		"cli_version":   c.version,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"device_id":     c.identity.DeviceID,
		"session_id":    c.identity.SessionID,
		"invocation_id": c.identity.InvocationID,
	}
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
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintln(s.out, "[telemetry] "+string(data))
}
