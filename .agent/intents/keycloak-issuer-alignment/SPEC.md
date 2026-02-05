# Spec: Keycloak Issuer Alignment

## Issuer Fix
- Set Keycloak hostname to a stable base URL (`http://localhost:8081`) so tokens use a consistent issuer.
- Use the existing Keycloak container (Nucleus) and restart it to apply the hostname configuration.

## Resolver Behavior
- Restore strict error handling for Nucleus endpoint lookups (no bypass for auth-context errors).

## Validation
- Decode a token minted from container access and confirm `iss` matches `http://localhost:8081/realms/nucleus`.
- Run unit + integration tests in Workspace.
