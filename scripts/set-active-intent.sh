#!/bin/sh
set -e

if [ -z "$1" ]; then
  echo "Usage: scripts/set-active-intent.sh <intent-slug>"
  exit 1
fi

INTENT_DIR=".agent/intents/$1"
if [ ! -d "$INTENT_DIR" ]; then
  echo "Intent not found: $INTENT_DIR"
  exit 1
fi

echo "$1" > .agent/ACTIVE_INTENT
printf "Active intent set to %s\n" "$1"
