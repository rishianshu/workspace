# Spec: Real Projects/Endpoints Wiring in Gateway

## Interfaces
- Gateway:
  - GET `/api/projects` → proxy to go-agent-service `/projects`.
  - GET `/api/projects/:id` → proxy to go-agent-service `/projects/:id` (new if missing).
  - GET `/api/endpoints` → proxy to go-agent-service `/endpoints` (new if missing).
- go-agent-service:
  - HTTP handlers for `/projects`, `/projects/:id`, `/endpoints` returning JSON.

## Behavior
- No stubbed project/endpoint objects in gateway.
- 404 is returned when project id is not found (from backend).
- Errors are surfaced as 502/503 at gateway when backend is unavailable.
