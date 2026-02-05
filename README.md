# Workspace

A full-stack AI agent platform featuring a Go-based agent service, Rust edge gateway, and Next.js web interface.

## 🏗️ Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  workspace-web  │────▶│  rust-gateway   │────▶│ go-agent-service│
│   (Next.js)     │     │    (Axum)       │     │ (gRPC/Temporal) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                         │
                              ┌──────────────────────────┼──────────────────────────┐
                              ▼                          ▼                          ▼
                       ┌───────────┐              ┌───────────┐              ┌───────────┐
                       │ PostgreSQL│              │  Temporal │              │ mcp-server│
                       │ (pgvector)│              │  Server   │              └─────┬─────┘
                       └───────────┘              └───────────┘                    ▼
                                                                                 ┌───────────┐
                                                                                 │ keystore  │
                                                                                 └─────┬─────┘
                                                                                       ▼
                                                                                 ┌───────────┐
                                                                                 │  Nucleus  │
                                                                                 │ (external)│
                                                                                 └───────────┘
```

## 📦 Components

### [go-agent-service](./go-agent-service)
Go-based AI agent service with multi-provider LLM support.

- **Stack**: Go 1.22, gRPC, Temporal, pgvector
- **Features**: 
  - Multi-provider LLM integration (Gemini, OpenAI, Groq)
  - Temporal workflows for agent orchestration
  - Vector embeddings with pgvector

### [rust-gateway](./rust-gateway)
High-performance edge gateway handling HTTP/WebSocket traffic.

- **Stack**: Rust, Axum, Tonic (gRPC client)
- **Features**:
  - SSE/WebSocket streaming
  - Request routing to agent service
  - CORS and compression middleware

### [workspace-web](./workspace-web)
Modern web interface for interacting with the AI agent.

- **Stack**: Next.js 14, React 18, TypeScript, TailwindCSS
- **Features**:
  - Monaco editor integration
  - Real-time streaming responses
  - Apollo Client for GraphQL

### Nucleus (external)
Source of truth for projects, endpoints, and brain/graph APIs (GraphQL + UCL gateway).

### [mcp-server](./mcp-server)
Standalone MCP service that wraps UCL + Nucleus brain APIs for tool discovery/execution.

### [keystore](./keystore)
Credential store service for user-scoped tokens (separate from endpoint config).

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.22+
- Rust 1.75+
- Node.js 20+

### Environment Variables
Create a `.env` file in the root directory:
```bash
GEMINI_API_KEY=your_gemini_api_key
NUCLEUS_URL=http://localhost:4000
NUCLEUS_API_URL=http://localhost:4000/graphql
NUCLEUS_UCL_URL=localhost:50051
NUCLEUS_USERNAME=dev-admin
NUCLEUS_PASSWORD=password
NUCLEUS_TENANT_ID=default
NUCLEUS_BEARER_TOKEN= # optional; if set, preferred over basic auth
KEYCLOAK_URL=http://localhost:8081
KEYCLOAK_REALM=nucleus
KEYCLOAK_CLIENT_ID= # required for token fetch (e.g., jira-plus-plus)
KEYCLOAK_CLIENT_SECRET= # optional
KEYCLOAK_USERNAME= # optional; falls back to NUCLEUS_USERNAME
KEYCLOAK_PASSWORD= # optional; falls back to NUCLEUS_PASSWORD
MCP_BEARER_TOKEN= # optional; auth for MCP server
```

### Nucleus GraphQL Auth (Keycloak)
Nucleus GraphQL expects a Bearer token. You can either provide `NUCLEUS_BEARER_TOKEN` or let the agent service fetch one via Keycloak.

Helper script:
```bash
scripts/fetch-keycloak-token.sh
```

Token retrieval (curl):
```bash
curl -s "http://localhost:8081/realms/nucleus/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "client_id=jira-plus-plus" \
  -d "username=$KEYCLOAK_USERNAME" \
  -d "password=$KEYCLOAK_PASSWORD"
```

Manual GraphQL example (replace token output):
```bash
TOKEN=$(scripts/fetch-keycloak-token.sh)
curl -s http://localhost:4010/graphql \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $TOKEN" \\
  -d '{"query":"{ metadataProjects { id } }"}'
```

### Credential Strategy
- **Now:** credentials are supplied via env and pre-registered tokens. Agents must not prompt for credentials at runtime.
- **Later:** Workspace will map logged-in users to Nucleus credentials so API calls are user-scoped without prompts.

### Running with Docker Compose
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

### Service Endpoints
| Service          | Port  | Description                |
|------------------|-------|----------------------------|
| workspace-web    | 3000  | Web frontend               |
| rust-gateway     | 8082  | Edge gateway (HTTP/WS)     |
| go-agent-service | 9002  | Agent gRPC service         |
| mcp-server       | 9100  | MCP tools API              |
| keystore         | 9200  | Credential store API       |
| nucleus          | n/a   | External Nucleus GraphQL/UCL|
| temporal-ui      | 8233  | Temporal Web UI            |
| temporal         | 7234  | Temporal gRPC              |
| postgres         | 5442  | PostgreSQL with pgvector   |

## 📚 Relevant Documentation

### Workspace Docs
- Ontology: [docs/ontology/INDEX.md](./docs/ontology/INDEX.md)

### Core Technologies
- [Google ADK for Go](https://github.com/google/genai-go) - AI agent development kit
- [Temporal.io](https://docs.temporal.io/) - Workflow orchestration
- [Axum](https://docs.rs/axum/latest/axum/) - Rust web framework
- [Next.js](https://nextjs.org/docs) - React framework
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search

### gRPC & Protobuf
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [Tonic](https://docs.rs/tonic/latest/tonic/) - Rust gRPC framework

### Additional Resources
- [Apollo Client](https://www.apollographql.com/docs/react/) - GraphQL client
- [Monaco Editor](https://microsoft.github.io/monaco-editor/) - Code editor

## 🛠️ Development

### Commit Workflow (Mandatory)
This repo uses a pre-commit hook to enforce requirements, unit tests, and Codex review logging.

Setup:
```bash
git config core.hooksPath .githooks
```

Usage:
```bash
scripts/set-active-intent.sh <intent-slug>
scripts/run-unit-tests.sh
scripts/run-codex-review.sh <intent-slug> "summary"
```

Default test commands can be set in:
```
.agent/test-commands/unit.sh
.agent/test-commands/integration.sh
```

Story completion (integration tests):
```bash
scripts/run-integration-tests.sh
scripts/verify-intent-complete.sh
```

### Individual Service Development

**Go Agent Service:**
```bash
cd go-agent-service
go run cmd/server/main.go
```

**Rust Gateway:**
```bash
cd rust-gateway
cargo run
```

**Workspace Web:**
```bash
cd workspace-web
npm install
npm run dev
```

**Nucleus (external):**
Point `NUCLEUS_API_URL` / `NUCLEUS_UCL_URL` at your Nucleus deployment.

## 📄 License

MIT
