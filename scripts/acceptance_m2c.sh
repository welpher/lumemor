#!/usr/bin/env bash
# M2c acceptance: verify the timeline & trash milestone end to end
# against the compose deployment -- the M2 milestone acceptance
# itself.
#
# The script first prepares the storage bind mount for the container's
# fixed uid 10001 -- mkdir as the host user, then chown through the
# image itself running as root, the sudo-less fresh-install contract
# documented on the bind mount in docker-compose.yml -- then builds
# the stack from the current source tree, waits for the server to
# become healthy, and walks the M2c surface in order: the first-run
# wizard, admin setup and web login (the M1/M2a flows), the multipart
# upload of THREE distinct photo payloads (distinct bytes, distinct
# checksums -- one file re-uploaded three times would collapse into
# the duplicate state of the owner-scoped checksum idempotency) with
# three distinct deviceCreatedAt values pinning the timeline order,
# the aggregated timeline newest-first, keyset pagination over the
# same three photos (two pages, full set, no overlap), the batch
# trash of one photo, the trash listing with its retention countdown,
# the batch restore, the batch hide and unhide with the hidden view,
# and the video leg: the tiny.mp4 fixture upload, the asset.ready
# event on the SSE activity stream once the video's pipeline jobs
# have drained, and the transcoded play stream answering a Range
# request with 206. The asset id chain is asserted end to end: every
# id the script acts on comes from a previous response, never a
# guess. The session credential rides the login Set-Cookie header
# into the cookie jar, which is never printed; no other credential
# exists in this flow. The script always tears the stack down
# including the database volume, so consecutive runs start from a
# clean state.
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message, and the teardown still runs. A passing run
# prints "M2C ACCEPTANCE PASS" as its final line.
#
# Requirements: docker with the compose plugin, curl, base64.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
sse_timeout_seconds=90
admin_password="correct-horse-battery"
# The Task 6 video fixture: a real 101796-byte MP4 container that the
# pipeline's ffprobe, poster and transcode jobs process for real.
video_fixture="$repo_root/server/internal/media/testdata/tiny.mp4"
# One timestamp for the run's unique identities (admin email, device
# asset ids) so reruns never collide with leftover data, and one for
# the elapsed-time reporting of the phases.
run_start="$(date +%s)"
run_stamp="$run_start"

fail() {
    echo "M2C ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# item_ids prints the ids of an items[] response body in document
# order -- the timeline, the trash and the hidden view all render
# their rows as objects whose "id" key is the first field, and the
# compact encoder keeps document order, so the ordered matches below
# assert both membership and sequence. The quoted "id" pattern cannot
# match "ownerId" or any other camel-case field: the leading quote
# anchors it against the key's start.
item_ids() {
    grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' "$1" \
        | sed 's/^"id"[[:space:]]*:[[:space:]]*"\([^"]*\)"$/\1/'
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
admin_email="admin-${run_stamp}@m2c-acceptance.invalid"
status="$(curl -sS -o "$work_dir/create_admin_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"M2c Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
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
# below (uploads, timeline, batch, trash, SSE stream, play) rides this
# jar, which is never echoed.
grep -q 'lumemor_session' "$work_dir/cookies" \
    || fail "POST /api/v1/auth/login did not set the lumemor_session cookie: $(cat "$work_dir/login_body")"
echo "    login accepted, session cookie captured"

echo "==> materializing the three distinct photo payloads"
# Three tiny real JPEGs (64x64 solid frames of three distinct colors),
# generated once with Go's image/jpeg and embedded here as base64:
# the M2c timeline flow needs three DISTINCT photos, and the golden
# fixture of M2b is one file -- re-uploading it would answer the
# duplicate state of the owner-scoped checksum identity, and three
# uploads that differ only in device metadata would collapse into one
# asset. The distinct bytes make the three uploads three separate
# assets under content addressing; the distinct deviceCreatedAt
# values below then pin the timeline order deterministically.
base64 -d >"$work_dir/photo-1.jpg" <<'EOF'
/9j/2wCEAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRQBAwQEBQQFCQUFCRQNCw0UFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFP/AABEIAEAAQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AOKooor8kP7FCiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKAP//Z
EOF
base64 -d >"$work_dir/photo-2.jpg" <<'EOF'
/9j/2wCEAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRQBAwQEBQQFCQUFCRQNCw0UFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFP/AABEIAEAAQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AMeiiiuI/nQKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooA//9k=
EOF
base64 -d >"$work_dir/photo-3.jpg" <<'EOF'
/9j/2wCEAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRQBAwQEBQQFCQUFCRQNCw0UFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFP/AABEIAEAAQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APJKKKK/qA/LwooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigD//Z
EOF
# Each payload must be a real JPEG before it rides an upload: the
# SOI magic (ff d8 ff) in the first three bytes, and enough body for
# the sniffer and the pipeline to work with.
for payload in photo-1 photo-2 photo-3; do
    payload_file="$work_dir/$payload.jpg"
    payload_size="$(wc -c < "$payload_file" | tr -d ' ')"
    [ "$payload_size" -gt 500 ] \
        || fail "$payload decoded to $payload_size bytes, want a real JPEG payload"
    payload_magic="$(head -c 3 "$payload_file" | od -An -tx1 | tr -d ' \n')"
    [ "$payload_magic" = "ffd8ff" ] \
        || fail "$payload does not start with the JPEG SOI magic, want ffd8ff, got $payload_magic"
    echo "    $payload: $payload_size bytes, a real JPEG"
