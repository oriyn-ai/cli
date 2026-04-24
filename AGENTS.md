<product>
Oriyn helps product teams understand user behavior, generate hypotheses, and run persona-grounded simulations before shipping changes.
</product>

<workflow>
- `go build ./...` - build
- `go test ./...` - run tests
- `go vet ./...` - lint
- `go install ./...` - install locally for manual testing
</workflow>

<rules>
- Prefer self-documenting code over explanatory comments.
- Keep comments only for public Go doc requirements, non-obvious contracts, security boundaries, external API quirks, concurrency, idempotency, or tooling requirements.
- The CLI is a thin client over `oriyn-api`.
- Shared product behavior belongs in `oriyn-api`, not the CLI.
- Commands write to `cmd.OutOrStdout()` so output is capturable in tests.
- API tokens live in the OS keychain; `ORIYN_ACCESS_TOKEN` is a CI escape hatch.
- No new dependencies without explicit justification logged in `/decisions/`.
</rules>

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health
