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
    ;;
  "")
    ;;
  *)
    echo "[workspace] Invalid model_reasoning_effort: $VALUE"
    echo "Allowed: minimal | low | medium | high"
    exit 1
    ;;
esac

MODEL=$(grep -E '^model' "$CONFIG" | sed -E 's/.*= *"([^"]+)".*/\1/' | head -n1)
case "$MODEL" in
  ""|gpt-5-codex)
    exit 0
    ;;
  gpt-5.3-codex)
    echo "[workspace] Unsupported Codex model in config: $MODEL"
    echo "Review scripts will use CODEX_REVIEW_MODEL=${CODEX_REVIEW_MODEL:-gpt-5-codex}."
    exit 0
    ;;
  *)
    echo "[workspace] Warning: unverified model in config: $MODEL"
    echo "Review script uses CODEX_REVIEW_MODEL=${CODEX_REVIEW_MODEL:-gpt-5-codex}."
    ;;
esac
