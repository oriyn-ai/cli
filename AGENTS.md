# oriyn CLI — agent guide

TypeScript CLI on Bun. Replaces the previous Go implementation.

<workflow>
- `bun install` — install deps
- `bun run src/index.ts <args>` — run during development (no transpile step)
- `bun test` — run unit + integration tests
- `bun x biome check .` — lint, format, import sort (one tool, owns formatting)
- `bun x biome check --write .` — autofix
- `bunx tsc --noEmit` — typecheck only (Bun handles emission)
- `bun build src/index.ts --target=bun --format=esm --outfile=dist/index.js --minify` — npm bundle
- `bun run scripts/build-binaries.ts` — cross-compile standalone binaries for darwin/linux/windows × x64/arm64
</workflow>

<storage>
- Auth credentials: `~/.config/oriyn/credentials.json` (mode 0600). NEVER fall back to OS keychain — that's the legacy Go pattern, gone.
- CLI prefs: `~/.config/oriyn/config.json` (mode 0644).
- Project link: `<repo>/oriyn.json` (mode 0644, committed to repo).
- `ORIYN_ACCESS_TOKEN` is the CI escape hatch — bypasses the credentials file entirely.
- `ORIYN_CONFIG_DIR` overrides the global config dir for tests.
</storage>

<rules>
- Bun-first. Use `Bun.serve`, `Bun.file`, native `fetch` where they make code clearer than the Node equivalent.
- TS strict mode + `noUncheckedIndexedAccess`. No `any`.
- Validate API responses with zod at the boundary. Trust internal types.
- Output mode: TTY → human (colors, tables, spinners); non-TTY → JSONL (one event per line). Override with `--human`.
- Never log secrets. All bearer tokens, JWTs, refresh tokens get scrubbed via `src/http/redact.ts` before stderr/Sentry/PostHog.
- Telemetry is silent opt-in: one-line announcement on first use, env override (`ORIYN_TELEMETRY=off`), CI auto-skip via vendor table. Never block on a prompt.
- Sentry + PostHog stay off in dev (`VERSION === '0.0.0-dev'`) and CI.
- Exit codes: `0` ok, `1` generic, `2` api, `3` auth, `4` network, `5` permission. Use `src/lib/handle-error.ts` to map errors → exit codes.
- Self-documenting code over comments. Comments only for non-obvious contracts (security boundaries, OAuth quirks, atomic-write semantics, race conditions).
- No new dependencies without justification logged in `decisions/`.
</rules>

<command-surface>
- `oriyn auth {login,logout,whoami,status}`
- `oriyn link [--product <id>] [--force]`, `oriyn unlink`
- `oriyn products`
- `oriyn personas [id] [--product <id>]`
- `oriyn patterns [--only hypothesis|bottleneck]`
- `oriyn experiments [id]`, `oriyn experiments run "<hypothesis>" [--agents N] [--product <id>]`
- `oriyn sync [--only synthesize|enrich]`
- `oriyn status`
- `oriyn config [key] [value]`
- `oriyn open [resource]`
- `oriyn upgrade`
- `oriyn completion <shell>`

When adding a new command:
1. Create `src/commands/<name>.ts` (or `<name>/index.ts` for a namespace) exporting `register<Name>(parent: Command): void`.
2. Wire it into `src/index.ts` eagerly (no lazy import — Bun's startup is fast enough).
3. Use `requireProduct()` from `src/lib/require-product.ts` for any command that needs a linked product.
4. Use `resolveMode()` to branch human vs JSONL output. Use `createJsonlEmitter()` for streams; `writeJson()` for one-shot.
5. Wrap actions in try/catch and call `reportAndExit(err)`. Never call `process.exit` directly except inside that helper.
6. Add a unit test for the command's pure logic; integration tests for HTTP-touching paths.
</command-surface>

<auth-flow>
OAuth 2.1 + PKCE direct to Clerk:

1. `generatePkce()` from `src/oauth/pkce.ts` (uses `oauth4webapi`).
2. `Bun.serve` callback on `127.0.0.1:0` (random port). Single-shot. 120s timeout.
3. `buildAuthorizeUrl()` → `https://clerk.oriyn.ai/oauth/authorize`.
4. On callback, validate `state` strictly. Exchange code via `exchangeCode()`.
5. Persist via `AuthStore.save()`.
6. Refresh handled transparently by `AuthStore.getValidAccessToken()`. Skews 60s, async-mutex'd, refresh-token-rotates on every call.
</auth-flow>

<output-rules>
- TTY: colored, tables via `renderTable`, spinners via `createSpinner`.
- Non-TTY: JSONL via `createJsonlEmitter` (`{type, ...}` events) or one-shot JSON via `writeJson`.
- Errors when piped: `{"error":"...","code":"...","exit":N}` to stderr.
- `NO_COLOR` and `FORCE_COLOR` honored.
- `ORIYN_AGENT=1` no longer exists — replaced by TTY detection. The legacy env was redundant with non-TTY pipes.
</output-rules>

## Skill routing

Match user requests to installed skills BEFORE answering directly:

- Bugs, errors, "why is this broken" → `investigate`
- Ship, deploy, push, create PR → `ship`
- Code review, check my diff → `review`
- Update docs after shipping → `document-release`
- Code quality, health check → `health`
