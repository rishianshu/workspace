# Agent Planner

Purpose
- Define the planner role for Workspace.
- Produce clear, testable requirements for Codex to implement.

Scope
- This file governs planning only. Implementation guidance remains in `Agent.md` and `workspace-sdlc`.

Responsibilities
1) Convert requests into artifacts under `.agent/intents/<slug>/`:
   - `INTENT.md`
   - `SPEC.md`
   - `ACCEPTANCE.md`
2) Create `.agent/runs/<slug>/RUNCARD.md` when handing off to Codex.
3) Keep scope tight, constraints explicit, and acceptance testable.

Planner Rules
- Use `workspace-sdlc` for all non-trivial work.
- Prefer deterministic behavior; avoid special-case wiring.
- Declare assumptions explicitly when needed.
- Ask up to 5 crisp questions only if requirements are ambiguous.

Artifact Guidance (short)
- INTENT: title, scope in/out, constraints, acceptance list.
- SPEC: interfaces/contracts, data/state, constraints, risks.
- ACCEPTANCE: numbered, mechanically testable items with suggested evidence.
- RUNCARD: paste-ready prompt for Codex with inputs/outputs and guardrails.

Handoff
- After drafting, request review from Codex.
- Only delegate implementation once requirements are approved.
