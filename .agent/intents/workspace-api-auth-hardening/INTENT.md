# Intent: Workspace API Auth Hardening

## Goal
Enforce server-side user/session validation for workspace-web app registry and keystore API routes so client-supplied `userId` and credential writes are scoped to the authenticated Keycloak user.

## Problem
- `/api/apps/*` and `/api/keystore/credentials` proxy requests without validating identity.
- Client can send arbitrary `userId`/`owner_id` in body/query.
- This creates cross-user write/read risk if browser/session is compromised.

## Scope (In)
- Add server-side bearer validation against Keycloak userinfo endpoint.
- Require authenticated user for `/api/apps/instances`, `/api/apps/users`, `/api/apps/projects`, `/api/keystore/credentials`.
- Enforce `userId` and `owner_id` consistency with authenticated user where applicable.
- Forward authenticated user headers to downstream services.
- Update workspace client calls to include bearer token on app-subscription API calls.

## Scope (Out)
- Full RBAC model in downstream agent service.
- Persisted server-side session store.
- Changes to Nucleus metadata schema.

## Acceptance (high-level)
- App/keystore API routes reject unauthenticated requests when Keycloak is configured.
- User-scoped routes reject mismatched user identifiers.
- Apps UI continues to work by forwarding bearer token.
- Unit/integration test scripts recorded for the story.
