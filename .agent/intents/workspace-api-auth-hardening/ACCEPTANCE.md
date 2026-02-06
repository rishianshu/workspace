# Acceptance: Workspace API Auth Hardening

1) App registry and keystore API routes enforce authentication when Keycloak is configured.
2) `/api/apps/users` rejects userId mismatches between request and authenticated user.
3) `/api/keystore/credentials` rejects owner mismatch and user-only constraints.
4) Apps UI forwards bearer token for app subscription operations.
5) Unit and integration test stamps are updated after implementation.
6) Story log includes a requirements-based review entry.
