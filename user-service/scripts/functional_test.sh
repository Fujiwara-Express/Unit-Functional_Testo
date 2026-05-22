#!/usr/bin/env bash
# scripts/functional_test.sh
# Functional / integration tests against a running user-service instance.
# Usage: BASE_URL=http://localhost:8080 bash scripts/functional_test.sh
#
# Exit codes:
#   0 – all tests passed
#   1 – one or more tests failed

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

# ── helpers ──────────────────────────────────────────────────────────────────

green() { printf '\033[0;32m✔ %s\033[0m\n' "$*"; }
red()   { printf '\033[0;31m✘ %s\033[0m\n' "$*"; }

assert_status() {
  local label="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    green "$label (HTTP $actual)"
    PASS=$((PASS + 1))
  else
    red "$label — expected HTTP $expected, got $actual"
    FAIL=$((FAIL + 1))
  fi
}

assert_field() {
  local label="$1" field="$2" body="$3"
  if echo "$body" | grep -q "\"$field\""; then
    green "$label — field '$field' present"
    PASS=$((PASS + 1))
  else
    red "$label — field '$field' missing in: $body"
    FAIL=$((FAIL + 1))
  fi
}

assert_no_field() {
  local label="$1" field="$2" body="$3"
  if echo "$body" | grep -q "\"$field\""; then
    red "$label — field '$field' must NOT be present: $body"
    FAIL=$((FAIL + 1))
  else
    green "$label — field '$field' absent (correct)"
    PASS=$((PASS + 1))
  fi
}

# Generate a unique email per run to avoid conflicts
UNIQUE="$(date +%s%N 2>/dev/null || date +%s)"
EMAIL="testuser_${UNIQUE}@example.com"
PASSWORD="SecurePass123!"
ADMIN_EMAIL="admin_${UNIQUE}@example.com"

echo ""
echo "=== Functional Tests: user-service ==="
echo "    BASE_URL : $BASE_URL"
echo "    Test user: $EMAIL"
echo ""

# ── TC-01: Register a new CUSTOMER ───────────────────────────────────────────
echo "--- TC-01: Register new CUSTOMER ---"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/users/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Test User\",\"email\":\"$EMAIL\",\"phone\":\"+6281234567890\",\"password\":\"$PASSWORD\",\"role\":\"CUSTOMER\"}")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)

assert_status "TC-01 Register CUSTOMER" "201" "$STATUS"
assert_field  "TC-01 user_id present"   "user_id" "$BODY"
USER_ID=$(echo "$BODY" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)

# ── TC-02: Duplicate email returns 409 ───────────────────────────────────────
echo "--- TC-02: Duplicate email ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/users/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Test User\",\"email\":\"$EMAIL\",\"phone\":\"+6281234567890\",\"password\":\"$PASSWORD\",\"role\":\"CUSTOMER\"}")
assert_status "TC-02 Duplicate email → 409" "409" "$STATUS"

# ── TC-03: Register with invalid email returns 400 ───────────────────────────
echo "--- TC-03: Invalid email ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/users/register" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bad","email":"not-an-email","phone":"+628","password":"SecurePass123!","role":"CUSTOMER"}')
assert_status "TC-03 Invalid email → 400" "400" "$STATUS"

# ── TC-04: Login with correct credentials ────────────────────────────────────
echo "--- TC-04: Login ---"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)

assert_status "TC-04 Login → 200"              "200" "$STATUS"
assert_field  "TC-04 access_token present"     "access_token"  "$BODY"
assert_field  "TC-04 refresh_token present"    "refresh_token" "$BODY"
assert_no_field "TC-04 password_hash absent"   "password_hash" "$BODY"

ACCESS_TOKEN=$(echo "$BODY"  | grep -o '"access_token":"[^"]*"'  | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

# ── TC-05: Login with wrong password returns 401 ─────────────────────────────
echo "--- TC-05: Wrong password ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"WrongPassword!\"}")
assert_status "TC-05 Wrong password → 401" "401" "$STATUS"

# ── TC-06: Get own profile (authenticated) ───────────────────────────────────
echo "--- TC-06: Get profile ---"
RESP=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/users/$USER_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)

assert_status   "TC-06 Get profile → 200"       "200"           "$STATUS"
assert_field    "TC-06 user_id present"          "user_id"       "$BODY"
assert_field    "TC-06 email present"            "email"         "$BODY"
assert_no_field "TC-06 password_hash absent"     "password_hash" "$BODY"

# ── TC-07: Get profile without token returns 401 ─────────────────────────────
echo "--- TC-07: Unauthenticated profile access ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/users/$USER_ID")
assert_status "TC-07 No token → 401" "401" "$STATUS"

# ── TC-08: Update profile ─────────────────────────────────────────────────────
echo "--- TC-08: Update profile ---"
RESP=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/users/$USER_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Name","phone":"+6289876543210"}')
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)

assert_status "TC-08 Update profile → 200" "200" "$STATUS"
assert_field  "TC-08 status present"       "status" "$BODY"

# ── TC-09: Refresh token ──────────────────────────────────────────────────────
echo "--- TC-09: Refresh token ---"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
BODY=$(echo "$RESP" | head -n -1)
STATUS=$(echo "$RESP" | tail -n 1)

assert_status "TC-09 Refresh → 200"           "200" "$STATUS"
assert_field  "TC-09 access_token present"    "access_token" "$BODY"
NEW_ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# ── TC-10: Admin endpoint rejected for non-admin ─────────────────────────────
echo "--- TC-10: Admin endpoint with CUSTOMER token ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X PUT "$BASE_URL/admin/users/$USER_ID/status" \
  -H "Authorization: Bearer $NEW_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"is_active":false}')
assert_status "TC-10 Non-admin → 403" "403" "$STATUS"

# ── TC-11: Logout ─────────────────────────────────────────────────────────────
echo "--- TC-11: Logout ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/logout" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
assert_status "TC-11 Logout → 200" "200" "$STATUS"

# ── TC-12: Refresh after logout returns 401 ───────────────────────────────────
echo "--- TC-12: Refresh after logout ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
assert_status "TC-12 Refresh after logout → 401" "401" "$STATUS"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
echo ""

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
