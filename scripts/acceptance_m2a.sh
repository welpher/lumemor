#!/usr/bin/env bash
# M2a acceptance: verify the auth & events foundation end to end
# against the compose deployment.
#
# The script builds the stack from the current source tree, waits for
# the server to become healthy, then walks the M2a surface in order:
# the first-run wizard and admin setup (the M1 flow), a web login that
# yields the session cookie, the session list through that cookie, the
# pairing exchange (generate with the cookie, consume without any
# credential for the bearer token), an authed server info call with
# the bearer, and the SSE activity stream. Every event published on
# the way -- the two session.started events, the pairing mint, and the
# login failure triggered after subscribing -- must arrive on the
# stream. The script always tears the stack down including the
# database volume, so consecutive runs start from a clean state.
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message, and the teardown still runs. A passing run
# prints "M2A ACCEPTANCE PASS" as its final line.
#
# Requirements: docker with the compose plugin, curl.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
admin_password="correct-horse-battery"

run_start="$(date +%s)"

fail() {
    echo "M2A ACCEPTANCE FAIL: $*" >&2
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
admin_email="admin-$(date +%s)@m2a-acceptance.invalid"
status="$(curl -sS -o "$work_dir/create_admin_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"M2a Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
    "$base_url/api/v1/server/setup/admin")" \
    || fail "POST /api/v1/server/setup/admin request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/server/setup/admin returned status $status, want 201: $(cat "$work_dir/create_admin_body")"
echo "    created administrator $admin_email"

echo "==> POST /api/v1/auth/login must return 200 and set the session cookie"
status="$(curl -sS -o "$work_dir/login_body" -c "$work_dir/cookies" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
    "$base_url/api/v1/auth/login")" \
    || fail "POST /api/v1/auth/login request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/auth/login returned status $status, want 200: $(cat "$work_dir/login_body")"
# The credential rides the Set-Cookie header only; asserting the jar
# received it guards the whole web-session half of the milestone.
grep -q 'lumemor_session' "$work_dir/cookies" \
    || fail "POST /api/v1/auth/login did not set the lumemor_session cookie: $(cat "$work_dir/login_body")"
echo "    login accepted, session cookie captured: $(cat "$work_dir/login_body")"

echo "==> GET /api/v1/auth/sessions with the cookie must return 200"
status="$(curl -sS -o "$work_dir/sessions_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/auth/sessions")" \
    || fail "GET /api/v1/auth/sessions request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/auth/sessions returned status $status, want 200: $(cat "$work_dir/sessions_body")"
sessions="$(cat "$work_dir/sessions_body")"
echo "    sessions: $sessions"
printf '%s\n' "$sessions" | grep -Fq '"kind":"web"' \
    || fail "GET /api/v1/auth/sessions did not list the web session: $sessions"

echo "==> POST /api/v1/auth/pairing/token with the cookie must return 200"
status="$(curl -sS -o "$work_dir/pairing_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -X POST "$base_url/api/v1/auth/pairing/token")" \
    || fail "POST /api/v1/auth/pairing/token request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/auth/pairing/token returned status $status, want 200: $(cat "$work_dir/pairing_body")"
pairing_body="$(cat "$work_dir/pairing_body")"
# The one-time secret exists exactly once, in this response; it is
# masked in the transcript and never echoed in the clear.
pairing_display="$(printf '%s\n' "$pairing_body" | sed 's/"token":"[^"]*"/"token":"<redacted>"/')"
echo "    pairing token minted: $pairing_display"
pairing_token="$(printf '%s\n' "$pairing_body" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$pairing_token" ] \
    || fail "POST /api/v1/auth/pairing/token response carries no token: $pairing_body"

