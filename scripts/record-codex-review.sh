#!/bin/sh
set -e

if [ -z "$1" ]; then
  echo "Usage: scripts/record-codex-review.sh <intent-slug> [summary]"
  exit 1
fi

INTENT_DIR=".agent/intents/$1"
LOG_FILE="$INTENT_DIR/LOG.md"
if [ ! -d "$INTENT_DIR" ]; then
  echo "Intent not found: $INTENT_DIR"
  exit 1
fi

SUMMARY="$2"
if [ -z "$SUMMARY" ]; then
  SUMMARY="review complete"
fi

mkdir -p "$INTENT_DIR"
if [ ! -f "$LOG_FILE" ]; then
  touch "$LOG_FILE"
fi

echo "Codex Review: $(date -u +"%Y-%m-%dT%H:%M:%SZ") - $SUMMARY" >> "$LOG_FILE"
