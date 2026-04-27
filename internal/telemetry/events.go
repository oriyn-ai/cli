package telemetry

import (
	"errors"
	"strings"
	"time"

	"github.com/oriyn-ai/cli/internal/apiclient"
)

// Event names. Adding a new constant is the first step in adding any
// new tracked event — the typed helpers below force every call site to
// reference one of these, mirroring Vercel's allowlist-by-method-name
// pattern.
const (
	EventCommandStarted    = "cli_command_started"
	EventCommandCompleted  = "cli_command_completed"
	EventLoginStateChanged = "cli_login_state_changed"
	EventOutputCount       = "cli_output_count"
)

// Outcome buckets the exit status of a command into a small enum so
// dashboards group meaningfully without exposing raw error strings.
type Outcome string

const (
	OutcomeSuccess         Outcome = "success"
	OutcomeUserError       Outcome = "user_error"       // bad flag, missing arg, etc.
	OutcomeAuthError       Outcome = "auth_error"       // not logged in, expired session
	OutcomePermissionError Outcome = "permission_error" // 403 with required-permission body
	OutcomeAPIError        Outcome = "api_error"        // 4xx/5xx from oriyn-api
	OutcomeNetworkError    Outcome = "network_error"    // DNS, connect, timeout
	OutcomeKeychainError   Outcome = "keychain_error"   // OS keyring failure
	OutcomeCanceled        Outcome = "canceled"         // ctx.Done / SIGINT
	OutcomeUnknownError    Outcome = "unknown_error"
)

// LoginState captures every transition in the browser-OAuth flow so
// product can see where users drop out of the funnel.
type LoginState string

const (
	LoginStateStarted          LoginState = "started"
	LoginStateBrowserOpened    LoginState = "browser_opened"
	LoginStateAwaitingCallback LoginState = "awaiting_callback"
	LoginStateCallbackReceived LoginState = "callback_received"
	LoginStateProfileFetched   LoginState = "profile_fetched"
	LoginStateSucceeded        LoginState = "succeeded"
	LoginStateTimedOut         LoginState = "timed_out"
	LoginStateCanceled         LoginState = "canceled"
	LoginStateFailed           LoginState = "failed"
)

// MaxServerMessageLen mirrors Vercel's 500-char ceiling on
// error_server_message. Server-generated messages are safe to capture
// but not infinitely long.
const MaxServerMessageLen = 500

// DurationBucket coarsens a duration into one of eight bands. Bucketing
// at the call site (rather than sending raw ms) protects against
// fingerprinting a slow user's machine via timing distribution while
// still letting us see "this command got slower in v0.5."
type DurationBucket string

const (
	BucketDur100ms   DurationBucket = "<100ms"
	BucketDur500ms   DurationBucket = "<500ms"
	BucketDur1s      DurationBucket = "<1s"
	BucketDur5s      DurationBucket = "<5s"
	BucketDur30s     DurationBucket = "<30s"
	BucketDur2m      DurationBucket = "<2m"
	BucketDur10m     DurationBucket = "<10m"
	BucketDurOver10m DurationBucket = ">=10m"
)

// BucketDuration returns a stable, low-cardinality bucket for d.
func BucketDuration(d time.Duration) DurationBucket {
	ms := d.Milliseconds()
	switch {
	case ms < 100:
		return BucketDur100ms
	case ms < 500:
		return BucketDur500ms
	case ms < 1_000:
		return BucketDur1s
	case ms < 5_000:
		return BucketDur5s
	case ms < 30_000:
		return BucketDur30s
	case ms < 120_000:
		return BucketDur2m
	case ms < 600_000:
		return BucketDur10m
	default:
		return BucketDurOver10m
	}
}

// CountBucket coarsens a result-set size for list-shape commands.
type CountBucket string

const (
	BucketCountZero CountBucket = "zero"
	BucketCountOne  CountBucket = "one"
	BucketCountFew  CountBucket = "few"  // 2–10
	BucketCountMany CountBucket = "many" // 11–100
	BucketCountHuge CountBucket = "huge" // 100+
)

