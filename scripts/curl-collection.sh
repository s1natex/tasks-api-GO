#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8081}"

echo "# health (app)"
curl -sS -i "$BASE_URL/health"
echo

echo "# create task"
curl -sS -i -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{"title":"from curl collection"}'
echo

echo "# create task (422)"
curl -sS -i -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{"title":""}'
echo

echo "# list tasks"
curl -sS -i "$BASE_URL/tasks"
echo

echo "# metrics (first 10 lines)"
curl -sS "$BASE_URL/metrics" | head -n 10
echo

echo "# cors preflight"
curl -sS -i -X OPTIONS "$BASE_URL/tasks" \
  -H "Origin: https://example.com" \
  -H "Access-Control-Request-Method: POST"
echo

echo "# health (drain)"
curl -sS -i "$HEALTH_URL/health"
echo
