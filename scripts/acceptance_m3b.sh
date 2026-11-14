#!/usr/bin/env bash
# M3b acceptance: the people surface's end-to-end acceptance -- the
# chained M3a acceptance (which itself chains the M2d chain and
# through it the whole M1/M2 line, every script owning its own
# compose stack and tearing it down fully, including the database
# volume) runs first and must be green before the M3b phases start.
# Then two stack lifetimes of this repository's own compose
# deployment walk the people story:
#
# Phase 1 (the people story over the API): the stack starts fresh,
# the first-run admin is created and signed in, the default model
# group downloads (through the same source mode the chained M3a
# used; the storage bind mount ./lumemor persists across the
# script's stack lifetimes, so the weights transfer once per mount
# lifetime and this phase's download re-verifies and registers them
# without re-fetching), and FOUR fixtures upload -- the face
# portrait, its EXIF-orientation twin of the SAME person, and TWO
# portraits of a DIFFERENT person (the clusterer's density rule
# needs two faces to birth a cluster, one per pair). The database
# carries the cluster structure the people surface is built on, and
# the asserts inside the database container (docker compose exec db
# psql) prove it: exactly two clusters, the same-person faces share
# their cluster, the cross-person faces sit in the other one -- with
# every pairwise embedding cosine distance printed as the first real
# cross-person separation evidence (the SFace-128d family-scale
# observation the chained M3a script printed).
#
# The story then walks the surface itself through the API: the
# unnamed region lists both clusters; PUT label names the
# same-person cluster "Alpha" (the named region gains it, the
# unnamed region loses it); the other cluster is named "Beta"; POST
# merge folds Beta into Alpha (the response's survivingClusterId and
# mergedClusterIds asserted, Beta's cluster gone WITH its label row
# -- the honest cost -- while Alpha's label survives and the merged
# person carries all four faces); PUT hidden flags Alpha (hidden
# people STAY listed with isHidden true); and DELETE of the assets
# empties the cluster until it dissolves -- Alpha's label dies with
# it (the Rehome/Dissolve discipline's label cascade, the
# partial-delete linkage): a partial delete keeps the person with
# the honest count, the last delete removes the person from every
# region and every table.
#
# Phase 2 (the e2e stack): a fresh stack whose models ride the SAME
# storage bind mount; the Playwright spec web/e2e/m3-people.spec.ts
# walks the people story of the web app through a real browser --
# the sign-in, the API-seeded library, the unnamed region's
# clusters, the naming dialog, the cluster detail, the merge flow of
# the detail's "merge into", and the viewer's face chips. The spec
# bootstraps itself (the first-run admin through the setup API, the
# model group's download through the admin API), exactly the way
# every prior e2e spec owns its stack's bootstrap.
#
# DOWNLOAD SOURCES (two modes, the M3a script's knobs -- the M3b
# script accepts the same M3A_SOURCE / M3A_HF_MIRROR variables plus
# M3B_SOURCE / M3B_HF_MIRROR aliases that win when set, so a CI
# workflow swapping the chain head needs no other change):
#   M3B_SOURCE=upstream (the default; what CI runs) -- the stack
#       downloads from the manifest's pinned upstreams directly; set
#       M3B_HF_MIRROR (or M3A_HF_MIRROR) to host[:port] (e.g.
#       hf-mirror.com) to route the huggingface.co URLs through an
#       HF domain mirror instead.
#   M3B_SOURCE=offline -- no network at all: the Hugging Face URL
#       path structure is reconstructed under
#       tools/models/cache/offline-root/ (symlinks to the local
#       weights cache) and served over TLS by a mirror service
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
# still runs. A passing run prints "M3B ACCEPTANCE PASS" as its final
# line.
#
# Requirements: docker with the compose plugin, curl, and (for the
# chained web e2e, which the m2d chain installs) pnpm 9.15; the
# offline mode additionally needs openssl and the lumemor-dev image
# (built by any make docker-test / docker-lint run) plus the local
# weights cache under tools/models/cache/.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
download_timeout_seconds=900
ml_timeout_seconds=420