done

echo "==> POST /api/v1/assets must return 201 with state created for each of the three photos"
# The taken times are pinned per photo -- January 1st, February 2nd,
# March 3rd of 2026 -- so the timeline's newest-first order below is
# deterministic (the photos carry no EXIF, so taken_at stays the
# device-reported value with takenAtSource import).
photo_ids=()
index=0
for spec in \
    "photo-1.jpg IMG_0001.jpg 2026-01-01T12:00:00Z" \
    "photo-2.jpg IMG_0002.jpg 2026-02-02T12:00:00Z" \
    "photo-3.jpg IMG_0003.jpg 2026-03-03T12:00:00Z"; do
    index=$((index + 1))
    read -r payload filename taken_at <<<"$spec"
    device_asset_id="m2c-acceptance-${run_stamp}-${index}"
    status="$(curl -sS -o "$work_dir/upload_body_$index" -b "$work_dir/cookies" -w '%{http_code}' \
        -F "deviceAssetId=$device_asset_id" \
        -F "fileName=$filename" \
        -F "deviceCreatedAt=$taken_at" \
        -F 'deviceTimeZone=UTC' \
        -F "photo=@$work_dir/$payload;filename=$filename;type=image/jpeg" \
        "$base_url/api/v1/assets")" \
        || fail "POST /api/v1/assets ($filename) request failed"
    [ "$status" = "201" ] \
        || fail "POST /api/v1/assets ($filename) returned status $status, want 201: $(cat "$work_dir/upload_body_$index")"
    upload_body="$(cat "$work_dir/upload_body_$index")"
    printf '%s\n' "$upload_body" | grep -Eq '"state"[[:space:]]*:[[:space:]]*"created"' \
        || fail "POST /api/v1/assets ($filename) did not report state created: $upload_body"
    asset_id="$(printf '%s\n' "$upload_body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
    [ -n "$asset_id" ] \
        || fail "POST /api/v1/assets ($filename) response carries no asset id: $upload_body"
    photo_ids+=("$asset_id")
    echo "    upload accepted ($filename, taken $taken_at): $upload_body"
done
january_id="${photo_ids[0]}"
february_id="${photo_ids[1]}"
march_id="${photo_ids[2]}"
# The id chain starts here: three distinct ids, one per distinct
# checksum -- a duplicate upload would have answered 200 and failed
# the 201 above, and the explicit distinctness pin documents the
# chain every later assertion walks.
distinct_ids="$(printf '%s\n' "${photo_ids[@]}" | sort -u | wc -l | tr -d ' ')"
[ "$distinct_ids" = "3" ] \
    || fail "the three uploads resolved to $distinct_ids distinct ids, want 3 (the checksums must differ): ${photo_ids[*]}"

echo "==> GET /api/v1/assets/timeline must return all three photos newest first"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline returned status $status, want 200: $(cat "$work_dir/timeline_body")"
timeline_ids="$(item_ids "$work_dir/timeline_body" | tr '\n' ' ')"
expected_ids="$(printf '%s ' "$march_id" "$february_id" "$january_id")"
[ "$timeline_ids" = "$expected_ids" ] \
    || fail "the timeline holds [$timeline_ids], want [$expected_ids] (March, February, January, newest first): $(cat "$work_dir/timeline_body")"
echo "    timeline returns the three photos newest first: March, February, January"

