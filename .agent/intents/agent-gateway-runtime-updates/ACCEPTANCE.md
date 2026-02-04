# Acceptance Criteria

1) go-agent-service exposes HTTP routes (chat/tools/brain/projects/apps/health) used by rust-gateway.
2) go-agent-service starts a Temporal worker and registers workflows/activities.
3) agent.proto and rust-gateway chat/stream support provider/model/history fields.
