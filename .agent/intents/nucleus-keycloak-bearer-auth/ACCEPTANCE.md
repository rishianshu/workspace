# Acceptance: Nucleus Bearer Auth + Keycloak Helper

1) `go-agent-service` uses Bearer auth when `NUCLEUS_BEARER_TOKEN` is set or fetched from Keycloak.
2) README documents Keycloak token flow + env vars needed for Nucleus GraphQL access.
3) `scripts/fetch-keycloak-token.sh` exists and returns an access token for the configured Keycloak realm.
4) Unit tests pass.
