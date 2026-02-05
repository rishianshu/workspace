# Acceptance Criteria

1) rust-gateway `/api/projects` and `/api/endpoints` no longer return stub data.
2) go-agent-service exposes HTTP `/projects`, `/projects/:id`, and `/endpoints`.
3) Gateway proxies these routes to go-agent-service and passes through responses.
