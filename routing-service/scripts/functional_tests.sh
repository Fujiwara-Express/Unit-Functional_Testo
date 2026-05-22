#!/usr/bin/env bash
# scripts/functional_tests.sh
# Functional / smoke tests against a running routing-service instance.
# Usage: BASE_URL=http://localhost:8080 bash scripts/functional_tests.sh
#
# Exit codes:
#   0 — all tests passed
#   1 — one or more tests failed

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

# ── helpers ───────────────────────────────────────────────────────────────────

green() { echo -e "\033[32m✔  $*\033[0m"; }
red()   { echo -e "\033[31m✘  $*\033[0m"; }

assert_status() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    green "$label (HTTP $actual)"
    PASS=$((PASS + 1))
  else
    red "$label — expected HTTP $expected, got $actual"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local label="$1"
  local needle="$2"
  local haystack="$3"
  if echo "$haystack" | grep -q "$needle"; then
    green "$label (contains '$needle')"
    PASS=$((PASS + 1))
  else
    red "$label — response does not contain '$needle'"
    echo "  Response: $haystack"
    FAIL=$((FAIL + 1))
  fi
}

# ── Test 1: GET /routing/nodes — should return 200 + array ───────────────────
echo ""
echo "=== Test 1: GET /routing/nodes ==="
RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/routing/nodes")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "GET /routing/nodes returns 200" 200 "$STATUS"
assert_contains "Response is a JSON array" "\[" "$BODY"

# ── Test 2: GET /routing/edges — should return 200 + array ───────────────────
echo ""
echo "=== Test 2: GET /routing/edges ==="
RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/routing/edges")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "GET /routing/edges returns 200" 200 "$STATUS"
assert_contains "Response is a JSON array" "\[" "$BODY"

# ── Test 3: POST /routing/nodes — create a new node ──────────────────────────
echo ""
echo "=== Test 3: POST /routing/nodes ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/routing/nodes" \
  -H "Content-Type: application/json" \
  -d '{"hub_id":"HUB_TEST_CI","city_code":"TST","latitude":-7.0,"longitude":110.0}')
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "POST /routing/nodes returns 201" 201 "$STATUS"
assert_contains "Response contains node_id" "node_id" "$BODY"

# ── Test 4: POST /routing/nodes — duplicate hub_id returns 409 ───────────────
echo ""
echo "=== Test 4: POST /routing/nodes duplicate ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/routing/nodes" \
  -H "Content-Type: application/json" \
  -d '{"hub_id":"HUB_JKT","city_code":"JKT","latitude":-6.2,"longitude":106.8}')
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "Duplicate hub_id returns 409" 409 "$STATUS"
assert_contains "Error is DUPLICATE_HUB" "DUPLICATE_HUB" "$BODY"

# ── Test 5: POST /routing/nodes — missing fields returns 400 ─────────────────
echo ""
echo "=== Test 5: POST /routing/nodes missing fields ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/routing/nodes" \
  -H "Content-Type: application/json" \
  -d '{"hub_id":"HUB_MISSING"}')
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "Missing fields returns 400" 400 "$STATUS"
assert_contains "Error is VALIDATION_ERROR" "VALIDATION_ERROR" "$BODY"

# ── Test 6: GET /routing/route — valid route ──────────────────────────────────
echo ""
echo "=== Test 6: GET /routing/route (valid) ==="
RESP=$(curl -s -w "\n%{http_code}" \
  "$BASE_URL/routing/route?origin=HUB_JKT&destination=HUB_BDG&service_type=REG")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "GET /routing/route returns 200" 200 "$STATUS"
assert_contains "Response contains origin" "origin" "$BODY"
assert_contains "Response contains route array" "route" "$BODY"
assert_contains "Response contains total_distance_km" "total_distance_km" "$BODY"

# ── Test 7: GET /routing/route — no route found returns 404 ──────────────────
echo ""
echo "=== Test 7: GET /routing/route (no path) ==="
RESP=$(curl -s -w "\n%{http_code}" \
  "$BASE_URL/routing/route?origin=HUB_NOWHERE&destination=HUB_ALSO_NOWHERE&service_type=REG")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "No route returns 404" 404 "$STATUS"
assert_contains "Error is NO_ROUTE_FOUND" "NO_ROUTE_FOUND" "$BODY"

