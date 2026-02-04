# Intent: Agent + Gateway Runtime Updates

## Goal
Align go-agent-service and rust-gateway for updated HTTP/streaming behavior, LLM routing fields, and workflow execution.

## Scope
- Add HTTP handlers alongside gRPC for gateway compatibility.
- Start a Temporal worker in the agent service.
- Extend chat/proto to carry provider/model/history.
- Update rust-gateway routes to call the new HTTP endpoints.

## Out of Scope
- MCP/keystore services (already handled separately).
- UI changes.
- Nucleus project selection flows.

## Acceptance
- Agent service exposes HTTP endpoints used by rust-gateway.
- Temporal worker is started in agent service.
- Proto and gateway support provider/model/history in chat.
