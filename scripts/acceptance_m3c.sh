#!/usr/bin/env bash
# M3c acceptance: the albums and sharing milestone's end-to-end
# acceptance -- the chained M3b acceptance (which itself chains the
# M3a chain and through it the whole M1/M2 line, every script owning
# its own compose stack and tearing it down fully, including the
# database volume) runs first and must be green before the M3c phases
# start. Then two stack lifetimes of this repository's own compose
# deployment walk the sharing milestone's story:
#
# Phase 1 (the sharing story over the API, two accounts): the stack
# starts fresh, the first-run admin is created and signed in (the
# session's identity face asserted: the login body and the session
# list both carry the account's userId and role -- the M3c identity
# seam), and a MEMBER account is provisioned through the admin API
# and signs in with its own session. The member uploads one photo of
# their own (their sync stream); the admin uploads the same-person
# face pair (the B4 people arm's cluster structure). The admin
# creates a virtual album and curates it: the member's photo answers
# the honest skip (the curation predicate is owner-based -- a
# foreign asset is never addable), the admin's own two photos land.
# The share grant follows, and the member's whole shared surface is
# asserted: the wall's shared group, the timeline's inclusion (the
# union's shared arm beside the member's own sync stream), the
# read-only detail (callerRole viewer), the per-user favorite, the
# timeline-exclusion toggle with its retraction and re-include, and
# the people surface's B4 sharing arm (the sharee sees the shared
# assets' faces as a cluster; labels stay per-user -- the sharee
# names it, the owner's unnamed region never hears about it). The
# admin's revoke then retracts EVERYTHING the share gave the member
# (the wall card, the timeline items, the detail read, the favorites
# view, the people surface -- while the member's own photo and the
# label row survive), and the album's delete cascades its rows
# (album_items, album_shares) without touching the photos.
#
# Phase 2 (the e2e stack): a fresh stack whose models ride the SAME
# storage bind mount; the Playwright spec web/e2e/m3-albums.spec.ts
# walks the sharing story of the web app through a real browser --
# the sign-in, the wall's create flow, the timeline's add-photos
# handoff, the share dialog (the identity seam's self filter), and
# the member's read-only perspective (the shared section, the
# read-only detail with its leave affordance, the timeline
# inclusion, the viewer drawer's identity-aware read-only state).
# The spec bootstraps itself (the first-run admin and the member
# account through the APIs, the model group's download through the
# admin API), exactly the way every prior e2e spec owns its stack's
# bootstrap.
#
# DOWNLOAD SOURCES (two modes, the m3b script's knobs -- the M3c
# script accepts the same M3B_SOURCE / M3B_HF_MIRROR variables plus
# M3C_SOURCE / M3C_HF_MIRROR aliases that win when set, so a CI
# workflow swapping the chain head needs no other change):
#   M3C_SOURCE=upstream (the default; what CI runs) -- the stack
#       downloads from the manifest's pinned upstreams directly; set
#       M3C_HF_MIRROR (or M3B_HF_MIRROR) to host[:port] (e.g.
#       hf-mirror.com) to route the huggingface.co URLs through an
#       HF domain mirror instead.
#   M3C_SOURCE=offline -- no network at all: the Hugging Face URL
#       path structure is reconstructed under
#       tools/models/cache/offline-root/ (symlinks to the Task 1
#       local weights cache) and served over TLS by a mirror service
#       of the stack (the dev image's python running
#       scripts/offline_mirror_server.py): the mirror rewrite keeps
#       the https scheme, so the local source must speak TLS, and
#       the self-signed certificate's CA is handed to the server
#       container through SSL_CERT_FILE (Go's crypto/x509 honors it
#       on Linux; the CA file itself rides the ./lumemor bind mount).
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message (the failing assertion's name; the stack-bound
# waits also print the server's compose log tail), and the teardown
# still runs. A passing run prints "M3C ACCEPTANCE PASS" as its final
# line.
#
# Requirements: docker with the compose plugin, curl, and (for the
# chained web e2e, which the m2d chain installs) pnpm 9.15; the
# offline mode additionally needs openssl and the lumemor-dev image
# (built by any make docker-test / docker-lint run) plus the Task 1
# weights cache under tools/models/cache/.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
download_timeout_seconds=900
ml_timeout_seconds=420

admin_password="correct-horse-battery"
member_password="correct-horse-battery"
face_portrait="$repo_root/server/internal/ml/testdata/face-portrait.jpg"
face_portrait_orientation="$repo_root/server/internal/ml/testdata/face-portrait-orientation6.jpg"
member_photo="$repo_root/server/internal/ml/testdata/ocr-text.png"

# The download source mode (see the header comment): the M3C_*
# aliases win over the M3B_* variables so this script stays a
# drop-in chain head -- CI's plain invocation (no variables set)
# runs the upstream default, and the mode is exported back as
# M3B_SOURCE (and M3A_SOURCE through the chained scripts) so the
# whole chain runs the SAME mode.
source_mode="${M3C_SOURCE:-${M3B_SOURCE:-upstream}}"
case "$source_mode" in
    upstream | offline) ;;
    *)
        echo "M3C ACCEPTANCE FAIL: M3C_SOURCE must be 'upstream' or 'offline', got '$source_mode'" >&2
        exit 1
        ;;
esac
# An upstream-mode HF domain mirror host (empty = the pinned upstreams
# directly).
hf_mirror="${M3C_HF_MIRROR:-${M3B_HF_MIRROR:-}}"
export M3B_SOURCE="$source_mode"
export M3B_HF_MIRROR="$hf_mirror"

models_cache="$repo_root/tools/models/cache"
offline_root="$models_cache/offline-root"
dev_image="lumemor-dev:latest"
mirror_service="m3a-offline-mirror"
mirror_port="8443"

# One timestamp for the run's unique identities (admin emails, device
# asset ids) so reruns never collide with leftover data, and one for
# the elapsed-time reporting of the phases.
run_start="$(date +%s)"
run_stamp="$run_start"

fail() {
    echo "M3C ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# item_ids prints the ids of an items[] response body in document
# order -- the timeline renders its rows as objects whose "id" key is
# the first field, and the compact encoder keeps document order. The
# quoted "id" pattern cannot match "ownerId" or any other camel-case
# field: the leading quote anchors it against the key's start.
item_ids() {
    grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' "$1" \
        | sed 's/^"id"[[:space:]]*:[[:space:]]*"\([^"]*\)"$/\1/'
}

