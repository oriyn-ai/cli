# Oriyn CLI

> Predict how users will respond to a change before you ship it.

Oriyn builds persona-grounded simulations from real user behavior so you can
test a UI change, a pricing tweak, or a new onboarding flow *before* writing
the code. The CLI is the primary way coding agents (Claude Code, Codex) and
humans run experiments from a terminal or CI pipeline.

This repo is a thin Go client over `oriyn-api` — all business logic lives there.

---

## Install

### Quick install (macOS / Linux)

```bash
curl -fsSL https://oriyn.ai/install.sh | bash
```

### From source

```bash
go install github.com/oriyn-ai/cli@latest
```

### From GitHub Releases

Grab the right binary from the [latest release](https://github.com/oriyn-ai/cli/releases/latest),
make it executable, put it on your `PATH`.

---

## Authenticate

```bash
oriyn login                 # interactive browser OAuth
oriyn login --no-browser    # headless — prints the URL to open manually
```

Tokens are stored in the OS keychain (macOS Keychain / Windows Credential Manager
/ libsecret on Linux). For non-interactive contexts (CI, coding agents, remote
shells), set `ORIYN_ACCESS_TOKEN` and skip `login` entirely.

```bash
oriyn whoami
oriyn doctor        # checks auth + API reachability (exits non-zero on failure)
```

---

## Core workflow: run an experiment before shipping a change

This is why the CLI exists.

```bash
# 1. Find the product
oriyn products list

# 2. Run the experiment — blocks until complete, emits JSON
oriyn experiment run \
  --product $PRODUCT_ID \
  --hypothesis "Move the trial CTA above the fold on /pricing" \
  --json

# Or pipe the hypothesis for longer proposals
cat proposal.md | oriyn experiment run --product $PRODUCT_ID --hypothesis-stdin --json
```

The JSON payload is the full `ExperimentResponse`:

```json
{
  "id": "…",
  "status": "complete",
  "summary": {
    "verdict": "ship" | "revise" | "reject",
    "convergence": 0.84,
    "summary": "…",
    "persona_breakdown": [
      {"persona": "Power users", "response": "strong_positive", "adoption_rate": 0.72, "reasoning": "…"}
    ],
    "question_results": { … },
    "agent_count": 50
  }
}
```

Agents branch on `summary.verdict` to decide whether to proceed with the
implementation. `convergence` below ~0.6 means the personas disagree — treat
the verdict as low-confidence and widen the proposal.

---

## Command reference

| Command | Purpose |
|---|---|
| `oriyn login` / `logout` / `whoami` | Auth (OAuth + keychain) |
| `oriyn doctor` | Verify env, auth, API reachability |
| `oriyn products list` / `ls` | List products in your org |
| `oriyn products get --product-id <id>` | Product details |
| `oriyn products context show / edit / history / version` | Inspect + edit synthesized context |
| `oriyn products scrape --product-id <id> --source-id <sid>` | Kick off a Firecrawl scrape of a source |
| `oriyn synthesize --product-id <id> [--wait]` | Trigger product-context synthesis |
| `oriyn enrich --product-id <id> [--wait]` | Trigger persona enrichment |
| `oriyn personas --product-id <id>` | Behavioral personas |
| `oriyn personas profile --product-id <id> --persona-id <pid>` | Supermemory-derived persona facts |
| `oriyn personas citations --product-id <id> --persona-id <pid> --trait-index N` | Evidence sessions for a trait |
| `oriyn hypotheses --product-id <id>` | Testable sequences mined from events |
| `oriyn knowledge search --product-id <id> --query "…"` | Semantic search over product knowledge graph |
| `oriyn timeline --product-id <id> --user-id <uid>` | Cross-provider timeline for one resolved user |
| `oriyn replay --product-id <id> --session-asset-id <sid>` | Raw rrweb events for a session |
| `oriyn experiment run / list / get / archive` | Simulation lifecycle |
| `oriyn telemetry --enable / --disable / --status` | Anonymous usage telemetry |

Every read-only command supports `--json`.

---

## Agent mode

Three equivalent ways to switch the CLI into machine-readable mode:

1. `--json` on any command that supports it.
2. `--quiet` on the root (suppresses color, progress dots, headers).
3. `ORIYN_AGENT=1` in the environment (implies the above globally).

Exit codes so agents can branch without parsing messages:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | User error (flag misuse, missing required input) |
| `2` | API returned 4xx/5xx — inspect the `error` field in stderr |
| `3` | Not logged in or session expired |
| `4` | Could not reach the API |

Error responses from the API are preserved: `402 insufficient_credits` returns
`credits_required` / `credits_available` / `plan`; `403 agent_count_exceeded`
returns `max_agent_count`. Agents can read these to self-correct (reduce
`--agents` or surface a billing prompt).

---

## Environment variables

| Variable | Purpose |
|---|---|
| `ORIYN_ACCESS_TOKEN` | Bypass keychain; use this token directly (CI/agent escape hatch) |
| `ORIYN_API_BASE` | Override API base URL (default `https://api.oriyn.ai`) |
| `ORIYN_WEB_BASE` | Override web app URL (default `https://app.oriyn.ai`) |
| `ORIYN_AGENT` | Set to `1` to force agent mode globally |
| `ORIYN_TELEMETRY` | Set to `0` / `false` / `off` to disable anonymous usage telemetry |

---

## For coding agents

If you're Claude Code or Codex, install the Oriyn skill so decision triggers
and worked examples are available on demand:

```bash
# Once the CLI is installed, one command does auth + skill + health check
oriyn init

# Or install just the skill:
oriyn skill install             # → ~/.claude/skills/oriyn
oriyn skill install --path .    # into the current repo

# Or use the skills.sh package manager, which reads from GitHub directly
npx skills add oriyn-ai/cli
```

See [`skills/oriyn/SKILL.md`](./skills/oriyn/SKILL.md) in this repo for the
full skill source. The skill files are also embedded in the `oriyn` binary,
so `oriyn skill install` works offline.

---

## Development

```bash
go build ./...      # compile
go test ./...       # run tests
go vet ./...        # lint
```
