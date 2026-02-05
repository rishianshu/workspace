# Spec: Projects/Endpoints Source-of-Truth + Tests

## Source of Truth
- Projects and endpoints are fetched from Nucleus via `go-agent-service/internal/nucleus/client.go`.
- Gateway only proxies to go-agent-service; it does not fabricate data.

## Tests (rust-gateway)
- `list_endpoints` returns 400 when `projectId` is missing.
- `get_project` returns 404 when backend returns 404.
- `list_endpoints` returns 200 when backend returns success payload.
