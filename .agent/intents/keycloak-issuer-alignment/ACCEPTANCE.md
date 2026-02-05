# Acceptance: Keycloak Issuer Alignment

1) Keycloak issues tokens with `iss` = http://localhost:8081/realms/nucleus even when fetched from containers.
2) MCP list tools no longer fails with missing tenant/project context.
3) Unit tests pass.
4) Integration tests pass.
