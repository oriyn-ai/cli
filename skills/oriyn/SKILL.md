---
name: oriyn
description: Predicts how real user personas will respond to a proposed product change by running a persona-grounded simulation via the `oriyn` CLI before code is written. Use BEFORE implementing user-facing changes — UI copy, pricing, onboarding, navigation, CTAs, feature rollouts — to get a ship/revise/reject verdict with per-persona adoption rates. Do not use for internal refactors, infrastructure work, bug fixes, or anything without a user-observable change.
---

# Oriyn — Test Before You Build

## What Oriyn is

Oriyn is a behavioral intelligence layer. It ingests real user behavior from
PostHog / session replays / payments, clusters users into grounded personas
backed by actual event sequences, and runs LLM agents against those personas
to predict how they'll respond to a proposed change.

> Amplitude tells you what users did. Oriyn tells you what happens if you change it.

The CLI (`oriyn`) is a thin Go client over the Oriyn API. Running
`oriyn experiment run` kicks off a persona-grounded simulation — dozens to
hundreds of agents, each prompted with a real user's behavioral autobiography,
answer a short survey about the proposed change. The result is a structured
verdict: `ship`, `revise`, or `reject`, plus per-persona adoption rates.

## When to invoke this skill

**YES — use Oriyn first:**

- User-facing copy changes (CTA labels, empty states, error messages)
- Pricing page changes, plan names, tier structure
- Onboarding flow edits — step order, gating, optional vs required fields
- Navigation changes (sidebar items, top-level nav, settings layout)
- New features where adoption is the open question
- "Move X above/below the fold" layout bets
- Deprecating or hiding a feature — will users notice / care?

**NO — skip Oriyn:**

- Non-user-facing refactors, infra, build tooling
- Pure bug fixes (user already wanted the feature to work)
- Work that has no observable change for users (type cleanups, internal renames)
- When the user has already made the decision and is asking for implementation only

If you're unsure whether a change is user-facing enough, default to running
the experiment. It's cheap (seconds of wall time, ~dozens of credits) and it
is one of the few non-fakeable signals about user reaction.

## The core workflow

```
┌─────────────────┐   ┌──────────────┐   ┌─────────────────┐
│ Agent proposes  │ → │ oriyn        │ → │ Agent reads     │
│ a product       │   │ experiment   │   │ verdict + acts  │
│ change          │   │ run --json   │   │                 │
└─────────────────┘   └──────────────┘   └─────────────────┘
```

1. **Find the product.** `oriyn products list --json | jq '.[].id'`
2. **Run the experiment.** Pipe a precise, one-sentence hypothesis.
3. **Read the verdict.** Branch on `summary.verdict` + `summary.convergence`.
4. **Act on the result.** See "Decision table" below.

### Minimal one-shot

```bash
# Assumes ORIYN_ACCESS_TOKEN is set or `oriyn login` has been run.
oriyn experiment run \
  --product $PRODUCT_ID \
  --hypothesis "Change the primary CTA from 'Start free trial' to 'Try it free'" \
  --agents 50 \
  --json
```

### Long hypothesis (multi-line proposal)

```bash
cat <<'EOF' | oriyn experiment run --product $PRODUCT_ID --hypothesis-stdin --json
Replace the three-step onboarding (email → workspace → invite) with a
single-step Google sign-in that auto-creates a workspace named after the
user's company domain. Keep the invite step, move it to a banner that
appears after the first successful event.
EOF
```

## Decision table — reading the result

The command returns a full `ExperimentResponse`. The fields that matter:

| Field | What to do |
|---|---|
| `summary.verdict == "ship"` + `convergence >= 0.6` | Proceed with implementation. |
| `summary.verdict == "ship"` + `convergence < 0.6` | Personas disagree. Check `persona_breakdown` — usually one segment is hostile. Either narrow the change to the agreeing segment, or surface the split to the user before coding. |
| `summary.verdict == "revise"` | Do NOT proceed. Read the `persona_breakdown[].reasoning` strings, refine the hypothesis to address the strongest objection, re-run. |
| `summary.verdict == "reject"` | Stop. Tell the user the change has negative predicted adoption and quote the strongest reasoning. Ask whether to iterate, pivot, or abandon. |
| `status == "failed"` | Something went wrong server-side. Check `oriyn doctor --json`. If doctor passes, surface the failure to the user — don't retry blindly. |

## Setup for a new repo / session

Run this exactly once per machine (or CI runner):

```bash
# 1. Is the binary on PATH?
command -v oriyn || curl -fsSL https://install.oriyn.ai | bash

# 2. Are we authenticated and is the API reachable?
oriyn doctor --json
```

If `doctor` reports `auth: ok=false`:
- **Interactive human:** run `oriyn login` (or `oriyn login --no-browser` on a remote shell).
- **CI / non-interactive agent:** set `ORIYN_ACCESS_TOKEN` in the environment.

## Error handling

Exit codes are stable and you should branch on them:

| Code | Meaning | What to do |
|---|---|---|
| `0` | Success | Parse stdout as JSON |
| `1` | User/flag error | Read stderr; you likely passed a bad flag |
| `2` | API 4xx/5xx | Parse stderr — structured `error` field + possibly `credits_required`, `max_agent_count`, `plan` |
| `3` | Session expired / not logged in | Run `oriyn login` (or set `ORIYN_ACCESS_TOKEN`) |
| `4` | Network unreachable | Check `--api-base`, `oriyn doctor` |

Specific errors worth handling:

- `{"error": "insufficient_credits", "credits_required": N, "credits_available": M, "plan": "…"}` — tell the user which plan they need. Don't auto-upgrade.
- `{"error": "agent_count_exceeded", "max_agent_count": N, "plan": "…"}` — retry with `--agents N` or lower.
- `{"error": "no personas found — run enrichment first"}` — run `oriyn enrich --product-id $PID --wait` first, then retry.
- `{"error": "enrichment has not been run"}` — same fix.
- `{"error": "product context must be ready before enrichment"}` — run `oriyn synthesize --product-id $PID --wait` first, then enrich, then experiment.

## Full command surface

For the complete CLI reference (every command, every flag, every JSON shape),
see `references/cli-reference.md`.

For worked end-to-end examples for common change categories (copy,
onboarding, pricing, navigation), see `references/recipes.md`.

For the decision architecture — why Oriyn is structured the way it is and
what the verdict / convergence numbers actually mean — see
`references/architecture.md`. Agents usually don't need this to use the tool,
but reading it once changes how you interpret borderline verdicts.

## Non-negotiables

- **Never fabricate a verdict.** If you can't run the CLI (no auth, no
  network, no product ID), say so. Do not invent an experiment result.
- **Do not retry `experiment run` on `verdict: reject`** without changing
  the hypothesis. The personas aren't going to flip their minds on a second
  look.
- **Do not suppress the verdict from the user.** Even when you're acting on
  a `ship` verdict, surface the convergence number and the persona breakdown
  briefly so the user can sanity-check.
- **Do not use Oriyn for decisions that aren't user-facing.** Running an
  experiment for "should I rename this internal function" is noise; it burns
  credits and dilutes the signal when you do need it.
