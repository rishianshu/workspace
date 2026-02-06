# AgentEngine Architecture (ReAct + Function Calling)

Status: phase 1 implemented (chat path), streaming pending

## Goals
- Provide a reusable agent runtime that can power multiple products.
- Align with ADK-style behavior: plan, act, observe, revise.
- Support tool calling with schema validation and deterministic execution.
- Use memory and context compaction as first-class behavior.

## Non-Goals (Phase 1)
- Multi-agent coordination or delegation.
- Automatic skill induction or autonomous long-running workflows.
- Hard dependency on any single LLM provider.

## Current Behavior (Phase 1)
- Chat path uses AgentEngine for non-stream responses.
- Planner gate is heuristic and deterministic.
- Tool calls execute via tools.Registry; results become observations.
- Tool execution uses a default timeout and validates required fields when schema is available.
- Observations are appended to the prompt before final response.
- Memory is recorded when episodic store is configured.
- StreamChat still uses the legacy Runner (to be unified next).

## Core Concepts

### AgentEngine
Orchestrates the full agent loop. Owns:
- Planner gate (should we use tools? which ones?).
- Tool calling loop (ReAct).
- Memory updates and compaction triggers.
- Policy enforcement (limits, tool allowlists).

### Planner
Returns a plan for the current turn:
- `Direct` response (no tools).
- `ToolCalls` list (structured, validated).
- `NeedClarification` (ask user).

Planner can be lightweight (classifier) or LLM-based. It must be deterministic at the boundary (same input -> same decision with same config).

### Tool Registry
Provides tool schemas and constraints. Sources:
- MCP registry (UCL + Nucleus brain tools).
- Store and internal tools.

### Tool Executor
Executes tools, applies retries/timeouts, records results as facts.

### Memory
Three tiers:
- Short-term (recent turns, session state).
- Episodic (pgvector turns + semantic search).
- Facts (entity facts).

### Context Assembler
Builds the prompt context:
1) System prompt
2) Rolling summary
3) Relevant turns (semantic)
4) Recent turns
5) Tool descriptions
6) Current request

### Policy
Guards cost, security, and determinism:
- Max tool calls per turn
- Allowed tool names
- Max tokens
- Timeout budget

## ReAct Loop (Target)
1) Build context from memory + tool registry.
2) Planner decides: direct response vs tool calls.
3) If tool calls:
   - Execute tool(s)
   - Store observations as facts
   - Append observations to context
   - Replan if needed
4) Generate final response.
5) Update memory + summary.

## Key Interfaces (Conceptual)
```
AgentEngine.Run(request) -> response
Planner.Plan(context) -> Plan
ToolRegistry.ListTools(user, project) -> ToolDef[]
ToolExecutor.Execute(toolCall) -> ToolResult
MemoryStore.AddTurn / SearchTurns / StoreFact
ContextAssembler.Build(...) -> prompt
```

## Integration With Current System
- **LLM Router** is used by the engine LLM adapter.
- **Orchestrator** injects KG context into the system prompt.
- **Tool Registry** remains source of tool schemas (MCP + Store).
- **Memory** uses existing pgvector store (episodic + facts) when configured.
- **Chat** uses AgentEngine. **StreamChat** is pending migration.

## Determinism Controls
- Use planner gate with fixed temperature.
- Tool schema validation before execution.
- Max step count and tool budget per turn.
- Log and attach trace IDs to each action and tool call.

## Phased Migration
1) Introduce AgentEngine package and interfaces.
2) Plug Chat path into AgentEngine (non-stream).
3) Plug StreamChat into AgentEngine (stream output).
4) Enable tool calling with schema validation.
5) Wire memory + compaction in engine loop.
