#!/usr/bin/env bash
# M1 acceptance: verify the skeleton milestone end to end against the
# compose deployment.
#
# The script builds the stack from the current source tree, waits for
# the server to become healthy, then exercises the public surface in
# order: the first-run wizard page, the admin setup API, and the server
# info capability report. It always tears the stack down including the
# database volume, so consecutive runs start from a clean state.
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message, and the teardown still runs. A passing run prints
# "M1 ACCEPTANCE PASS" as its final line.
#
# Requirements: docker with the compose plugin, curl.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120

run_start="$(date +%s)"

fail() {
    echo "M1 ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# Brought-up flag for the teardown trap: a failed assertion must still
# remove the containers and the pgdata volume, otherwise a rerun would
# hit an instance that is already initialized.
stack_up=0

teardown() {
    if [ "$stack_up" -ne 0 ]; then
        stack_up=0
        echo "==> tearing down the stack and removing volumes"
        docker compose down -v --remove-orphans
    fi
}

work_dir="$(mktemp -d)"
cleanup() {
    teardown
    rm -rf "$work_dir"
}
trap cleanup EXIT

# Start from a clean stack state so the assertions below run against a
# fresh, uninitialized instance even if a previous run was interrupted.
echo "==> resetting to a clean stack state"
docker compose down -v --remove-orphans
stack_up=1

echo "==> building and starting the stack"
docker compose up -d --build || fail "docker compose up -d --build failed"

echo "==> waiting for GET /healthz (up to ${health_timeout_seconds}s)"
deadline="$(( $(date +%s) + health_timeout_seconds ))"
until curl -fsS "$base_url/healthz" >/dev/null 2>&1; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        docker compose logs --no-color db lumemor || true
        fail "GET /healthz did not succeed within ${health_timeout_seconds}s"
    fi
    sleep 2
done
echo "    server healthy after $(elapsed)s"

echo "==> GET /setup must return 200"
status="$(curl -sS -o /dev/null -w '%{http_code}' "$base_url/setup")" \
    || fail "GET /setup request failed"
[ "$status" = "200" ] \
    || fail "GET /setup returned status $status, want 200"

echo "==> POST /api/v1/server/setup/admin must return 201"
# The email is unique per run so reruns never collide with leftover
# data even if a previous teardown failed.
admin_email="admin-$(date +%s)@m1-acceptance.invalid"
status="$(curl -sS -o "$work_dir/create_admin_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"M1 Acceptance\",\"email\":\"$admin_email\",\"password\":\"correct-horse-battery\"}" \
    "$base_url/api/v1/server/setup/admin")" \
    || fail "POST /api/v1/server/setup/admin request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/server/setup/admin returned status $status, want 201: $(cat "$work_dir/create_admin_body")"
echo "    created administrator $admin_email"

echo "==> GET /api/v1/server/info must report the instance as initialized"
info="$(curl -sS "$base_url/api/v1/server/info")" \
    || fail "GET /api/v1/server/info request failed"
echo "    server info: $info"
printf '%s\n' "$info" | grep -Eq '"initialized"[[:space:]]*:[[:space:]]*true' \
    || fail "GET /api/v1/server/info did not report initialized true: $info"

teardown
echo "==> acceptance flow completed in $(elapsed)s"
echo "M1 ACCEPTANCE PASS"
