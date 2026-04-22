package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/apiclient"
	"github.com/oriyn-ai/cli/internal/auth"
)

// Exit codes used by the CLI so coding agents can branch deterministically.
// 0 reserved for success via cobra's default. Anything else is surfaced by the
// Execute loop — see root.go.
const (
	ExitUserError        = 1 // flag misuse, missing required input
	ExitAPIError         = 2 // API returned 4xx/5xx
	ExitSessionExpired   = 3 // credentials missing or refused by Supabase
	ExitNetworkError     = 4 // couldn't reach the API at all
	ExitPermissionDenied = 5 // API returned 403 with a permission payload
)

// agentMode returns true when the caller wants machine-readable output with no
// interactive niceties. Any of these opt in:
//   - --json on the command (callers wire this flag)
//   - --quiet on the root command
//   - ORIYN_AGENT=1 in the environment
//   - stdout is not a TTY (common when piped through jq)
func agentMode(cmd *cobra.Command, explicitJSON bool) bool {
	if explicitJSON {
		return true
	}
	if quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet"); quiet {
		return true
	}
	if v := strings.ToLower(os.Getenv("ORIYN_AGENT")); v == "1" || v == "true" {
		return true
	}
	return false
}

// printJSON writes the given value as compact JSON followed by a newline.
// All agent-mode output flows through here so the contract is uniform.
func printJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

// classifyError maps an error into a CLI exit code. Keep the mapping tight —
// the goal is to let agents branch on reason without parsing messages.
func classifyError(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, auth.ErrNotLoggedIn) || errors.Is(err, auth.ErrSessionExpired) {
		return ExitSessionExpired
	}
	var permErr *apiclient.PermissionError
	if errors.As(err, &permErr) {
		return ExitPermissionDenied
	}
	var apiErr *apiclient.APIError
	if errors.As(err, &apiErr) {
		return ExitAPIError
	}
	if strings.Contains(err.Error(), "reaching oriyn API") {
		return ExitNetworkError
	}
	return ExitUserError
}

// readHypothesis returns the hypothesis string from --hypothesis or stdin.
// Agents that generate long multiline proposals usually prefer to pipe them.
func readHypothesis(cmd *cobra.Command, flagValue string, fromStdin bool) (string, error) {
	if fromStdin {
		if flagValue != "" {
			return "", fmt.Errorf("--hypothesis and --hypothesis-stdin are mutually exclusive")
		}
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("reading hypothesis from stdin: %w", err)
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", fmt.Errorf("hypothesis from stdin is empty")
		}
		return text, nil
	}
	if flagValue == "" {
		return "", fmt.Errorf("--hypothesis is required (or use --hypothesis-stdin to pipe)")
	}
	return flagValue, nil
}

func joinStrings(ss []string, sep string) string {
	return strings.Join(ss, sep)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
