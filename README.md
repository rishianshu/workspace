# Workspace

A full-stack AI agent platform featuring a Go-based agent service, Rust edge gateway, and Next.js web interface.

## 🏗️ Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  workspace-web  │────▶│  rust-gateway   │────▶│ go-agent-service│
│   (Next.js)     │     │    (Axum)       │     │ (gRPC/Temporal) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                         │
                              ┌──────────────────────────┴──────────────────────────┐
                              ▼                          ▼                          ▼
                       ┌───────────┐              ┌───────────┐              ┌───────────┐
                       │ PostgreSQL│              │  Temporal │              │  Nucleus  │
                       │ (pgvector)│              │  Server   │              │   Stub    │
                       └───────────┘              └───────────┘              └───────────┘
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

### [nucleus-stub](./nucleus-stub)
Stub implementation of the Nucleus data layer.

- **Stack**: Node.js, Apollo Server, gRPC
- **Features**:
  - GraphQL API
  - gRPC service endpoints

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
```

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
| rust-gateway     | 8080  | Edge gateway (HTTP/WS)     |
| go-agent-service | 9000  | Agent gRPC service         |
| nucleus-stub     | 4000  | Nucleus data layer         |
| temporal-ui      | 8233  | Temporal Web UI            |
| temporal         | 7233  | Temporal gRPC              |
| postgres         | 5432  | PostgreSQL with pgvector   |

## 📚 Relevant Documentation

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

**Nucleus Stub:**
```bash
cd nucleus-stub
npm install
npm start
```

## 📄 License

MIT
