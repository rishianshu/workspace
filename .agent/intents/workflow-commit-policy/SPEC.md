# Spec: Commit Workflow Enforcement

## Interfaces
- `.githooks/pre-commit`: enforces policy before commit.
- `.agent/ACTIVE_INTENT`: points to active intent slug.
- `.agent/test-commands/unit.sh`: default unit command set.
- `.agent/test-commands/integration.sh`: default integration smoke checks.
- Scripts:
  - `scripts/set-active-intent.sh`
  - `scripts/record-unit-test.sh`
  - `scripts/record-integration-test.sh`
  - `scripts/record-codex-review.sh`
  - `scripts/verify-intent-complete.sh`

## Behavior
- `record-*-test.sh` runs a provided command or defaults to the script in `.agent/test-commands/`.
- Pre-commit enforces:
  - active intent exists
  - INTENT/SPEC/ACCEPTANCE present
  - LOG.md contains a Codex Review entry
  - unit test stamp exists and is < 6 hours old

## Risks
- Default unit command may be slow; can be edited.
- Integration smoke checks require local stack to be up.
