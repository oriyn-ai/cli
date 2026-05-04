package telemetry

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oriyn-ai/cli/internal/apiclient"
)

func TestBucketDuration_Boundaries(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want DurationBucket
	}{
		{0, BucketDur100ms},
		{99 * time.Millisecond, BucketDur100ms},
		{100 * time.Millisecond, BucketDur500ms},
		{499 * time.Millisecond, BucketDur500ms},
		{500 * time.Millisecond, BucketDur1s},
		{999 * time.Millisecond, BucketDur1s},
		{1 * time.Second, BucketDur5s},
		{5 * time.Second, BucketDur30s},
		{30 * time.Second, BucketDur2m},
		{2 * time.Minute, BucketDur10m},
		{10 * time.Minute, BucketDurOver10m},
		{2 * time.Hour, BucketDurOver10m},
	}
	for _, c := range cases {
		if got := BucketDuration(c.d); got != c.want {
			t.Errorf("BucketDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestBucketCount_Boundaries(t *testing.T) {
	cases := []struct {
		n    int
		want CountBucket
	}{
		{-1, BucketCountZero},
		{0, BucketCountZero},
		{1, BucketCountOne},
		{2, BucketCountFew},
		{10, BucketCountFew},
		{11, BucketCountMany},
		{100, BucketCountMany},
		{101, BucketCountHuge},
	}
	for _, c := range cases {
		if got := BucketCount(c.n); got != c.want {
			t.Errorf("BucketCount(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestClassifyError_Nil(t *testing.T) {
	info := ClassifyError(nil)
	if info.Outcome != OutcomeSuccess {
		t.Errorf("nil error → %q, want success", info.Outcome)
	}
}

func TestClassifyError_APIErrorExtractsFields(t *testing.T) {
	credits := 100
	max := 5
	apiErr := &apiclient.APIError{
		StatusCode:      402,
		Message:         "  not enough credits\n   to run this experiment  ",
		Plan:            "free",
		CreditsRequired: &credits,
		MaxAgentCount:   &max,
	}
	wrapped := fmt.Errorf("create experiment: %w", apiErr)

	info := ClassifyError(wrapped)
	if info.Outcome != OutcomeAPIError {
		t.Errorf("Outcome = %q, want api_error", info.Outcome)
	}
	if info.Status != 402 {
		t.Errorf("Status = %d, want 402", info.Status)
	}
	if info.Plan != "free" {
		t.Errorf("Plan = %q, want free", info.Plan)
	}
	if !info.HasCreditsRequired || !info.HasMaxAgentCount {
		t.Errorf("missing has_* booleans: %+v", info)
	}
	if info.ServerMessage != "not enough credits to run this experiment" {
		t.Errorf("ServerMessage not normalized: %q", info.ServerMessage)
	}
}

func TestClassifyError_PermissionErrorExtractsFields(t *testing.T) {
	permErr := &apiclient.PermissionError{
		StatusCode:         403,
		RequiredPermission: "org:content:write",
		Role:               "org:member",
	}
	info := ClassifyError(permErr)
	if info.Outcome != OutcomePermissionError {
		t.Errorf("Outcome = %q, want permission_error", info.Outcome)
	}
	if info.RequiredPermission != "org:content:write" || info.Role != "org:member" {
		t.Errorf("permission fields not extracted: %+v", info)
	}
}

func TestClassifyError_ServerMessageTruncates(t *testing.T) {
	long := strings.Repeat("a", MaxServerMessageLen+200)
	apiErr := &apiclient.APIError{Message: long}
	info := ClassifyError(apiErr)
	if len(info.ServerMessage) != MaxServerMessageLen {
		t.Errorf("ServerMessage length = %d, want %d", len(info.ServerMessage), MaxServerMessageLen)
	}
}

func TestClassifyError_NeverLeaksRawErrorMessage(t *testing.T) {
	// A path-like error string must never appear in the ErrorInfo. The
	// classifier maps it into a fixed-vocabulary outcome instead.
	err := errors.New("dial tcp /Users/shivam/secret-stuff/sock: connection refused")
	info := ClassifyError(err)

	if info.Outcome != OutcomeNetworkError {
		t.Errorf("Outcome = %q, want network_error", info.Outcome)
	}
	if strings.Contains(info.ServerMessage, "/Users/") || strings.Contains(info.ServerMessage, "secret-stuff") {
		t.Errorf("ServerMessage leaked path: %q", info.ServerMessage)
	}
}

func TestClassifyError_KeywordFallbacks(t *testing.T) {
	cases := []struct {
		msg     string
		outcome Outcome
	}{
		{"not logged in", OutcomeAuthError},
		{"session expired", OutcomeAuthError},
		{"keychain access denied", OutcomeKeychainError},
		{"context canceled", OutcomeCanceled},
		{"deadline exceeded: timed out", OutcomeNetworkError},
		{"connection refused", OutcomeNetworkError},
		{"no such host", OutcomeNetworkError},
		{"something else entirely", OutcomeUnknownError},
	}
	for _, c := range cases {
		got := ClassifyError(errors.New(c.msg))
		if got.Outcome != c.outcome {
			t.Errorf("ClassifyError(%q).Outcome = %q, want %q", c.msg, got.Outcome, c.outcome)
		}
	}
}

func TestTruncateServerMessage_CollapsesWhitespace(t *testing.T) {
	got := truncateServerMessage("  multi\n\t\n  line\n  msg  ")
	if got != "multi line msg" {
		t.Errorf("got %q, want %q", got, "multi line msg")
	}
}
