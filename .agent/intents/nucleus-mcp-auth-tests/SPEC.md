# Spec: Nucleus + MCP Auth Tests and Credential Strategy

## Credential Strategy
- **Now:** credentials provided via env (NUCLEUS_BEARER_TOKEN or Keycloak password grant). No runtime prompts.
- **Later:** map Workspace user identity to Nucleus credentials so calls are scoped to the logged-in user (no prompt).
- API surfaces must accept user/project context without requiring credential parameters from the agent.

## Unit Tests
- Nucleus client:
  - Bearer token is preferred when present.
  - Keycloak password grant fetch is used when Bearer absent and Keycloak config present.
  - Basic auth fallback is used when Bearer and Keycloak are not available.
- MCP client:
  - List tools call uses Authorization header if configured.
  - Execute tool call uses Authorization header and forwards payload.

## Integration Tests (local)
- Fetch Keycloak token.
- Query Nucleus GraphQL for projects and endpoints.
- Call MCP list tools for a project and verify non-empty response.

## Documentation
- Add a short “Credential Strategy” section in README or docs/ontology.
- Clarify that agents should never ask for credentials; they rely on pre-registered tokens.
