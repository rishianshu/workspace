#!/usr/bin/env bash
set -euo pipefail

KEYCLOAK_URL=${KEYCLOAK_URL:-http://localhost:8081}
KEYCLOAK_REALM=${KEYCLOAK_REALM:-nucleus}
KEYCLOAK_CLIENT_ID=${KEYCLOAK_CLIENT_ID:-}
KEYCLOAK_CLIENT_SECRET=${KEYCLOAK_CLIENT_SECRET:-}
USERNAME=${KEYCLOAK_USERNAME:-${NUCLEUS_USERNAME:-}}
PASSWORD=${KEYCLOAK_PASSWORD:-${NUCLEUS_PASSWORD:-}}

normalize_local_url() {
  local input="${1:-}"
  if [ -z "$input" ]; then
    printf '%s' "$input"
    return 0
  fi
  case "$input" in
    *host.docker.internal*)
      if python3 - <<'PY' >/dev/null 2>&1
import socket
socket.getaddrinfo("host.docker.internal", None)
PY
      then
        printf '%s' "$input"
      else
        printf '%s' "$input" | sed 's/host\.docker\.internal/localhost/g'
      fi
      ;;
    *)
      printf '%s' "$input"
      ;;
  esac
}

KEYCLOAK_URL="$(normalize_local_url "$KEYCLOAK_URL")"

if [ -z "$KEYCLOAK_CLIENT_ID" ]; then
  echo "KEYCLOAK_CLIENT_ID is required" >&2
  exit 1
fi
if [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; then
  echo "KEYCLOAK_USERNAME/KEYCLOAK_PASSWORD or NUCLEUS_USERNAME/NUCLEUS_PASSWORD are required" >&2
  exit 1
fi

TOKEN_URL="${KEYCLOAK_URL%/}/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token"

RESP=$(curl -sS -X POST "$TOKEN_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "client_id=${KEYCLOAK_CLIENT_ID}" \
  -d "username=${USERNAME}" \
  -d "password=${PASSWORD}" \
  $( [ -n "$KEYCLOAK_CLIENT_SECRET" ] && printf -- "-d client_secret=%s" "$KEYCLOAK_CLIENT_SECRET" ))

python3 - <<PY
import json, sys
resp = json.loads('''$RESP''')
if 'access_token' not in resp:
    print(resp)
    sys.exit(1)
print(resp['access_token'])
PY