// BucketCount returns a stable bucket for an integer count.
func BucketCount(n int) CountBucket {
	switch {
	case n <= 0:
		return BucketCountZero
	case n == 1:
		return BucketCountOne
	case n <= 10:
		return BucketCountFew
	case n <= 100:
		return BucketCountMany
	default:
		return BucketCountHuge
	}
}

// ErrorInfo holds the structured fields extracted from a Go error for
// transmission. All string fields are bounded; nothing here passes
// through user paths, args, or env contents — those would have leaked
// in via err.Error() if we used it directly.
type ErrorInfo struct {
	Outcome            Outcome
	Status             int    // HTTP status when applicable
	ServerMessage      string // truncated + whitespace-collapsed
	Plan               string // from APIError when present
	HasCreditsRequired bool   // boolean to avoid leaking the value
	HasMaxAgentCount   bool
	RequiredPermission string // safe: enum-shaped
	Role               string // safe: enum-shaped (member|admin|owner)
	HasOrgID           bool   // boolean only — actual ID is PII
}

// ClassifyError produces an ErrorInfo for telemetry. Never returns the
// raw err.Error() string. Falls back to keyword classification on
// unknown errors using the same rules as the prior cli_error path.
func ClassifyError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{Outcome: OutcomeSuccess}
	}

	// 1) APIError: structured server response.
	var apiErr *apiclient.APIError
	if errors.As(err, &apiErr) {
		return ErrorInfo{
			Outcome:            OutcomeAPIError,
			Status:             apiErr.StatusCode,
			ServerMessage:      truncateServerMessage(apiErr.Message),
			Plan:               apiErr.Plan,
			HasCreditsRequired: apiErr.CreditsRequired != nil,
			HasMaxAgentCount:   apiErr.MaxAgentCount != nil,
		}
	}

	// 2) PermissionError: 403 with permission body.
	var permErr *apiclient.PermissionError
	if errors.As(err, &permErr) {
		return ErrorInfo{
			Outcome:            OutcomePermissionError,
			Status:             permErr.StatusCode,
			RequiredPermission: permErr.RequiredPermission,
			Role:               permErr.Role,
			HasOrgID:           permErr.OrgID != "",
		}
	}

	// 3) Last resort — keyword classification on the lowercased message.
	// We never transmit err.Error(); only the bucket and a fixed
	// server_message label leave this function.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not logged in"):
		return ErrorInfo{Outcome: OutcomeAuthError, ServerMessage: "not_logged_in"}
	case strings.Contains(msg, "session expired"):
		return ErrorInfo{Outcome: OutcomeAuthError, ServerMessage: "session_expired"}
	case strings.Contains(msg, "keychain"):
		return ErrorInfo{Outcome: OutcomeKeychainError}
	case strings.Contains(msg, "required flag"),
		strings.Contains(msg, "unknown command"),
		strings.Contains(msg, "unknown flag"),
		strings.Contains(msg, "invalid argument"),
		strings.Contains(msg, "accepts at most"),
		strings.Contains(msg, "requires at least"):
		return ErrorInfo{Outcome: OutcomeUserError, ServerMessage: "bad_invocation"}
	case strings.Contains(msg, "canceled"), strings.Contains(msg, "context canceled"):
		return ErrorInfo{Outcome: OutcomeCanceled}
	case strings.Contains(msg, "timed out"), strings.Contains(msg, "timeout"):
		return ErrorInfo{Outcome: OutcomeNetworkError, ServerMessage: "timeout"}
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"),
		strings.Contains(msg, "network is unreachable"):
		return ErrorInfo{Outcome: OutcomeNetworkError, ServerMessage: "unreachable"}
	default:
		return ErrorInfo{Outcome: OutcomeUnknownError}
	}
}

// truncateServerMessage applies Vercel's exact normalization rule:
// trim, collapse whitespace, truncate to MaxServerMessageLen.
func truncateServerMessage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > MaxServerMessageLen {
		s = s[:MaxServerMessageLen]
	}
	return s
}
