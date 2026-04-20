# Oriyn — Architecture & Signal Provenance

Optional reading for agents. You can use the CLI effectively without this.
The material below changes how you interpret borderline verdicts and weak
convergence scores — worth a read if you're running Oriyn in a high-stakes
loop (merge gate, auto-refactor).

## Contents

- The data pipeline — PostHog + replays + payments → ClickHouse → personas → simulation
- How the verdict is computed — weighted tallies, entropy convergence, threshold-based verdict
- What this means for agent behavior — when to trust, when to distrust, how to re-run
- Cross-provider wedge — why Oriyn's signal is non-fakeable
- What the CLI doesn't do — caching, streaming, Supabase access
- What to do if something feels wrong — doctor, product status, persona status

## The data pipeline

```
PostHog events ─┐
Session replays ├─► ClickHouse (raw events, TTL'd)
Payments       ─┤
                │
                ▼
         session_signals        (append-only, per-session)
         user_behavioral_profiles  (upsert, per resolved user)
                │
                ▼
         LLM enrichment → personas
         (each persona carries source_users: a list of resolved user IDs)
                │
                ▼
         simulation (agents drawn from source_users, prompted with
                    their behavioral autobiography)
                │
                ▼
         aggregation (pure Python, no LLM) → verdict + convergence
```

Key invariant: agents are **survey respondents**, not role-players. Each
agent's system prompt is a two-layer prose:
1. The persona description (grounded LLM prose from enrichment).
2. One sampled real user's `user_behavioral_profile` rendered to first-person
   prose via deterministic Python formatting — no LLM in this step.

Agents only return categorical choices (JSON schema with enum constraints).
No numbers, scores, or confidence values come from the LLM anywhere in the
simulation path.

## How the verdict is computed

Pure Python aggregation lives server-side in `services/aggregation.py`. The
agent (you) does not see the raw per-agent answers — only the aggregated
summary. The math:

- **Weighted tallies.** Each agent's vote is weighted by its persona's
  `size_estimate / sum(size_estimates)`, then split across the agents drawn
  from that persona. Big personas dominate, as they should.
- **Entropy-based convergence.** `1 - (entropy / log(k))` per question,
  averaged across questions. 0 means the personas are 50/50; 1 means
  unanimous. Below ~0.6 is "personas disagree" territory.
- **Verdict thresholds.** `ship` if ≥ 60% weighted positive adoption,
  `reject` if ≥ 40% weighted negative, else `revise`.
- **Representative reasoning via TF cosine centroid.** The `reasoning`
  string surfaced per persona is the one closest to the cluster centroid —
  it's the *most-typical* justification, not a cherry-pick.
- **Min completion rate = 80%.** If fewer than 80% of agents return a valid
  answer, the run fails rather than returning a low-confidence verdict.

## What this means for agent behavior

- **Trust `ship` verdicts with `convergence >= 0.6`.** The math is stable
  and the agents you're talking to aren't the ones producing the vote.
- **Distrust `ship` verdicts with `convergence < 0.6`.** The aggregate says
  yes but the personas split. Read `persona_breakdown` — one segment is
  usually dragging the average.
- **Never claim a verdict is "close to shipping" or "almost a reject."**
  The thresholds are discrete. Either it's above, or it isn't.
- **Re-running an identical hypothesis is noise.** The personas are stable;
  the agents will give you roughly the same answer. If the verdict was
  `revise`, change the hypothesis — don't re-roll.

## Cross-provider wedge (why this works)

Oriyn's unique signal comes from crossing providers. Amplitude sees clicks.
Stripe sees payments. FullStory sees replays. No single provider has the
sequence "upgrade page view → trial click → frustration replay →
cancellation 6 days later." Oriyn's ClickHouse layer joins them by
resolved user identity, which means the hypotheses mined from
`oriyn hypotheses` are things no single provider's UI could show you.

## What the CLI doesn't do

- The CLI does not call Supabase directly. Every operation flows through
  `oriyn-api`.
- The CLI does not generate hypotheses itself. LLM work happens server-side.
- The CLI does not cache. Every read hits the API. Rate limits apply.
- The CLI does not stream — `experiment run` polls at your chosen
  `--poll-interval`. The `--no-wait` flag lets you poll on your own.

## What to do if something feels wrong

1. Run `oriyn doctor --json`. Auth + API + `/v1/me` — if any fail, fix those
   first.
2. Check `oriyn products get --product-id $PID` — `context_status` and
   `enrichment_status` should both be `"ready"`.
3. Check `oriyn personas --product-id $PID` — each persona should have a
   non-zero `size_estimate` and a populated `source_users` (present but
   not shown in the CLI output; failure to populate raises a clear error
   on the experiment side).
4. If `experiment run` consistently fails with
   `"no personas found — run enrichment first"` but enrichment has in fact
   run, inspect the API logs server-side (not your job as an agent —
   surface to the user).