echo "==> POST /api/v1/auth/pairing/consume without credentials must return 200"
status="$(curl -sS -o "$work_dir/consume_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"token\":\"$pairing_token\",\"deviceName\":\"M2a Acceptance Phone\",\"deviceId\":\"m2a-acceptance-device-1\"}" \
    "$base_url/api/v1/auth/pairing/consume")" \
    || fail "POST /api/v1/auth/pairing/consume request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/auth/pairing/consume returned status $status, want 200: $(cat "$work_dir/consume_body")"
consume_body="$(cat "$work_dir/consume_body")"
consume_display="$(printf '%s\n' "$consume_body" | sed 's/"token":"[^"]*"/"token":"<redacted>"/')"
echo "    bearer session established: $consume_display"
bearer_token="$(printf '%s\n' "$consume_body" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$bearer_token" ] \
    || fail "POST /api/v1/auth/pairing/consume response carries no token: $consume_body"
printf '%s\n' "$consume_body" | grep -Fq '"kind":"mobile"' \
    || fail "POST /api/v1/auth/pairing/consume did not create a mobile session: $consume_body"

echo "==> GET /api/v1/server/info with the bearer token must return 200"
status="$(curl -sS -o "$work_dir/info_body" -w '%{http_code}' \
    -H "Authorization: Bearer $bearer_token" \
    "$base_url/api/v1/server/info")" \
    || fail "GET /api/v1/server/info request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/server/info returned status $status, want 200: $(cat "$work_dir/info_body")"
info="$(cat "$work_dir/info_body")"
echo "    server info: $info"
printf '%s\n' "$info" | grep -Eq '"initialized"[[:space:]]*:[[:space:]]*true' \
    || fail "GET /api/v1/server/info did not report initialized true: $info"

echo "==> GET /api/v1/events must stream the session, pairing and login-failure events"
# The bearer-authenticated stream runs in the background. Everything
# published so far is still inside the five-minute retention ring, so
# the subscription replays the M2a lifecycle immediately; the wrong
# password attempt below then publishes auth.login_failed onto the
# live feed. curl's --max-time bounds the wait -- the stream never
# ends on its own, so exit status 28 (timeout) is the expected
# outcome, not a failure.
curl -sS -N --max-time 10 \
    -H "Authorization: Bearer $bearer_token" \
    "$base_url/api/v1/events" >"$work_dir/sse_stream" 2>"$work_dir/sse_err" &
sse_pid=$!
sleep 2

echo "==> POST /api/v1/auth/login with a wrong password must return 401"
status="$(curl -sS -o "$work_dir/failed_login_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$admin_email\",\"password\":\"wrong-horse-battery\"}" \
    "$base_url/api/v1/auth/login")" \
    || fail "POST /api/v1/auth/login (wrong password) request failed"
[ "$status" = "401" ] \
    || fail "POST /api/v1/auth/login with a wrong password returned status $status, want 401: $(cat "$work_dir/failed_login_body")"
echo "    rejected: $(cat "$work_dir/failed_login_body")"

sse_status=0
wait "$sse_pid" || sse_status=$?
if [ "$sse_status" != "0" ] && [ "$sse_status" != "28" ]; then
    fail "GET /api/v1/events curl exited $sse_status, want 0 or 28: $(cat "$work_dir/sse_err")"
fi
echo "    event stream (curl exit $sse_status):"
sed 's/^\(.\)/    \1/' "$work_dir/sse_stream"
sse_stream="$(cat "$work_dir/sse_stream")"
printf '%s\n' "$sse_stream" | grep -Fq 'event: session.started' \
    || fail "the event stream did not deliver session.started: $sse_stream"
printf '%s\n' "$sse_stream" | grep -Fq 'event: pairing.token_created' \
    || fail "the event stream did not deliver pairing.token_created: $sse_stream"
printf '%s\n' "$sse_stream" | grep -Fq 'event: auth.login_failed' \
    || fail "the event stream did not deliver auth.login_failed: $sse_stream"

teardown
echo "==> acceptance flow completed in $(elapsed)s"
echo "M2A ACCEPTANCE PASS"
