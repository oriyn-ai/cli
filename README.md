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
go install github.com/oriyn-ai/cli@latest
```

## Usage

### Authenticate

```bash
oriyn login
```

This starts an OAuth flow and securely stores your token in the OS keychain.

### Query products

```bash
oriyn products                          # list all products
oriyn products get --product-id <ID>    # get product details
```

### View enrichment data

```bash
oriyn personas --product-id <ID>        # behavioral personas
oriyn patterns --product-id <ID>        # behavioral patterns
oriyn direction --product-id <ID>       # prescriptive direction
```

### Run experiments

```bash
oriyn experiment run --product <ID> --hypothesis "Users prefer dark mode"
oriyn experiment list --product <ID>
oriyn experiment get --product <ID> --experiment <EID>
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--api-base <URL>` | Override the Oriyn API base URL | `https://api.oriyn.ai` |
| `--web-base <URL>` | Override the Oriyn web app base URL | `https://app.oriyn.ai` |

## Development

```bash
go build ./...          # compile
go test ./...           # run tests
go vet ./...            # lint
```
