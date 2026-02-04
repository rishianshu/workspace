#!/bin/sh
set -e

ACTIVE_FILE=".agent/ACTIVE_INTENT"
if [ ! -f "$ACTIVE_FILE" ]; then
  echo "ACTIVE_INTENT is not set."
  exit 1
fi

INTENT_SLUG=$(cat "$ACTIVE_FILE" | tr -d '[:space:]')
INTENT_DIR=".agent/intents/$INTENT_SLUG"

if [ ! -d "$INTENT_DIR" ]; then
  echo "Intent not found: $INTENT_DIR"
  exit 1
fi

for f in INTENT.md SPEC.md ACCEPTANCE.md LOG.md; do
  if [ ! -f "$INTENT_DIR/$f" ]; then
    echo "Missing $f in $INTENT_DIR"
    exit 1
  fi
done

if ! grep -qi "codex review" "$INTENT_DIR/LOG.md"; then
  echo "LOG.md missing Codex review entry"
  exit 1
fi

if [ ! -f ".agent/test-stamps/unit.txt" ]; then
  echo "Unit tests not recorded."
  exit 1
fi

if [ ! -f ".agent/test-stamps/integration.txt" ]; then
  echo "Integration tests not recorded."
  exit 1
fi

echo "Intent verification passed for $INTENT_SLUG"
