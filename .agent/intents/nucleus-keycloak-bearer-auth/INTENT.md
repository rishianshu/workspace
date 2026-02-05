# Intent: Nucleus Bearer Auth + Keycloak Helper

## Goal
Ensure Workspace can authenticate to Nucleus GraphQL using Keycloak-issued Bearer tokens, and document the auth flow for local dev.

## Scope
- Add config + client support for Bearer auth (Keycloak password grant) in go-agent-service.
- Document the Keycloak → Bearer token flow and required env vars.
- Add a helper script to fetch a token for manual GraphQL calls.

## Out of Scope
- Changing Nucleus auth policy or schema.
- UI auth changes.

## Acceptance
- go-agent-service can use `NUCLEUS_BEARER_TOKEN` or fetch a token via Keycloak config.
- README documents the Keycloak token flow and env vars.
- Helper script exists to fetch a Bearer token for local GraphQL usage.
- Unit tests pass.
