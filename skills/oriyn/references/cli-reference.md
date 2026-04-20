# Oriyn CLI — Complete Reference

Every command the agent can invoke. All read-only commands support `--json`,
and every command respects `ORIYN_AGENT=1` / `--quiet` for non-interactive use.

## Contents

- Global flags
- Auth & diagnostics — `login`, `logout`, `whoami`, `doctor`
- Products — `products list`, `products get`, `products context`, `products scrape`
- Synthesis & enrichment — `synthesize`, `enrich`
- Personas — `personas`, `personas profile`, `personas citations`
- Hypotheses — `hypotheses`
- Knowledge graph — `knowledge search`
- Timeline & replay — `timeline`, `replay`
- Experiments — `experiment run`, `experiment list`, `experiment get`, `experiment archive`
- Telemetry — `telemetry`

## Global flags

| Flag | Default | Purpose |
|---|---|---|
| `--api-base` | `https://api.oriyn.ai` | Override the API base URL (also `ORIYN_API_BASE`) |
| `--web-base` | `https://app.oriyn.ai` | Override the web app base URL (also `ORIYN_WEB_BASE`) |
| `--quiet` | `false` | Suppress non-essential output; implies `--json` behavior |

## Auth & diagnostics

### `oriyn login [--no-browser]`
Starts a local OAuth callback server and opens the browser (or prints the URL
with `--no-browser`). Stores tokens in the OS keychain. Not needed if
`ORIYN_ACCESS_TOKEN` is set.

### `oriyn logout`
Removes stored credentials.

### `oriyn whoami`
Prints the authenticated user. Good liveness check inside scripts.

### `oriyn doctor [--json]`
One-shot health report: auth present, API reachable, `/v1/me` succeeds.
Exits non-zero on any failure. JSON shape:

```json
{
  "ok": false,
  "version": "0.3.0",
  "commit": "…",
  "os": "darwin",
  "arch": "arm64",
  "api_base": "https://api.oriyn.ai",
  "checks": [
    {"name": "auth",          "ok": true,  "detail": "token present"},
    {"name": "api-reachable", "ok": true,  "detail": "api version 0.1.0"},
    {"name": "whoami",        "ok": true,  "detail": "user@example.com"}
  ]
}
```

## Products

### `oriyn products list` / `oriyn products ls [--json]`
List products your org owns.

```json
[{"id":"…","name":"Acme","context_status":"ready"}]
```

### `oriyn products get --product-id <id> [--json]`
Full product detail — `id`, `name`, `context_status`, `enrichment_status`,
`created_at`, and (when synthesized) a typed `context` object with
`company`, `product_summary`, `core_features`, `target_users`,
`value_proposition`, `use_cases`.

### `oriyn products context show --product-id <id> [--json]`
Just the `context` portion of the product.

### `oriyn products context edit --product-id <id> [--field … --value …] [--json-body …] [--json]`
Patch context fields. For scalar edits use `--field` + `--value`. For list
fields (`core_features`, `use_cases`) or multi-field patches use `--json-body`
(or `--json-body -` to read from stdin).

### `oriyn products context history --product-id <id> [--json]`
List all context versions (manual edits + synthesis snapshots).

### `oriyn products context version --product-id <id> --version-id <vid> [--json]`
Fetch a specific version's snapshot.

### `oriyn products scrape --product-id <id> --source-id <sid>`
Kick off a Firecrawl scrape for a source row. Returns `202 {"status":"pending"}`.

## Synthesis & enrichment

### `oriyn synthesize --product-id <id> [--wait] [--timeout DUR] [--poll-interval DUR]`
Triggers LLM product-context synthesis. With `--wait`, blocks until
`context_status` reaches `ready` or `failed`.

### `oriyn enrich --product-id <id> [--wait] [--timeout DUR] [--poll-interval DUR]`
Triggers persona clustering. Requires `context_status == "ready"`. With
`--wait`, blocks until `enrichment_status` is `ready` or `failed`.

## Personas

