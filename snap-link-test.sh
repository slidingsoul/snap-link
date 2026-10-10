#!/usr/bin/env bash
# snap-link end-to-end smoke test (dev only — test user below is not for production).
# Replaces snap-link.http. Aborts on first failure (set -e), non-zero exit for CI.
#
# Prereqs:
#   1. Neon Console → Branch → Auth enabled on the SAME branch as DATABASE_URL.
#   2. Trusted domains include $AUTH_ORIGIN (default http://localhost:8081),
#      else auth rejects with MISSING_ORIGIN / untrusted origin.
#   3. Server running: go run main.go (needs DATABASE_URL + NEON_AUTH_BASE_URL).
#   4. curl + jq installed.
#
# Auth path note: NEON_AUTH_BASE_URL already ends in /auth
# (e.g. https://ep-xxx.aws.neon.tech/neondb/auth), so endpoints are
# appended directly: $AUTH_BASE/sign-up/email.
# If you get 404 from auth, retry with $AUTH_BASE/auth/sign-up/email
# (means your base URL does NOT include the /auth suffix).
#
# Usage:
#   export NEON_AUTH_BASE_URL="https://ep-xxx.../neondb/auth"
#   ./snap-link-test.sh
#   # optional overrides:
#   API_BASE=http://localhost:8081 AUTH_ORIGIN=http://localhost:8081 TEST_EMAIL=dev@example.com TEST_PASSWORD=dev-password-123 EXPECTED_LIMIT=5 ./snap-link-test.sh
# NOTE: restart the server right before running for a clean fixed window —
# leftover POST /snap counts from a prior run shift every assertion below.
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8081}"
# Expected per-IP limit (must match the running server's internal/ratelimit).
EXPECTED_LIMIT="${EXPECTED_LIMIT:-5}"
# Better Auth requires Origin (browsers send it; curl does not).
# Must be listed in Neon Console → Branch → Auth → trusted domains.
AUTH_ORIGIN="${AUTH_ORIGIN:-http://localhost:8081}"
AUTH_BASE="${NEON_AUTH_BASE_URL:?set NEON_AUTH_BASE_URL to your branch .../auth URL}"
EMAIL="${TEST_EMAIL:-dev@example.com}"
PASS="${TEST_PASSWORD:-dev-password-123}"
NAME="Dev User"
TARGET_URL="https://github.com/golang/go"

