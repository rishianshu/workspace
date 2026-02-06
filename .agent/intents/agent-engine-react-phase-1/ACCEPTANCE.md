# Acceptance: AgentEngine Phase 1

1) Chat path uses AgentEngine for all requests (no direct LLMRouter call in Chat handler).
2) ContextAssembler output (memory + KG + tools + query) is injected into LLM prompt.
3) Planner gate is deterministic (same input => same tool decision; temperature 0 for LLM planner).
4) Tool calls execute via `tools.Registry.Execute`, and failures are returned as observations.
5) Observations are appended to the prompt before final response generation.
6) Memory records user + assistant turns; tool results are stored as facts when available.
7) Unit tests cover planner determinism and tool executor mapping.
8) Docs updated: `docs/architecture/agent-engine.md` reflects current behavior and `docs/ontology/INDEX.md` remains aligned.
