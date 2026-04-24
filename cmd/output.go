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

const (
	ExitUserError        = 1 // flag misuse, missing required input
	ExitAPIError         = 2 // API returned 4xx/5xx
	ExitSessionExpired   = 3 // credentials missing or refused by Supabase
	ExitNetworkError     = 4 // couldn't reach the API at all
	ExitPermissionDenied = 5 // API returned 403 with a permission payload
)

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

func printJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

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
