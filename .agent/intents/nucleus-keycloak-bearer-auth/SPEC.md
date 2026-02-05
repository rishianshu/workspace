# Spec: Nucleus Bearer Auth + Keycloak Helper

## Config
- `NUCLEUS_BEARER_TOKEN`: If set, prefer for Nucleus GraphQL auth.
- `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_CLIENT_SECRET`:
  Used to fetch a token with the password grant when `NUCLEUS_BEARER_TOKEN` is not set.

## Behavior
- Nucleus client should attach `Authorization: Bearer <token>` when available.
- If Bearer is absent and Keycloak is configured, request token via password grant.
- Fall back to basic auth only when Bearer not available.

## Docs
- README documents the token flow, required env vars, and a sample curl for token retrieval.
- Helper script in `scripts/` fetches and prints the access token.
