# Intent: AgentEngine Phase 1 (ReAct + Function Calling)

## Goal
Align the agent runtime with ADK-style behavior by introducing a unified AgentEngine loop (plan → act → observe → respond) that is reusable across products and tool sets.

## Problem
- Chat and Stream paths are split and inconsistent.
- Context builder + memory exist but are not injected into the main LLM path.
- Tool registry is only descriptive (no structured tool calling loop).

## Scope (In)
- Wire AgentEngine into the primary Chat path.
- Implement adapters for LLM, Tool Registry, Tool Executor, Memory, and Context Assembler.
- Add deterministic planner gate + tool-call loop (initial, minimal).
- Record observations and persist memory/facts.
- Update architecture docs to match behavior.

## Scope (Out)
- Multi-agent coordination.
- Full autonomous planning beyond basic tool calls.
- Redis/external short-term memory.
- Full streaming parity (may follow in next phase).

## Constraints
- Deterministic planner decisions (temperature 0).
- Preserve existing HTTP/gRPC contracts.
- No new external dependencies for Phase 1.

## Acceptance (high-level)
- AgentEngine drives Chat path end-to-end.
- Planner gate decides tool usage deterministically.
- Tool calls execute via registry and results are stored as observations/facts.
- Memory + context are injected into LLM prompt.
- Docs and ontology remain aligned.
