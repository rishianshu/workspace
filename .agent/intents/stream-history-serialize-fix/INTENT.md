# Intent: Stream History Serialization Fix

## Goal
Fix unit test failure in rust-gateway by making stream HistoryMessage serializable.

## Scope
- Add serde Serialize to stream HistoryMessage.

## Out of Scope
- Refactors of streaming logic.
- API contract changes.

## Acceptance
- Unit tests pass.
- HistoryMessage in stream route implements Serialize.