work_dir="$(mktemp -d)"
work_dir="$(cd "$work_dir" && pwd -P)"

# The compose file set: the repository's compose file, plus -- in
# offline mode -- a generated override that adds the mirror service
# and the CA trust env. The override is written before any compose
# invocation runs.
compose_override="$work_dir/compose-offline.yml"
compose_args=(-f docker-compose.yml)
if [ "$source_mode" = "offline" ]; then
    compose_args+=(-f "$compose_override")
fi

compose() {
    docker compose "${compose_args[@]}" "$@"
}

# Brought-up flag for the teardown trap: a failed step must still
# remove the containers and the pgdata volume, otherwise a rerun would
# hit an instance that is already initialized.
stack_up=0

teardown() {
    if [ "$stack_up" -ne 0 ]; then
        stack_up=0
        echo "==> tearing down the stack and removing volumes"
        compose down -v --remove-orphans || true
    fi
    rm -rf "$work_dir"
}
trap teardown EXIT

# ---------------------------------------------------------------------------
# The offline download source.
# ---------------------------------------------------------------------------

# link_offline_file places one cached weight at its Hugging Face URL
# path under the offline root, as a relative symlink that resolves
# inside the cache tree (the whole cache dir is what the mirror
# container mounts, so the links stay valid in both file systems).
link_offline_file() {
    local url_path="$1" cached_name="$2"
    local dir="$offline_root/$(dirname "$url_path")"
    mkdir -p "$dir"
    local depth ups=""
    depth="$(printf '%s\n' "$(dirname "$url_path")" | awk -F/ '{print NF}')"
    local i
    for i in $(seq 1 $((depth + 1))); do
        ups="../$ups"
    done
    ln -sfn "${ups}${cached_name}" "$offline_root/$url_path"
}

# prepare_offline_source writes the offline root, the mirror
# certificate, the staged CA and the compose override. Must run before
# any compose invocation.
prepare_offline_source() {
    echo "==> preparing the offline download source (no network)"
    local weight
    for weight in \
        face_detection_yunet_2023mar.onnx \
        face_recognition_sface_2021dec.onnx \
        cn_clip_vision.onnx \
        zihuv_text.onnx \
        cn_clip_vocab.txt \
        ch_PP-OCRv4_det_infer.onnx \
        ch_PP-OCRv4_rec_infer.onnx; do
        [ -f "$models_cache/$weight" ] \
            || fail "the Task 1 weights cache is missing $models_cache/$weight -- the offline mode needs it"
    done
    docker image inspect "$dev_image" >/dev/null 2>&1 \
        || fail "the offline mirror rides the dev image $dev_image, which is absent -- run any make docker-test first to build it"
    command -v openssl >/dev/null 2>&1 \
        || fail "the offline mode needs openssl to mint the mirror certificate"

    # The seven pinned upstream artifacts at their Hugging Face URL
    # paths (the M3d text tower and its tokenizer vocabulary among
    # them): what the mirror rewrite makes the download resolve.
    link_offline_file \
        "opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx" \
        "face_detection_yunet_2023mar.onnx"
    link_offline_file \
        "opencv/opencv_zoo/resolve/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx" \
        "face_recognition_sface_2021dec.onnx"
    link_offline_file \
        "felixdu/chinese-clip-vit-base-patch16-onnx/resolve/main/cn_clip_vision.onnx" \
        "cn_clip_vision.onnx"
    link_offline_file \
        "zihuv/chinese-clip-vit-base-patch16-onnx/resolve/main/text.onnx" \
        "zihuv_text.onnx"
    link_offline_file \
        "zihuv/chinese-clip-vit-base-patch16-onnx/resolve/main/vocab.txt" \
        "cn_clip_vocab.txt"
    link_offline_file \
        "SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_det_infer.onnx" \
        "ch_PP-OCRv4_det_infer.onnx"
    link_offline_file \
        "SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_rec_infer.onnx" \
        "ch_PP-OCRv4_rec_infer.onnx"
    local url_path
    for url_path in \
        "opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx" \
        "opencv/opencv_zoo/resolve/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx" \
        "felixdu/chinese-clip-vit-base-patch16-onnx/resolve/main/cn_clip_vision.onnx" \
        "zihuv/chinese-clip-vit-base-patch16-onnx/resolve/main/text.onnx" \
        "zihuv/chinese-clip-vit-base-patch16-onnx/resolve/main/vocab.txt" \
        "SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_det_infer.onnx" \
        "SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_rec_infer.onnx"; do
        [ -e "$offline_root/$url_path" ] \
            || fail "the offline root link $offline_root/$url_path does not resolve"
    done

    # The mirror certificate: a self-signed CA whose SAN is the mirror
    # service's DNS name on the compose network.
    cat >"$work_dir/mirror-openssl.cnf" <<'EOF'
[req]
distinguished_name=dn
x509_extensions=v3
prompt=no
[dn]
CN=m3a-offline-mirror
[v3]
subjectAltName=DNS:m3a-offline-mirror
basicConstraints=critical,CA:TRUE
keyUsage=critical,keyCertSign,digitalSignature
EOF
    openssl req -x509 -newkey rsa:2048 -nodes -days 2 \
        -keyout "$work_dir/mirror-key.pem" -out "$work_dir/mirror-cert.pem" \
        -config "$work_dir/mirror-openssl.cnf" -subj "/CN=$mirror_service" >/dev/null 2>&1 \
        || fail "minting the offline mirror certificate failed"

    # The CA rides the storage bind mount (./lumemor), so the override
    # only wires the env that points Go's x509 at it.
    mkdir -p lumemor
    cp "$work_dir/mirror-cert.pem" lumemor/m3a-offline-ca.pem

    cat >"$compose_override" <<EOF
# Generated by scripts/acceptance_m3c.sh (offline download mode): the
# local mirror service and the CA trust of the server container. The
# CA file itself rides the storage bind mount (./lumemor, staged by
# the script); the weights tree is mounted read-only.
services:
  $mirror_service:
    image: $dev_image
    volumes:
      - $models_cache:/srv:ro
      - $work_dir/mirror-cert.pem:/certs/cert.pem:ro
      - $work_dir/mirror-key.pem:/certs/key.pem:ro
      - $repo_root/scripts/offline_mirror_server.py:/mirror/offline_mirror_server.py:ro
    command: >-
        python3 /mirror/offline_mirror_server.py /srv/offline-root
        $mirror_port /certs/cert.pem /certs/key.pem
  lumemor:
    environment:
      SSL_CERT_FILE: /var/lib/lumemor/m3a-offline-ca.pem
EOF

    # The stack env: the mirror rewrite replaces the huggingface.co
    # host names with the mirror service's network name.
    export LUMEMOR_MODELS_HF_MIRROR="$mirror_service:$mirror_port"
    echo "    offline source ready under $offline_root (mirror service $mirror_service:$mirror_port)"
}

