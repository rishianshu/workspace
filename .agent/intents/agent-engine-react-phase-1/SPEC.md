# Spec: AgentEngine Phase 1 (ReAct + Function Calling)

## Overview
Introduce a reusable AgentEngine that executes a ReAct loop with a planner gate and structured tool calling. Phase 1 focuses on integrating the engine into the primary Chat path (non-stream) and wiring minimal adapters to existing components.

## Architecture

### AgentEngine (new package)
Location: `go-agent-service/internal/agentengine`
Responsibilities:
- Build prompt context (memory + KG + tools).
- Decide if tools are needed (planner gate).
- Execute tool calls with validation/policy.
- Append observations to prompt.
- Generate final response via LLM.
- Persist memory + facts.

### Adapters (new)

#### LLMClient adapter
- Wraps existing `LLMRouter`.
- Accepts full prompt + observations and returns LLM response.
- Uses provider/model from request.

#### ToolRegistry adapter
- Wraps existing `tools.Registry`.
- Converts tool definitions to `ToolDef` with schemas.

#### ToolExecutor adapter
- Uses `tools.Registry.Execute` for tool calls.
- Applies timeouts + policy checks.

#### Memory adapter
- Wraps `memory.MemoryStore` for AddTurn/StoreFact.
- No-op if memory store not configured.

#### ContextAssembler
- Combines:
  - System prompt
  - KG context from Orchestrator
  - Memory summary + relevant turns + recent turns
  - Tool descriptions + schemas
  - Current user request
- Outputs a single prompt string.

### Planner Gate
- Deterministic plan decision.
- Phase 1: heuristic classifier + optional LLM planner (temp=0) when provider configured.
- Plan output:
  - Direct response
  - Tool calls list (structured)
  - Clarification question

### Tool Calls
- Tool calls must map to `tools.Registry.Execute`.
- Actions must be validated against schema when available.
- Observations stored as facts.

## Data Flow (Chat)
1) Build context via ContextAssembler.
2) Planner decides tool use.
3) Execute tools; append observations.
4) Generate final response via LLM.
5) Store memory turns + facts.

## Error Handling
- Tool errors become observations, not fatal unless policy blocks all tools.
- LLM errors surfaced to caller with trace context.

## Tests
- Unit tests for:
  - Planner decision determinism.
  - Tool executor mapping to registry.
  - Context assembler includes memory + tools + KG.
- Integration: Chat request returns response with tool observation when tool call is forced.

## Docs
- Update `docs/architecture/agent-engine.md` as implementation evolves.
- Keep `docs/ontology/INDEX.md` aligned.
