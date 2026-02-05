# Intent: Nucleus + MCP Auth Tests and Credential Strategy

## Goal
Ensure agentic conversations do not request credentials at runtime by validating Nucleus/MCP auth flows and documenting the credential strategy.

## Scope
- Add unit tests for Nucleus GraphQL client auth (Bearer + Keycloak + Basic fallback).
- Add unit tests for MCP client auth/tool calls.
- Add integration tests that hit live Keycloak + Nucleus + MCP (read-only).
- Document the credential strategy (env now, user-scoped mapping later) and make the API surface explicit.

## Out of Scope
- Implementing user-scoped credential storage/mapping in this repo.
- UI changes.

## Acceptance
- Unit tests cover Nucleus auth header behavior and Keycloak token fetch flow.
- Unit tests cover MCP list/execute flows and auth header usage.
- Integration tests run against local stack and validate projects/endpoints and MCP tool discovery.
- Docs clearly describe the credential strategy and how agent runtime avoids credential prompts.