if [ "$source_mode" = "offline" ]; then
    prepare_offline_source
else
    # The upstream mode: empty keeps the pinned upstreams, a host
    # name routes the HF URLs through a domain mirror.
    export LUMEMOR_MODELS_HF_MIRROR="$hf_mirror"
    if [ -n "$hf_mirror" ]; then
        echo "==> download source: upstream through the HF mirror $hf_mirror"
    else
        echo "==> download source: the pinned upstreams directly"
    fi
fi

# ---------------------------------------------------------------------------
# Shared stack and API helpers (the m3b patterns).
# ---------------------------------------------------------------------------

psql_query() {
    compose exec -T db psql -U lumemor -d lumemor -tA -c "$1"
}

wait_healthy() {
    local deadline="$(( $(date +%s) + health_timeout_seconds ))"
    until curl -fsS "$base_url/healthz" >/dev/null 2>&1; do
        if [ "$(date +%s)" -ge "$deadline" ]; then
            compose logs --no-color db lumemor || true
            fail "GET /healthz did not succeed within ${health_timeout_seconds}s"
        fi
        sleep 2
    done
    echo "    server healthy after $(elapsed)s"
}

prepare_bind_mount() {
    # The container runs as the image's fixed uid/gid 10001 and the
    # bind source must be writable by it before the server starts.
    # Docker only creates a missing source as root:root, so the
    # directory is created by the host user first and then chowned
    # through the image itself running as root -- the fresh-install
    # contract without sudo. The chown rides the app image, so it must
    # exist: build it here (a cached no-op when the image is current;
    # `up -d --build` below re-checks it).
    mkdir -p lumemor
    compose build || fail "docker compose build failed"
    compose run --rm --no-deps --entrypoint /bin/sh -u 0 lumemor \
        -c 'chown -R 10001:10001 /var/lib/lumemor' \
        || fail "chown of ./lumemor to 10001:10001 through the image failed"
}

start_stack() {
    echo "==> building and starting the stack"
    compose up -d --build || fail "docker compose up -d --build failed"
    stack_up=1
    wait_healthy
}

reset_stack_state() {
    echo "==> resetting to a clean stack state"
    compose down -v --remove-orphans
    stack_up=1
}

teardown_stack() {
    echo "==> tearing down the phase stack and removing volumes"
    compose down -v --remove-orphans
    stack_up=0
}

# probe_offline_mirror fetches one pinned artifact from the mirror the
# way the server's download will (same network, same CA trust), so a
# broken mirror fails here with a clear cause instead of inside the
# download job.
probe_offline_mirror() {
    if [ "$source_mode" != "offline" ]; then
        return 0
    fi
    local network
    network="$(docker network ls --filter label=com.docker.compose.project=lumemor -q | head -1)"
    [ -n "$network" ] || fail "the compose network of the stack was not found for the mirror probe"
    docker run --rm --network "$network" \
        -e SSL_CERT_FILE=/certs/cert.pem \
        -v "$work_dir/mirror-cert.pem:/certs/cert.pem:ro" \
        "$dev_image" \
        python3 -c 'import sys, urllib.request; urllib.request.urlopen(sys.argv[1])' \
        "https://$mirror_service:$mirror_port/opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx" \
        || { compose logs --no-color --tail 50 "$mirror_service" || true
             fail "the offline mirror did not serve a pinned artifact over TLS (see the mirror logs above)"; }
    echo "    the offline mirror serves the pinned artifacts over TLS"
}

# create_admin_and_login bootstraps the first-run administrator and
# captures the session cookie for the phase's API calls. The phase
# label keeps every stack's identities distinct. The identity seam's
# acceptance assert rides the login body: the session's userId and
# role name whose credentials this is.
cookie_jar="$work_dir/cookies"
create_admin_and_login() {
    local label="$1"
    local admin_email="admin-${run_stamp}-${label}@m3c-acceptance.invalid"
    cookie_jar="$work_dir/cookies_$label"

    echo "==> GET /setup must return 200"
    local status
    status="$(curl -sS -o /dev/null -w '%{http_code}' "$base_url/setup")" \
        || fail "GET /setup request failed"
    [ "$status" = "200" ] \
        || fail "GET /setup returned status $status, want 200"

    echo "==> POST /api/v1/server/setup/admin must return 201"
    status="$(curl -sS -o "$work_dir/create_admin_body" -w '%{http_code}' \
        -H 'Content-Type: application/json' \
        -d "{\"name\":\"M3c Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
        "$base_url/api/v1/server/setup/admin")" \
        || fail "POST /api/v1/server/setup/admin request failed"
    [ "$status" = "201" ] \
        || fail "POST /api/v1/server/setup/admin returned status $status, want 201: $(cat "$work_dir/create_admin_body")"
    admin_user_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/create_admin_body")"
    [ -n "$admin_user_id" ] \
        || fail "the created administrator carries no id: $(cat "$work_dir/create_admin_body")"
    echo "    created administrator $admin_email ($admin_user_id)"

    echo "==> POST /api/v1/auth/login must return 200 and set the session cookie"
    status="$(curl -sS -o "$work_dir/login_body" -c "$cookie_jar" -w '%{http_code}' \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
        "$base_url/api/v1/auth/login")" \
        || fail "POST /api/v1/auth/login request failed"
    [ "$status" = "200" ] \
        || fail "POST /api/v1/auth/login returned status $status, want 200: $(cat "$work_dir/login_body")"
    grep -q 'lumemor_session' "$cookie_jar" \
        || fail "POST /api/v1/auth/login did not set the lumemor_session cookie: $(cat "$work_dir/login_body")"
    echo "    login accepted, session cookie captured"

    # The identity seam: the login body carries the account's
    # identity (the web's login-time source).
    grep -q "\"userId\":\"$admin_user_id\"" "$work_dir/login_body" \
        || fail "the login body does not carry the administrator's userId ($admin_user_id): $(cat "$work_dir/login_body")"
    grep -q '"role":"admin"' "$work_dir/login_body" \
        || fail "the login body does not carry the admin role: $(cat "$work_dir/login_body")"
    echo "    the login body carries the identity: userId + role admin"
}

