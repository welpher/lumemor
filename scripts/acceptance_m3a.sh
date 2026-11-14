#!/usr/bin/env bash
# M3a acceptance: the chained ML milestone acceptance -- the M2d
# chain (which itself runs the M1, M2a, M2b and M2c scripts
# sequentially, each owning its own compose stack and tearing it down
# fully, including the database volume) runs first and must be green
# before the M3a phases start. Then three stack lifetimes of this
# repository's own compose deployment walk the ML story:
#
# Phase 1 (the degraded stack, no models): the stack starts with an
# empty model registry, the first-run admin is created and signed in,
# the face portrait fixture uploads and its asset reaches the ready
# state with the timeline listing it -- while the database carries
# ZERO intelligence rows (no faces, no clusters, no embeddings, no
# OCR text, no job runs). The readiness seam never dispatches ML work
# for an unfilled slot and nothing breaks: the real degraded
# load.
#
# Phase 2 (the real-model full chain): the default model group
# downloads through the production download service (POST
# /api/v1/admin/models/download, the job_runs row polled to
# succeeded), the listing shows all five models active with their
# files present (the CLIP text tower among them), and three fixtures upload -- the face portrait, its
# EXIF-orientation twin of the SAME person, and the text-bearing OCR
# image -- after which the ML products are asserted inside the
# database container (docker compose exec db psql): faces with
# non-null embeddings exist, a face cluster exists, clip_embeddings
# rows exist with non-zero vectors, and ocr_texts carries non-empty
# text. The same-person cluster facts (cluster membership and the
# pair's cosine distance) are printed as the SFace-128d family-scale
# observation for the people chain.
#
# Phase 3 (the e2e stack): a fresh stack whose models ride the SAME
# storage bind mount (./lumemor persists across the script's stack
# lifetimes -- the established compose stack arrangement; the
# already-verified skip means the weights transfer once per mount
# lifetime, and the phase-3 download re-verifies and registers them
# without re-fetching). The script stages two self-made constant
# clip-embedding fixture models into the models dir (the activation
# dialog scenes need a second model in the clip slot), then the
# Playwright spec web/e2e/m3-models.spec.ts walks the admin model
# management story: login, the empty-state CTA, the download's
# progress completing, the task panel's succeeded row, both activation
# dialog branches, and the activation badge.
#
# DOWNLOAD SOURCES (two modes):
#   M3A_SOURCE=upstream (the default; what CI runs) -- the stack
#       downloads from the manifest's pinned upstreams directly; set
#       M3A_HF_MIRROR=host[:port] (e.g. hf-mirror.com) to route the
#       huggingface.co URLs through an HF domain mirror instead.
#   M3A_SOURCE=offline -- no network at all: the Hugging Face URL
#       path structure is reconstructed under
#       tools/models/cache/offline-root/ (symlinks to the local
#       weights cache) and served over TLS by a mirror service
#       of the stack (the dev image's python running
#       scripts/offline_mirror_server.py): the mirror rewrite keeps
#       the https scheme, so the local source must speak TLS, and the
#       self-signed certificate's CA is handed to the server
#       container through SSL_CERT_FILE (Go's crypto/x509 honors it
#       on Linux; the CA file itself rides the ./lumemor bind mount).
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message (the failing assertion's name; the stack-bound
# waits also print the server's compose log tail), and the teardown
# still runs. A passing run prints "M3A ACCEPTANCE PASS" as its final
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
ocr_text_image="$repo_root/server/internal/ml/testdata/ocr-text.png"

# The download source mode (see the header comment).
source_mode="${M3A_SOURCE:-upstream}"
case "$source_mode" in
    upstream | offline) ;;
    *)
        echo "M3A ACCEPTANCE FAIL: M3A_SOURCE must be 'upstream' or 'offline', got '$source_mode'" >&2
        exit 1
        ;;
esac
# An upstream-mode HF domain mirror host (empty = the pinned upstreams
# directly).
hf_mirror="${M3A_HF_MIRROR:-}"

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
    echo "M3A ACCEPTANCE FAIL: $*" >&2
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

# file_sha256 digests a file on macOS (shasum) and Linux (sha256sum)
# alike.
file_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
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
# Generated by scripts/acceptance_m3a.sh (offline download mode): the
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
# The suites never reach the outside world: the release check is off
# at the deployment level for every acceptance stack (its own request
# would be the only outbound one the stories do not stub).
export LUMEMOR_UPDATE_CHECK_DISABLED=1
    if [ -n "$hf_mirror" ]; then
        echo "==> download source: upstream through the HF mirror $hf_mirror"
    else
        echo "==> download source: the pinned upstreams directly"
    fi
