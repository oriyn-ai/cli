# Oriyn CLI

Command-line tool for querying customer behavioral intelligence from Oriyn.

## Installation

### Quick install (macOS / Linux)

```bash
curl -fsSL https://install.oriyn.ai | bash
```

### From GitHub Releases

Download the appropriate binary for your platform from the
[latest release](https://github.com/oriyn-ai/cli/releases/latest),
make it executable, and place it on your `PATH`.

### From source

```bash
cargo install --git https://github.com/oriyn-ai/cli.git
```

## Usage

### Authenticate

```bash
oriyn login
```

This starts an OAuth flow and securely stores your token in the OS keychain.

### Query

```bash
oriyn query "Which customers churned last month?"
```

Send a natural-language prompt to the Oriyn API and print the response.

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--api-base <URL>` | Override the Oriyn API base URL | `https://api.oriyn.ai` |

## Development

```bash
cargo build          # compile
cargo test           # run tests
cargo clippy -- -D warnings  # lint
cargo fmt            # format
```