# create_member_and_login provisions the member account through the
# admin API and signs it in with its own cookie jar. The member's
# user id lands in the member_user_id global (the function's stdout
# carries only the progress lines the log wants).
create_member_and_login() {
    local label="$1" email="$2" name="$3"
    local member_email="${email}-${run_stamp}@m3c-acceptance.invalid"

    echo "==> POST /api/v1/admin/users must return 201 with the member's id"
    local status
    status="$(curl -sS -o "$work_dir/create_member_body" -b "$cookie_jar" -w '%{http_code}' \
        -H 'Content-Type: application/json' \
        -d "{\"name\":\"$name\",\"email\":\"$member_email\",\"password\":\"$member_password\"}" \
        "$base_url/api/v1/admin/users")" \
        || fail "POST /api/v1/admin/users request failed"
    [ "$status" = "201" ] \
        || fail "POST /api/v1/admin/users returned status $status, want 201: $(cat "$work_dir/create_member_body")"
    member_user_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/create_member_body")"
    [ -n "$member_user_id" ] \
        || fail "the created member carries no id: $(cat "$work_dir/create_member_body")"
    grep -q '"role":"user"' "$work_dir/create_member_body" \
        || fail "the created member's role is not the default user: $(cat "$work_dir/create_member_body")"
    echo "    created member $member_email ($member_user_id)"

    cookie_jar="$work_dir/cookies_$label"
    echo "==> the member's POST /api/v1/auth/login must return 200 and carry the member identity"
    status="$(curl -sS -o "$work_dir/member_login_body" -c "$cookie_jar" -w '%{http_code}' \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"$member_email\",\"password\":\"$member_password\"}" \
        "$base_url/api/v1/auth/login")" \
        || fail "the member's POST /api/v1/auth/login request failed"
    [ "$status" = "200" ] \
        || fail "the member's POST /api/v1/auth/login returned status $status, want 200: $(cat "$work_dir/member_login_body")"
    grep -q 'lumemor_session' "$cookie_jar" \
        || fail "the member's login did not set the lumemor_session cookie: $(cat "$work_dir/member_login_body")"
    grep -q "\"userId\":\"$member_user_id\"" "$work_dir/member_login_body" \
        || fail "the member's login body does not carry the member's userId: $(cat "$work_dir/member_login_body")"
    grep -q '"role":"user"' "$work_dir/member_login_body" \
        || fail "the member's login body does not carry the user role: $(cat "$work_dir/member_login_body")"
    echo "    the member's login carries the identity: userId + role user"

    # The restore arm of the seam: the session list the web's boot
    # probe reads answers the identity for an existing session.
    echo "==> the member's GET /api/v1/auth/sessions must carry the identity on every entry"
    status="$(curl -sS -o "$work_dir/member_sessions_body" -b "$cookie_jar" -w '%{http_code}' \
        "$base_url/api/v1/auth/sessions")" \
        || fail "the member's GET /api/v1/auth/sessions request failed"
    [ "$status" = "200" ] \
        || fail "the member's GET /api/v1/auth/sessions returned status $status, want 200: $(cat "$work_dir/member_sessions_body")"
    local entries with_identity
    entries="$(grep -o '"userId"' "$work_dir/member_sessions_body" | wc -l | tr -d ' ')"
    with_identity="$(grep -o "\"userId\":\"$member_user_id\"" "$work_dir/member_sessions_body" | wc -l | tr -d ' ')"
    [ "$entries" -ge 1 ] && [ "$entries" = "$with_identity" ] \
        || fail "the member's session list carries the identity on $with_identity of $entries entries: $(cat "$work_dir/member_sessions_body")"
    echo "    the session list carries the member's identity on every entry (the restore arm)"
}

# upload_fixture uploads one fixture file; the new asset's id is
# printed on stdout (the id chain: never a guess).
upload_fixture() {
    local payload="$1" filename="$2" mime="$3" device_id="$4" taken_at="$5" label="$6" jar="$7"
    local body="$work_dir/upload_body_$label"
    local status
    status="$(curl -sS -o "$body" -b "$jar" -w '%{http_code}' \
        -F "deviceAssetId=$device_id" \
        -F "fileName=$filename" \
        -F "deviceCreatedAt=$taken_at" \
        -F 'deviceTimeZone=UTC' \
        -F "photo=@$payload;filename=$filename;type=$mime" \
        "$base_url/api/v1/assets")" \
        || fail "POST /api/v1/assets ($label) request failed"
    [ "$status" = "201" ] \
        || fail "POST /api/v1/assets ($label) returned status $status, want 201: $(cat "$body")"
    local body_text
    body_text="$(cat "$body")"
    printf '%s\n' "$body_text" | grep -Eq '"state"[[:space:]]*:[[:space:]]*"created"' \
        || fail "POST /api/v1/assets ($label) did not report state created: $body_text"
    local asset_id
    asset_id="$(printf '%s\n' "$body_text" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
    [ -n "$asset_id" ] \
        || fail "POST /api/v1/assets ($label) response carries no asset id: $body_text"
    echo "$asset_id"
}

# wait_asset_ready polls the asset's detail until isReady turns true
# (the pipeline's pending-jobs counter drained -- the seam ML dispatch
# rides the same completion). The OWNER's cookie reads the detail.
wait_asset_ready() {
    local asset_id="$1" label="$2" jar="$3"
    local timeout_s=120
    local deadline="$(( $(date +%s) + timeout_s ))"
    until curl -fsS -b "$jar" "$base_url/api/v1/assets/$asset_id" 2>/dev/null \
        | grep -q '"isReady":true'; do
        if [ "$(date +%s)" -ge "$deadline" ]; then
            compose logs --no-color --tail 200 lumemor || true
            fail "the $label asset ($asset_id) did not become ready within ${timeout_s}s"
        fi
        sleep 2
    done
    echo "    the $label asset is ready ($asset_id)"
}

