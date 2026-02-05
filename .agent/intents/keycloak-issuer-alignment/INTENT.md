# Intent: Keycloak Issuer Alignment

## Goal
Ensure Nucleus GraphQL accepts tokens minted from Workspace containers by aligning Keycloak issuer settings.

## Scope
- Configure Keycloak to emit a stable issuer matching metadata-api expectations.
- Remove temporary resolver bypass for missing tenant/project context.
- Re-run unit + integration tests to validate MCP tool discovery.

## Out of Scope
- Changes to metadata-api auth logic.
- UI updates.

## Acceptance
- Tokens minted via container access resolve to issuer http://localhost:8081/realms/nucleus.
- MCP list tools succeeds without auth-context errors.
- Unit + integration tests pass.
