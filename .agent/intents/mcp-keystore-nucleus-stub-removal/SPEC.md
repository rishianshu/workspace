# Spec: MCP + Keystore Services and Nucleus Stub Removal

## Services
- `mcp-server`: MCP tools API wrapping UCL + Nucleus brain APIs.
- `keystore`: credential storage API used by MCP for user-scoped tokens.
- `nucleus`: external GraphQL/UCL endpoints configured via env vars.

## Interfaces
- docker-compose services for `mcp-server` and `keystore` with ports exposed.
- `go-agent-service` talks to `mcp-server` via `MCP_SERVER_URL`.
- `mcp-server` talks to Nucleus via `NUCLEUS_API_URL` and `NUCLEUS_UCL_URL`.

## Behavior
- Local stack runs without the nucleus-stub service.
- Nucleus endpoints are configured via env (host.docker.internal defaults).
- README lists the new services and env vars.
