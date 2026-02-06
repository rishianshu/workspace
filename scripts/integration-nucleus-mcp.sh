#!/usr/bin/env bash
set -euo pipefail

NUCLEUS_API_URL=${NUCLEUS_API_URL:-http://localhost:4010/graphql}
MCP_URL=${MCP_URL:-http://localhost:9100}
GATEWAY_URL=${GATEWAY_URL:-http://localhost:8082}
PROJECT_ID=${PROJECT_ID:-global}
USER_ID=${USER_ID:-dev-admin}

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

NUCLEUS_API_URL="$(normalize_local_url "$NUCLEUS_API_URL")"
MCP_URL="$(normalize_local_url "$MCP_URL")"
GATEWAY_URL="$(normalize_local_url "$GATEWAY_URL")"

export TOKEN
TOKEN=$(scripts/fetch-keycloak-token.sh)

python3 - <<'PY'
import base64, json, os, sys

token = os.environ.get("TOKEN", "")
expected = os.environ.get("KEYCLOAK_ISSUER_EXPECTED", "http://localhost:8081/realms/nucleus")
if not token:
    print("Missing TOKEN")
    sys.exit(1)
parts = token.split(".")
if len(parts) < 2:
    print("Invalid TOKEN format")
    sys.exit(1)
payload = parts[1]
payload += "=" * (-len(payload) % 4)
try:
    data = json.loads(base64.urlsafe_b64decode(payload.encode()))
except Exception as exc:
    print("Failed to decode TOKEN payload:", exc)
    sys.exit(1)
issuer = data.get("iss")
if issuer != expected:
    print(f"Unexpected token issuer: {issuer} (expected {expected})")
    sys.exit(1)
PY

python3 - <<'PY'
import json, os, shlex, sys, urllib.request

token = os.environ.get("TOKEN", "")
endpoint = os.environ.get("NUCLEUS_API_URL", "http://localhost:4010/graphql")
if not token:
    print("Missing TOKEN")
    sys.exit(1)

query = '{ metadataProjects { id } }'
req = urllib.request.Request(endpoint, data=json.dumps({'query': query}).encode(), method='POST')
req.add_header('Content-Type','application/json')
req.add_header('Authorization', f'Bearer {token}')
with urllib.request.urlopen(req) as r:
    data = json.loads(r.read().decode())
projects = data.get('data', {}).get('metadataProjects', [])
if not projects:
    print('No projects returned')
    sys.exit(1)
PY

python3 - <<'PY'
import json, os, sys, urllib.request

token = os.environ.get("TOKEN", "")
endpoint = os.environ.get("NUCLEUS_API_URL", "http://localhost:4010/graphql")
project_id = os.environ.get("PROJECT_ID", "global")
if not token:
    print("Missing TOKEN")
    sys.exit(1)

query = '{ metadataEndpoints(projectId: "%s") { id name } }' % project_id
req = urllib.request.Request(endpoint, data=json.dumps({'query': query}).encode(), method='POST')
req.add_header('Content-Type','application/json')
req.add_header('Authorization', f'Bearer {token}')
with urllib.request.urlopen(req) as r:
    data = json.loads(r.read().decode())
endpoints = data.get('data', {}).get('metadataEndpoints', [])
if not endpoints:
    print('No endpoints returned')
    sys.exit(1)
PY

TOOLS_URL="${MCP_URL%/}/v1/tools?userId=${USER_ID}&projectId=${PROJECT_ID}"
# Create app registry entries using a real Nucleus endpoint
APP_INFO=$(python3 - <<'PY'
import json, os, shlex, sys, urllib.request

token = os.environ.get("TOKEN", "")
endpoint = os.environ.get("NUCLEUS_API_URL", "http://localhost:4010/graphql")
project_id = os.environ.get("PROJECT_ID", "global")
if not token:
    print("Missing TOKEN")
    sys.exit(1)

query = '{ metadataEndpoints(projectId: "%s", includeDeleted: false) { id name config } }' % project_id
req = urllib.request.Request(endpoint, data=json.dumps({'query': query}).encode(), method='POST')
req.add_header('Content-Type','application/json')
req.add_header('Authorization', f'Bearer {token}')
with urllib.request.urlopen(req) as r:
    data = json.loads(r.read().decode())
endpoints = data.get('data', {}).get('metadataEndpoints', [])
for ep in endpoints:
    cfg = ep.get('config') or {}
    template_id = cfg.get('templateId') or cfg.get('template_id')
    if template_id:
        name = ep.get('name') or ep.get('id')
        print("ENDPOINT_ID=%s" % shlex.quote(str(ep.get('id'))))
        print("ENDPOINT_NAME=%s" % shlex.quote(name.replace("\\n", " ")))
        print("TEMPLATE_ID=%s" % shlex.quote(str(template_id)))
        sys.exit(0)
print("No endpoint with templateId found for project:", project_id)
sys.exit(1)
PY
)

eval "$APP_INFO"

export APP_INSTANCE_JSON
APP_INSTANCE_JSON=$(curl -sS -X POST "${GATEWAY_URL%/}/api/apps/instances" \
  -H "Content-Type: application/json" \
  -d "{\"templateId\":\"${TEMPLATE_ID}\",\"instanceKey\":\"${ENDPOINT_ID}\",\"displayName\":\"${ENDPOINT_NAME}\"}" || true)
APP_INSTANCE_ID=$(python3 - <<'PY'
import json, os, sys
raw = os.environ.get("APP_INSTANCE_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    print("")
    sys.exit(1)
print(data.get("id") or data.get("ID",""))
PY
)

if [ -z "$APP_INSTANCE_ID" ]; then
  echo "App instance create failed: $APP_INSTANCE_JSON"
  exit 1
fi

export USER_APP_JSON
USER_APP_JSON=$(curl -sS -X POST "${GATEWAY_URL%/}/api/apps/users" \
  -H "Content-Type: application/json" \
  -d "{\"userId\":\"${USER_ID}\",\"appInstanceId\":\"${APP_INSTANCE_ID}\",\"credentialRef\":\"env\"}" || true)
USER_APP_ID=$(python3 - <<'PY'
import json, os, sys
raw = os.environ.get("USER_APP_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    print("")
    sys.exit(1)
print(data.get("id") or data.get("ID",""))
PY
)

if [ -z "$USER_APP_ID" ]; then
  echo "User app create failed: $USER_APP_JSON"
  exit 1
fi

export PROJECT_APP_JSON
PROJECT_APP_JSON=$(curl -sS -X POST "${GATEWAY_URL%/}/api/apps/projects" \
  -H "Content-Type: application/json" \
  -d "{\"projectId\":\"${PROJECT_ID}\",\"userAppId\":\"${USER_APP_ID}\",\"endpointId\":\"${ENDPOINT_ID}\",\"isDefault\":true}" || true)
PROJECT_APP_ID=$(python3 - <<'PY'
import json, os, sys
raw = os.environ.get("PROJECT_APP_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    print("")
    sys.exit(1)
print(data.get("id") or data.get("ID",""))
PY
)

if [ -z "$PROJECT_APP_ID" ]; then
  echo "Project app create failed: $PROJECT_APP_JSON"
  exit 1
fi

export RESP
RESP=$(curl -sS --http0.9 "$TOOLS_URL" || true)
python3 - <<'PY'
import json, os, sys
resp_raw = os.environ.get("RESP", "")
if not resp_raw:
    print("MCP tools response is empty")
    sys.exit(1)
try:
    resp = json.loads(resp_raw)
except Exception as exc:
    print("MCP tools response is not JSON:", resp_raw)
    sys.exit(1)
if not isinstance(resp, list):
    print("MCP tools response is not a list:", resp)
    sys.exit(1)
if len(resp) == 0:
    print("MCP tools list is empty")
    sys.exit(1)
if not any(isinstance(item, dict) and str(item.get("name","")).startswith("app/") for item in resp):
    print("MCP tools list missing app/* tools")
    sys.exit(1)
PY

echo "Integration auth checks passed"