# wait_model_download triggers the default group's download and waits
# for the job run to reach succeeded (the m3a phase-2 discipline: the
# warm mount's re-verification is seconds, the cold transfer minutes).
wait_model_download() {
    echo "==> POST /api/v1/admin/models/download must return 202 with a job run id"
    local status
    status="$(curl -sS -o "$work_dir/download_body" -b "$cookie_jar" -w '%{http_code}' \
        -X POST "$base_url/api/v1/admin/models/download")" \
        || fail "POST /api/v1/admin/models/download request failed"
    [ "$status" = "202" ] \
        || fail "POST /api/v1/admin/models/download returned status $status, want 202: $(cat "$work_dir/download_body")"
    grep -q '"jobRunId":"[^"]*"' "$work_dir/download_body" \
        || fail "POST /api/v1/admin/models/download carries no jobRunId: $(cat "$work_dir/download_body")"
    echo "    download started: $(cat "$work_dir/download_body")"

    echo "==> the model_download job run must reach succeeded (up to ${download_timeout_seconds}s)"
    download_row() {
        grep -o '"kind":"model_download"[^}]*' "$work_dir/jobs_body" | head -1
    }
    local deadline row state
    deadline="$(( $(date +%s) + download_timeout_seconds ))"
    while :; do
        status="$(curl -sS -o "$work_dir/jobs_body" -b "$cookie_jar" -w '%{http_code}' \
            "$base_url/api/v1/admin/jobs")" \
            || fail "GET /api/v1/admin/jobs request failed"
        [ "$status" = "200" ] \
            || fail "GET /api/v1/admin/jobs returned status $status, want 200: $(cat "$work_dir/jobs_body")"
        row="$(download_row)"
        state="$(printf '%s\n' "$row" | sed -n 's/.*"state":"\([a-z]*\)".*/\1/p')"
        if [ "$state" = "succeeded" ]; then
            break
        fi
        if [ "$(date +%s)" -ge "$deadline" ]; then
            compose logs --no-color --tail 200 lumemor || true
            fail "the model_download run did not succeed within ${download_timeout_seconds}s (last row: $row)"
        fi
        if [ "$state" = "failed" ]; then
            compose logs --no-color --tail 200 lumemor || true
            fail "the model_download run failed: $row (see the server logs above)"
        fi
        sleep 3
    done
    echo "    the download run succeeded after $(elapsed)s: $row"
}

# ---------------------------------------------------------------------------
# The chained regressions: the M3b acceptance (which chains the M3a
# chain, which chains the M2d chain, which chains the M1, M2a, M2b and
# M2c acceptances -- the full M1 to M3b serial line, each script
# owning its own stacks).
# ---------------------------------------------------------------------------

echo "==> running the prior milestone acceptance (the M3b chain owns its stacks)"
bash scripts/acceptance_m3b.sh \
    || fail "scripts/acceptance_m3b.sh failed (see its output above)"
echo "    acceptance_m3b (and its chained M1/M2/M3a line) passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 1: the sharing story over the API (two accounts, the whole
# share lifecycle).
# ---------------------------------------------------------------------------

echo "==> phase 1/2: the sharing story over the API (source: $source_mode)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror
create_admin_and_login sharing
create_member_and_login member grace "Grace Hopper"
admin_jar="$work_dir/cookies_sharing"
member_jar="$work_dir/cookies_member"
cookie_jar="$admin_jar"
wait_model_download

echo "==> the member uploads one photo of their own (their sync stream)"
[ -f "$member_photo" ] || fail "fixture missing: $member_photo"
member_photo_id="$(upload_fixture \
    "$member_photo" IMG_M3C_MEMBER.png image/png \
    "m3c-acceptance-${run_stamp}-member" "2026-06-01T12:00:00Z" member-own "$member_jar")"
wait_asset_ready "$member_photo_id" member-own "$member_jar"

echo "==> the admin uploads the same-person face pair (the people arm's cluster structure)"
[ -f "$face_portrait" ] || fail "fixture missing: $face_portrait"
[ -f "$face_portrait_orientation" ] || fail "fixture missing: $face_portrait_orientation"
portrait_id="$(upload_fixture \
    "$face_portrait" IMG_M3C_0001.jpg image/jpeg \
    "m3c-acceptance-${run_stamp}-p1" "2026-05-01T12:00:00Z" portrait "$admin_jar")"
twin_id="$(upload_fixture \
    "$face_portrait_orientation" IMG_M3C_0002.jpg image/jpeg \
    "m3c-acceptance-${run_stamp}-p2-twin" "2026-05-02T12:00:00Z" twin "$admin_jar")"
wait_asset_ready "$portrait_id" portrait "$admin_jar"
wait_asset_ready "$twin_id" orientation-twin "$admin_jar"

