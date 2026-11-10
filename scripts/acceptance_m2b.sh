#!/usr/bin/env bash
# M2b acceptance: verify the upload & media pipeline end to end
# against the compose deployment.
#
# The script first prepares the storage bind mount for the container's
# fixed uid 10001 -- mkdir as the host user, then chown through the
# image itself running as root, the sudo-less fresh-install contract
# documented on the bind mount in docker-compose.yml -- then builds
# the stack from the current source tree, waits for the server to
# become healthy, and walks the M2b surface in order: the first-run
# wizard, admin setup and web login (the M1/M2a flows), the multipart
# upload of the golden JPEG fixture carrying its device identity, the
# asset.ready event on the SSE activity stream once the pipeline's
# metadata and thumbnail jobs have drained, the tile thumbnail served
# as AVIF bytes, and the idempotent re-upload of the same bytes, which
# resolves as duplicate and names the existing asset. The script
# always tears the stack down including the database volume, so
# consecutive runs start from a clean state.
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message, and the teardown still runs. A passing run
# prints "M2B ACCEPTANCE PASS" as its final line.
#
# Requirements: docker with the compose plugin, curl.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
sse_timeout_seconds=60
admin_password="correct-horse-battery"
# The golden fixture: a real 64x64 JPEG whose EXIF carries
# capture time with offset, GPS position and the camera fields, so the
# metadata job has genuine tags to resolve.
golden_fixture="$repo_root/server/internal/jobs/testdata/golden.jpg"

run_start="$(date +%s)"

fail() {
    echo "M2B ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# Brought-up flag for the teardown trap: a failed assertion must still
# remove the containers and the pgdata volume, otherwise a rerun would
# hit an instance that is already initialized.
stack_up=0
# PID of the background SSE reader, emptied once reaped, so the exit
# trap never kills a stale (possibly recycled) process id.
sse_pid=""

teardown() {
    if [ "$stack_up" -ne 0 ]; then
        stack_up=0
        echo "==> tearing down the stack and removing volumes"
        docker compose down -v --remove-orphans
    fi
}

work_dir="$(mktemp -d)"
cleanup() {
    if [ -n "$sse_pid" ]; then
        kill "$sse_pid" 2>/dev/null || true
    fi
    teardown
    rm -rf "$work_dir"
}
trap cleanup EXIT

# Start from a clean stack state so the assertions below run against a
# fresh, uninitialized instance even if a previous run was interrupted.
echo "==> resetting to a clean stack state"
docker compose down -v --remove-orphans
stack_up=1

echo "==> preparing the storage bind mount for uid 10001 (sudo-less)"
# The container runs as the image's fixed uid/gid 10001 and the bind
# source must be writable by it before the server starts. Docker only
# creates a missing source as root:root, so the directory is created
# by the host user first and then chowned through the image itself
# running as root -- the fresh-install contract without sudo. The
# chown rides the app image, so it must exist: build it here (a cached
# no-op when the image is current; `up -d --build` below re-checks it).
mkdir -p lumemor
docker compose build || fail "docker compose build failed"
docker compose run --rm --no-deps --entrypoint /bin/sh -u 0 lumemor \
    -c 'chown -R 10001:10001 /var/lib/lumemor' \
    || fail "chown of ./lumemor to 10001:10001 through the image failed"

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
admin_email="admin-$(date +%s)@m2b-acceptance.invalid"
status="$(curl -sS -o "$work_dir/create_admin_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"M2b Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
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
# The credential rides the Set-Cookie header only; every authed step
# below (upload, SSE stream, thumbnail) rides this jar.
grep -q 'lumemor_session' "$work_dir/cookies" \
    || fail "POST /api/v1/auth/login did not set the lumemor_session cookie: $(cat "$work_dir/login_body")"
echo "    login accepted, session cookie captured"

echo "==> POST /api/v1/assets must return 201 with state created"
[ -f "$golden_fixture" ] \
    || fail "golden JPEG fixture missing: $golden_fixture"
# The device identity is unique per run; the device-reported capture
# time and zone mirror the fixture's EXIF (2023-07-14 10:15:30 +05:30,
# Asia/Kolkata) that the metadata job resolves from the real tags.
device_asset_id="m2b-acceptance-$(date +%s)"
status="$(curl -sS -o "$work_dir/upload_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -F "deviceAssetId=$device_asset_id" \
    -F 'fileName=IMG_0001.jpg' \
    -F 'deviceCreatedAt=2023-07-14T10:15:30+05:30' \
    -F 'deviceTimeZone=Asia/Kolkata' \
    -F "photo=@$golden_fixture;filename=IMG_0001.jpg;type=image/jpeg" \
    "$base_url/api/v1/assets")" \
    || fail "POST /api/v1/assets request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/assets returned status $status, want 201: $(cat "$work_dir/upload_body")"
upload_body="$(cat "$work_dir/upload_body")"
printf '%s\n' "$upload_body" | grep -Eq '"state"[[:space:]]*:[[:space:]]*"created"' \
    || fail "POST /api/v1/assets did not report state created: $upload_body"
