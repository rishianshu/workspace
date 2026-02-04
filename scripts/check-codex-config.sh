#!/bin/sh
set -e

CONFIG="$HOME/.codex/config.toml"
if [ ! -f "$CONFIG" ]; then
  echo "[workspace] Codex config not found: $CONFIG"
  exit 1
fi

VALUE=$(grep -E '^model_reasoning_effort' "$CONFIG" | sed -E 's/.*= *"([^"]+)".*/\1/' | head -n1)
case "$VALUE" in
  minimal|low|medium|high)
    exit 0
    ;;
  *)
    echo "[workspace] Invalid model_reasoning_effort: $VALUE"
    echo "Allowed: minimal | low | medium | high"
    exit 1
    ;;
esac