admin_password="correct-horse-battery"
face_portrait="$repo_root/server/internal/ml/testdata/face-portrait.jpg"
face_portrait_orientation="$repo_root/server/internal/ml/testdata/face-portrait-orientation6.jpg"
face_portrait_other="$repo_root/server/internal/ml/testdata/face-portrait-other.jpg"
face_portrait_other_second="$repo_root/server/internal/ml/testdata/face-portrait-other-second.jpg"

# The download source mode (see the header comment): the M3B_*
# aliases win over the M3A_* variables so this script stays a
# drop-in chain head -- CI's plain invocation (no variables set)
# runs the upstream default, and the mode is exported back as
# M3A_SOURCE so the chained acceptance_m3a.sh runs the SAME mode.
source_mode="${M3B_SOURCE:-${M3A_SOURCE:-upstream}}"
case "$source_mode" in
    upstream | offline) ;;
    *)
        echo "M3B ACCEPTANCE FAIL: M3B_SOURCE must be 'upstream' or 'offline', got '$source_mode'" >&2
        exit 1
        ;;
esac
# An upstream-mode HF domain mirror host (empty = the pinned upstreams
# directly).
hf_mirror="${M3B_HF_MIRROR:-${M3A_HF_MIRROR:-}}"
export M3A_SOURCE="$source_mode"
export M3A_HF_MIRROR="$hf_mirror"

models_cache="$repo_root/tools/models/cache"
offline_root="$models_cache/offline-root"
dev_image="lumemor-dev:latest"
mirror_service="m3a-offline-mirror"
mirror_port="8443"

# The suites never reach the outside world: the release check is off
# at the deployment level for every acceptance stack (its own request
# would be the only outbound one the stories do not stub). Exported
# before either source mode is prepared: the offline mirror boots the
# check exactly like the upstream mode does.
export LUMEMOR_UPDATE_CHECK_DISABLED=1

# One timestamp for the run's unique identities (admin emails, device
# asset ids) so reruns never collide with leftover data, and one for
# the elapsed-time reporting of the phases.
run_start="$(date +%s)"
run_stamp="$run_start"

fail() {
    echo "M3B ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
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
            || fail "the local weights cache is missing $models_cache/$weight -- the offline mode needs it"
    done
    docker image inspect "$dev_image" >/dev/null 2>&1 \
        || fail "the offline mirror rides the dev image $dev_image, which is absent -- run any make docker-test first to build it"
    command -v openssl >/dev/null 2>&1 \
        || fail "the offline mode needs openssl to mint the mirror certificate"

    # The seven pinned upstream artifacts at their Hugging Face URL
    # paths (the CLIP text tower and its tokenizer vocabulary among
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
# Generated by scripts/acceptance_m3b.sh (offline download mode): the
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
# Shared stack and API helpers (the m3a patterns).
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
# label keeps every stack's identities distinct.
cookie_jar="$work_dir/cookies"
create_admin_and_login() {
    local label="$1"
    local admin_email="admin-${run_stamp}-${label}@m3b-acceptance.invalid"
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
        -d "{\"name\":\"M3b Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
        "$base_url/api/v1/server/setup/admin")" \
        || fail "POST /api/v1/server/setup/admin request failed"
    [ "$status" = "201" ] \
        || fail "POST /api/v1/server/setup/admin returned status $status, want 201: $(cat "$work_dir/create_admin_body")"
    echo "    created administrator $admin_email"

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
}

