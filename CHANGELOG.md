# Changelog

All notable changes to the Oriyn CLI are documented here.

Format: `## [version] - YYYY-MM-DD` followed by Added / Changed / Fixed sections as relevant.

---

## [0.6.0](https://github.com/oriyn-ai/cli/compare/v0.5.0...v0.6.0) (2026-04-29)


### Features

* **auth:** switch CLI to OAuth 2.0 + PKCE (Clerk) ([2662872](https://github.com/oriyn-ai/cli/commit/2662872bf0b9e73dac705c732976eae43fef5c74))
* **auth:** switch CLI to OAuth 2.0 + PKCE (Clerk) ([06bd251](https://github.com/oriyn-ai/cli/commit/06bd2518b2fcc20bcc850345f33b47af6c4e864e))


### Bug Fixes

* **login_test:** close response bodies, return error last ([25f2d75](https://github.com/oriyn-ai/cli/commit/25f2d752340cbc3ec3eb45c71df6aaa384d990f5))
* **login_test:** close response bodies, return error last ([9218cfd](https://github.com/oriyn-ai/cli/commit/9218cfd02159c2ae82ff36e01271b12a24779456))

## [0.5.0](https://github.com/oriyn-ai/cli/compare/v0.4.0...v0.5.0) (2026-04-28)


### Features

* switch CLI auth to Clerk JWT ([7116b49](https://github.com/oriyn-ai/cli/commit/7116b49de1539a9699bc32f98f9dc00fc8645e89))
* switch CLI auth to Clerk JWT ([1455f78](https://github.com/oriyn-ai/cli/commit/1455f78fbae20fbacc917587076b8c36ab6e35f2))
* **telemetry:** rebuild around Vercel CLI pattern ([50c39ce](https://github.com/oriyn-ai/cli/commit/50c39cea9046804009d1b700ace7862b0ef44606))
* **telemetry:** rebuild around Vercel CLI pattern ([4db60a5](https://github.com/oriyn-ai/cli/commit/4db60a5897be9b46701032d0a3cc4b332eccece0))
* **telemetry:** structured diagnostics — command lifecycle, error classifier, login funnel ([6ef0c8b](https://github.com/oriyn-ai/cli/commit/6ef0c8b1f96dc500aa613bb63dc2cd0cf85ca757))
* **telemetry:** structured diagnostics — lifecycle + error classifier + login funnel ([6534ec0](https://github.com/oriyn-ai/cli/commit/6534ec03527b84301158eb9e8adb1938d18b9b91))
* **uninstall:** add `oriyn uninstall` + `install.sh --uninstall` ([5cc7f2f](https://github.com/oriyn-ai/cli/commit/5cc7f2f5b3a4b1cc523b78511d2d60118cac96cd))
* **uninstall:** add `oriyn uninstall` + `install.sh --uninstall` ([c40088e](https://github.com/oriyn-ai/cli/commit/c40088e1dec77175b6ab3f0424ce96a029da2cb5))


### Bug Fixes

* **apiclient,telemetry:** align with flat 403 error shape ([dadae18](https://github.com/oriyn-ai/cli/commit/dadae18077af9a43daec3a0283a66d8f49be6cc0))
* **ci:** pin go 1.26.2 + golangci-lint v2.11.4 ([ddc4168](https://github.com/oriyn-ai/cli/commit/ddc4168c0a258237a9d42707ec18613764e1a867))
* **install:** hoist temp vars so EXIT trap doesn't trip set ([573d14a](https://github.com/oriyn-ai/cli/commit/573d14a70fad7b6b47558afde26095867ef752d7))
* **telemetry:** make ORIYN_TELEMETRY=log override CI auto-skip; scrub CI env in dispatch test ([cb7f618](https://github.com/oriyn-ai/cli/commit/cb7f6184cd100c22e47ed7af7f1825308d1060f3))

## [Unreleased]

## [0.4.0] - 2026-04-24

### Changed
- **BREAKING:** `oriyn skill install` now fetches the skill from `https://oriyn.ai/skill.md` on every invocation. No copy of the skill is embedded in the binary. First install requires network; after that, the agent reads the installed copy offline. Use `--url <file-path>` to install from a local file for development or air-gapped environments.
- Single source of truth: the marketing app's `public/skill.md` is the only editable skill source. Drift between the CLI's embedded copy and the published URL is structurally impossible now — see `decisions/skill-remote-fetch-2026-04-24.md`.

### Added
- `oriyn skill update` — idempotent re-fetch of the skill from oriyn.ai. Equivalent to `oriyn skill install --force`. Use when the remote skill has been updated.
- `oriyn skill print --url <path>` — print any skill source (remote URL or local file) to stdout without installing.

### Removed
- `cli/skills/oriyn/` (SKILL.md + README.md + references/) and `cli/embedded.go` — no content ships in the binary anymore.

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
