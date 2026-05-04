# Contributing to oriyn

Thanks for picking this up. This file is the short version for drive-by
contributors. The full architectural guide lives in [AGENTS.md](./AGENTS.md) —
read it before changing anything non-trivial.

## Getting set up

Requires **Bun ≥ 1.2**.

```bash
git clone https://github.com/oriyn-ai/cli.git && cd cli
bun install
bun run src/index.ts --help
```

## Day-to-day

| Task            | Command                                  |
| --------------- | ---------------------------------------- |
| Run during dev  | `bun run src/index.ts <args>`            |
| Unit tests      | `bun test`                               |
| Lint + format   | `bun x biome check .` (`--write` to fix) |
| Typecheck       | `bunx tsc --noEmit`                      |
| Build npm bundle| `bun run build`                          |

CI runs all four on every push; please run them locally before opening a PR.

## Pull requests

- Branch from `main`. Keep PRs focused — one concern per PR.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for the PR
  title (`feat:`, `fix:`, `chore:`, `docs:`, etc.). The release pipeline reads
  these to generate the changelog.
- Add a unit test for any pure logic; an integration test for any new HTTP
  path.
- If you add a non-trivial dependency or make a structural decision, drop a
  short note in `decisions/`.

## Adding a command

The pattern is documented in [AGENTS.md](./AGENTS.md#command-surface). The
short version:

1. Create `src/commands/<name>.ts` exporting `register<Name>(parent: Command)`.
2. Wire it into `src/index.ts`.
3. Use `requireProduct()` if it needs a linked product, `resolveMode()` to
   branch human vs JSONL output, and `reportAndExit()` for errors.

## Reporting issues

Use [GitHub Issues](https://github.com/oriyn-ai/cli/issues). Include:

- `oriyn --version`
- Your OS + Bun version (`bun --version`)
- The exact command, with secrets redacted
- Output of `oriyn status` if relevant

## Code of conduct

Be kind. Assume good faith. We follow the spirit of the
[Contributor Covenant](https://www.contributor-covenant.org/).

## License

By contributing, you agree that your contributions will be licensed under the
[Apache 2.0 License](./LICENSE).