echo "==> the recognition must land two embedded faces in one cluster (up to ${ml_timeout_seconds}s)"
ml_facts() {
    printf '%s %s %s %s\n' \
        "$(psql_query 'SELECT count(*) FROM faces')" \
        "$(psql_query 'SELECT count(embedding) FROM faces')" \
        "$(psql_query 'SELECT count(*) FROM face_clusters')" \
        "$(psql_query 'SELECT count(*) FROM faces WHERE cluster_id IS NOT NULL')"
}
deadline="$(( $(date +%s) + ml_timeout_seconds ))"
while :; do
    read -r faces embedded clusters clustered <<<"$(ml_facts)"
    if [ "$faces" -ge 2 ] && [ "$embedded" -ge 2 ] \
        && [ "$clusters" -ge 1 ] && [ "$clustered" -ge 2 ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the faces did not land within ${ml_timeout_seconds}s: faces=$faces embedded=$embedded clusters=$clusters clustered=$clustered"
    fi
    sleep 3
done
echo "    faces=$faces (embedded $embedded), face_clusters=$clusters (clustered faces $clustered)"

echo "==> the member's people surface stays empty before the share (they own no face)"
status="$(curl -sS -o "$work_dir/member_unnamed_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (member, pre-share) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (member, pre-share) returned status $status, want 200: $(cat "$work_dir/member_unnamed_body")"
grep -q '"items":\[\]' "$work_dir/member_unnamed_body" \
    || fail "the member's unnamed region is not empty before the share: $(cat "$work_dir/member_unnamed_body")"
echo "    the member sees no cluster yet"

echo "==> POST /api/v1/albums must create the admin's virtual album"
status="$(curl -sS -o "$work_dir/album_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d '{"title":"M3c Acceptance Album","description":"The sharing story"}' \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/albums returned status $status, want 201: $(cat "$work_dir/album_body")"
grep -q '"callerRole":"owner"' "$work_dir/album_body" \
    || fail "the created album's card is not the owner's: $(cat "$work_dir/album_body")"
grep -q '"itemCount":0' "$work_dir/album_body" \
    || fail "the created album's count is not zero: $(cat "$work_dir/album_body")"
album_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/album_body")"
[ -n "$album_id" ] \
    || fail "the created album carries no id: $(cat "$work_dir/album_body")"
echo "    the album exists ($album_id), owned by the admin"

echo "==> POST /api/v1/albums/{id}/items: the member's photo answers the honest skip, the admin's pair lands"
status="$(curl -sS -o "$work_dir/items_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"assetIds\":[\"$member_photo_id\"]}" \
    "$base_url/api/v1/albums/$album_id/items")" \
    || fail "POST /api/v1/albums/$album_id/items (member photo) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/albums/$album_id/items (member photo) returned status $status, want 200: $(cat "$work_dir/items_body")"
grep -q '"added":0' "$work_dir/items_body" \
    || fail "the member's photo was added to the admin's album: $(cat "$work_dir/items_body")"
grep -q '"skipped":1' "$work_dir/items_body" \
    || fail "the member's photo did not answer the honest skip: $(cat "$work_dir/items_body")"
echo "    the member's photo is never addable (the owner-based curation predicate)"
status="$(curl -sS -o "$work_dir/items_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"assetIds\":[\"$portrait_id\",\"$twin_id\"]}" \
    "$base_url/api/v1/albums/$album_id/items")" \
    || fail "POST /api/v1/albums/$album_id/items (admin pair) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/albums/$album_id/items (admin pair) returned status $status, want 200: $(cat "$work_dir/items_body")"
grep -q '"added":2' "$work_dir/items_body" \
    || fail "the admin's pair did not land: $(cat "$work_dir/items_body")"
grep -q '"skipped":0' "$work_dir/items_body" \
    || fail "the admin's pair was partly skipped: $(cat "$work_dir/items_body")"
echo "    the admin's two photos are members"

echo "==> POST /api/v1/albums/{id}/shares must grant the member the view"
status="$(curl -sS -o "$work_dir/share_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"userId\":\"$member_user_id\"}" \
    "$base_url/api/v1/albums/$album_id/shares")" \
    || fail "POST /api/v1/albums/$album_id/shares request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/albums/$album_id/shares returned status $status, want 200: $(cat "$work_dir/share_body")"
grep -q "\"userId\":\"$member_user_id\"" "$work_dir/share_body" \
    || fail "the share row does not name the member: $(cat "$work_dir/share_body")"
echo "    the member holds the view"

echo "==> the member's wall must list the album in the shared group"
status="$(curl -sS -o "$work_dir/member_wall_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums")" \
    || fail "GET /api/v1/albums (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums (member) returned status $status, want 200: $(cat "$work_dir/member_wall_body")"
grep -q "\"$album_id\"" "$work_dir/member_wall_body" \
    || fail "the member's wall does not list the shared album: $(cat "$work_dir/member_wall_body")"
grep -q '"callerRole":"viewer"' "$work_dir/member_wall_body" \
    || fail "the member's wall card is not the viewer's: $(cat "$work_dir/member_wall_body")"
grep -q '"itemCount":2' "$work_dir/member_wall_body" \
    || fail "the member's wall card does not carry the honest count: $(cat "$work_dir/member_wall_body")"
echo "    the shared card lists with the viewer role and the live count"

echo "==> the member's timeline must include the shared items beside their own photo"
status="$(curl -sS -o "$work_dir/member_timeline_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (member) returned status $status, want 200: $(cat "$work_dir/member_timeline_body")"
for wanted_id in "$member_photo_id" "$portrait_id" "$twin_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/member_timeline_body" \
        || fail "the member's timeline does not list $wanted_id: $(cat "$work_dir/member_timeline_body")"
done
member_timeline_ids="$(item_ids "$work_dir/member_timeline_body" | tr '\n' ' ')"
[ "$member_timeline_ids" = "$(printf '%s ' "$member_photo_id" "$twin_id" "$portrait_id")" ] \
    || fail "the member's timeline holds [$member_timeline_ids], want the member's photo then the shared pair newest-first: $(cat "$work_dir/member_timeline_body")"
echo "    three items: the member's own photo and the two shared ones (the union's shared arm)"

echo "==> the member's album detail must read as viewer"
status="$(curl -sS -o "$work_dir/member_detail_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$album_id")" \
    || fail "GET /api/v1/albums/$album_id (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/$album_id (member) returned status $status, want 200: $(cat "$work_dir/member_detail_body")"
grep -q '"callerRole":"viewer"' "$work_dir/member_detail_body" \
    || fail "the member's detail is not the viewer's: $(cat "$work_dir/member_detail_body")"
grep -q "\"$portrait_id\"" "$work_dir/member_detail_body" \
    || fail "the member's detail does not list the shared items: $(cat "$work_dir/member_detail_body")"
grep -q '"inTimeline":true' "$work_dir/member_detail_body" \
    || fail "the member's detail does not carry the excludable inTimeline field: $(cat "$work_dir/member_detail_body")"
echo "    the detail answers the viewer role with the items page"

echo "==> the member favorites a shared asset (per-user state)"
status="$(curl -sS -o "$work_dir/member_favorite_body" -b "$member_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"favorite\",\"ids\":[\"$portrait_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (favorite) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (favorite) returned status $status, want 200: $(cat "$work_dir/member_favorite_body")"
grep -q '"affected":1' "$work_dir/member_favorite_body" \
    || fail "the member's favorite did not affect the shared asset: $(cat "$work_dir/member_favorite_body")"
