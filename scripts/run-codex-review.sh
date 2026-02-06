#!/bin/sh
set -e

if [ -z "$1" ]; then
  echo "Usage: scripts/run-codex-review.sh <intent-slug> [summary]"
  exit 1
fi

INTENT_SLUG="$1"
SUMMARY="$2"
if [ -z "$SUMMARY" ]; then
  SUMMARY="requirements-based review"
fi
REVIEW_MODEL="${CODEX_REVIEW_MODEL:-gpt-5-codex}"

/Users/rishikeshkumar/Development/Workspace/scripts/check-codex-config.sh

INTENT_DIR=".agent/intents/$INTENT_SLUG"
LOG_FILE="$INTENT_DIR/LOG.md"
if [ ! -d "$INTENT_DIR" ]; then
  echo "Intent not found: $INTENT_DIR"
  exit 1
fi

REQ_INTENT="$INTENT_DIR/INTENT.md"
REQ_SPEC="$INTENT_DIR/SPEC.md"
REQ_ACCEPT="$INTENT_DIR/ACCEPTANCE.md"

for f in "$REQ_INTENT" "$REQ_SPEC" "$REQ_ACCEPT"; do
  if [ ! -f "$f" ]; then
    echo "Missing requirement doc: $f"
    exit 1
  fi
done

mkdir -p "$INTENT_DIR"
if [ ! -f "$LOG_FILE" ]; then
  touch "$LOG_FILE"
fi

printf "\nCodex Review Output (%s)\n" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "$LOG_FILE"
printf '%s\n' "--------------------------------------------------" >> "$LOG_FILE"

DIFF=$(git diff --no-color)

codex exec --model "$REVIEW_MODEL" "You are reviewing a commit against explicit requirements. Use the full context below.

INTENT:\n$(cat "$REQ_INTENT")\n\nSPEC:\n$(cat "$REQ_SPEC")\n\nACCEPTANCE:\n$(cat "$REQ_ACCEPT")\n\nGIT DIFF:\n$DIFF\n\nTask: For each Acceptance criterion, mark PASS or FAIL and cite evidence with file paths and line references when possible. Then list any additional risks/regressions not covered by Acceptance." >> "$LOG_FILE"

/Users/rishikeshkumar/Development/Workspace/scripts/record-codex-review.sh "$INTENT_SLUG" "$SUMMARY"