# The asset id is asserted from the response and drives both the SSE
# match and the thumbnail fetch below -- never a guessed value.
asset_id="$(printf '%s\n' "$upload_body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$asset_id" ] \
    || fail "POST /api/v1/assets response carries no asset id: $upload_body"
echo "    upload accepted: $upload_body"

echo "==> GET /api/v1/events must deliver asset.ready for asset $asset_id (up to ${sse_timeout_seconds}s)"
# The cookie-authenticated stream runs in the background and the
# captured output is polled: the upload has already been answered, so
# asset.ready either replays from the retention ring or arrives live
# once the metadata and thumbnail jobs drain the asset's pending-jobs
# counter. curl's --max-time is a backstop only -- the poll deadline
# below bounds the actual wait.
curl -sS -N --max-time 120 -b "$work_dir/cookies" \
    "$base_url/api/v1/events" >"$work_dir/sse_stream" 2>"$work_dir/sse_err" &
sse_pid=$!
deadline="$(( $(date +%s) + sse_timeout_seconds ))"
until awk -v want="$asset_id" '
    $0 == "event: asset.ready" {
        getline data
        if (data ~ "\"assetId\":\"" want "\"") found = 1
    }
    END { exit(found ? 0 : 1) }
' "$work_dir/sse_stream"; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        kill "$sse_pid" 2>/dev/null || true
        wait "$sse_pid" 2>/dev/null || true
        sse_pid=""
        fail "asset.ready for asset $asset_id was not delivered within ${sse_timeout_seconds}s; stream so far: $(cat "$work_dir/sse_stream")"
    fi
    if ! kill -0 "$sse_pid" 2>/dev/null; then
        sse_status=0
        wait "$sse_pid" || sse_status=$?
        sse_pid=""
        fail "GET /api/v1/events curl exited $sse_status before asset.ready arrived: $(cat "$work_dir/sse_err") stream: $(cat "$work_dir/sse_stream")"
    fi
    sleep 1
done
kill "$sse_pid" 2>/dev/null || true
wait "$sse_pid" 2>/dev/null || true
sse_pid=""
echo "    event stream (the upload lifecycle, asset.ready included):"
sed 's/^/    /' "$work_dir/sse_stream"

echo "==> GET /api/v1/assets/$asset_id/thumbnail?size=tile must return 200 image/avif with AVIF bytes"
status="$(curl -sS -o "$work_dir/thumb_body" -D "$work_dir/thumb_headers" \
    -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/$asset_id/thumbnail?size=tile")" \
    || fail "GET thumbnail request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/$asset_id/thumbnail?size=tile returned status $status, want 200"
grep -Fq 'Content-Type: image/avif' "$work_dir/thumb_headers" \
    || fail "the tile thumbnail is not image/avif: $(cat "$work_dir/thumb_headers")"
# An AVIF file is an ISOBMFF container: bytes 4-8 spell the ftyp box
# type, so the first 12 bytes must carry it.
head -c 12 "$work_dir/thumb_body" | LC_ALL=C grep -aq 'ftyp' \
    || fail "the tile thumbnail does not carry the ftyp box in its first 12 bytes: $(head -c 12 "$work_dir/thumb_body" | od -An -tx1 | tr -d ' \n')"
thumb_bytes="$(wc -c < "$work_dir/thumb_body" | tr -d ' ')"
thumb_magic="$(head -c 12 "$work_dir/thumb_body" | od -An -tx1 | tr -d ' \n')"
echo "    served $thumb_bytes bytes of image/avif, magic $thumb_magic"

echo "==> re-uploading the same file must return 200 with state duplicate"
status="$(curl -sS -o "$work_dir/reupload_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -F "deviceAssetId=$device_asset_id" \
    -F 'fileName=IMG_0001.jpg' \
    -F 'deviceCreatedAt=2023-07-14T10:15:30+05:30' \
    -F 'deviceTimeZone=Asia/Kolkata' \
    -F "photo=@$golden_fixture;filename=IMG_0001.jpg;type=image/jpeg" \
    "$base_url/api/v1/assets")" \
    || fail "re-upload request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets (re-upload) returned status $status, want 200: $(cat "$work_dir/reupload_body")"
reupload_body="$(cat "$work_dir/reupload_body")"
printf '%s\n' "$reupload_body" | grep -Eq '"state"[[:space:]]*:[[:space:]]*"duplicate"' \
    || fail "the re-upload did not report state duplicate: $reupload_body"
duplicate_id="$(printf '%s\n' "$reupload_body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ "$duplicate_id" = "$asset_id" ] \
    || fail "the re-upload named asset $duplicate_id, want the existing $asset_id: $reupload_body"
echo "    re-upload resolved as duplicate for asset $asset_id"

teardown
echo "==> acceptance flow completed in $(elapsed)s"
echo "M2B ACCEPTANCE PASS"