# ── Test 8: GET /routing/route — missing params returns 400 ──────────────────
echo ""
echo "=== Test 8: GET /routing/route (missing params) ==="
RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/routing/route?origin=HUB_JKT")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "Missing params returns 400" 400 "$STATUS"
assert_contains "Error is VALIDATION_ERROR" "VALIDATION_ERROR" "$BODY"

# ── Test 9: POST /routing/edges — create edge ────────────────────────────────
echo ""
echo "=== Test 9: POST /routing/edges ==="
# Get node IDs first
NODES=$(curl -s "$BASE_URL/routing/nodes")
FROM_ID=$(echo "$NODES" | python3 -c "import sys,json; nodes=json.load(sys.stdin); print(next(n['node_id'] for n in nodes if n['hub_id']=='HUB_JKT'))" 2>/dev/null || echo "")
TO_ID=$(echo "$NODES"   | python3 -c "import sys,json; nodes=json.load(sys.stdin); print(next(n['node_id'] for n in nodes if n['hub_id']=='HUB_BDG'))" 2>/dev/null || echo "")

if [ -n "$FROM_ID" ] && [ -n "$TO_ID" ]; then
  RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/routing/edges" \
    -H "Content-Type: application/json" \
    -d "{\"from_node_id\":\"$FROM_ID\",\"to_node_id\":\"$TO_ID\",\"distance_km\":155.0,\"avg_transit_hours\":3.5,\"transport_mode\":\"DARAT\"}")
  BODY=$(echo "$RESP" | head -n -1)
  STATUS=$(echo "$RESP" | tail -n 1)
  assert_status "POST /routing/edges returns 201" 201 "$STATUS"
  assert_contains "Response contains edge_id" "edge_id" "$BODY"
else
  red "POST /routing/edges — could not resolve node IDs, skipping"
  FAIL=$((FAIL + 1))
fi

# ── Test 10: PATCH /routing/edges/:id — update edge ──────────────────────────
echo ""
echo "=== Test 10: PATCH /routing/edges/:id ==="
EDGES=$(curl -s "$BASE_URL/routing/edges")
EDGE_ID=$(echo "$EDGES" | python3 -c "import sys,json; edges=json.load(sys.stdin); print(edges[0]['edge_id'])" 2>/dev/null || echo "")

if [ -n "$EDGE_ID" ]; then
  RESP=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/routing/edges/$EDGE_ID" \
    -H "Content-Type: application/json" \
    -d '{"avg_transit_hours":4.0}')
  BODY=$(echo "$RESP" | head -n -1)
  STATUS=$(echo "$RESP" | tail -n 1)
  assert_status "PATCH /routing/edges/:id returns 200" 200 "$STATUS"
  assert_contains "Response contains edge_id" "edge_id" "$BODY"
else
  red "PATCH /routing/edges/:id — no edges found, skipping"
  FAIL=$((FAIL + 1))
fi

# ── Test 11: PATCH /routing/edges/:id — non-existent edge returns 404 ────────
echo ""
echo "=== Test 11: PATCH /routing/edges (not found) ==="
RESP=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/routing/edges/non-existent-id" \
  -H "Content-Type: application/json" \
  -d '{"avg_transit_hours":5.0}')
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)
assert_status "Non-existent edge returns 404" 404 "$STATUS"
assert_contains "Error is EDGE_NOT_FOUND" "EDGE_NOT_FOUND" "$BODY"

# ── Test 12: Cache hit — second call to same route should be fast ─────────────
echo ""
echo "=== Test 12: Cache hit (second route request) ==="
START=$(date +%s%N)
RESP=$(curl -s -w "\n%{http_code}" \
  "$BASE_URL/routing/route?origin=HUB_JKT&destination=HUB_BDG&service_type=REG")
END=$(date +%s%N)
STATUS=$(echo "$RESP" | tail -n 1)
ELAPSED_MS=$(( (END - START) / 1000000 ))
assert_status "Cached route returns 200" 200 "$STATUS"
if [ "$ELAPSED_MS" -lt 500 ]; then
  green "Cached response latency: ${ELAPSED_MS}ms (< 500ms)"
  PASS=$((PASS + 1))
else
  red "Cached response latency: ${ELAPSED_MS}ms (>= 500ms threshold)"
  FAIL=$((FAIL + 1))
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "════════════════════════════════════════"
echo "  Functional Test Results"
echo "  Passed: $PASS  |  Failed: $FAIL"
echo "════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