# upload_fixture uploads one fixture file; the new asset's id is
# printed on stdout (the id chain: never a guess).
upload_fixture() {
    local payload="$1" filename="$2" mime="$3" device_id="$4" taken_at="$5" label="$6"
    local body="$work_dir/upload_body_$label"
    local status
    status="$(curl -sS -o "$body" -b "$cookie_jar" -w '%{http_code}' \
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
# rides the same completion).
wait_asset_ready() {
    local asset_id="$1" label="$2"
    local timeout_s=120
    local deadline="$(( $(date +%s) + timeout_s ))"
    until curl -fsS -b "$cookie_jar" "$base_url/api/v1/assets/$asset_id" 2>/dev/null \
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

# face_cluster_of prints the cluster id of the (single) face of one
# asset -- empty when the asset carries no clustered face yet.
face_cluster_of() {
    psql_query "SELECT cluster_id::text FROM faces WHERE asset_id = '$1'"
}

# pairwise_distance prints the cosine distance of the two single faces
# of two assets (0 = identical embeddings).
pairwise_distance() {
    psql_query "SELECT round((a.embedding <=> b.embedding)::numeric, 6) FROM faces a, faces b WHERE a.asset_id = '$1' AND b.asset_id = '$2'"
}

# ---------------------------------------------------------------------------
# The chained regressions: the M3a acceptance (which chains the M2d
# chain, which chains the M1, M2a, M2b and M2c acceptances -- the full
# M1 to M3a serial line, each script owning its own stacks).
# ---------------------------------------------------------------------------

echo "==> running the prior acceptance (the M3a chain owns its stacks)"
bash scripts/acceptance_m3a.sh \
    || fail "scripts/acceptance_m3a.sh failed (see its output above)"
echo "    acceptance_m3a (and its chained M1/M2/M2d line) passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 1: the people story over the API (the cluster structure, the
# label/merge/hide flows, the asset-delete dissolution).
# ---------------------------------------------------------------------------

echo "==> phase 1/2: the people story over the API (source: $source_mode)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror
create_admin_and_login people
wait_model_download

echo "==> uploading the same-person pair and the different-person pair"
[ -f "$face_portrait" ] || fail "fixture missing: $face_portrait"
[ -f "$face_portrait_orientation" ] || fail "fixture missing: $face_portrait_orientation"
[ -f "$face_portrait_other" ] || fail "fixture missing: $face_portrait_other"
[ -f "$face_portrait_other_second" ] || fail "fixture missing: $face_portrait_other_second"
portrait_id="$(upload_fixture \
    "$face_portrait" IMG_M3B_0001.jpg image/jpeg \
    "m3b-acceptance-${run_stamp}-p1" "2026-01-01T12:00:00Z" portrait)"
twin_id="$(upload_fixture \
    "$face_portrait_orientation" IMG_M3B_0002.jpg image/jpeg \
    "m3b-acceptance-${run_stamp}-p2-twin" "2026-02-02T12:00:00Z" twin)"
other_id="$(upload_fixture \
    "$face_portrait_other" IMG_M3B_0003.jpg image/jpeg \
    "m3b-acceptance-${run_stamp}-p3-other" "2026-03-03T12:00:00Z" other)"
other_second_id="$(upload_fixture \
    "$face_portrait_other_second" IMG_M3B_0004.jpg image/jpeg \
    "m3b-acceptance-${run_stamp}-p4-other-second" "2026-04-04T12:00:00Z" other-second)"
wait_asset_ready "$portrait_id" portrait
wait_asset_ready "$twin_id" orientation-twin
wait_asset_ready "$other_id" other-portrait
wait_asset_ready "$other_second_id" second-other-portrait

echo "==> the recognition must land four embedded faces in two clusters (up to ${ml_timeout_seconds}s)"
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
    if [ "$faces" -ge 4 ] && [ "$embedded" -ge 4 ] \
        && [ "$clusters" -ge 2 ] && [ "$clustered" -ge 4 ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the faces did not land within ${ml_timeout_seconds}s: faces=$faces embedded=$embedded clusters=$clusters clustered=$clustered"
    fi
    sleep 3
done
echo "    faces=$faces (embedded $embedded), face_clusters=$clusters (clustered faces $clustered)"

echo "==> the cluster structure: two clusters, the same-person pair co-clustered, the cross-person pairs separated"
clusterA1="$(face_cluster_of "$portrait_id")"
clusterA2="$(face_cluster_of "$twin_id")"
clusterB1="$(face_cluster_of "$other_id")"
clusterB2="$(face_cluster_of "$other_second_id")"
[ -n "$clusterA1" ] || fail "the portrait's face carries no cluster id (cluster structure assert)"
[ -n "$clusterA2" ] || fail "the orientation twin's face carries no cluster id (cluster structure assert)"
[ -n "$clusterB1" ] || fail "the other portrait's face carries no cluster id (cluster structure assert)"
[ -n "$clusterB2" ] || fail "the second other portrait's face carries no cluster id (cluster structure assert)"
[ "$clusterA1" = "$clusterA2" ] \
    || fail "the same-person faces sit in different clusters ($clusterA1 vs $clusterA2 -- co-clustering assert)"
[ "$clusterB1" = "$clusterB2" ] \
    || fail "the different person's two faces sit in different clusters ($clusterB1 vs $clusterB2 -- co-clustering assert)"
[ "$clusterA1" != "$clusterB1" ] \
    || fail "the cross-person faces share one cluster ($clusterA1) -- separation assert"
cluster_count="$(psql_query 'SELECT count(*) FROM face_clusters')"
[ "$cluster_count" = "2" ] \
    || fail "the library holds $cluster_count face clusters, want exactly 2 (the seeded pairs)"
clusterA="$clusterA1"
clusterB="$clusterB1"
facesA="$(psql_query "SELECT count(*) FROM faces WHERE cluster_id = '$clusterA'")"
facesB="$(psql_query "SELECT count(*) FROM faces WHERE cluster_id = '$clusterB'")"
[ "$facesA" = "2" ] || fail "cluster A holds $facesA faces, want 2 (cluster structure assert)"
[ "$facesB" = "2" ] || fail "cluster B holds $facesB faces, want 2 (cluster structure assert)"
echo "    two clusters of two faces each: $clusterA (person A), $clusterB (person B)"

echo "==> the embedding evidence: every pairwise cosine distance (0 = identical embeddings)"
echo "    same person A (IMG_M3B_0001.jpg x IMG_M3B_0002.jpg): $(pairwise_distance "$portrait_id" "$twin_id")"
echo "    same person B (IMG_M3B_0003.jpg x IMG_M3B_0004.jpg): $(pairwise_distance "$other_id" "$other_second_id")"
echo "    cross person   (IMG_M3B_0001.jpg x IMG_M3B_0003.jpg): $(pairwise_distance "$portrait_id" "$other_id")"
echo "    cross person   (IMG_M3B_0001.jpg x IMG_M3B_0004.jpg): $(pairwise_distance "$portrait_id" "$other_second_id")"
echo "    cross person   (IMG_M3B_0002.jpg x IMG_M3B_0003.jpg): $(pairwise_distance "$twin_id" "$other_id")"
echo "    cross person   (IMG_M3B_0002.jpg x IMG_M3B_0004.jpg): $(pairwise_distance "$twin_id" "$other_second_id")"

echo "==> GET /api/v1/people/unnamed must list both clusters"
status="$(curl -sS -o "$work_dir/unnamed_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed returned status $status, want 200: $(cat "$work_dir/unnamed_body")"
unnamed_items="$(grep -o '"clusterId"' "$work_dir/unnamed_body" | wc -l | tr -d ' ' || true)"
[ "$unnamed_items" = "2" ] \
    || fail "the unnamed region lists $unnamed_items clusters, want 2: $(cat "$work_dir/unnamed_body")"
grep -q "\"$clusterA\"" "$work_dir/unnamed_body" \
    || fail "the unnamed region does not list cluster A ($clusterA): $(cat "$work_dir/unnamed_body")"
grep -q "\"$clusterB\"" "$work_dir/unnamed_body" \
    || fail "the unnamed region does not list cluster B ($clusterB): $(cat "$work_dir/unnamed_body")"
echo "    the unnamed region lists both clusters"

echo "==> PUT /api/v1/people/{clusterA}/label must name person A Alpha"
status="$(curl -sS -o "$work_dir/label_body" -b "$cookie_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"name":"Alpha"}' \
    "$base_url/api/v1/people/$clusterA/label")" \
    || fail "PUT /api/v1/people/$clusterA/label request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/people/$clusterA/label returned status $status, want 200: $(cat "$work_dir/label_body")"
grep -q '"name":"Alpha"' "$work_dir/label_body" \
    || fail "the label response does not carry the name Alpha: $(cat "$work_dir/label_body")"
grep -q '"isHidden":false' "$work_dir/label_body" \
    || fail "the fresh label is not unhidden: $(cat "$work_dir/label_body")"
echo "    Alpha is named"

echo "==> the named region gains Alpha while the unnamed region keeps only cluster B"
status="$(curl -sS -o "$work_dir/people_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people returned status $status, want 200: $(cat "$work_dir/people_body")"
grep -q "\"$clusterA\"" "$work_dir/people_body" \
    || fail "the named region does not list Alpha's cluster: $(cat "$work_dir/people_body")"
grep -q '"name":"Alpha"' "$work_dir/people_body" \
    || fail "the named region does not carry Alpha's name: $(cat "$work_dir/people_body")"
grep -q '"faceCount":2' "$work_dir/people_body" \
    || fail "Alpha's honest face count is missing: $(cat "$work_dir/people_body")"
grep -q "\"$clusterB\"" "$work_dir/people_body" \
    && fail "cluster B is listed as named before its label exists: $(cat "$work_dir/people_body")" || true
status="$(curl -sS -o "$work_dir/unnamed_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed returned status $status, want 200: $(cat "$work_dir/unnamed_body")"
unnamed_items="$(grep -o '"clusterId"' "$work_dir/unnamed_body" | wc -l | tr -d ' ' || true)"
[ "$unnamed_items" = "1" ] \
    || fail "the unnamed region lists $unnamed_items clusters after naming Alpha, want 1: $(cat "$work_dir/unnamed_body")"
grep -q "\"$clusterB\"" "$work_dir/unnamed_body" \
    || fail "the unnamed region does not keep cluster B: $(cat "$work_dir/unnamed_body")"
grep -q "\"$clusterA\"" "$work_dir/unnamed_body" \
    && fail "the named Alpha still sits in the unnamed region: $(cat "$work_dir/unnamed_body")" || true
echo "    the regions split honestly: named Alpha, unnamed cluster B"

echo "==> PUT label must name person B Beta, then POST merge folds Beta into Alpha"
status="$(curl -sS -o "$work_dir/label_body" -b "$cookie_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"name":"Beta"}' \
    "$base_url/api/v1/people/$clusterB/label")" \
    || fail "PUT /api/v1/people/$clusterB/label request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/people/$clusterB/label returned status $status, want 200: $(cat "$work_dir/label_body")"
grep -q '"name":"Beta"' "$work_dir/label_body" \
    || fail "the label response does not carry the name Beta: $(cat "$work_dir/label_body")"
echo "    Beta is named"

status="$(curl -sS -o "$work_dir/merge_body" -b "$cookie_jar" -w '%{http_code}' \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"sourceClusterIds\":[\"$clusterB\"],\"targetClusterId\":\"$clusterA\"}" \
    "$base_url/api/v1/people/merge")" \
    || fail "POST /api/v1/people/merge request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/people/merge returned status $status, want 200: $(cat "$work_dir/merge_body")"
