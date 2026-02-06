#!/bin/sh
set -e

DEFAULT_CMD=".agent/test-commands/integration.sh"

if [ -z "$1" ]; then
  if [ ! -f "$DEFAULT_CMD" ]; then
    echo "Usage: scripts/record-integration-test.sh \"<integration test command>\""
    exit 1
  fi
  CMD="$DEFAULT_CMD"
else
  CMD="$*"
fi

echo "[workspace] Running integration tests: $CMD"

# Load local environment defaults for scripted runs.
if [ -f ".env" ]; then
  set -a
  . ./.env
  set +a
fi

if [ "$CMD" = "$DEFAULT_CMD" ]; then
  sh "$CMD"
else
  sh -lc "$CMD"
fi

STAMP_DIR=".agent/test-stamps"
mkdir -p "$STAMP_DIR"

date -u +"%Y-%m-%dT%H:%M:%SZ" > "$STAMP_DIR/integration.txt"

echo "Integration tests recorded at $(cat $STAMP_DIR/integration.txt)"
