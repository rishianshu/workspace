# Acceptance Criteria

1) rust-gateway chat uses AGENT_SERVICE_URL (no hardcoded localhost).
2) go-agent-service registers /action and handles it.
3) HTTP chat handler processes attachedFiles into query context.
4) workflow handlers return 503 when workflow engine is unavailable.