echo "==> GET /api/v1/assets/timeline?limit=2 must page 2 then 1 with the full set and no overlap"
status="$(curl -sS -o "$work_dir/page1_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline?limit=2")" \
    || fail "GET /api/v1/assets/timeline?limit=2 request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline?limit=2 returned status $status, want 200: $(cat "$work_dir/page1_body")"
page1_ids="$(item_ids "$work_dir/page1_body" | tr '\n' ' ')"
[ "$page1_ids" = "$(printf '%s ' "$march_id" "$february_id")" ] \
    || fail "timeline page 1 holds [$page1_ids], want [$(printf '%s ' "$march_id" "$february_id")]: $(cat "$work_dir/page1_body")"
next_cursor="$(sed -n 's/.*"nextCursor"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/page1_body")"
[ -n "$next_cursor" ] \
    || fail "timeline page 1 carries no nextCursor while another page follows: $(cat "$work_dir/page1_body")"
# The cursor is base64 of the raw URL alphabet, safe to interpolate.
status="$(curl -sS -o "$work_dir/page2_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline?limit=2&cursor=$next_cursor")" \
    || fail "GET /api/v1/assets/timeline (page 2) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (page 2) returned status $status, want 200: $(cat "$work_dir/page2_body")"
page2_ids="$(item_ids "$work_dir/page2_body" | tr '\n' ' ')"
[ "$page2_ids" = "$january_id " ] \
    || fail "timeline page 2 holds [$page2_ids], want [$january_id ]: $(cat "$work_dir/page2_body")"
if grep -q '"nextCursor"' "$work_dir/page2_body"; then
    fail "timeline page 2 carries a nextCursor, want none on the last page: $(cat "$work_dir/page2_body")"
fi
# The union of the two pages is the full upload set, with no overlap:
# the sorted id lists must be identical.
printf '%s\n' "${photo_ids[@]}" | sort >"$work_dir/expected_set"
{ item_ids "$work_dir/page1_body"; item_ids "$work_dir/page2_body"; } | sort >"$work_dir/paged_set"
diff "$work_dir/expected_set" "$work_dir/paged_set" >/dev/null \
    || fail "the two pages' union [$( { item_ids "$work_dir/page1_body"; item_ids "$work_dir/page2_body"; } | tr '\n' ' ')] does not match the three uploaded ids ${photo_ids[*]}"
echo "    pages 2+1 cover the full set of three with no overlap, the last page carries no cursor"

echo "==> POST /api/v1/assets/batch (trash) must remove the January photo from the timeline"
status="$(curl -sS -o "$work_dir/batch_trash_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"trash\",\"ids\":[\"$january_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (trash) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (trash) returned status $status, want 200: $(cat "$work_dir/batch_trash_body")"
batch_trash_body="$(cat "$work_dir/batch_trash_body")"
affected="$(printf '%s\n' "$batch_trash_body" | sed -n 's/.*"affected"[[:space:]]*:[[:space:]]*\([0-9]\{1,\}\).*/\1/p')"
[ "$affected" = "1" ] \
    || fail "the trash batch affected $affected ids, want 1: $batch_trash_body"
printf '%s\n' "$batch_trash_body" | grep -q '"missing":\[\]' \
    || fail "the trash batch reported missing ids, want none: $batch_trash_body"
echo "    batch trash applied: $batch_trash_body"

echo "==> GET /api/v1/assets/timeline must now show the two survivors"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (after trash) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (after trash) returned status $status, want 200: $(cat "$work_dir/timeline_body")"
timeline_ids="$(item_ids "$work_dir/timeline_body" | tr '\n' ' ')"
[ "$timeline_ids" = "$(printf '%s ' "$march_id" "$february_id")" ] \
    || fail "the timeline after the trash holds [$timeline_ids], want [$(printf '%s ' "$march_id" "$february_id")]: $(cat "$work_dir/timeline_body")"
echo "    the trashed photo left the timeline, two remain"

echo "==> GET /api/v1/trash must list the January photo with a retention countdown"
status="$(curl -sS -o "$work_dir/trash_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/trash")" \
    || fail "GET /api/v1/trash request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/trash returned status $status, want 200: $(cat "$work_dir/trash_body")"
trash_ids="$(item_ids "$work_dir/trash_body" | tr '\n' ' ')"
[ "$trash_ids" = "$january_id " ] \
    || fail "the trash holds [$trash_ids], want [$january_id ]: $(cat "$work_dir/trash_body")"
