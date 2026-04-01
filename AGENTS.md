## Repository
- GitHub org: `oriyn-ai`
- CLI repo: `oriyn-ai/cli`
- GitHub Releases URL pattern: `https://github.com/oriyn-ai/cli/releases/download/vX.Y.Z/oriyn-<target>`

## Project Structure
- Binary crate: `src/main.rs` is the entrypoint only — logic lives in modules
- `src/commands/` — one file per subcommand
- `src/api/` — HTTP client and request/response types
- `src/auth.rs` — keychain read/write logic

## Dependencies
- Error handling: `thiserror` for library errors, `anyhow` for binary errors
- HTTP: `reqwest` with `json` and `rustls-tls` features (avoid openssl dependency)
- Auth storage: `keyring` — never write credentials to flat files
- Async: `tokio` with `full` features unless you have a reason to be selective

## Async
- Mark all I/O functions `async`
- Never call blocking code inside async context — use `tokio::task::spawn_blocking` if needed

## Error Handling
- No `unwrap()` or `expect()` outside of tests
- Propagate errors with `?`
- Surface user-facing errors with context: `anyhow::Context::context()`

## Release & Versioning
- Semver: `MAJOR.MINOR.PATCH`
- Tag format for GitHub Actions release trigger: `v1.2.3`
- Cross-compilation targets: x86_64/aarch64 for linux and darwin, x86_64 for windows

## Security
- API tokens stored in OS keychain via `keyring`, never in files or env vars committed to repo
- Secrets in CI go in GitHub Actions secrets, referenced as `${{ secrets.NAME }}`

## Decision Logging

When choosing between alternatives that affect more than today's task — a library, an architecture pattern, an API design, or deciding NOT to do something — log it in `/decisions/{topic}-YYYY-MM-DD.md`:

## Decision: {what you decided}
## Context: {why this came up}
## Alternatives considered: {what else was on the table}
## Reasoning: {why this option won}
## Trade-offs accepted: {what you gave up}
## Status: {active | superseded by {topic}-YYYY-MM-DD | revisit when}

Before making a similar decision, grep `/decisions/` for prior choices and follow them unless new information invalidates the reasoning.
