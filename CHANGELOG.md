# Changelog

All notable changes to the Oriyn CLI are documented here.

Format: `## [version] - YYYY-MM-DD` followed by Added / Changed / Fixed sections as relevant.

---

## [Unreleased]

## [0.3.0] - 2026-04-20

### Added
- `oriyn init` — one-shot onboarding: login + skill install + doctor
- `oriyn skill install` / `oriyn skill print` — lay the embedded Oriyn skill down at `~/.claude/skills/oriyn` (or `--path`); skill files are embedded in the binary so install works offline
- `oriyn doctor` — one-shot health check (auth + API reachability + `/v1/me`)
- `oriyn products context show / edit / history / version` — inspect and patch synthesized product context
- `oriyn products scrape` — kick off a Firecrawl scrape of a product source
- `oriyn personas profile` — Supermemory static + dynamic persona facts
- `oriyn personas citations --trait-index N` — evidence sessions for a persona trait
- `oriyn knowledge search` — semantic search across the product knowledge graph
- `oriyn timeline` — cross-provider per-user event timeline
- `oriyn replay` — raw rrweb events for a stored session asset (plus `--output FILE` to avoid bloating agent context)
- `oriyn timeline --output FILE` — write full JSON response to disk instead of stdout
- `oriyn experiment archive` — archive a completed experiment
- `oriyn experiment run --agents N` — plan-aware agent-count override
- `oriyn experiment run --hypothesis-stdin` — pipe long proposals from stdin
- `oriyn experiment run --no-wait / --poll-interval / --timeout` — tunable polling
- `oriyn synthesize --wait` and `oriyn enrich --wait` — block until terminal status
- Login `--no-browser` flag for headless and remote-shell contexts
- `ORIYN_AGENT=1`, `--quiet` global flag, `ORIYN_API_BASE` / `ORIYN_WEB_BASE` env vars
- Structured `APIError` surfacing credits/agent-count details for agent self-correction
- Distinct exit codes (1 user, 2 API, 3 session, 4 network) for scripted use

### Changed
- `ProductDetail` no longer carries `description` / `urls` (stale — API dropped them)
- `ExperimentListItem` now includes `title` and `convergence`
- `PersonaItem.behavioral_traits` is a `[]string` (was `json.RawMessage`)
- All list/get commands honor agent mode (`--json` / `--quiet` / `ORIYN_AGENT`)
- Root `Execute` returns `int` — main propagates it as the process exit code
- URLs in API paths now escaped via `url.PathEscape` to handle exotic IDs safely

## [0.2.0] - 2026-04-08

### Changed
- Rewritten from Rust to Go for simpler cross-compilation and easier contribution
- CLI framework: clap -> cobra
- HTTP client: reqwest -> resty
- Keychain: keyring-rs -> go-keyring
- Release: cross-rs + manual CI -> GoReleaser
- Installer script updated for Go binary naming (oriyn-{os}-{arch})
- Login flow consolidated from two /v1/me calls to one
- Telemetry flags made mutually exclusive

### Removed
- `query` command (API endpoint not yet available)

## [0.1.0] - 2026-04-05

### Added
- Initial release
- `oriyn login` — OAuth device flow, token stored in OS keychain
- `oriyn query` — natural language queries against Oriyn API
- `--api-base` flag to override API URL
- Cross-compiled binaries for x86_64/aarch64 Linux and macOS, x86_64 Windows
