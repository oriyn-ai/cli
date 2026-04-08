# Changelog

All notable changes to the Oriyn CLI are documented here.

Format: `## [version] - YYYY-MM-DD` followed by Added / Changed / Fixed sections as relevant.

---

## [Unreleased]

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