### `oriyn personas --product-id <id> [--json]`
List all personas for a product. JSON shape:

```json
{
  "enrichment_status": "ready",
  "data": [
    {
      "id": "…",
      "name": "Power users",
      "description": "…",
      "behavioral_traits": ["logs in daily", "creates > 5 workspaces"],
      "size_estimate": 18,
      "status": "active",
      "generated_at": "…",
      "updated_at": "…",
      "trait_citation_counts": [12, 4, 7]
    }
  ]
}
```

### `oriyn personas profile --product-id <id> --persona-id <pid> [--json]`
Supermemory-derived profile:
- `static_facts` — long-term identity claims (job, team size, domain)
- `dynamic_facts` — recent simulation-context facts

### `oriyn personas citations --product-id <id> --persona-id <pid> --trait-index N [--json]`
List the evidence sessions that back persona's Nth behavioral trait. Each
citation includes `external_session_id`, `session_summary`, `frustration_score`,
`duration_ms`, `replay_url` (when configured), and `has_stored_replay`.

## Hypotheses

### `oriyn hypotheses --product-id <id> [--json]`
Mine-on-read testable sequences — recurring cross-provider event patterns
shared by multiple users. Each item:

```json
{
  "sequence": ["pageview:/pricing", "event:trial_start", "event:plan_upgrade"],
  "rendered_sequence": ["Pricing page", "Trial start", "Upgrade plan"],
  "frequency": 42,
  "user_count": 17,
  "significance_pct": 4.1,
  "source_users": ["…"]
}
```

Use these as experiment seeds when you don't have a specific hypothesis in mind.

## Knowledge graph

### `oriyn knowledge search --product-id <id> --query "…" [--limit 10] [--threshold 0.5] [--rerank] [--json]`
Semantic search across the product's Supermemory graph. Returns chunks with
`content`, `score`, `metadata`, `created_at`.

## Timeline & replay

### `oriyn timeline --product-id <id> --user-id <uid> [--limit 60] [--json]`
Cross-provider per-user timeline — events + session replays + revenue, joined
in chronological order. Session-kind items carry `session_summary`,
`frustration_score`, `session_asset_id`.

### `oriyn replay --product-id <id> --session-asset-id <sid> [--json]`
Raw rrweb events for a stored session. Payloads are not human-readable —
treat as opaque arrays and ship them to a replay player (or save to disk).

## Experiments

### `oriyn experiment run --product <id> (--hypothesis "…" | --hypothesis-stdin) [--agents N] [--no-wait] [--poll-interval DUR] [--timeout DUR] [--json]`
Creates an experiment, fans out simulation agents against personas, waits for
completion. Returns the full `ExperimentResponse`:

```json
{
  "id": "…",
  "product_id": "…",
  "hypothesis": "…",
  "status": "complete",
  "created_by_email": "…",
  "created_at": "…",
  "summary": {
    "verdict": "ship" | "revise" | "reject",
    "convergence": 0.84,
    "summary": "human-readable prose summary",
    "persona_breakdown": [
      {
        "persona": "Power users",
        "response": "strong_positive",
        "adoption_rate": 0.72,
        "reasoning": "…"
      }
    ],
    "question_results": { "adoption_likelihood": {...}, "primary_reaction": {...}, ... },
    "agent_count": 50
  }
}
```

Use `--no-wait` to return immediately with just `{"experiment_id": "…"}`,
then poll `oriyn experiment get` yourself.

### `oriyn experiment list --product <id> [--json]`
List all experiments for a product with `id`, `title`, `hypothesis`,
`status`, `verdict`, `convergence`, `created_by_email`, `created_at`.

### `oriyn experiment get --product <id> --experiment <eid> [--json]`
Full detail for one experiment.

### `oriyn experiment archive --product <id> --experiment <eid>`
Archive a completed experiment (hides from default list views).

## Telemetry

### `oriyn telemetry --enable | --disable | --status`
Controls anonymous PostHog usage telemetry. Dev builds (`version == "dev"`)
never send telemetry regardless of setting.
