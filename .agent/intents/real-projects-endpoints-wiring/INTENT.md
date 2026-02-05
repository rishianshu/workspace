# Intent: Real Projects/Endpoints Wiring in Gateway

## Goal
Replace stubbed `/api/projects` and `/api/endpoints` responses in rust-gateway with real backend wiring to go-agent-service/Nucleus.

## Scope
- Remove stub responses in `rust-gateway/src/routes/tools.rs` for projects/endpoints.
- Add/verify corresponding HTTP handlers in go-agent-service.
- Ensure gateway routes call real backend endpoints.

## Out of Scope
- UI changes beyond consuming existing routes.
- Authentication/authorization.
- Nucleus data model changes.

## Acceptance
- `/api/projects` returns real data via backend (no stubbed objects).
- `/api/projects/:id` returns real data or a clear 404 from backend.
- `/api/endpoints` returns real data via backend (no stubbed objects).
- Gateway contains no stubbed project/endpoint responses.
