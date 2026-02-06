# Workspace Ontology Index

Purpose
- Provide a shared vocabulary for Workspace services, contracts, and dependencies.
- Make system behavior predictable for humans and agents by anchoring terminology.

Scope
- Applies to runtime services, control-plane APIs, agent workflows, and data stores.
- Planned services are explicitly marked as such.
- Architecture spec: `docs/architecture/agent-adk.md` (internal ADK runtime).
- Agent engine spec: `docs/architecture/agent-engine.md` (ReAct + function calling).

## Core Entities

- **User**: Human operator of Workspace.
- **Session**: A user’s interactive run in the UI (chat + tool usage).
- **Agent**: Orchestrates tools, workflows, and LLM calls to fulfill a session.
- **Tool**: A callable capability exposed to the agent (internal or external).
- **Action**: A concrete operation executed by a tool (e.g., search, create, write).
- **Run**: A unit of execution (workflow or tool invocation) with traceable logs.
- **Artifact**: Persisted output from runs (logs, traces, results, embeddings).
- **App Instance**: Non-secret configuration identity for an external app (template + instance key).
- **User App**: A user-scoped binding to an app instance plus credential reference (`appId`).
- **Project App**: Link between a project and a user app with the Nucleus endpoint ID.
- **Provider**: External LLM provider or model backend (Gemini/OpenAI/Groq).
- **Gateway**: Edge service for HTTP/WebSocket streaming and routing.
- **Registry**: Source of truth for tools/actions and their schemas.

## Services (Runtime)

- **workspace-web**: UI for sessions and agent interaction.
- **rust-gateway**: Edge gateway for routing, streaming, and protocol translation.
- **go-agent-service**: Agent runtime, tool registry, workflow execution.
- **mcp-server**: MCP tool surface for UCL + Nucleus brain APIs.
- **keystore**: Credential storage service for user-level tokens.

## Services (Planned)

- **nucleus-upstream**: MCP service will eventually move into Nucleus proper.

## Service Details (Runtime)

### workspace-web
- Purpose: Next.js UI for sessions, chat, and tool-driven workflows.
- Entry: `workspace-web/src/app` (App Router).
- Key areas:
  - `workspace-web/src/components`: UI surfaces.
  - `workspace-web/src/lib`: client utilities, config, API helpers.
  - `workspace-web/src/protos`: shared proto files for gateway/UCL usage.
  - `workspace-web/src/app/api`: API routes (proxy to gateway or Nucleus).
- Dependencies:
  - Talks to `rust-gateway` for streaming and agent operations.
  - Talks to Nucleus GraphQL via API routes for project/context metadata.
  - Agent reasoning tools use MCP (UCL + brain APIs) instead of direct UI calls.
  - Uses Keycloak (same realm, separate client) for user login and bearer token injection.
  - App catalog + subscription flow uses app registry + keystore via API routes.
  - App/credential API proxies enforce bearer validation and user-scoped mutations.

### rust-gateway
- Purpose: Edge HTTP/WS gateway, streaming responses, routing to agent service.
- Entry: `rust-gateway/src/main.rs`.
- Key areas:
  - `rust-gateway/src/routes`: HTTP and tool routes.
  - `rust-gateway/src/middleware`: CORS, compression, tracing.
  - `rust-gateway/src/proxy`: request forwarding and streaming helpers.
- Dependencies:
  - Calls `go-agent-service` over gRPC/HTTP.
  - Calls Nucleus for graph/tool surfaces.

### go-agent-service
- Purpose: Agent runtime, tool registry, workflow orchestration.
- Entry: `go-agent-service/cmd/server/main.go`.
- Key areas:
  - `go-agent-service/internal/server`: HTTP/gRPC handlers.
  - `go-agent-service/internal/tools`: tool registry and tool execution.
  - `go-agent-service/internal/nucleus`: GraphQL client for Nucleus.
  - `go-agent-service/internal/ucl`: UCL gateway client and MCP surfaces.
  - `go-agent-service/internal/context`: context assembly and orchestration.
  - `go-agent-service/internal/store`: store core client.
  - App registry endpoints: `/apps/instances`, `/apps/users`, `/apps/projects`.
