# Intent: Projects/Endpoints Source-of-Truth + Tests

## Goal
Document that projects/endpoints are sourced from Nucleus and add tests covering the gateway wiring behavior.

## Scope
- Update intent/spec/acceptance to state Nucleus as the source of truth.
- Add rust-gateway tests for /api/projects/:id and /api/endpoints behavior.

## Out of Scope
- Changes to Nucleus APIs or data model.
- UI changes.

## Acceptance
- Intent/spec explicitly state Nucleus as the source of truth for projects/endpoints.
- Tests cover: /api/endpoints requires projectId, /api/projects/:id proxies backend 404, /api/endpoints proxies backend success.
