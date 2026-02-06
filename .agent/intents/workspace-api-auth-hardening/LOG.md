
Codex Review Output (2026-02-06T06:48:37Z)
--------------------------------------------------
**Acceptance**
- PASS – 1) `requireAuthenticatedUser` enforces Keycloak bearer validation (or x-user-id fallback only when Keycloak is absent) and each proxy for apps/keystore routes now invokes it before forwarding headers downstream, so unauthenticated traffic is rejected when Keycloak is configured (workspace-web/src/lib/auth/server-auth.ts:33-152, workspace-web/src/app/api/apps/instances/route.ts:20-33, workspace-web/src/app/api/apps/projects/route.ts:47-103, workspace-web/src/app/api/apps/users/route.ts:20-68, workspace-web/src/app/api/keystore/credentials/route.ts:6-40).
- PASS – 2) `/api/apps/users` checks that any `userId` in the query/body matches the authenticated user and overwrites the body value before proxying, ensuring cross-user access is blocked (workspace-web/src/app/api/apps/users/route.ts:20-68).
- PASS – 3) The keystore credential proxy now enforces `owner_type === "user"` and requires `owner_id` to match the authenticated user (defaulting it when absent), rejecting mismatches before forwarding (workspace-web/src/app/api/keystore/credentials/route.ts:6-34).
- PASS – 4) The Apps UI derives `Authorization: Bearer <token>` headers (and JSON variants) and reuses them for all `/api/apps/*` and `/api/keystore/credentials` fetches during subscription flows, keeping browser calls authenticated (workspace-web/src/components/apps/apps-page.tsx:274-548).
- PASS – 5) Unit and integration test stamps were updated to `2026-02-06T06:48:06Z`, indicating the required scripts were rerun after the change (/.agent/test-stamps/unit.txt:1, /.agent/test-stamps/integration.txt:1).
- FAIL – 6) The story log lacks the required requirements-based review note; it only contains an empty header entry with no review details (/.agent/intents/workspace-api-auth-hardening/LOG.md:1-3).

**Risks**
- Keycloak load: `requireAuthenticatedUser` calls the Keycloak userinfo endpoint for every API request with no caching; the Apps page issues multiple proxied calls per interaction, so this could become a latency or rate-limit bottleneck under heavy use (workspace-web/src/lib/auth/server-auth.ts:71-123, workspace-web/src/components/apps/apps-page.tsx:344-548).

Next step: add the missing requirements-based review entry to `.agent/intents/workspace-api-auth-hardening/LOG.md` so acceptance item 6 can pass.
Codex Review: 2026-02-06T06:50:36Z - Server-side auth guard for app/keystore routes
Codex Review: 2026-02-06T06:50:48Z - Server-side auth guard for app/keystore routes
