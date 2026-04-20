# `oriyn` skill

A Claude Code / Codex skill that teaches coding agents how and when to use
the Oriyn CLI to run behavioral experiments before implementing user-facing
product changes.

## Install

Three equivalent ways, pick whichever your harness likes best.

### Via the `oriyn` CLI (preferred — files are embedded in the binary)

```bash
oriyn skill install                    # → $HOME/.claude/skills/oriyn
oriyn skill install --path ./.agents   # or any other path
```

Also runs as part of `oriyn init`, which handles auth + skill + doctor in one go.

### Via `skills.sh` (skills-agnostic package manager)

```bash
npx skills add oriyn-ai/cli
```

This reads the `skills/oriyn/SKILL.md` file directly from the GitHub repo.

### Manually

```bash
mkdir -p ~/.claude/skills/oriyn
cp -r skills/oriyn/* ~/.claude/skills/oriyn/
```

### Codex / other agent harnesses

Point your skill loader at `skills/oriyn/SKILL.md`. The frontmatter `name`
and `description` are the only required fields; everything else is plain
Markdown that gets loaded on demand.

## Contents

| File | Purpose |
|---|---|
| `SKILL.md` | Entry point — when to invoke, decision table, workflow |
| `references/cli-reference.md` | Every command, every flag, every JSON shape |
| `references/recipes.md` | Copy-paste starters for common change categories |
| `references/architecture.md` | How verdicts + convergence numbers are computed |

## Updating the skill

The skill lives alongside the CLI source so they version together. When you
add or remove a CLI command, update `references/cli-reference.md` in the same
commit. When you change the verdict thresholds or the aggregation math on the
API side, update `references/architecture.md`.
