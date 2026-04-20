package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oriyn-ai/cli/internal/auth"
)

// newDoctorCmd is an agent-first health check — one command that tells a
// coding agent whether it can actually use Oriyn right now. Prints a series
// of checks, exits non-zero on any failure so CI / agent scripts can gate on it.
func newDoctorCmd(app *App, version, commit string) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify environment, authentication, and API reachability",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			results := runDoctorChecks(ctx, app, version, commit)
			w := cmd.OutOrStdout()
			if agentMode(cmd, jsonOutput) {
				return printJSON(w, results)
			}
			for _, r := range results.Checks {
				marker := color.GreenString("✓")
				if !r.OK {
					marker = color.RedString("✗")
				}
				fmt.Fprintf(w, "%s %-24s %s\n", marker, r.Name, r.Detail)
			}
			if !results.OK {
				return fmt.Errorf("one or more checks failed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	return cmd
}

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type doctorReport struct {
	OK      bool          `json:"ok"`
	Version string        `json:"version"`
	Commit  string        `json:"commit"`
	OS      string        `json:"os"`
	Arch    string        `json:"arch"`
	APIBase string        `json:"api_base"`
	Checks  []doctorCheck `json:"checks"`
}

func runDoctorChecks(ctx context.Context, app *App, version, commit string) doctorReport {
	report := doctorReport{
		Version: version,
		Commit:  commit,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		APIBase: app.APIBase,
	}

	// 1. Auth — present in keychain or env override
	authCheck := doctorCheck{Name: "auth"}
	if _, err := app.AuthStore.GetValidAccessToken(ctx); err != nil {
		authCheck.Detail = err.Error()
		if errors.Is(err, auth.ErrNotLoggedIn) {
			authCheck.Detail = "not logged in — run `oriyn login`"
		} else if errors.Is(err, auth.ErrSessionExpired) {
			authCheck.Detail = "session expired — run `oriyn login`"
		}
	} else {
		authCheck.OK = true
		authCheck.Detail = "token present"
	}
	report.Checks = append(report.Checks, authCheck)

	// 2. API reachability via /version (no auth required) — called directly
	// so the check works even when the keychain is empty.
	reachCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	apiCheck := doctorCheck{Name: "api-reachable"}
	if v, err := fetchAPIVersion(reachCtx, app.APIBase); err != nil {
		apiCheck.Detail = err.Error()
	} else {
		apiCheck.OK = true
		apiCheck.Detail = fmt.Sprintf("api version %s", v)
	}
	report.Checks = append(report.Checks, apiCheck)

	// 3. /v1/me as the end-to-end auth-through-API check (only if auth passed)
	meCheck := doctorCheck{Name: "whoami"}
	if authCheck.OK {
		if me, err := app.API.GetMe(ctx); err != nil {
			meCheck.Detail = err.Error()
		} else {
			meCheck.OK = true
			meCheck.Detail = me.Email
		}
	} else {
		meCheck.Detail = "skipped (auth failed)"
	}
	report.Checks = append(report.Checks, meCheck)

	report.OK = true
	for _, c := range report.Checks {
		if !c.OK {
			report.OK = false
			break
		}
	}
	return report
}

func fetchAPIVersion(ctx context.Context, apiBase string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	return v.Version, nil
}
