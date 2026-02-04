# Agent Workflow

## Delegation Rules
- **Codex CLI**: Always invoke `codex` to delegate implementation and review tasks when appropriate.

## SDLC Workflow (Global)
1. **Use Skill**: Always apply the `workspace-sdlc` skill for any non-trivial work.
2. **Draft Requirement**: Create INTENT/SPEC/ACCEPTANCE under `.agent/intents/<slug>/`.
3. **Requirement Review**: Submit the drafted requirement to `codex` for review.
4. **Implementation Delegation**: Submit the approved requirement to `codex` for implementation.
5. **Code Review**: Submit the implemented solution to `codex` for a separate review. You must also perform an independent review.
6. **Completion**: Once verified and passed, close the task/slug and proceed to the next one.

## Commit Policy (Mandatory)
- **Active intent required**: set via `scripts/set-active-intent.sh <slug>` and tracked in `.agent/ACTIVE_INTENT`.
- **Requirement docs required**: `INTENT.md`, `SPEC.md`, `ACCEPTANCE.md` must exist for the active intent.
- **Unit tests per commit**: run via `scripts/record-unit-test.sh "<unit test command>"` before each commit.
- **Codex CLI review per commit**: log to `.agent/intents/<slug>/LOG.md` with a `Codex Review:` entry.
- **Integration tests per story**: run via `scripts/record-integration-test.sh "<integration test command>"` before story close, and verify with `scripts/verify-intent-complete.sh`.
- **Default test commands**: set in `.agent/test-commands/unit.sh` and `.agent/test-commands/integration.sh` (optional).
- **Preferred execution**: use scripts instead of ad‑hoc commands:
  - `scripts/run-unit-tests.sh`
  - `scripts/run-codex-review.sh <slug> "summary"`
  - `scripts/run-integration-tests.sh`

## Doc/Ontology Policy
- Keep code, docs, and `docs/ontology/INDEX.md` aligned for any contract or service change.

## Planner Role
- Use `AGENT_PLANNER.md` for planning and requirements artifacts.

## Skills Repository
- **Reference**: When initialized, checking `.agent/skills` for available skills and capabilities.
- **Learning**: Maintain and update skills in `.agent/skills` to preserve knowledge of how tools and workflows operate.