grep -q "\"survivingClusterId\":\"$clusterA\"" "$work_dir/merge_body" \
    || fail "the merge response's surviving cluster is not Alpha: $(cat "$work_dir/merge_body")"
grep -q "\"mergedClusterIds\":\[\"$clusterB\"\]" "$work_dir/merge_body" \
    || fail "the merge response's merged ids do not name exactly Beta's cluster: $(cat "$work_dir/merge_body")"
echo "    merge committed: surviving $clusterA, merged [$clusterB]"

echo "==> the merged person carries all four faces, Beta's cluster and label died, Alpha's label survives"
status="$(curl -sS -o "$work_dir/people_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people returned status $status, want 200: $(cat "$work_dir/people_body")"
grep -q '"faceCount":4' "$work_dir/people_body" \
    || fail "the merged Alpha does not carry four faces: $(cat "$work_dir/people_body")"
grep -q "\"$clusterA\"" "$work_dir/people_body" \
    || fail "the named region does not list the merged Alpha: $(cat "$work_dir/people_body")"
status="$(curl -sS -o "$work_dir/unnamed_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed returned status $status, want 200: $(cat "$work_dir/unnamed_body")"
grep -q '"items":\[\]' "$work_dir/unnamed_body" \
    || fail "the unnamed region is not empty after the merge: $(cat "$work_dir/unnamed_body")"
