# Intent: Untracked Source Cleanup

## Goal
Bring untracked source files under version control and ignore build artifacts.

## Scope
- Add new source files (agent LLM router, endpoints store, workflow temporal helpers, start-dev script).
- Add generated gRPC/proto Go files required for builds.
- Ignore local build artifact `go-agent-service/agent-service`.

## Out of Scope
- Regenerating protobufs.
- Reworking build tooling.

## Acceptance
- All necessary untracked source and generated files are committed.
- Build artifacts are added to .gitignore.
- Repo has no untracked files except intentionally ignored outputs.
