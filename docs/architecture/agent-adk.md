# Internal ADK Agent Architecture (Workspace)

Status: active (internal implementation, not an external SDK)

## Goals
- Provide a predictable, tool-first agent runtime for Workspace.
- Support multi-provider LLMs (Gemini/OpenAI/Groq) behind a single interface.
- Maintain conversational context via a three-tier memory design.
- Keep UI and agent runtime decoupled (gateway + service boundary).

## Non-Goals (Current)
- Full autonomous planning and tool execution loops.
- Long-horizon memory across large org datasets without explicit retrieval.
- Zero-latency real-time embedding or summarization at high scale.

## Core Components
- **Agent Server** (`go-agent-service/internal/server/agent_server.go`)
  - gRPC/HTTP entrypoints for chat, streaming, and actions.
- **LLM Router** (`go-agent-service/internal/agent/llm_router.go`)
  - Selects provider/model and routes requests.
- **Runner** (`go-agent-service/internal/agent/runner.go`)
  - ADK-style execution engine (used by streaming).
- **Context Orchestrator** (`go-agent-service/internal/context/orchestrator.go`)
  - Extracts entities and pulls KG context from Nucleus.
- **Context Builder** (`go-agent-service/internal/context/builder.go`)
  - Assembles memory + summary + tools + query into a prompt.
- **Memory System** (`go-agent-service/internal/memory/*`)
  - Short-term (in-memory), episodic (pgvector), facts (pgvector).
- **Tool Registry** (`go-agent-service/internal/tools/*`)
  - Discovers MCP tools and schemas for LLM awareness.
- **Workflow Engine** (`go-agent-service/internal/workflow/*`)
  - Temporal-backed workflow execution.
- **Gateway** (`rust-gateway/*`)
  - HTTP/WS edge that proxies to the agent service.

## Request Flows

### Non-Streaming Chat (Primary)
1) workspace-web → rust-gateway → go-agent-service Chat.
2) Orchestrator extracts entities → Nucleus graph retrieval.
3) Tool registry enumerates available tools → appended to system prompt.
4) LLM Router calls provider → returns response.
5) Response is returned to UI.

### Streaming Chat (Secondary)
1) workspace-web → rust-gateway → go-agent-service StreamChat.
2) Runner processes the request (pattern-based response today).
3) Reasoning + tokenized output streamed to the UI.

## Memory & Context
- **Short-term**: In-memory store with TTL semantics.
- **Episodic**: PostgreSQL + pgvector for turns and semantic search.
- **Facts**: Structured entity facts with vector search.
- **Embeddings**: Gemini embedder (blocking call per turn).
- **Summarization**: Compressor updates rolling summaries (currently not wired to chat path).

Context assembly in builder order:
1) System prompt
2) Rolling summary
3) Relevant semantic turns
4) Recent turns
5) Tool descriptions
6) Current request

## Tooling
- Dynamic tool registry is populated from MCP (UCL + Nucleus brain tools).
- Tools are exposed to the LLM as prompt text (not function calls).
- Actions are executed via `ExecuteAction` using the tool registry.

## Configuration
Key env vars (not exhaustive):
- LLM: `GEMINI_API_KEY`, `OPENAI_API_KEY`
- Nucleus: `NUCLEUS_API_URL`, `NUCLEUS_UCL_URL`, `NUCLEUS_*`
- MCP: `MCP_SERVER_URL`
- Infra: `POSTGRES_URL`, `TEMPORAL_HOST`

## Design Scrutiny (Current Gaps)
- **Two execution paths**: Chat uses LLM Router + Orchestrator, StreamChat uses Runner.
- **Memory not integrated in main chat path**: ContextBuilder and episodic memory are not used in the primary LLM flow.
- **ContextBuilder output unused**: Built context is logged but not injected into LLM prompts.
- **Tools are descriptive only**: No function-calling or tool planning loop.
- **Embedding latency**: Synchronous embedding on write and query; no batching or async pipeline.
- **Short-term store is not distributed**: Horizontal scaling loses session memory.
- **Groq support incomplete**: Models exist, but no env wiring for Groq keys.

## Scale Readiness (Current)
Best fit today: small to mid-sized dev usage, low concurrency, and limited long-term memory needs.

### Required for Higher Scale
- Externalize short-term memory (Redis or DB-backed session store).
- Asynchronous embedding + summarization pipelines.
- Unify Chat and StreamChat into a single LLM execution pipeline.
- Introduce tool-call planning or function calling with schema validation.
- Add observability (trace IDs, structured metrics, and request-level SLIs).

## Roadmap (Proposed)
1) Unify Chat + StreamChat pipeline (Runner uses LLM Router).
2) Inject ContextBuilder output into system prompt.
3) Make memory + summarization first-class in the non-stream path.
4) Implement tool-call planning and execution.
5) Add Groq key wiring for free LLMs.