beta_rows="$(psql_query "SELECT count(*) FROM face_clusters WHERE id = '$clusterB'")"
[ "$beta_rows" = "0" ] \
    || fail "Beta's cluster row survived the merge ($beta_rows rows) -- dissolution assert"
beta_labels="$(psql_query "SELECT count(*) FROM user_face_labels WHERE cluster_id = '$clusterB'")"
[ "$beta_labels" = "0" ] \
    || fail "Beta's label row survived the merge ($beta_labels rows) -- the label-death assert"
alpha_labels="$(psql_query "SELECT count(*) FROM user_face_labels WHERE cluster_id = '$clusterA'")"
[ "$alpha_labels" = "1" ] \
    || fail "Alpha's label did not survive the merge ($alpha_labels rows) -- the survival assert"
alpha_name="$(psql_query "SELECT name FROM user_face_labels WHERE cluster_id = '$clusterA'")"
[ "$alpha_name" = "Alpha" ] \
    || fail "Alpha's surviving label carries the name '$alpha_name', want Alpha"
cluster_rows="$(psql_query 'SELECT count(*) FROM face_clusters')"
[ "$cluster_rows" = "1" ] \
    || fail "the library holds $cluster_rows clusters after the merge, want 1 (the survivor)"
merged_faces="$(psql_query "SELECT count(*) FROM faces WHERE cluster_id = '$clusterA'")"
[ "$merged_faces" = "4" ] \
    || fail "the surviving cluster holds $merged_faces faces, want 4 (the merged set)"
