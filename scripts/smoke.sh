#!/usr/bin/env bash
# Brings up the whole stack with Docker Compose and checks it through the proxy, the way a visitor and an API client
# would. Needs Docker with the compose plugin. Tears the stack down at the end, even on failure.
#
#   scripts/smoke.sh                  build the images and test
#   SMOKE_NO_BUILD=1 scripts/smoke.sh reuse images that are already built
set -euo pipefail
cd "$(dirname "$0")/.."

BASE="http://127.0.0.1:8080"
fail=0
ok()   { printf '  ok   %s\n' "$1"; }
bad()  { printf '  FAIL %s\n' "$1"; fail=1; }
check() { # description, expected status, curl args...
  local desc=$1 want=$2; shift 2
  local got; got=$(curl -s -o /dev/null -w '%{http_code}' -m 15 "$@" || true)
  if [ "$got" = "$want" ]; then ok "$desc ($got)"; else bad "$desc: wanted $want, got $got"; fi
}

python3 scripts/smoke_env.py
KEY=$(grep '^API_KEYS=' env/backend/.env | cut -d= -f2)

cleanup() { docker compose down -v >/dev/null 2>&1 || true; }
trap cleanup EXIT

up_args=(-d --wait --wait-timeout 300)
[ -n "${SMOKE_NO_BUILD:-}" ] && up_args+=(--no-build) || up_args+=(--build)
docker compose up "${up_args[@]}"

echo "website"
for page in / /compress-jpg /compress-png /compress-image-to-100kb /privacy /sitemap.xml /robots.txt /icon.svg; do
  check "GET $page" 200 "$BASE$page"
done
check "unknown page is a 404" 404 "$BASE/no-such-page"
check "no ads.txt without a publisher id" 404 "$BASE/ads.txt"
curl -s -m 15 "$BASE/compress-jpg" | grep -q '<title>Compress JPG' && ok "page title is rendered" || bad "page title"
curl -sI -m 15 "$BASE/" | grep -qi '^x-content-type-options: nosniff' && ok "security headers" || bad "security headers"

echo "api"
check "health" 200 "$BASE/api/v1/health"
check "ready (database reachable)" 200 "$BASE/api/v1/ready"
check "jobs need a key" 401 -X POST "$BASE/api/v1/jobs" -d '{}'
check "private address is refused" 422 -X POST "$BASE/api/v1/jobs" -H "Authorization: Bearer $KEY" -d '{"source_urls":["http://127.0.0.1/x.jpg"]}'
check "job creation" 202 -X POST "$BASE/api/v1/jobs" -H "Authorization: Bearer $KEY" -d '{"source_urls":["https://example.com/a.jpg"]}'
check "job list" 200 "$BASE/api/v1/jobs" -H "Authorization: Bearer $KEY"

echo "state survives a backend restart"
docker compose restart backend >/dev/null 2>&1
for _ in $(seq 1 30); do curl -sf -m 3 "$BASE/api/v1/ready" >/dev/null && break; sleep 2; done
curl -s -m 15 "$BASE/api/v1/jobs" -H "Authorization: Bearer $KEY" | grep -q '"total":1' && ok "the job is still there" || bad "job lost after restart"

[ "$fail" = 0 ] && echo "smoke test passed" || { echo "smoke test FAILED"; docker compose logs --tail 40 || true; exit 1; }
