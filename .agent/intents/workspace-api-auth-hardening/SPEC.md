# Spec: Workspace API Auth Hardening

## Server Auth
- Add `workspace-web/src/lib/auth/server-auth.ts` with:
  - Bearer extraction from `Authorization` header.
  - Keycloak config resolution from `NEXT_PUBLIC_KEYCLOAK_URL` + `NEXT_PUBLIC_KEYCLOAK_REALM`.
  - User validation via `GET /realms/{realm}/protocol/openid-connect/userinfo`.
  - Helpers:
    - `requireAuthenticatedUser(req)`
    - `unauthorized()/forbidden()/badRequest()` response helpers.
- Behavior:
  - If Keycloak is configured: missing/invalid bearer => `401`.
  - If Keycloak is not configured: allow passthrough for local dev with `x-user-id` fallback.

## Route Hardening
- `/api/apps/instances`:
  - Require authenticated user.
  - Forward `Authorization` and `x-user-id`.
- `/api/apps/users`:
  - GET: if `userId` query exists and mismatches auth user => `403`.
  - POST: body `userId` must equal auth user => `403`.
- `/api/apps/projects`:
  - GET: enforce optional `userId` query consistency.
  - POST: require auth; forward `x-user-id`.
- `/api/keystore/credentials`:
  - Require auth.
  - `owner_type` must be `user` (when provided).
  - `owner_id` must match auth user (or be defaulted to auth user when missing).

## Client Wiring
- Update `workspace-web/src/components/apps/apps-page.tsx` fetches to include
  `Authorization: Bearer <token>` when calling `/api/apps/*` and `/api/keystore/credentials`.

## Testing
- Run `scripts/run-unit-tests.sh`.
- Run `scripts/run-integration-tests.sh`.
- Append requirement-based review note to story log.
