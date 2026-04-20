# Recipes — Worked Examples

Copy-paste starting points for common change categories. Each recipe shows
the full shell flow an agent runs end to end, with the JSON you should
parse and the decision logic downstream of the verdict.

## Contents

- Pattern: CTA copy change — short one-line hypothesis, 50 agents
- Pattern: pricing-page restructure — high-stakes, 200 agents, inspect breakdown
- Pattern: multi-line hypothesis from stdin — onboarding / flow redesign
- Pattern: "no personas — enrichment not run" — synthesize → enrich → experiment chain
- Pattern: read the hypotheses Oriyn already mined — agent picks from `oriyn hypotheses`
- Pattern: check credits before running — handle `insufficient_credits` / `agent_count_exceeded`
- Pattern: one-shot in CI — gate PR merges on Oriyn verdict

## Pattern: CTA copy change

**Situation:** the user asked you to change a button label. Before editing
the code, run an experiment.

```bash
PRODUCT_ID=$(oriyn products list --json | jq -r '.[0].id')

oriyn experiment run \
  --product "$PRODUCT_ID" \
  --hypothesis "Change the pricing-page CTA from 'Start free trial' to 'Try it free'" \
  --agents 50 \
  --json > /tmp/exp.json

VERDICT=$(jq -r '.summary.verdict' /tmp/exp.json)
CONVERGENCE=$(jq -r '.summary.convergence' /tmp/exp.json)

case "$VERDICT" in
  ship)   echo "Shipping. Convergence=$CONVERGENCE" ;;
  revise) jq '.summary.persona_breakdown[] | select(.response | startswith("neg")) | .reasoning' /tmp/exp.json ;;
  reject) echo "Do not ship. Top objection:"; jq '.summary.persona_breakdown[0].reasoning' /tmp/exp.json ;;
esac
```

The agent typically then either (a) implements the change, (b) proposes a
revised wording to the user and re-runs, or (c) reports the objection.

## Pattern: pricing-page restructure

Pricing changes have high blast radius — run more agents (more stable
verdict) and inspect the per-persona breakdown carefully.

```bash
oriyn experiment run \
  --product "$PRODUCT_ID" \
  --hypothesis "Rename the 'Team' plan to 'Business' and raise the price from \$29 to \$49/seat, keeping all features the same" \
  --agents 200 \
  --json \
  | jq '{
      verdict: .summary.verdict,
      convergence: .summary.convergence,
      by_persona: .summary.persona_breakdown
    }'
```

If `convergence < 0.6`, the personas disagree — that's the signal, not the
noise. Report the split to the user instead of proceeding.

## Pattern: multi-line hypothesis from stdin

For onboarding or flow redesigns, the hypothesis is usually a paragraph.
Pipe it in — `--hypothesis` becomes unwieldy for long strings, and shell
quoting gets error-prone.

```bash
cat <<'EOF' | oriyn experiment run --product "$PRODUCT_ID" --hypothesis-stdin --json
Collapse the three onboarding steps (email → workspace → invite) into a
single Google sign-in that auto-provisions a workspace named after the user's
email domain. Move the team-invite step to a post-signup banner that appears
after the first successful project creation.
EOF
```

## Pattern: "no personas — enrichment not run"

First run against a new product fails with `enrichment has not been run`.
Chain synthesize → enrich → experiment. Each step can take minutes; use
`--wait` and set the timeout high enough.

```bash
oriyn synthesize --product-id "$PRODUCT_ID" --wait --timeout 5m
oriyn enrich     --product-id "$PRODUCT_ID" --wait --timeout 10m
oriyn experiment run --product "$PRODUCT_ID" --hypothesis "…" --json
```

## Pattern: read the hypotheses Oriyn already mined

Sometimes the user says "run an experiment on something interesting you
find." Let Oriyn suggest: `oriyn hypotheses` returns testable sequences
mined from real event data. Pick one with high `user_count` /
`significance_pct`, render it into a hypothesis, run it.

```bash
oriyn hypotheses --product-id "$PRODUCT_ID" --json \
  | jq 'map(select(.user_count >= 10)) | sort_by(-.significance_pct) | .[0]'
```

## Pattern: check credits before running

Agent-count overrides cost credits. Before fanning out 1000 agents, check
the plan.

```bash
RESULT=$(oriyn experiment run --product "$PRODUCT_ID" --hypothesis "…" --agents 1000 --json 2>&1)
EXIT=$?

if [ $EXIT -eq 2 ]; then
  # API error — parse the JSON that the API sent back
  echo "$RESULT" | jq
  # Expect: {"error":"insufficient_credits","credits_required":X,"credits_available":Y,"plan":"…"}
  # or:     {"error":"agent_count_exceeded","max_agent_count":N,"plan":"…"}
fi
```

If `max_agent_count` is returned, retry with `--agents <max_agent_count>`.
If `credits_required > credits_available`, do NOT auto-upgrade — surface the
gap to the user with a clear "you need the X plan" message.

## Pattern: one-shot in CI

For a repo's CI pipeline to gate merges on Oriyn verdict:

```bash
# In a workflow step, with ORIYN_ACCESS_TOKEN set from a secret:
oriyn doctor --json >/dev/null || exit 3

VERDICT=$(oriyn experiment run \
  --product "$ORIYN_PRODUCT_ID" \
  --hypothesis "$(cat proposal.md)" \
  --agents 50 \
  --json | jq -r '.summary.verdict')

case "$VERDICT" in
  ship)   exit 0 ;;
  revise) exit 1 ;;  # block merge; humans iterate
  reject) exit 1 ;;
  *)      exit 2 ;;
esac
```
