# mcp-server

Purpose
- Wrap UCL + Nucleus APIs into MCP tool surfaces for agents.

Entry Point
- `go-agent-service/cmd/mcp-server/main.go`

Port
- 9100 (default `MCP_PORT`)

Expected Responsibilities
- Tool discovery and schema exposure.
- Tool execution routing to UCL + Nucleus APIs.
- Auth integration via keystore.
- Publish input/output schemas and read APIs (catalog/preview).
- Expose Nucleus metadata helpers (projects, endpoints) for endpoint resolution.
- Resolve app bindings via Workspace app registry (app_instances, user_apps, project_apps).

Notes
- No stub fallback; requires live UCL + Nucleus APIs.
- Endpoint/credential binding is resolved outside action schemas (Nucleus + keystore).
- Tool discovery requires `userId` + `projectId` query params and returns app-bound tool names (`app/{appId}/{action}`).
