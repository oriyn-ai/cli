# Switch CLI auth from Clerk JWT templates to OAuth 2.0 + PKCE

**Date**: 2026-04-28
**Status**: Accepted

## Context

The CLI's first Clerk integration (v0.5.0) used a custom Clerk **JWT template** (`cli`)
plus a Next.js mint endpoint at `app.oriyn.ai/auth/cli/login` to issue tokens. The
flow:

1. CLI opens browser → web's `/auth/cli/login?port=X&state=Y`.
2. Web ensures the user has a Clerk session, calls `clerkClient().sessions.getToken(sid, "cli")`.
3. Web redirects back to `127.0.0.1:X/callback?token=<jwt>&state=Y`.
4. CLI saves the JWT, sends it as a Bearer to `api.oriyn.ai`.
5. API verifies the JWT signature against a static PEM (networkless), extracts
   `o.id` / `o.rol` / `o.per` from the claims, gates on `azp`.

This shipped, then immediately broke. The investigation traced through:

- The Clerk dashboard UI displayed a sample claims block with the toggle "Show
  default claims example" enabled. The actual saved Claims field was the literal
  unexpanded string `{"o": "{{org}}"}`. Tokens minted from the template carried
  `"o": "{{org}}"` as a *string*, not the org object the API expected.
- Once the template was changed to `{"o": {"id": "{{org.id}}", "rol":
  "{{org.role}}", ...}}`, those shortcodes worked — but `{{org.permissions}}`
  came through as an unexpanded string too, and Clerk treats `azp` as a
  reserved claim and refuses to populate it on Backend-API-minted tokens
  (only on session tokens carrying a request-origin context).
- The API's `azp` allow-list check therefore failed every CLI request before
  the org check even ran.
- Multiple shape mismatches between session tokens (default claims format) and
  custom-template tokens (no `azp`, no `per`, prefixed `org:admin` role).

The root issue is one level deeper than any single mismatch: **custom JWT
templates are not the right architectural fit for "CLI authenticates a user".**
Clerk's templates are designed for third-party integrations (Convex, Hasura,
Supabase Third-Party Auth) where the consumer expects a particular JWT shape.
For CLIs Clerk's own product (`clerk/cli`) uses standard OAuth 2.0 + PKCE
against the `https://clerk.<domain>/oauth/*` endpoints — no templates,
no custom claims, no shape coupling between the CLI binary, the web app, and
the API.

## Decision

Replace the JWT-template flow with OAuth 2.0 Authorization Code + PKCE
(RFC 6749 + RFC 7636 + RFC 8252):

1. Register `Oriyn CLI` as a public OAuth application on the production
   Clerk instance. Redirect URI `http://127.0.0.1/callback` (loopback —
   any port is permitted by RFC 8252 §7.3). Client ID is non-sensitive
   and embedded in the binary; PKCE provides the proof of possession
   that a `client_secret` would for confidential clients.
2. CLI generates a `code_verifier` per login attempt, derives an
   S256 `code_challenge`, opens `https://clerk.oriyn.ai/oauth/authorize`
   with the challenge + `state`. User signs in on Clerk's hosted page.
3. Clerk redirects to `127.0.0.1:port/callback?code=...&state=...`.
4. CLI POSTs `code + verifier` to `https://clerk.oriyn.ai/oauth/token`,
   receives `{access_token, refresh_token, expires_in}`.
5. Tokens stored in OS keychain. Access tokens auto-refresh against
   `/oauth/token` when within 60s of expiry. Refresh tokens rotate.
6. API validates the access token (RS256 JWT signed by the Clerk
   instance) networklessly using `CLERK_JWT_KEY`. `sub` → user_id.
   Org membership is resolved via Clerk Backend API
   (`/v1/users/{sub}/organization_memberships`), cached per-user 60s.
7. Web `/auth/cli/login` route is **deleted**. Web app no longer
   brokers CLI auth; the CLI talks directly to Clerk.

## Why this beats custom JWT templates

