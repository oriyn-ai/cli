# oriyn CLI

Predict how users respond to product changes before shipping — from your terminal or any AI agent.

The CLI is built around a single headline command:

```
oriyn experiments run "Should we move pricing before signup?"
```

It auto-resolves the product from a nearby `oriyn.json`, streams progress as JSONL when piped (or shows a spinner in a TTY), and prints a structured verdict with per-persona breakdown.

## Install

```bash
curl -fsSL https://oriyn.ai/install.sh | bash
# or, with Bun already installed:
bun add -g oriyn
```

The CLI requires Bun ≥ 1.2 when installed via npm. The curl installer falls back to a precompiled standalone binary for users without Bun.

## Quickstart

```bash
oriyn auth login                           # browser PKCE; tokens at ~/.config/oriyn/credentials.json (0600)
cd <your repo>
oriyn link                                 # interactive picker → writes oriyn.json (commit it)
oriyn experiments run "<your hypothesis>"  # the headline flow
```

`oriyn.json` lives at the project root and is shared with your team. The CLI walks up from `cwd` to find it (just like `package.json`), so monorepos with multiple linked products work out of the box.

## Commands

```
oriyn auth login [--no-browser]      Log in via browser (OAuth 2.1 + PKCE)
oriyn auth logout                    Forget stored credentials
oriyn auth whoami                    Show the logged-in account
oriyn auth status                    Token validity + expiry

oriyn link                           Interactive product picker → oriyn.json
oriyn link --product <id>            Non-interactive
oriyn unlink                         Remove oriyn.json from cwd

oriyn products                       List products in the org
oriyn personas                       List behavioral personas
oriyn personas <id>                  Persona detail (profile + facts)
oriyn patterns                       Mined hypotheses + bottlenecks
oriyn experiments                    List experiments
oriyn experiments <id>               Get one experiment
oriyn experiments run "<hypothesis>" Run experiment, stream progress

oriyn sync                           Idempotent synthesize → enrich
oriyn status                         One-screen diagnostic
oriyn config [key] [value]           Show or update CLI config
oriyn open [resource]                Open the web app for the linked product
oriyn upgrade                        Upgrade via bun add -g oriyn@latest
oriyn completion <bash|zsh|fish>     Print shell completion script
```

## Agent flow (CI / Claude Code / Codex)

For sandboxed agents, set one env var and commit `oriyn.json`:

```bash
export ORIYN_ACCESS_TOKEN=<token>          # from app.oriyn.ai → Settings
oriyn experiments run "<hypothesis>"       # streams JSONL to stdout
```

The CLI infers JSONL mode from non-TTY stdout. Each line is an event:

```json
{"type":"step","name":"create-experiment","ts":"…"}
{"type":"progress","message":"status: running","ts":"…"}
{"type":"result","data":{ "summary": { "verdict": "ship", "convergence": 0.86, "persona_breakdown": [ … ] } }}
```

Exit codes: `0` ok · `2` api · `3` auth · `4` network · `5` permission · `1` other.

## Configuration

| Path | Purpose |
|------|---------|
| `~/.config/oriyn/credentials.json` | Auth tokens (`0600`) |
| `~/.config/oriyn/config.json` | CLI prefs (telemetry, default product) |
| `<repo>/oriyn.json` | Project link `{ orgId, productId }` (commit it) |

Override with env vars:

| Env | Effect |
|-----|--------|
| `ORIYN_ACCESS_TOKEN` | Skip credentials file (CI escape hatch) |
| `ORIYN_API_BASE` | Override API base URL |
| `ORIYN_PRODUCT` / `ORIYN_ORG` | Override link resolution |
| `ORIYN_CONFIG_DIR` | Move the global config dir |
| `ORIYN_TELEMETRY=off` | Disable PostHog usage events |
| `NO_COLOR=1` | Disable colors |
| `FORCE_COLOR=1` | Enforce colors when piped |

## Develop

```bash
git clone git@github.com:oriyn-ai/cli.git && cd cli
bun install
bun test
bun run src/index.ts --help
```

Workflow: `bun test` (unit), `bun x biome check .` (lint+format), `bunx tsc --noEmit` (typecheck), `bun run scripts/build-binaries.ts` (cross-compile).

## License

Pre-launch / not yet licensed.
