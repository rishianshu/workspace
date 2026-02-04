#!/bin/sh
set -e

DEFAULT_CMD=".agent/test-commands/unit.sh"

if [ -z "$1" ]; then
  if [ ! -f "$DEFAULT_CMD" ]; then
    echo "Usage: scripts/record-unit-test.sh \"<unit test command>\""
    exit 1
  fi
  CMD="$DEFAULT_CMD"
else
  CMD="$*"
fi

echo "[workspace] Running unit tests: $CMD"

if [ "$CMD" = "$DEFAULT_CMD" ]; then
  sh "$CMD"
else
  sh -lc "$CMD"
fi

STAMP_DIR=".agent/test-stamps"
mkdir -p "$STAMP_DIR"

date -u +"%Y-%m-%dT%H:%M:%SZ" > "$STAMP_DIR/unit.txt"

echo "Unit tests recorded at $(cat $STAMP_DIR/unit.txt)"
