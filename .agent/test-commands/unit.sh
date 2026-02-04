#!/bin/sh
set -e

# Default unit test command set. Adjust as needed.
# workspace-web lint is currently disabled due to ESLint config export error.

(
  cd go-agent-service
  go test ./...
)

(
  cd rust-gateway
  cargo test --workspace
)