- Dependencies:
  - PostgreSQL (state, embeddings).
  - Temporal (workflow orchestration).
  - LLM providers (Gemini/OpenAI/Groq via API keys).
  - MCP server (tool discovery + execution).
  - Nucleus API (GraphQL + UCL gateway).
  - Uses MCP to access Nucleus brain/tool surfaces; direct Nucleus calls are limited to UI/context metadata.

### mcp-server
- Purpose: Wrap UCL + Nucleus APIs into MCP tool surfaces for agents.
- Entry: `go-agent-service/cmd/mcp-server/main.go`.
- Dependencies:
  - Nucleus API (GraphQL + UCL).
  - Keystore for credential lookups.
- Notes:
  - Injects user credentials from keystore into UCL tool calls.
  - No stub fallback: MCP requires live UCL and Nucleus APIs.
  - Publishes input/output schemas for actions, plus read APIs (list datasets, schema, preview).
  - Endpoint/credential binding is resolved outside action schemas (via Nucleus + keystore).
  - Tool discovery requires `userId` + `projectId` and returns app-bound tool names (`app/{appId}/{action}`).

### keystore
- Purpose: Credential storage and retrieval for user-scoped tokens (separate from endpoint config).
- Entry: `go-agent-service/cmd/keystore/main.go`.
- Dependencies:
  - Database for metadata (tokens, scopes, ownership).
  - KMS for encryption at rest (future).
- Notes:
  - Enables per-user audit trails for actions executed via MCP/UCL.
  - Workspace uses user-level credentials; Nucleus org-level credentials remain separate.

### nucleus (external)
- Purpose: Source of truth for projects, endpoints, configs, and brain/graph APIs.
- Provided by the Nucleus repo (metadata-api + UCL gateway).
- Workspace treats Nucleus as an external dependency; configure via `NUCLEUS_*` env vars.

## Data Stores & Dependencies

- **PostgreSQL (pgvector)**: Agent persistence + embeddings.
- **Temporal**: Workflow orchestration for agent runs.
- **LLM Providers**: External APIs used by the agent service.
- **App Registry (Postgres)**: `app_instances`, `user_apps`, `project_apps` used by MCP to resolve app bindings.

## Code Layout

```
.
├── Agent.md                # Implementor workflow (default)
├── AGENT_PLANNER.md        # Planner workflow
├── README.md               # Project overview
├── docker-compose.yml      # Local infra dependencies
├── start-dev.sh            # Local dev bootstrap
├── docs/ontology/          # Shared vocabulary and contracts
├── go-agent-service/       # Go agent runtime
├── keystore/               # Keystore service docs
├── mcp-server/             # MCP service docs
├── rust-gateway/           # Rust edge gateway
├── workspace-web/          # Next.js UI
└── docs/                   # Documentation
```

## Relationships (High-Level)

- UI → Gateway → Agent Service.
- Agent Service → MCP → (UCL + Nucleus brain tools).
- Agent Service → Postgres (state, embeddings) + Temporal (workflows).
- MCP → Keystore (user credentials) and Nucleus (brain + UCL).

## Invariants

- Prefer explicit schemas and stable error codes for public contracts.
- Avoid hardcoded IDs; rely on declared capabilities and config.
- Deterministic behavior unless explicitly required otherwise.
- Conversations are created only with a valid Nucleus project ID; creation is blocked until a project is selected.

## Credential Strategy
- **Now:** credentials are supplied via env and pre-registered tokens; agents must not prompt for credentials.
- **Later:** Workspace maps logged-in users to Nucleus credentials so calls are user-scoped without prompts.

## Auth + App Subscription Flow
1) User logs in via Keycloak (Workspace client).
2) Workspace fetches projects + endpoints via Nucleus GraphQL using the bearer token.
3) App catalog shows endpoints and template auth modes per project.
4) Subscribing an app:
   - Upserts app instance (template + instance key = endpoint ID).
   - Stores credentials in keystore → key token.
   - Upserts user app (user + app instance + credential ref).
   - Upserts project app (project + user app + endpoint ID).