echo "    Alpha carries 4 faces with its label; Beta's cluster and label are gone"

echo "==> PUT /api/v1/people/{clusterA}/hidden must hide Alpha, who STAYS listed"
status="$(curl -sS -o "$work_dir/hidden_body" -b "$cookie_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"isHidden":true}' \
    "$base_url/api/v1/people/$clusterA/hidden")" \
    || fail "PUT /api/v1/people/$clusterA/hidden request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/people/$clusterA/hidden returned status $status, want 200: $(cat "$work_dir/hidden_body")"
grep -q '"isHidden":true' "$work_dir/hidden_body" \
    || fail "the hidden response does not carry isHidden true: $(cat "$work_dir/hidden_body")"
grep -q '"name":"Alpha"' "$work_dir/hidden_body" \
    || fail "the hidden response lost the name: $(cat "$work_dir/hidden_body")"
status="$(curl -sS -o "$work_dir/people_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people returned status $status, want 200: $(cat "$work_dir/people_body")"
grep -q "\"$clusterA\"" "$work_dir/people_body" \
    || fail "the hidden Alpha left the named region (hidden people stay listed): $(cat "$work_dir/people_body")"
grep -q '"isHidden":true' "$work_dir/people_body" \
    || fail "the hidden Alpha is listed without the isHidden flag: $(cat "$work_dir/people_body")"
