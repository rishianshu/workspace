# Acceptance: Nucleus + MCP Auth Tests and Credential Strategy

1) Nucleus client unit tests validate Bearer, Keycloak fetch, and Basic fallback behavior.
2) MCP client unit tests validate list/execute calls and auth header usage.
3) Integration test script validates Keycloak token retrieval, Nucleus projects/endpoints, and MCP tool discovery.
4) Docs state the credential strategy and explicitly say agents should not prompt for credentials.
5) Unit tests pass.