remaining_days="$(sed -n 's/.*"remainingDays"[[:space:]]*:[[:space:]]*\([0-9]\{1,\}\).*/\1/p' "$work_dir/trash_body")"
case "$remaining_days" in
    ''|*[!0-9]*)
        fail "the trash item carries no numeric remainingDays: $(cat "$work_dir/trash_body")"
        ;;
esac
[ "$remaining_days" -ge 0 ] \
    || fail "the trash item's remainingDays is $remaining_days, want at least 0: $(cat "$work_dir/trash_body")"
echo "    the trash lists the January photo, $remaining_days days until expiry (the default retention is 30)"

echo "==> POST /api/v1/assets/batch (restore) must return the January photo to the timeline"
status="$(curl -sS -o "$work_dir/batch_restore_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"restore\",\"ids\":[\"$january_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (restore) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (restore) returned status $status, want 200: $(cat "$work_dir/batch_restore_body")"
batch_restore_body="$(cat "$work_dir/batch_restore_body")"
affected="$(printf '%s\n' "$batch_restore_body" | sed -n 's/.*"affected"[[:space:]]*:[[:space:]]*\([0-9]\{1,\}\).*/\1/p')"
[ "$affected" = "1" ] \
    || fail "the restore batch affected $affected ids, want 1: $batch_restore_body"
echo "    batch restore applied: $batch_restore_body"

echo "==> GET /api/v1/assets/timeline must show all three photos again"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (after restore) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (after restore) returned status $status, want 200: $(cat "$work_dir/timeline_body")"
timeline_ids="$(item_ids "$work_dir/timeline_body" | tr '\n' ' ')"
[ "$timeline_ids" = "$expected_ids" ] \
    || fail "the timeline after the restore holds [$timeline_ids], want [$expected_ids]: $(cat "$work_dir/timeline_body")"
echo "    the restored photo re-entered the timeline, three again"

echo "==> POST /api/v1/assets/batch (hide) must move the February photo to the hidden view"
status="$(curl -sS -o "$work_dir/batch_hide_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"hide\",\"ids\":[\"$february_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (hide) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (hide) returned status $status, want 200: $(cat "$work_dir/batch_hide_body")"
batch_hide_body="$(cat "$work_dir/batch_hide_body")"
affected="$(printf '%s\n' "$batch_hide_body" | sed -n 's/.*"affected"[[:space:]]*:[[:space:]]*\([0-9]\{1,\}\).*/\1/p')"
[ "$affected" = "1" ] \
    || fail "the hide batch affected $affected ids, want 1: $batch_hide_body"
echo "    batch hide applied: $batch_hide_body"

echo "==> GET /api/v1/assets/hidden must list the February photo"
status="$(curl -sS -o "$work_dir/hidden_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/hidden")" \
    || fail "GET /api/v1/assets/hidden request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/hidden returned status $status, want 200: $(cat "$work_dir/hidden_body")"
hidden_ids="$(item_ids "$work_dir/hidden_body" | tr '\n' ' ')"
[ "$hidden_ids" = "$february_id " ] \
    || fail "the hidden view holds [$hidden_ids], want [$february_id ]: $(cat "$work_dir/hidden_body")"
echo "    the hidden view lists the February photo"

echo "==> GET /api/v1/assets/timeline must show the two visible photos"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (after hide) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (after hide) returned status $status, want 200: $(cat "$work_dir/timeline_body")"
timeline_ids="$(item_ids "$work_dir/timeline_body" | tr '\n' ' ')"
[ "$timeline_ids" = "$(printf '%s ' "$march_id" "$january_id")" ] \
    || fail "the timeline after the hide holds [$timeline_ids], want [$(printf '%s ' "$march_id" "$january_id")]: $(cat "$work_dir/timeline_body")"
echo "    the hidden photo left the timeline, two remain"

echo "==> POST /api/v1/assets/batch (unhide) must return the February photo to the timeline"
status="$(curl -sS -o "$work_dir/batch_unhide_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"unhide\",\"ids\":[\"$february_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (unhide) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (unhide) returned status $status, want 200: $(cat "$work_dir/batch_unhide_body")"
batch_unhide_body="$(cat "$work_dir/batch_unhide_body")"
affected="$(printf '%s\n' "$batch_unhide_body" | sed -n 's/.*"affected"[[:space:]]*:[[:space:]]*\([0-9]\{1,\}\).*/\1/p')"
[ "$affected" = "1" ] \
    || fail "the unhide batch affected $affected ids, want 1: $batch_unhide_body"
echo "    batch unhide applied: $batch_unhide_body"

