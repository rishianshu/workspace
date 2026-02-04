#!/bin/sh
set -e

# Default integration smoke checks. Requires stack running.

curl -fsS http://localhost:8082/health > /dev/null
curl -fsS http://localhost:9002/health > /dev/null
curl -fsS http://localhost:9100/health > /dev/null

printf "Integration smoke checks passed\n"