echo "    Alpha is hidden and still listed with isHidden true"

echo "==> deleting the assets empties the cluster until it dissolves and Alpha's label dies"
status="$(curl -sS -o /dev/null -b "$cookie_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/assets/$portrait_id")" \
    || fail "DELETE /api/v1/assets/$portrait_id request failed"
[ "$status" = "204" ] \
    || fail "DELETE /api/v1/assets/$portrait_id returned status $status, want 204"
status="$(curl -sS -o "$work_dir/people_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people returned status $status, want 200: $(cat "$work_dir/people_body")"
grep -q '"faceCount":3' "$work_dir/people_body" \
    || fail "Alpha's face count did not drop to 3 after the first delete: $(cat "$work_dir/people_body")"
grep -q "\"$clusterA\"" "$work_dir/people_body" \
    || fail "Alpha left the named region after a partial delete: $(cat "$work_dir/people_body")"
echo "    one asset deleted: the cluster survives with 3 faces (the Rehome discipline)"
for doomed in "$twin_id" "$other_id" "$other_second_id"; do
    status="$(curl -sS -o /dev/null -b "$cookie_jar" -w '%{http_code}' \
        -X DELETE "$base_url/api/v1/assets/$doomed")" \
        || fail "DELETE /api/v1/assets/$doomed request failed"
    [ "$status" = "204" ] \
        || fail "DELETE /api/v1/assets/$doomed returned status $status, want 204"
done
status="$(curl -sS -o "$work_dir/people_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people")" \
    || fail "GET /api/v1/people request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people returned status $status, want 200: $(cat "$work_dir/people_body")"
grep -q '^\[\]$' "$work_dir/people_body" \
    || fail "the named region is not empty after the cluster dissolved: $(cat "$work_dir/people_body")"
status="$(curl -sS -o "$work_dir/unnamed_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed returned status $status, want 200: $(cat "$work_dir/unnamed_body")"
grep -q '"items":\[\]' "$work_dir/unnamed_body" \
    || fail "the unnamed region is not empty after the cluster dissolved: $(cat "$work_dir/unnamed_body")"
status="$(curl -sS -o "$work_dir/detail_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/$clusterA")" \
    || fail "GET /api/v1/people/$clusterA request failed"
[ "$status" = "404" ] \
    || fail "the dissolved cluster's detail returned status $status, want 404: $(cat "$work_dir/detail_body")"
leftover_faces="$(psql_query 'SELECT count(*) FROM faces')"
[ "$leftover_faces" = "0" ] \
    || fail "$leftover_faces faces survived the asset deletes, want 0"
leftover_clusters="$(psql_query 'SELECT count(*) FROM face_clusters')"
[ "$leftover_clusters" = "0" ] \
    || fail "$leftover_clusters clusters survived the asset deletes, want 0 (dissolution assert)"
leftover_labels="$(psql_query 'SELECT count(*) FROM user_face_labels')"
[ "$leftover_labels" = "0" ] \
    || fail "$leftover_labels label rows survived the cluster's dissolution, want 0 (Alpha's label-death assert)"
echo "    the cluster dissolved: no faces, no clusters, no labels -- Alpha's label died with it"
if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the people phase (see compose logs)"
fi
teardown_stack
echo "    phase 1 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 2: the e2e stack (the people story of the web app).
# ---------------------------------------------------------------------------

echo "==> phase 2/2: the e2e stack (the people story of the web app)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror

echo "==> running the Playwright spec (web/e2e/m3-people.spec.ts)"
if ! (cd web && pnpm e2e e2e/m3-people.spec.ts); then
    echo "==> server logs at the time of the failure:"
    compose logs --no-color --tail 200 lumemor || true
    fail "the people e2e spec failed (see the report above; artifacts under web/test-results/)"
fi
teardown_stack
echo "    phase 2 passed after $(elapsed)s"

echo "==> acceptance flow completed in $(elapsed)s"
echo "M3B ACCEPTANCE PASS"