fi

# ---------------------------------------------------------------------------
# Shared stack and API helpers (the m2c patterns).
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

# reset_bind_mount_ownership returns the storage bind mount's tree to
# the HOST user. prepare_bind_mount leaves ./lumemor owned by the
# image's fixed uid 10001 (the container contract); a later phase that
# stages files from the host side (the phase 3 fixture models) must
# write into that tree again first -- on a permissive runtime (the
# macOS desktop mapping) the write happens to work, but the uid
# discipline of a Linux CI runner denies it. The reverse chown rides
# the same image-as-root trick, so it needs the built image (a cached
# no-op after any phase's prepare_bind_mount).
reset_bind_mount_ownership() {
    if [ ! -d lumemor ]; then
        return 0
    fi
    compose run --rm --no-deps --entrypoint /bin/sh -u 0 \
        -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" lumemor \
        -c 'chown -R "$HOST_UID:$HOST_GID" /var/lib/lumemor' \
        || fail "returning ./lumemor to the host user failed"
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
    local admin_email="admin-${run_stamp}-${label}@m3a-acceptance.invalid"
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
        -d "{\"name\":\"M3a Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
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

# ---------------------------------------------------------------------------
# The chained regressions: the M2d acceptance (which chains the M1,
# M2a, M2b and M2c acceptances and drives the M2 web story itself).
# ---------------------------------------------------------------------------

echo "==> running the prior acceptances (the M2d chain owns its stacks)"
bash scripts/acceptance_m2d.sh \
    || fail "scripts/acceptance_m2d.sh failed (see its output above)"
echo "    acceptance_m2d (and its chained M1/M2a/M2b/M2c) passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 1: the degraded stack (no models) -- the degraded load.
# ---------------------------------------------------------------------------

echo "==> phase 1/3: the degraded stack (no models)"
reset_stack_state
prepare_bind_mount
start_stack
create_admin_and_login phase1

echo "==> GET /api/v1/admin/models must return the honest empty state"
status="$(curl -sS -o "$work_dir/models_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/admin/models")" \
    || fail "GET /api/v1/admin/models request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/admin/models returned status $status, want 200: $(cat "$work_dir/models_body")"
grep -q '"models":\[\]' "$work_dir/models_body" \
    || fail "the models listing is not empty on a fresh stack: $(cat "$work_dir/models_body")"
grep -q '"download":null' "$work_dir/models_body" \
    || fail "the models listing reports a download run on a fresh stack: $(cat "$work_dir/models_body")"
echo "    the models listing is empty and no download run exists"

echo "==> the face portrait uploads and reaches ready with zero ML products"
[ -f "$face_portrait" ] || fail "fixture missing: $face_portrait"
portrait1_id="$(upload_fixture \
    "$face_portrait" IMG_M3A_0001.jpg image/jpeg \
    "m3a-acceptance-${run_stamp}-p1" "2026-01-01T12:00:00Z" portrait-p1)"
wait_asset_ready "$portrait1_id" portrait

echo "==> the timeline lists the ready asset"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline returned status $status, want 200: $(cat "$work_dir/timeline_body")"
grep -q "\"$portrait1_id\"" "$work_dir/timeline_body" \
    || fail "the timeline does not list the ready portrait asset: $(cat "$work_dir/timeline_body")"
echo "    the timeline holds the portrait"

echo "==> the degraded stack carries zero intelligence rows"
degraded_faces="$(psql_query 'SELECT count(*) FROM faces')"
[ "$degraded_faces" = "0" ] \
    || fail "the degraded stack holds $degraded_faces faces, want 0 (no model, no ML dispatch)"
degraded_clusters="$(psql_query 'SELECT count(*) FROM face_clusters')"
[ "$degraded_clusters" = "0" ] \
    || fail "the degraded stack holds $degraded_clusters face clusters, want 0"
degraded_clip="$(psql_query 'SELECT count(*) FROM clip_embeddings')"
[ "$degraded_clip" = "0" ] \
    || fail "the degraded stack holds $degraded_clip clip embeddings, want 0"
degraded_ocr="$(psql_query 'SELECT count(*) FROM ocr_texts')"
[ "$degraded_ocr" = "0" ] \
    || fail "the degraded stack holds $degraded_ocr OCR texts, want 0"
degraded_runs="$(psql_query 'SELECT count(*) FROM job_runs')"
[ "$degraded_runs" = "0" ] \
    || fail "the degraded stack holds $degraded_runs job runs, want 0"
if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the degraded phase (see compose logs)"
fi
echo "    no faces, no clusters, no embeddings, no OCR rows, no job runs -- readiness untouched"
teardown_stack
echo "    phase 1 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 2: the real-model full chain.
# ---------------------------------------------------------------------------

echo "==> phase 2/3: the real-model full chain (source: $source_mode)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror
create_admin_and_login phase2

echo "==> POST /api/v1/admin/models/download must return 202 with a job run id"
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

echo "==> GET /api/v1/admin/models must list the five default models active"
status="$(curl -sS -o "$work_dir/models_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/admin/models")" \
    || fail "GET /api/v1/admin/models request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/admin/models returned status $status, want 200: $(cat "$work_dir/models_body")"
active_count="$(grep -o '"isActive":true' "$work_dir/models_body" | wc -l | tr -d ' ' || true)"
[ "$active_count" = "5" ] \
    || fail "the model listing shows $active_count active models, want 5: $(cat "$work_dir/models_body")"
files_present="$(grep -o '"filesPresent":true' "$work_dir/models_body" | wc -l | tr -d ' ' || true)"
[ "$files_present" = "5" ] \
    || fail "the model listing shows $files_present models with files present, want 5: $(cat "$work_dir/models_body")"
for model_id in yunet-2023mar sface-2021dec cnclip-vit-b16-vision cnclip-vit-b16-text ppocr-v4-mobile-ch; do
    grep -q "\"id\":\"$model_id\"" "$work_dir/models_body" \
        || fail "the model listing does not carry $model_id: $(cat "$work_dir/models_body")"
done
echo "    five models active with their files present (the text tower among them)"

echo "==> uploading the portrait, its same-person orientation twin, and the text image"
[ -f "$face_portrait" ] || fail "fixture missing: $face_portrait"
[ -f "$face_portrait_orientation" ] || fail "fixture missing: $face_portrait_orientation"
[ -f "$ocr_text_image" ] || fail "fixture missing: $ocr_text_image"
portrait2_id="$(upload_fixture \
    "$face_portrait" IMG_M3A_0002.jpg image/jpeg \
    "m3a-acceptance-${run_stamp}-p2-portrait" "2026-02-02T12:00:00Z" portrait-p2)"
twin_id="$(upload_fixture \
    "$face_portrait_orientation" IMG_M3A_0003.jpg image/jpeg \
    "m3a-acceptance-${run_stamp}-p2-twin" "2026-03-03T12:00:00Z" twin)"
ocr_id="$(upload_fixture \
    "$ocr_text_image" IMG_M3A_0004.png image/png \
    "m3a-acceptance-${run_stamp}-p2-text" "2026-04-04T12:00:00Z" ocr-text)"
wait_asset_ready "$portrait2_id" portrait
wait_asset_ready "$twin_id" orientation-twin
wait_asset_ready "$ocr_id" text-image

echo "==> the timeline lists all three assets"
status="$(curl -sS -o "$work_dir/timeline_body" -b "$cookie_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline returned status $status, want 200: $(cat "$work_dir/timeline_body")"
for wanted_id in "$portrait2_id" "$twin_id" "$ocr_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/timeline_body" \
        || fail "the timeline does not list the asset $wanted_id: $(cat "$work_dir/timeline_body")"
done
echo "    the timeline holds the portrait, the twin and the text image"

echo "==> the ML products must land in the database (up to ${ml_timeout_seconds}s)"
ml_facts() {
    printf '%s %s %s %s %s %s %s\n' \
        "$(psql_query 'SELECT count(*) FROM faces')" \
        "$(psql_query 'SELECT count(embedding) FROM faces')" \
        "$(psql_query 'SELECT count(*) FROM face_clusters')" \
        "$(psql_query 'SELECT count(*) FROM faces WHERE cluster_id IS NOT NULL')" \
        "$(psql_query 'SELECT count(*) FROM clip_embeddings')" \
        "$(psql_query "SELECT count(*) FROM clip_embeddings WHERE embedding::text ~ '[1-9]'")" \
        "$(psql_query 'SELECT count(*) FROM ocr_texts WHERE length(text) > 0')"
}
deadline="$(( $(date +%s) + ml_timeout_seconds ))"
while :; do
    read -r faces embedded clusters clustered clip clipnz ocr <<<"$(ml_facts)"
    if [ "$faces" -ge 1 ] && [ "$embedded" -ge 1 ] \
        && [ "$clusters" -ge 1 ] && [ "$clustered" -ge 1 ] \
        && [ "$clip" -ge 1 ] && [ "$clipnz" -eq "$clip" ] \
        && [ "$ocr" -ge 1 ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the ML products did not land within ${ml_timeout_seconds}s: faces=$faces embedded=$embedded clusters=$clusters clustered=$clustered clip=$clip (non-zero $clipnz) ocr=$ocr"
    fi
    sleep 3
done
echo "    faces=$faces (embedded $embedded), face_clusters=$clusters (clustered faces $clustered), clip_embeddings=$clip (all non-zero), ocr_texts with text=$ocr"

echo "==> the cluster-quality observation (the same-person faces)"
echo "    cluster membership:"
psql_query "SELECT COALESCE(cluster_id::text, 'noise') || ': ' || count(*) || ' face(s)' FROM faces GROUP BY cluster_id ORDER BY 1" | sed 's/^/        /'
echo "    the same-person pair's cosine distance (0 = identical embeddings):"
psql_query "SELECT round((a.embedding <=> b.embedding)::numeric, 6) FROM faces a JOIN faces b ON a.id < b.id ORDER BY 1" | sed 's/^/        /'
echo "    the OCR text of the text-bearing fixture:"
psql_query "SELECT text FROM ocr_texts WHERE length(text) > 0 ORDER BY created_at" | sed 's/^/        /'
echo "    the clip embedding dimensions:"
psql_query "SELECT count(*) || ' rows x ' || min(array_length(string_to_array(embedding::text, ','), 1)) || ' dims' FROM clip_embeddings" | sed 's/^/        /'
if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the ML phase (see compose logs)"
fi
teardown_stack
echo "    phase 2 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 3: the e2e stack (the model management story).
# ---------------------------------------------------------------------------

# stage_fixture_model copies one self-made clip-embedding fixture into
# the models dir with a manifest that pins its digest: the activation
# dialog scenes need a second (and a third, cross-dimension) model in
# the clip slot, and scanning registers exactly these.
stage_fixture_model() {
    local id="$1" dim="$2" source_fixture="$3"
    local dir="lumemor/config/models/$id"
    mkdir -p "$dir"
    cp "$source_fixture" "$dir/"
    local name sha bytes
    name="$(basename "$source_fixture")"
    sha="$(file_sha256 "$source_fixture")"
    bytes="$(wc -c <"$source_fixture" | tr -d ' ')"
    cat >"$dir/manifest.json" <<EOF
{
  "id": "$id",
  "slot": "clip-embed",
  "version": "fixture",
  "license": "none",
  "sourceUrl": "",
  "fallbackUrl": null,
  "notice": "Self-made constant-embedding fixture ($name, see server/internal/ml/testdata/README.md): no upstream artifact. Staged by scripts/acceptance_m3a.sh for the activation dialog scenes.",
  "inputSize": { "width": 224, "height": 224 },
  "dim": $dim,
  "params": {},
  "files": [
    { "name": "$name", "sha256": "$sha", "bytes": $bytes }
  ]
}
EOF
}

echo "==> phase 3/3: the e2e stack (the model management story)"
reset_stack_state
# Phases 1-2 left ./lumemor chowned to the image uid; the fixture
# staging below writes into it as the host user, so the tree comes
# back to us first (prepare_bind_mount re-applies the container uid
# after the staging).
reset_bind_mount_ownership
echo "==> staging the self-made clip fixture models for the dialog scenes"
stage_fixture_model clip-fixture-512 512 "$repo_root/server/internal/ml/testdata/const512.onnx"
stage_fixture_model clip-fixture-768 768 "$repo_root/server/internal/ml/testdata/const768.onnx"
prepare_bind_mount
start_stack
probe_offline_mirror

echo "==> running the Playwright spec (web/e2e/m3-models.spec.ts)"
if ! (cd web && pnpm e2e e2e/m3-models.spec.ts); then
    echo "==> server logs at the time of the failure:"
    compose logs --no-color --tail 200 lumemor || true
    fail "the models e2e spec failed (see the report above; artifacts under web/test-results/)"
fi
teardown_stack
echo "    phase 3 passed after $(elapsed)s"

echo "==> acceptance flow completed in $(elapsed)s"
echo "M3A ACCEPTANCE PASS"
