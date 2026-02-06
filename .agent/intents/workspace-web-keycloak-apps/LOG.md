

Codex Review Output (2026-02-06T06:20:24Z)
--------------------------------------------------
**Acceptance**

- 1) PASS – Keycloak env vars drive a client-side config and Keycloak instance (`workspace-web/src/lib/auth/keycloak.ts:9-40`). `AuthProvider` boots Keycloak, exposes `user/token`, and persists them for the rest of the app (`workspace-web/src/lib/auth/auth-context.tsx:45-145`). The root layout wraps the entire UI in `AuthGate`, which blocks rendering until the user authenticates (`workspace-web/src/app/layout.tsx:28-48`, `workspace-web/src/components/auth/auth-gate.tsx:6-44`). Once authenticated, the header shows the user identity and offers sign-in/out actions (`workspace-web/src/components/layout/global-header.tsx:84-110`).

- 2) PASS – Projects are fetched through `nucleusGraphql` with the logged-in user’s bearer token and tenant/user headers (`workspace-web/src/lib/workspace-context.tsx:217-277`). The apps catalog fetches endpoints with the same helper, sending the project ID plus tenant/user context (`workspace-web/src/components/apps/apps-page.tsx:282-333`). `nucleusGraphql` injects `Authorization: Bearer …` and `/api/nucleus/graphql` forwards it to Nucleus unchanged (`workspace-web/src/lib/nucleus-graphql.ts:12-44`, `workspace-web/src/app/api/nucleus/graphql/route.ts:10-33`).

- 3) PASS – The apps page queries both `metadataEndpoints(projectId)` and `endpointTemplates` (`workspace-web/src/components/apps/apps-page.tsx:77-117,282-297`), lets the user pick a project, and renders each endpoint with its template title and auth modes (`workspace-web/src/components/apps/apps-page.tsx:579-693`). Connected endpoints are tagged, and the list updates per selected project.

- 4) PASS – Opening an endpoint modal upserts/loads the app instance, discovers existing user apps, and builds the credential form from template-required fields (`workspace-web/src/components/apps/apps-page.tsx:374-444`). Connecting stores credentials in the keystore, creates/updates the user app, and links the project app (with distinct POSTs to `/api/keystore/credentials`, `/api/apps/users`, and `/api/apps/projects`) before refreshing state (`workspace-web/src/components/apps/apps-page.tsx:459-559`). The API routes proxy those calls to the agent service/keystore, and the keystore store writes scopes via `pq.Array` (`workspace-web/src/app/api/apps/instances/route.ts:1-34`, `workspace-web/src/app/api/apps/users/route.ts:1-34`, `workspace-web/src/app/api/apps/projects/route.ts:1-34`, `workspace-web/src/app/api/keystore/credentials/route.ts:1-19`, `go-agent-service/internal/keystore/store.go:77-115`).

- 5) PASS – Configuration artifacts now document the required services: `.env.example` and `docker-compose.yml` add the Keycloak, keystore, and agent service env vars for workspace-web (`.env.example:7-46`, `docker-compose.yml:147-163`). The ontology index describes workspace-web’s Keycloak dependency, keystore usage, and the complete auth + app subscription workflow (`docs/ontology/INDEX.md:43-109,155-167`).

- 6) PASS – The recorded timestamps show both `scripts/record-unit-test.sh` and `scripts/record-integration-test.sh` were run for this intent (`.agent/test-stamps/unit.txt:1`, `.agent/test-stamps/integration.txt:1`).

**Additional Risks**

- `keystore.Get` still scans the `scopes` column directly into a `[]string`, but inserts now persist non-null `TEXT[]` via `pq.Array`; once scopes are stored, reads will fail unless `Scan` also uses `pq.Array(&cred.Scopes)` (`go-agent-service/internal/keystore/store.go:77-140`).

- The new Next.js API proxies for `/api/apps/*` and `/api/keystore/credentials` blindly forward whatever `userId`/`projectId` the client supplies and never validate the caller’s Keycloak session (`workspace-web/src/app/api/apps/users/route.ts:1-34`, `workspace-web/src/app/api/apps/projects/route.ts:1-34`, `workspace-web/src/app/api/keystore/credentials/route.ts:1-19`). A compromised browser session could therefore write credentials or app links on behalf of other users.
Codex Review: 2026-02-06T06:26:44Z - Config + keystore/chat compatibility fixes
Codex Review: 2026-02-06T06:28:48Z - Follow-up fix: keystore scopes read path uses pq.Array scan
**Acceptance**
- PASS – Keycloak auth is wired through `AuthProvider` to initialize the Keycloak JS client, persist the token, and expose `user`, `token`, and `login/logout` hooks, while `AuthGate` blocks the UI until a session exists and `GlobalHeader` shows the signed‑in user identity (`workspace-web/src/lib/auth/auth-context.tsx:45-145`, `workspace-web/src/components/auth/auth-gate.tsx:6-44`, `workspace-web/src/components/layout/global-header.tsx:15-110`).
- PASS – Project and endpoint queries go through the `/api/nucleus/graphql` proxy with the Keycloak bearer token so Nucleus sees the user context (`workspace-web/src/lib/workspace-context.tsx:217-272`, `workspace-web/src/components/apps/apps-page.tsx:282-321`, `workspace-web/src/lib/nucleus-graphql.ts:12-44`, `workspace-web/src/app/api/nucleus/graphql/route.ts:10-33`).
- PASS – The apps catalog is project-scoped, resolves endpoint templates, and renders available auth modes plus credential forms per template, matching the spec (`workspace-web/src/components/apps/apps-page.tsx:256-746`).
- PASS – The subscribe flow upserts the app instance, stores credentials in the keystore, creates the user app, and links the project app via the new API proxies backed by the agent service and keystore storage layer (`workspace-web/src/components/apps/apps-page.tsx:360-541`, `workspace-web/src/app/api/apps/instances/route.ts:1-33`, `workspace-web/src/app/api/apps/users/route.ts:1-34`, `workspace-web/src/app/api/apps/projects/route.ts:1-33`, `workspace-web/src/app/api/keystore/credentials/route.ts:1-19`, `go-agent-service/internal/server/http_handler.go:522-687`, `go-agent-service/internal/keystore/store.go:21-198`).
- PASS – Configuration/docs were updated for the new Keycloak and keystore requirements, including env defaults, docker compose wiring, and ontology docs describing the auth + subscription workflow (`.env.example:7-46`, `docker-compose.yml:147-170`, `docs/ontology/INDEX.md:43-167`).
- PASS – Unit and integration workflows were recorded with fresh timestamps per the required scripts (`.agent/test-stamps/unit.txt:1`, `.agent/test-stamps/integration.txt:1`).

**Additional Risks**
- `workspace-web/src/app/api/apps/*/route.ts` and `/api/keystore/credentials` simply proxy requests without verifying the Keycloak session; a malicious client could forge requests for another `userId` because the routes trust whatever payload the browser sends (`workspace-web/src/app/api/apps/users/route.ts:1-34`, `workspace-web/src/app/api/apps/projects/route.ts:1-33`, `workspace-web/src/app/api/apps/instances/route.ts:1-33`, `workspace-web/src/app/api/keystore/credentials/route.ts:1-19`).
- The Nucleus GraphQL proxy still defaults to port 4000 even though the env templates now reference 4010, so local setups that rely on the default could fail without additional configuration (`workspace-web/src/app/api/nucleus/graphql/route.ts:4-8`, `.env.example:7-9`).
Codex Review: 2026-02-06T06:32:48Z - Final review after keystore scope fix