grep -q '"missing":\[\]' "$work_dir/member_favorite_body" \
    || fail "the member's favorite reported missing ids: $(cat "$work_dir/member_favorite_body")"
status="$(curl -sS -o "$work_dir/member_favorites_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/favorites")" \
    || fail "GET /api/v1/assets/favorites (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/favorites (member) returned status $status, want 200: $(cat "$work_dir/member_favorites_body")"
grep -q "\"$portrait_id\"" "$work_dir/member_favorites_body" \
    || fail "the member's favorites view does not list the shared asset: $(cat "$work_dir/member_favorites_body")"
echo "    the shared asset is the member's favorite"

echo "==> the timeline-exclusion toggle must retract and re-include the shared items"
status="$(curl -sS -o "$work_dir/member_pref_body" -b "$member_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"inTimeline":false}' \
    "$base_url/api/v1/albums/$album_id/timeline-preference")" \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference returned status $status, want 200: $(cat "$work_dir/member_pref_body")"
grep -q '"inTimeline":false' "$work_dir/member_pref_body" \
    || fail "the preference response does not carry the stored switch: $(cat "$work_dir/member_pref_body")"
status="$(curl -sS -o "$work_dir/member_timeline_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (member, excluded) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (member, excluded) returned status $status, want 200: $(cat "$work_dir/member_timeline_body")"
member_timeline_ids="$(item_ids "$work_dir/member_timeline_body" | tr '\n' ' ')"
[ "$member_timeline_ids" = "$(printf '%s ' "$member_photo_id")" ] \
    || fail "the excluded timeline holds [$member_timeline_ids], want the member's own photo alone (the sync exemption): $(cat "$work_dir/member_timeline_body")"
echo "    the shared items left; the member's own photo stayed (the sync arm's exemption)"
status="$(curl -sS -o "$work_dir/member_pref_body" -b "$member_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"inTimeline":true}' \
    "$base_url/api/v1/albums/$album_id/timeline-preference")" \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (re-include) request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (re-include) returned status $status, want 200: $(cat "$work_dir/member_pref_body")"
status="$(curl -sS -o "$work_dir/member_timeline_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (member, re-included) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (member, re-included) returned status $status, want 200: $(cat "$work_dir/member_timeline_body")"
for wanted_id in "$member_photo_id" "$portrait_id" "$twin_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/member_timeline_body" \
        || fail "the re-included timeline does not list $wanted_id: $(cat "$work_dir/member_timeline_body")"
done
echo "    the shared items returned with the re-include"

echo "==> the people surface's B4 sharing arm: the sharee sees the shared faces, labels stay per-user"
status="$(curl -sS -o "$work_dir/member_unnamed_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (member) returned status $status, want 200: $(cat "$work_dir/member_unnamed_body")"
member_cluster_id="$(sed -n 's/.*"clusterId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/member_unnamed_body")"
[ -n "$member_cluster_id" ] \
    || fail "the member's unnamed region lists no cluster: $(cat "$work_dir/member_unnamed_body")"
grep -q '"faceCount":2' "$work_dir/member_unnamed_body" \
    || fail "the member's cluster does not carry both shared faces: $(cat "$work_dir/member_unnamed_body")"
echo "    the sharee sees the shared assets' cluster (2 faces)"
status="$(curl -sS -o "$work_dir/member_label_body" -b "$member_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"name":"Sharee Pal"}' \
    "$base_url/api/v1/people/$member_cluster_id/label")" \
    || fail "PUT /api/v1/people/$member_cluster_id/label (member) request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/people/$member_cluster_id/label (member) returned status $status, want 200: $(cat "$work_dir/member_label_body")"
grep -q '"name":"Sharee Pal"' "$work_dir/member_label_body" \
    || fail "the member's label response does not carry the name: $(cat "$work_dir/member_label_body")"
status="$(curl -sS -o "$work_dir/member_people_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people (member) returned status $status, want 200: $(cat "$work_dir/member_people_body")"
grep -q '"name":"Sharee Pal"' "$work_dir/member_people_body" \
    || fail "the member's named region does not list their label: $(cat "$work_dir/member_people_body")"
echo "    the sharee named the shared cluster"
status="$(curl -sS -o "$work_dir/admin_unnamed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (admin) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (admin) returned status $status, want 200: $(cat "$work_dir/admin_unnamed_body")"
grep -q "\"$member_cluster_id\"" "$work_dir/admin_unnamed_body" \
    || fail "the owner's unnamed region lost the cluster after the sharee's label: $(cat "$work_dir/admin_unnamed_body")"
status="$(curl -sS -o "$work_dir/admin_people_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people (admin) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people (admin) returned status $status, want 200: $(cat "$work_dir/admin_people_body")"
grep -q '^\[\]$' "$work_dir/admin_people_body" \
    || fail "the owner's named region is not empty (the sharee's label leaked): $(cat "$work_dir/admin_people_body")"
echo "    labels stay per-user: the owner's unnamed region kept the cluster, their named region stayed empty"

echo "==> the admin's revoke must retract everything the share gave the member"
status="$(curl -sS -o /dev/null -b "$admin_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$album_id/shares/$member_user_id")" \
    || fail "DELETE /api/v1/albums/$album_id/shares/$member_user_id request failed"
[ "$status" = "204" ] \
    || fail "DELETE /api/v1/albums/$album_id/shares/$member_user_id returned status $status, want 204"

status="$(curl -sS -o "$work_dir/member_wall_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums")" \
    || fail "GET /api/v1/albums (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums (member, revoked) returned status $status, want 200: $(cat "$work_dir/member_wall_body")"
# The revoke's honest remainder: the member's own default album (the
# pinned sync card carrying their own upload) survives -- the shared
# card must be gone. The wall answers exactly one album, the member's
# own sync card, with the shared album's id absent.
shared_gone="$(grep -c "\"id\":\"$album_id\"" "$work_dir/member_wall_body" || true)"
[ "$shared_gone" = "0" ] \
    || fail "the member's wall still lists the revoked album: $(cat "$work_dir/member_wall_body")"
