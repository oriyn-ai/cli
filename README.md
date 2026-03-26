# Bridge CLI

Command-line tool for querying customer behavioral intelligence from Bridge.

## Installation

### Quick install (macOS / Linux)

```bash
curl -fsSL https://install.trybridge.dev | bash
```

### From GitHub Releases

Download the appropriate binary for your platform from the
[latest release](https://github.com/try-bridge/cli/releases/latest),
make it executable, and place it on your `PATH`.

### From source

```bash
cargo install --git https://github.com/try-bridge/cli.git
```

## Usage

### Authenticate

```bash
bridge login
```

This starts an OAuth flow and securely stores your token in the OS keychain.

### Query

```bash
bridge query "Which customers churned last month?"
```

Send a natural-language prompt to the Bridge API and print the response.

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--api-base <URL>` | Override the Bridge API base URL | `https://api.bridge.com` |

## Development

```bash
cargo build          # compile
cargo test           # run tests
cargo clippy -- -D warnings  # lint
cargo fmt            # format
```