command -v curl >/dev/null || { echo "❌ curl is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "❌ jq is required (brew install jq / apt install jq)" >&2; exit 1; }

JAR="$(mktemp)"
trap 'rm -f "$JAR"' EXIT

pass() { printf '✅ %s\n' "$1"; }
fail() { printf '❌ %s (got %s, want %s)\n' "$1" "$2" "$3" >&2; exit 1; }

# assert_status <actual> <expected> <label>
assert_status() {
  if [ "$1" = "$2" ]; then
    pass "$3 → $1"
  else
    fail "$3" "$1" "$2"
  fi
}

echo "→ API: $API_BASE"
echo "→ AUTH: $AUTH_BASE"
echo "→ ORIGIN: $AUTH_ORIGIN"

# 1. Health check (public, proves server + DB)
code=$(curl -s -o /tmp/snap-health.json -w "%{http_code}" "$API_BASE/health")
assert_status "$code" "200" "GET /health"
grep -q healthy /tmp/snap-health.json || fail "GET /health body" "$(cat /tmp/snap-health.json)" '{"status":"healthy"}'
pass "GET /health body healthy"

# 2. Register (tolerate already-exists on re-runs)
code=$(curl -s -o /tmp/snap-signup.json -w "%{http_code}" \
  -H 'Content-Type: application/json' -H "Origin: $AUTH_ORIGIN" \
  -d "$(jq -n --arg n "$NAME" --arg e "$EMAIL" --arg p "$PASS" '{name:$n,email:$e,password:$p}')" \
  "$AUTH_BASE/sign-up/email")
if [ "$code" = "200" ] || [ "$code" = "201" ]; then
  pass "POST sign-up/email → $code"
elif grep -qiE 'already exists|already registered|duplicate|taken' /tmp/snap-signup.json; then
  pass "POST sign-up/email → already registered (re-run OK)"
else
  echo "--- sign-up body ---" >&2; cat /tmp/snap-signup.json >&2
  fail "POST sign-up/email" "$code" "200 (or already-exists)"
fi

# 3. Login (session cookie into jar)
code=$(curl -s -o /tmp/snap-signin.json -w "%{http_code}" -c "$JAR" -b "$JAR" \
  -H 'Content-Type: application/json' -H "Origin: $AUTH_ORIGIN" \
  -d "$(jq -n --arg e "$EMAIL" --arg p "$PASS" '{email:$e,password:$p}')" \
  "$AUTH_BASE/sign-in/email")
assert_status "$code" "200" "POST sign-in/email"

# 4. Raw JWT (Bearer token for snap-link, 15-min expiry)
code=$(curl -s -o /tmp/snap-token.json -w "%{http_code}" -b "$JAR" -H "Origin: $AUTH_ORIGIN" "$AUTH_BASE/token")
assert_status "$code" "200" "GET token"
TOKEN="$(jq -r '.token // .session.token // empty' /tmp/snap-token.json)"
if [ -z "$TOKEN" ]; then
  echo "--- token body ---" >&2; cat /tmp/snap-token.json >&2
  echo "❌ empty token: if email verification is required on your branch, verify inbox first." >&2
  echo "   Fallback: GET $AUTH_BASE/get-session and read the set-auth-jwt response header." >&2
  exit 1
fi
pass "GET token → JWT received (${#TOKEN} chars)"

# 5. Create short link (Neon JWT required)
code=$(curl -s -o /tmp/snap-create.json -w "%{http_code}" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" \
  -d "$(jq -n --arg l "$TARGET_URL" '{link:$l}')" \
  "$API_BASE/snap")
assert_status "$code" "201" "POST /snap"
SHORT_CODE="$(jq -r '.link // empty' /tmp/snap-create.json)"
[ -n "$SHORT_CODE" ] || { echo "--- create body ---" >&2; cat /tmp/snap-create.json >&2; fail "POST /snap body" "empty" '{"link":"..."}'; }
pass "POST /snap → code $SHORT_CODE"

# 6. Follow short link (public, expect 302 + Location, no -L flag)
headers=$(curl -s -D - -o /dev/null "$API_BASE/$SHORT_CODE")
code=$(printf '%s' "$headers" | awk 'NR==1{print $2}')
assert_status "$code" "302" "GET /$SHORT_CODE"
location=$(printf '%s' "$headers" | grep -i '^location:' | tr -d '\r' | awk '{print $2}')
[ "$location" = "$TARGET_URL" ] || fail "GET /$SHORT_CODE Location" "$location" "$TARGET_URL"
pass "GET /$SHORT_CODE Location correct"

# 7. Negative: no token -> 401
code=$(curl -s -o /dev/null -w "%{http_code}" \
  -H 'Content-Type: application/json' \
  -d '{"link":"https://example.com/no-auth"}' \
  "$API_BASE/snap")
assert_status "$code" "401" "POST /snap without token"

# 8. Negative: bad URL -> 400
code=$(curl -s -o /dev/null -w "%{http_code}" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" \
  -d '{"link":"ftp://example.com/not-allowed"}' \
  "$API_BASE/snap")
assert_status "$code" "400" "POST /snap bad URL"

# 9. Negative: unknown code -> 404
code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/abcdef")
assert_status "$code" "404" "GET /abcdef"

# 10. Negative: malformed code -> 400
code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/ab")
assert_status "$code" "400" "GET /ab"

# 11. Rate limit: adaptive to EXPECTED_LIMIT (fixed 5/min per IP by default)
# 3 POST /snap already consumed above (create + no-token 401 + bad-URL 400 —
# the limiter runs before auth, so even rejected requests count).
# On a fresh window: first (EXPECTED_LIMIT - 3) probes -> 201, then 429s.
CONSUMED=3
ALLOW=$((EXPECTED_LIMIT - CONSUMED))
if [ "$ALLOW" -lt 1 ]; then
  fail "rate-limit setup" "$EXPECTED_LIMIT" "EXPECTED_LIMIT > $CONSUMED (restart server for a clean window)"
fi
i=1
while [ "$i" -le "$ALLOW" ]; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" \
    -d "{\"link\":\"https://example.com/rate-limit-probe-$i\"}" \
    "$API_BASE/snap")
  [ "$code" = "201" ] || fail "probe $i (allowed)" "$code" "201"
  pass "probe $i (allowed) → 201"
  i=$((i + 1))
done
for j in 1 2; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" \
    -d "{\"link\":\"https://example.com/rate-limit-over-$j\"}" \
    "$API_BASE/snap")
  assert_status "$code" "429" "over-limit probe $j/2"
done

echo "All smoke tests passed ✅"
