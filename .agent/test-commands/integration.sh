#!/bin/sh
set -e

# Default integration smoke checks. Requires stack running.

curl -fsS --http0.9 http://localhost:8082/health > /dev/null
curl -fsS --http0.9 http://localhost:9100/health > /dev/null

scripts/integration-nucleus-mcp.sh

printf "Integration smoke checks passed\n"
