## Repository
- GitHub org: `try-bridge`
- CLI repo: `try-bridge/cli`
- GitHub Releases URL pattern: `https://github.com/try-bridge/cli/releases/download/vX.Y.Z/bridge-<target>`

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