sync_only="$(grep -c '"type":"sync"' "$work_dir/member_wall_body" || true)"
[ "$sync_only" = "1" ] \
    || fail "the member's wall after the revoke should carry exactly their own sync card, found $sync_only: $(cat "$work_dir/member_wall_body")"
echo "    the shared card is gone; the member's own default album remains"

status="$(curl -sS -o "$work_dir/member_timeline_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (member, revoked) returned status $status, want 200: $(cat "$work_dir/member_timeline_body")"
member_timeline_ids="$(item_ids "$work_dir/member_timeline_body" | tr '\n' ' ')"
[ "$member_timeline_ids" = "$(printf '%s ' "$member_photo_id")" ] \
    || fail "the revoked timeline holds [$member_timeline_ids], want the member's own photo alone: $(cat "$work_dir/member_timeline_body")"
echo "    the timeline keeps only the member's own photo"

status="$(curl -sS -o "$work_dir/member_detail_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$album_id")" \
    || fail "GET /api/v1/albums/$album_id (member, revoked) request failed"
[ "$status" = "404" ] \
    || fail "the revoked album's detail returned status $status, want 404: $(cat "$work_dir/member_detail_body")"
echo "    the detail answers the single-shape 404"

status="$(curl -sS -o "$work_dir/member_favorites_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/favorites")" \
    || fail "GET /api/v1/assets/favorites (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/favorites (member, revoked) returned status $status, want 200: $(cat "$work_dir/member_favorites_body")"
grep -q '"items":\[\]' "$work_dir/member_favorites_body" \
    || fail "the member's favorites view still lists the retracted asset: $(cat "$work_dir/member_favorites_body")"
echo "    the favorites view lost the shared asset (the row survives per-user, the view reads the union)"

status="$(curl -sS -o "$work_dir/member_people_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people (member, revoked) returned status $status, want 200: $(cat "$work_dir/member_people_body")"
grep -q '^\[\]$' "$work_dir/member_people_body" \
    || fail "the member's named region survived the revoke: $(cat "$work_dir/member_people_body")"
status="$(curl -sS -o "$work_dir/member_unnamed_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (member, revoked) returned status $status, want 200: $(cat "$work_dir/member_unnamed_body")"
grep -q '"items":\[\]' "$work_dir/member_unnamed_body" \
    || fail "the member's unnamed region survived the revoke: $(cat "$work_dir/member_unnamed_body")"
status="$(curl -sS -o "$work_dir/member_cluster_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/$member_cluster_id")" \
    || fail "GET /api/v1/people/$member_cluster_id (member, revoked) request failed"
[ "$status" = "404" ] \
    || fail "the member's cluster detail returned status $status, want 404 after the retraction: $(cat "$work_dir/member_cluster_body")"
member_label_rows="$(psql_query "SELECT count(*) FROM user_face_labels WHERE cluster_id = '$member_cluster_id'")"
[ "$member_label_rows" = "1" ] \
    || fail "the member's label row did not survive the retraction ($member_label_rows rows, want 1)"
echo "    the people surface retracted; the member's label row survives (the per-user discipline)"

status="$(curl -sS -o /dev/null -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/$member_photo_id")" \
    || fail "GET /api/v1/assets/$member_photo_id (member, revoked) request failed"
[ "$status" = "200" ] \
    || fail "the member's own asset detail returned status $status, want 200 (the sync stream never retracts)"
echo "    the member's own photo stays readable"

echo "==> the album's delete must cascade its rows without touching the photos"
status="$(curl -sS -o /dev/null -b "$admin_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$album_id")" \
    || fail "DELETE /api/v1/albums/$album_id request failed"
[ "$status" = "204" ] \
    || fail "DELETE /api/v1/albums/$album_id returned status $status, want 204"
status="$(curl -sS -o "$work_dir/admin_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$album_id")" \
    || fail "GET /api/v1/albums/$album_id (admin, deleted) request failed"
[ "$status" = "404" ] \
    || fail "the deleted album's detail returned status $status, want 404: $(cat "$work_dir/admin_detail_body")"
album_rows="$(psql_query "SELECT count(*) FROM albums WHERE id = '$album_id'")"
[ "$album_rows" = "0" ] \
    || fail "the album row survived its own delete ($album_rows rows)"
item_rows="$(psql_query "SELECT count(*) FROM album_items WHERE album_id = '$album_id'")"
[ "$item_rows" = "0" ] \
    || fail "$item_rows album_items rows survived the delete (the cascade)"
share_rows="$(psql_query "SELECT count(*) FROM album_shares WHERE album_id = '$album_id'")"
[ "$share_rows" = "0" ] \
    || fail "$share_rows album_shares rows survived the delete (the cascade)"
admin_assets="$(psql_query "SELECT count(*) FROM assets WHERE id IN ('$portrait_id', '$twin_id') AND deleted_at IS NULL")"
[ "$admin_assets" = "2" ] \
    || fail "the album delete touched the photos ($admin_assets of 2 live)"
echo "    the album, its membership rows and its share rows are gone; the photos stay"

status="$(curl -sS -o "$work_dir/admin_unnamed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (admin, after delete) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (admin, after delete) returned status $status, want 200: $(cat "$work_dir/admin_unnamed_body")"
grep -q "\"$member_cluster_id\"" "$work_dir/admin_unnamed_body" \
    || fail "the owner lost the faces when the album left (the photos stay): $(cat "$work_dir/admin_unnamed_body")"
echo "    the owner's people surface keeps the faces (the photos never left)"
if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the sharing phase (see compose logs)"
fi
teardown_stack
echo "    phase 1 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 2: the e2e stack (the M3c sharing story of the web app).
# ---------------------------------------------------------------------------

echo "==> phase 2/2: the e2e stack (the M3c sharing story of the web app)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror

echo "==> running the Playwright spec (web/e2e/m3-albums.spec.ts)"
if ! (cd web && pnpm e2e e2e/m3-albums.spec.ts); then
    echo "==> server logs at the time of the failure:"
    compose logs --no-color --tail 200 lumemor || true
    fail "the M3c e2e spec failed (see the report above; artifacts under web/test-results/)"
fi
teardown_stack
echo "    phase 2 passed after $(elapsed)s"

echo "==> acceptance flow completed in $(elapsed)s"
echo "M3C ACCEPTANCE PASS"
