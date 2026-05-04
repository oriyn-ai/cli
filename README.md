<h1 align="center">oriyn</h1>

<p align="center">
  Predict how users respond to product changes — <em>before</em> you ship.<br/>
  From your terminal. Or any AI agent.
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/oriyn"><img src="https://img.shields.io/npm/v/oriyn?color=cb3837&label=npm" alt="npm version"></a>
  <a href="https://www.npmjs.com/package/oriyn"><img src="https://img.shields.io/npm/dm/oriyn?color=cb3837" alt="npm downloads"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/npm/l/oriyn?color=blue" alt="license"></a>
  <a href="https://github.com/oriyn-ai/cli/actions/workflows/ci.yml"><img src="https://github.com/oriyn-ai/cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://bun.com"><img src="https://img.shields.io/badge/runtime-bun%20%E2%89%A5%201.2-fbf0df" alt="Bun ≥ 1.2"></a>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quickstart">Quickstart</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#agents--ci">Agents &amp; CI</a> ·
  <a href="#configuration">Config</a> ·
  <a href="https://oriyn.ai/docs">Docs</a>
</p>

---

The CLI is built around a single headline command:

```bash
oriyn experiments run "Should we move pricing before signup?"
```

It auto-resolves the product from a nearby `oriyn.json`, streams progress as
JSONL when piped (or shows a spinner in a TTY), and prints a structured
verdict with per-persona breakdown.

## Why oriyn

- **Predict, don't guess.** Run an experiment against your real users' synthesized personas and get a verdict in seconds — before you build the variant.
- **Agent-native.** Non-TTY → JSONL. Designed for Claude Code, Codex, CI, and shell pipelines from day one.
- **Cross-provider.** Personas are built from your product analytics, session replays, and payments — not from a single tool's view.
- **Local-first auth.** OAuth 2.1 + PKCE direct to your browser. Tokens at `~/.config/oriyn/credentials.json` (`0600`). No keychain, no daemons.
- **Honest output.** No telemetry in dev or CI. `ORIYN_TELEMETRY=off` kills it everywhere else.

## Install

```bash
# npm
npm i -g oriyn

# bun
bun add -g oriyn

# one-liner (falls back to a precompiled binary if Bun isn't installed)
curl -fsSL https://oriyn.ai/install.sh | bash
```

The npm/bun install requires **Bun ≥ 1.2** at runtime. The curl installer
ships precompiled standalone binaries for macOS and Linux (x64 + arm64) — no
runtime needed.

## Quickstart

```bash
oriyn auth login                            # browser PKCE
cd <your repo>
oriyn link                                  # interactive picker → oriyn.json
oriyn experiments run "<your hypothesis>"   # the headline flow
```

`oriyn.json` lives at the project root and is shared with your team. The CLI
walks up from `cwd` to find it (just like `package.json`), so monorepos with
multiple linked products work out of the box.

## Commands

```text
oriyn auth login [--no-browser]      Log in via browser (OAuth 2.1 + PKCE)
oriyn auth logout                    Forget stored credentials
oriyn auth whoami                    Show the logged-in account
oriyn auth status                    Token validity + expiry

oriyn link [--product <id>]          Link a product → writes oriyn.json
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
oriyn upgrade                        Upgrade to the latest version
oriyn completion <bash|zsh|fish>     Print shell completion script
```

Run `oriyn <command> --help` for full flags on any command.

## Agents & CI

For sandboxed agents (Claude Code, Codex, GitHub Actions, etc.), set one env
var and commit `oriyn.json`:

```bash
export ORIYN_ACCESS_TOKEN=<token>           # from app.oriyn.ai → Settings
oriyn experiments run "<hypothesis>"        # streams JSONL to stdout
```

The CLI infers JSONL mode from a non-TTY stdout. Each line is one event:

```jsonc
{"type":"step","name":"create-experiment","ts":"…"}
{"type":"progress","message":"status: running","ts":"…"}
{"type":"result","data":{"summary":{"verdict":"ship","convergence":0.86,"persona_breakdown":[…]}}}
```

Force the mode explicitly with `--human` or `--json` if you need to override
TTY detection.

**Exit codes:** `0` ok · `2` api · `3` auth · `4` network · `5` permission · `1` other.

## Configuration

| Path                                | Purpose                                            |
| ----------------------------------- | -------------------------------------------------- |
| `~/.config/oriyn/credentials.json`  | Auth tokens (mode `0600`)                          |
| `~/.config/oriyn/config.json`       | CLI prefs (telemetry, default product)             |
| `<repo>/oriyn.json`                 | Project link `{ orgId, productId }` — commit it    |

| Env var               | Effect                                            |
| --------------------- | ------------------------------------------------- |
| `ORIYN_ACCESS_TOKEN`  | Skip credentials file (CI escape hatch)           |
| `ORIYN_API_BASE`      | Override API base URL                             |
| `ORIYN_PRODUCT`       | Override the linked product                       |
| `ORIYN_ORG`           | Override the linked org                           |
| `ORIYN_CONFIG_DIR`    | Move the global config dir (useful for tests)     |
| `ORIYN_TELEMETRY=off` | Disable PostHog usage events                      |
| `NO_COLOR=1`          | Disable colors                                    |
| `FORCE_COLOR=1`       | Force colors when piped                           |

## Telemetry

Anonymous usage events are sent to PostHog to help us prioritize. They are:

- **Off** in dev (`VERSION === '0.0.0-dev'`) and on CI (auto-detected)
- **Announced** with a one-line notice on first use
- **Disabled** instantly with `ORIYN_TELEMETRY=off` or `oriyn config telemetry off`
- **Scrubbed** of all bearer tokens, JWTs, and refresh tokens before send

No request bodies, no hypotheses, no persona content — only command name,
exit code, and duration.

## Local development

```bash
git clone https://github.com/oriyn-ai/cli.git && cd cli
bun install
bun test
bun run src/index.ts --help
```

| Task            | Command                              |
| --------------- | ------------------------------------ |
| Run during dev  | `bun run src/index.ts <args>`        |
| Unit tests      | `bun test`                           |
| Lint + format   | `bun x biome check .` (`--write` to fix) |
| Typecheck       | `bunx tsc --noEmit`                  |
| Build npm bundle| `bun run build`                      |
| Cross-compile   | `bun run build:bin:all`              |

See [AGENTS.md](./AGENTS.md) for the full contributor guide and architecture
notes, and [CONTRIBUTING.md](./CONTRIBUTING.md) for PR expectations.

## Security

Found a vulnerability? Please **do not** open a public issue — email
[shivam@oriyn.ai](mailto:shivam@oriyn.ai) instead. See [SECURITY.md](./SECURITY.md)
for the full policy.

## License

[Apache 2.0](./LICENSE) © Pontis Inc.
