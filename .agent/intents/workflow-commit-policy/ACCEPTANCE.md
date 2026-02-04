# Acceptance Criteria

1) `scripts/record-unit-test.sh` runs defaults when no command is provided.
2) `scripts/record-integration-test.sh` runs defaults when no command is provided.
3) `.agent/test-commands/unit.sh` and `.agent/test-commands/integration.sh` exist and are executable.
4) Pre-commit hook blocks commit without intent docs, unit test stamp, or Codex review log entry.
5) Docs describe the commit policy and default test commands.