echo "==> GET /api/v1/assets/timeline must show all three photos once more"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$work_dir/cookies" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (after unhide) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (after unhide) returned status $status, want 200: $(cat "$work_dir/timeline_body")"
timeline_ids="$(item_ids "$work_dir/timeline_body" | tr '\n' ' ')"
[ "$timeline_ids" = "$expected_ids" ] \
    || fail "the timeline after the unhide holds [$timeline_ids], want [$expected_ids]: $(cat "$work_dir/timeline_body")"
echo "    the unhidden photo re-entered the timeline, three again"

echo "==> POST /api/v1/assets must accept the video fixture with state created"
[ -f "$video_fixture" ] \
    || fail "video fixture missing: $video_fixture"
video_device_id="m2c-acceptance-video-${run_stamp}"
status="$(curl -sS -o "$work_dir/video_body" -b "$work_dir/cookies" -w '%{http_code}' \
    -F "deviceAssetId=$video_device_id" \
    -F 'fileName=VID_0001.mp4' \
    -F 'deviceCreatedAt=2026-04-04T12:00:00Z' \
    -F 'deviceTimeZone=UTC' \
    -F "photo=@$video_fixture;filename=VID_0001.mp4;type=video/mp4" \
    "$base_url/api/v1/assets")" \
    || fail "POST /api/v1/assets (video) request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/assets (video) returned status $status, want 201: $(cat "$work_dir/video_body")"
video_body="$(cat "$work_dir/video_body")"
printf '%s\n' "$video_body" | grep -Eq '"state"[[:space:]]*:[[:space:]]*"created"' \
    || fail "POST /api/v1/assets (video) did not report state created: $video_body"
video_id="$(printf '%s\n' "$video_body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$video_id" ] \
    || fail "POST /api/v1/assets (video) response carries no asset id: $video_body"
echo "    video upload accepted: $video_body"

echo "==> GET /api/v1/events must deliver asset.ready for the video asset (up to ${sse_timeout_seconds}s)"
# The cookie-authenticated stream runs in the background and the
# captured output is polled: the upload has already been answered, so
# asset.ready either replays from the retention ring or arrives live
# once the video's three pipeline jobs (metadata, poster, transcode)
# drain its pending-jobs counter. curl's --max-time is a backstop
# only -- the poll deadline below bounds the actual wait.
curl -sS -N --max-time 150 -b "$work_dir/cookies" \
    "$base_url/api/v1/events" >"$work_dir/sse_stream" 2>"$work_dir/sse_err" &
sse_pid=$!
deadline="$(( $(date +%s) + sse_timeout_seconds ))"
until awk -v want="$video_id" '
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
        fail "asset.ready for the video asset $video_id was not delivered within ${sse_timeout_seconds}s; stream so far: $(cat "$work_dir/sse_stream")"
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
echo "    event stream (the video's readiness included):"
sed 's/^/    /' "$work_dir/sse_stream"

echo "==> GET /api/v1/assets/$video_id/play with Range bytes=0-99 must return 206 video/mp4"
status="$(curl -sS -o "$work_dir/play_body" -D "$work_dir/play_headers" \
    -b "$work_dir/cookies" -w '%{http_code}' \
    -H 'Range: bytes=0-99' \
    "$base_url/api/v1/assets/$video_id/play")" \
    || fail "GET /api/v1/assets/$video_id/play request failed"
[ "$status" = "206" ] \
    || fail "GET /api/v1/assets/$video_id/play returned status $status, want 206: $(cat "$work_dir/play_headers") $(head -c 200 "$work_dir/play_body")"
grep -Fq 'Content-Type: video/mp4' "$work_dir/play_headers" \
    || fail "the play stream is not video/mp4: $(cat "$work_dir/play_headers")"
grep -Eq 'Content-Range: bytes 0-99/[0-9]+' "$work_dir/play_headers" \
    || fail "the play stream carries no Content-Range for bytes 0-99: $(cat "$work_dir/play_headers")"
play_bytes="$(wc -c < "$work_dir/play_body" | tr -d ' ')"
[ "$play_bytes" = "100" ] \
    || fail "the play stream served $play_bytes bytes, want the 100 the range asked for"
echo "    served 100 of the range's bytes as video/mp4: $(grep -i 'content-range' "$work_dir/play_headers" | tr -d '\r')"

teardown
echo "==> acceptance flow completed in $(elapsed)s"
echo "M2C ACCEPTANCE PASS"