| Concern | Custom JWT template | OAuth 2.0 + PKCE |
|---|---|---|
| Token in URL | Yes (`?token=<jwt>`) — visible in history, proxy logs, web-server access logs | No — only a one-time `code` is in the URL; the token comes back over a server-to-server POST that requires the verifier |
| Refresh tokens | None — 24h expiry forces re-login | Native — refresh until the user revokes |
| Claim shape sync | Template config in Clerk dashboard must agree with API expectations across versions | Standard OIDC claims; opaque to consumers |
| Verifying the token | Custom for each app | OIDC discovery + `/userinfo` is standard |
| Failure modes | Silent (template shortcode doesn't expand → unhelpful 401) | Loud (OAuth errors come back via `?error=...&error_description=...` and we surface them) |
| Vendor lock-in | Heavy — the template lives in Clerk's UI | Light — switching auth providers means swapping endpoints, not rewriting both ends |
| Operational risk | Web + CLI + API + dashboard config all have to agree | Web is uninvolved; CLI ↔ Clerk ↔ API |

## Blast radius

- **`cli/`**: net-new `internal/oauth/` package; `cmd/login.go` rewritten;
  `internal/auth/auth.go` gains a refresh path and a `RefreshToken` field.
- **`web/`**: `apps/app/app/auth/cli/login/route.ts` deleted.
  `mintCliToken` removed from `@oriyn/auth/server`, the `AuthProvider`
  interface, and the Clerk provider.
- **`api/`**: `auth/clerk.py` rewritten to verify OAuth access tokens
  and resolve org membership via Backend API + 60s cache.
  `CLERK_AUTHORIZED_PARTIES` removed; `CLERK_SECRET_KEY` and
  `CLERK_BACKEND_API_URL` added.
- **Clerk dashboard**: OAuth application registered. Custom JWT
  template `cli` (`jtmp_3CzLXA5go20XSVn8JFUwaMQRj6h`) deleted only
  after the new flow ships; deleting it pre-deploy strands existing
  v0.5.0 binaries that haven't upgraded yet.

## Rollout

PRs land in dependency order:

1. **api**: deploy first. The new auth provider accepts OAuth tokens but
   v0.5.0 CLIs are sending JWT-template tokens — those will continue to
   401 (which they already do today). No regression introduced; the
   pre-fix state was already broken for these users.
2. **web**: deploy. The CLI mint route is gone; v0.5.0 CLIs hitting
   `/auth/cli/login` get a 404. Acceptable because the route was
   already broken end-to-end.
3. **cli**: tag a new release. Goreleaser publishes v0.6.0;
   `install.sh` users get the OAuth client. Login works.
4. **dashboard cleanup**: delete the unused `cli` JWT template.

There is no migration step on the user side beyond `install.sh && oriyn login`.
Existing keychain credentials become invalid (different shape, different
refresh story) and are replaced by the next login.

## Trade-offs accepted

- The API now makes one Clerk Backend API call per user every 60s of
  activity. At expected CLI usage volumes (one-digit RPS per active user,
  if that) this is negligible. The cache TTL is the dial.
- Org-membership cache means role flips and removals take up to 60s to
  affect API authorization. Acceptable for a CLI; if it stops being
  acceptable we shorten the TTL or invalidate via Clerk webhook.
- Multi-org users get the first membership returned by Clerk. We do not
  yet support `oriyn switch-org`; when we add it, the CLI will pass an
  `X-Org-Id` header and the API will verify membership before accepting it.

## Future cleanup

- Add a `decisions/` mirror in `web/` and `api/` so the architectural
  rationale is visible from each repo's tree, not just `cli/`.
- Reconsider whether the `users` mirror should be expanded to include
  org memberships, removing the per-request Backend API call entirely.
  Defer until membership flips become a real ops pain.

## Reference

- Clerk's own CLI uses this same pattern:
  https://github.com/clerk/cli (`packages/cli-core/src/commands/auth/login.ts`,
  `packages/cli-core/src/lib/auth-server.ts`).
- RFC 8252, "OAuth 2.0 for Native Apps": loopback redirect (§7.3),
  any-port matching, PKCE recommendation for public clients.
- RFC 7636, "Proof Key for Code Exchange": S256 method.
