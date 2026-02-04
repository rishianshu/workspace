# Intent: Commit Workflow Enforcement

## Goal
Enforce a consistent workflow where each commit is tied to an intent, includes requirement docs, unit tests, and Codex CLI review logging. Integration tests are required at story completion.

## Scope
- Add default unit/integration test command files.
- Update scripts to use defaults when no command is provided.
- Update docs to reflect policy and defaults.
- Ensure repo is a single Git repo (remove nested .git).

## Out of Scope
- CI/CD enforcement.
- Enforcing test selection by subsystem.
- Automating Codex CLI execution itself.

## Acceptance
- Pre-commit hook blocks commits when intent docs or unit test stamp or review log are missing.
- Default test commands are defined and used when no command is passed.
- README/Agent docs describe the policy and default test commands.
