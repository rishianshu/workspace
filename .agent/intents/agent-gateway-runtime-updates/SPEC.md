# Spec: Agent + Gateway Runtime Updates

## Interfaces
- Agent HTTP routes: `/chat`, `/tools`, `/brain/search`, `/projects`, `/apps/*`, `/health`.
- gRPC still available on `GRPC_PORT`.
- rust-gateway calls agent HTTP for chat/streaming.

## Behavior
- Agent runs gRPC and HTTP in the same process.
- Temporal worker is started with workflow + activities registered.
- Chat requests include optional `provider`, `model`, and `history`.
