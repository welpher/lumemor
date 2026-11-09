#!/usr/bin/env bash
# M3d acceptance: the search and places milestone's end-to-end
# acceptance -- the chained M3c acceptance (which itself chains the
# M3b and M3a chains and through them the whole M1/M2 line, every
# script owning its own compose stack and tearing it down fully,
# including the database volume) runs first and must be green before
# the M3d phases start. Then two stack lifetimes of this repository's
# own compose deployment walk the search milestone's story -- the
# M3 milestone's own finale line, "a search for 海边 answers photos":
#
# Phase 1 (the search and places story over the API, two accounts):
# the stack starts fresh, the first-run admin is created and signed
# in, a MEMBER account is provisioned through the admin API and signs
# in with its own session, and the default model group downloads
# through the admin API -- the FIVE-model group now, whose newest
# member is the CLIP TEXT TOWER (the semantic arm's query encoder):
# the listing asserts it active, and an explicit PATCH through the
# models API re-asserts the activation the download's empty-slot rule
# already performed. The member then owns the whole library, six
# seeded photos: the Beijing GPS photo (the golden derivative whose
# EXIF coordinates resolve to Beijing), the Chinese OCR photo (the
# rendered text image whose extracted text carries 海边), the
# same-person face pair (the semantically-clustered subject set), the
# wildcard decoy (a filename the live-wildcard bug would glob but the
# literal query does not match), and the private photo (a
# filename-only text hit with a place of its own).
#
# The search semantics are then asserted arm by arm:
#   - The FTS arm: a Chinese keyword (海边) hits the OCR text; an
#     English substring (M3D_FACE_A) hits a filename; the wildcards
#     stay literal (50% matches the literal "50%" filename text,
#     50_ matches nothing -- asserted through the query mode's facet
#     domain, the one observable a live wildcard would change).
#   - The semantic arm: a descriptive query (女宇航员, "female
#     astronaut") matching no text at all answers the embedded library
#     with the face pair leading -- the text tower's own ranking.
#   - The merge: the 海边 query hits the OCR photo through BOTH arms
#     (the OCR text and its embedding), and the fused page answers
#     every visible asset exactly once, the both-arms photo first,
#     byte-identically on the repeat (determinism), and the
#     mode-tagged cursor walks the same order in pages.
#   - The filters and facets: the person filter (the member's
#     labeled face cluster), the place filter (Beijing's GeoNames
#     id), and the seven facet dimensions' honest counts.
#   - The tags domain: the idempotent create (201 then 200 with the
#     same id), the batch assign's honest skips (a bogus id, then the
#     already-tagged pair), the tag-as-vocabulary FTS arm (a query
#     hitting the tag NAME), the detail's caller-scoped tags row, and
#     the tag filter.
#   - The visibility (B4 and the union predicate): the member opens a
#     directory album over the source fixture (the fixture's four
#     photos are byte-identical to four of the member's uploads, so
#     the scan's owner+checksum dedup answers each file with the
#     honest skip: the EXISTING rows gain the album's membership and
#     the shared surface is exactly those four, no second assets)
#     and grants the ADMIN the view at create (the audience rides
#     the POST), and the sharee's search NEVER answers the member's
#     private photo from EITHER arm -- its filename is the only text
#     hit of the visibility query and it would rank first if either
#     arm leaked -- while the sharee's tag facet stays empty and the
#     places surface drops the private photo's city. The member's
#     own search answers it first.
#   - The places domain: the GPS photo resolves to Beijing (en
#     "Beijing", zh-Hans 北京 -- the locale join), the private photo
#     to its own city, and GET /places counts caller-visible.
#
# Phase 2 (the e2e stack): a fresh stack whose models ride the SAME
# storage bind mount; the Playwright spec web/e2e/m3-search.spec.ts
# walks the search story of the web app through a real browser -- the
# topbar's search box navigating to /search, the flat rank-ordered
# wall of a Chinese query (no month headers: grouping would lie about
# the rank order), the facet chips' toggle-and-re-execute, the browse
# mode's month-grouped wall, the places page's city grid with its
# deep-link into the filtered browse, and the timeline's batch add-tag
# flow. The spec bootstraps itself (the first-run admin and the
# library through the APIs, the model group's download through the
# admin API), exactly the way every prior e2e spec owns its stack's
# bootstrap.
#
# DOWNLOAD SOURCES (two modes, the m3c script's knobs -- the M3d
# script accepts the same M3C_SOURCE / M3C_HF_MIRROR variables plus
# M3D_SOURCE / M3D_HF_MIRROR aliases that win when set, so a CI
# workflow swapping the chain head needs no other change):
#   M3D_SOURCE=upstream (the default; what CI runs) -- the stack
#       downloads from the manifest's pinned upstreams directly; set
#       M3D_HF_MIRROR (or M3C_HF_MIRROR) to host[:port] (e.g.
#       hf-mirror.com) to route the huggingface.co URLs through an
#       HF domain mirror instead.
#   M3D_SOURCE=offline -- no network at all: the Hugging Face URL
#       path structure is reconstructed under
#       tools/models/cache/offline-root/ (symlinks to the Task 1
#       local weights cache, the M3d text tower and its vocabulary
#       included) and served over TLS by a mirror service of the
#       stack (the dev image's python running
#       scripts/offline_mirror_server.py): the mirror rewrite keeps
#       the https scheme, so the local source must speak TLS, and the
#       self-signed certificate's CA is handed to the server
#       container through SSL_CERT_FILE (Go's crypto/x509 honors it
#       on Linux; the CA file itself rides the ./lumemor bind mount).
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message (the failing assertion's name; the stack-bound
# waits also print the server's compose log tail), and the teardown
# still runs. A passing run prints "M3D ACCEPTANCE PASS" as its final
# line.
#
# Requirements: docker with the compose plugin, curl, python3 (the
# GPS fixture's checksum probe below only prints), and (for the
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
rescan_timeout_seconds=180

admin_password="correct-horse-battery"
member_password="correct-horse-battery"

# The story's fixtures (all committed to the repository):
#   gps-beijing.jpg -- the golden derivative whose EXIF GPS is
#       Tiananmen Square (resolves to Beijing, the zh-Hans-named
#       city) and whose DateTimeOriginal reads 2026:06:20 +05:30 (a
#       deterministic taken-at distinct from every other photo's).
#   ocr-zh-text.png -- the rendered Chinese text image (the lines
#       "杭州西湖景区" and "海边门票") whose extracted OCR text carries
#       the milestone's finale keyword.
#   face-portrait.jpg / face-portrait-orientation6.jpg -- the
#       same-person face pair: the people facet's cluster and the
#       semantic arm's subject set.
#   face-portrait-other.jpg -- the cross-person portrait: one lone
#       face, cluster-less noise (the clusterer's minPts rule), and
#       the wildcard-escaping decoy through its filename.
#   golden.jpg -- the private photo: a filename-only text hit whose
#       GPS resolves to its own (zh-less) city, Paris 16 Passy.
gps_photo="$repo_root/server/internal/jobs/testdata/gps-beijing.jpg"
ocr_photo="$repo_root/server/internal/ml/testdata/ocr-zh-text.png"
face_portrait="$repo_root/server/internal/ml/testdata/face-portrait.jpg"
face_portrait_orientation="$repo_root/server/internal/ml/testdata/face-portrait-orientation6.jpg"
face_portrait_other="$repo_root/server/internal/ml/testdata/face-portrait-other.jpg"
private_photo="$repo_root/server/internal/jobs/testdata/golden.jpg"

# The E4 source fixture: the repository's compose file binds the e2e
# fixture tree (web/e2e/fixtures/m3-source, see its README) into the
# container read-only at /mnt/e2e-source, and the story's member
# opens trip-2026 under it as the album's source -- the four
# byte-identical copies of the member's own uploads above, so the
# scan's owner+checksum dedup is the story's own arm (the honest
# skip: the existing rows gain the membership, never a second
# asset). The read-only bind is the zero-copy pin held by
# construction.
mount_root="/mnt/e2e-source"
source_dir="/mnt/e2e-source/trip-2026"

# The expected places of the two GPS-bearing photos (the pinned
# GeoNames bake): Beijing (zh-Hans 北京) and Paris 16 Passy (no zh
# name -- the canonical fallback).
beijing_place_id=1816670
beijing_name_en="Beijing"
beijing_name_zh="北京"
paris_name="Paris 16 Passy"

# The story's queries: the finale keyword (the OCR text), the English
# filename substring, the two wildcard literals, the visibility
# query (the private photo's filename text -- the only text hit), and
# the descriptive semantic query (matching no text at all -- "female
# astronaut", the face pair's subject). The query strings below
# carry the percent-encoded forms (the queries are constants, so the
# encodings are too).
query_seaside="海边"
query_seaside_encoded="%E6%B5%B7%E8%BE%B9"
query_filename="M3D_FACE_A"
query_percent="50%"
query_percent_encoded="50%25"
query_underscore="50_"
query_visibility="人脸"
query_visibility_encoded="%E4%BA%BA%E8%84%B8"
query_semantic="女宇航员"
query_semantic_encoded="%E5%A5%B3%E5%AE%87%E8%88%AA%E5%91%98"
query_tag_text_encoded="%E5%90%88%E9%9B%86"

# The download source mode (see the header comment): the M3D_*
# aliases win over the M3C_* variables so this script stays a
# drop-in chain head -- CI's plain invocation (no variables set)
# runs the upstream default, and the mode is exported back as
# M3C_SOURCE (and M3B_SOURCE/M3A_SOURCE through the chained scripts)
# so the whole chain runs the SAME mode.
source_mode="${M3D_SOURCE:-${M3C_SOURCE:-${M3B_SOURCE:-upstream}}}"
case "$source_mode" in
    upstream | offline) ;;
    *)
        echo "M3D ACCEPTANCE FAIL: M3D_SOURCE must be 'upstream' or 'offline', got '$source_mode'" >&2
        exit 1
        ;;
esac
# An upstream-mode HF domain mirror host (empty = the pinned upstreams
# directly).
hf_mirror="${M3D_HF_MIRROR:-${M3C_HF_MIRROR:-${M3B_HF_MIRROR:-}}}"
export M3C_SOURCE="$source_mode"
export M3C_HF_MIRROR="$hf_mirror"

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
    echo "M3D ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# item_ids prints the ids of an items[] response body in document
# order -- the timeline renders its rows as objects whose "id" key is
# the first field, and the compact encoder keeps document order. The
# quoted "id" pattern cannot match "ownerId" or any other camel-case
# field: the leading quote anchors it against the key's start. The
# body is the file argument, or stdin when piped (search_item_ids).
item_ids() {
    # /dev/stdin: the pipe's reader when no file argument arrives
    # (BSD grep has no "-" convention for stdin).
    grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' "${1:-/dev/stdin}" \
        | sed 's/^"id"[[:space:]]*:[[:space:]]*"\([^"]*\)"$/\1/'
}

# search_item_ids prints the ids of ONE SEARCH response's items in
# the page's order. The search response marshals its keys
# alphabetically ("facets" precedes "items" and "nextCursor" follows
# them), and the facets block's people/cameras/albums/tags entries
# own "id" fields of their own: the page's order must never read
# them. The items array carries no nested arrays (the timeline item
# shape is flat), so the prefix cut at '"items":[' plus the tail cut
# at the nextCursor (or the closing brace) isolates exactly the
# items, in the page's own order.
search_item_ids() {
    sed -e 's/^.*"items":\[//' \
        -e 's/\],"nextCursor":"[^"]*"}$//' \
        -e 's/\]}$//' "$1" | item_ids
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
# Generated by scripts/acceptance_m3d.sh (offline download mode): the
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
# Shared stack and API helpers (the m3c patterns).
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
    local admin_email="admin-${run_stamp}-${label}@m3d-acceptance.invalid"
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
        -d "{\"name\":\"M3d Acceptance\",\"email\":\"$admin_email\",\"password\":\"$admin_password\"}" \
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
}

# create_member_and_login provisions the member account through the
# admin API and signs it in with its own cookie jar. The member's
# user id lands in the member_user_id global (the function's stdout
# carries only the progress lines the log wants).
create_member_and_login() {
    local label="$1" email="$2" name="$3"
    local member_email="${email}-${run_stamp}@m3d-acceptance.invalid"

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
    echo "==> the member's POST /api/v1/auth/login must return 200 and set the session cookie"
    status="$(curl -sS -o "$work_dir/member_login_body" -c "$cookie_jar" -w '%{http_code}' \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"$member_email\",\"password\":\"$member_password\"}" \
        "$base_url/api/v1/auth/login")" \
        || fail "the member's POST /api/v1/auth/login request failed"
    [ "$status" = "200" ] \
        || fail "the member's POST /api/v1/auth/login returned status $status, want 200: $(cat "$work_dir/member_login_body")"
    grep -q 'lumemor_session' "$cookie_jar" \
        || fail "the member's login did not set the lumemor_session cookie: $(cat "$work_dir/member_login_body")"
    echo "    the member's login accepted, session cookie captured"
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
    local timeout_s=180
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

# search_get runs one GET /api/v1/search and asserts the 200; the
# body lands in $work_dir/search_body for the caller's greps.
search_get() {
    local jar="$1" query_string="$2" label="$3"
    local status
    status="$(curl -sS -o "$work_dir/search_body" -b "$jar" -w '%{http_code}' \
        "$base_url/api/v1/search?$query_string")" \
        || fail "GET /api/v1/search ($label) request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/search ($label) returned status $status, want 200: $(cat "$work_dir/search_body")"
}

# search_get_raw runs one GET /api/v1/search and prints ONLY the
# status code (the negative arms assert the code, not the body).
search_get_raw() {
    local jar="$1" query_string="$2"
    curl -sS -o "$work_dir/search_raw_body" -b "$jar" -w '%{http_code}' \
        "$base_url/api/v1/search?$query_string" \
        || fail "GET /api/v1/search request failed"
}

# ---------------------------------------------------------------------------
# The chained regressions: the M3c acceptance (which chains the M3b
# chain, which chains the M3a chain, which chains the M2d chain, which
# chains the M1, M2a, M2b and M2c acceptances -- the full M1 to M3c
# serial line, each script owning its own stacks).
# ---------------------------------------------------------------------------

echo "==> running the prior milestone acceptance (the M3c chain owns its stacks)"
bash scripts/acceptance_m3c.sh \
    || fail "scripts/acceptance_m3c.sh failed (see its output above)"
echo "    acceptance_m3c (and its chained M1/M2/M3a/M3b line) passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 1: the search and places story over the API (two accounts, the
# whole search surface: the two arms, the merge, the filters, the
# visibility, the tags and the places).
# ---------------------------------------------------------------------------

echo "==> phase 1/2: the search and places story over the API (source: $source_mode)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror
create_admin_and_login m3d
create_member_and_login member grace "Grace Hopper"
admin_jar="$work_dir/cookies_m3d"
member_jar="$work_dir/cookies_member"
cookie_jar="$admin_jar"
wait_model_download

echo "==> GET /api/v1/admin/models must list the five default models active (the text tower among them)"
status="$(curl -sS -o "$work_dir/models_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/admin/models")" \
    || fail "GET /api/v1/admin/models request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/admin/models returned status $status, want 200: $(cat "$work_dir/models_body")"
active_count="$(grep -o '"isActive":true' "$work_dir/models_body" | wc -l | tr -d ' ' || true)"
[ "$active_count" = "5" ] \
    || fail "the model listing shows $active_count active models, want 5: $(cat "$work_dir/models_body")"
for model_id in yunet-2023mar sface-2021dec cnclip-vit-b16-vision cnclip-vit-b16-text ppocr-v4-mobile-ch; do
    grep -q "\"id\":\"$model_id\"" "$work_dir/models_body" \
        || fail "the model listing does not carry $model_id: $(cat "$work_dir/models_body")"
done
echo "    five models active (the clip-text tower of the semantic arm included)"

echo "==> PATCH /api/v1/admin/models/cnclip-vit-b16-text must answer the activation (the models API face)"
status="$(curl -sS -o "$work_dir/activate_body" -b "$admin_jar" -w '%{http_code}' \
    -X PATCH -H 'Content-Type: application/json' -d '{}' \
    "$base_url/api/v1/admin/models/cnclip-vit-b16-text")" \
    || fail "PATCH /api/v1/admin/models/cnclip-vit-b16-text request failed"
[ "$status" = "200" ] \
    || fail "PATCH /api/v1/admin/models/cnclip-vit-b16-text returned status $status, want 200: $(cat "$work_dir/activate_body")"
grep -q '"activated":{"id":"cnclip-vit-b16-text","slot":"clip_text"}' "$work_dir/activate_body" \
    || fail "the activation response does not name the text tower active in its slot: $(cat "$work_dir/activate_body")"
echo "    the text tower is ACTIVE through the models API (the semantic arm is armed)"

echo "==> the member uploads the seeded library (six photos, the member owns them all)"
for fixture in "$gps_photo" "$ocr_photo" "$face_portrait" "$face_portrait_orientation" "$face_portrait_other" "$private_photo"; do
    [ -f "$fixture" ] || fail "fixture missing: $fixture"
done
gps_id="$(upload_fixture \
    "$gps_photo" IMG_M3D_GPS.jpg image/jpeg \
    "m3d-acceptance-${run_stamp}-gps" "2026-06-20T12:00:00Z" gps "$member_jar")"
ocr_id="$(upload_fixture \
    "$ocr_photo" IMG_M3D_OCR_50%.png image/png \
    "m3d-acceptance-${run_stamp}-ocr" "2026-07-15T12:00:00Z" ocr "$member_jar")"
portrait_id="$(upload_fixture \
    "$face_portrait" IMG_M3D_FACE_A.jpg image/jpeg \
    "m3d-acceptance-${run_stamp}-portrait" "2026-05-10T12:00:00Z" portrait "$member_jar")"
twin_id="$(upload_fixture \
    "$face_portrait_orientation" IMG_M3D_FACE_B.jpg image/jpeg \
    "m3d-acceptance-${run_stamp}-twin" "2026-05-11T12:00:00Z" twin "$member_jar")"
decoy_id="$(upload_fixture \
    "$face_portrait_other" IMG_M3D_508.jpg image/jpeg \
    "m3d-acceptance-${run_stamp}-decoy" "2026-08-01T12:00:00Z" decoy "$member_jar")"
private_id="$(upload_fixture \
    "$private_photo" IMG_M3D_人脸.jpg image/jpeg \
    "m3d-acceptance-${run_stamp}-private" "2026-04-01T12:00:00Z" private "$member_jar")"
for pair in "gps:$gps_id" "ocr:$ocr_id" "portrait:$portrait_id" "twin:$twin_id" "decoy:$decoy_id" "private:$private_id"; do
    label="${pair%%:*}"
    id="${pair#*:}"
    wait_asset_ready "$id" "$label" "$member_jar"
done

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
    if [ "$faces" -ge 3 ] && [ "$embedded" -ge 3 ] \
        && [ "$clusters" -ge 1 ] && [ "$clustered" -ge 2 ] \
        && [ "$clip" -ge 6 ] && [ "$clipnz" -eq "$clip" ] \
        && [ "$ocr" -ge 1 ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the ML products did not land within ${ml_timeout_seconds}s: faces=$faces embedded=$embedded clusters=$clusters clustered=$clustered clip=$clip (non-zero $clipnz) ocr=$ocr"
    fi
    sleep 3
done
echo "    faces=$faces (embedded $embedded), face_clusters=$clusters (clustered faces $clustered -- the lone decoy face is noise), clip_embeddings=$clip (all non-zero), ocr_texts with text=$ocr"

echo "==> the OCR text of the Chinese fixture must carry the finale keyword"
ocr_text="$(psql_query "SELECT text FROM ocr_texts WHERE asset_id = '$ocr_id'")"
echo "    the extracted OCR text: $ocr_text"
printf '%s' "$ocr_text" | grep -q "$query_seaside" \
    || fail "the OCR text of the fixture does not carry $query_seaside: $ocr_text"

echo "==> the GPS photo must resolve to Beijing and the private photo to its own city"
deadline="$(( $(date +%s) + 120 ))"
while :; do
    gps_place="$(psql_query "SELECT place_id FROM assets WHERE id = '$gps_id'")"
    private_place="$(psql_query "SELECT place_id FROM assets WHERE id = '$private_id'")"
    [ "$gps_place" = "$beijing_place_id" ] && [ "$private_place" = "2988394" ] && break
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 100 lumemor || true
        fail "the GPS resolution did not land: gps=$gps_place (want $beijing_place_id), private=$private_place (want 2988394)"
    fi
    sleep 2
done
echo "    the GPS photo resolves Beijing ($beijing_place_id); the private photo resolves $paris_name (2988394)"

# ---------------------------------------------------------------------------
# The FTS arm: the Chinese keyword, the English substring, the literal
# wildcards.
# ---------------------------------------------------------------------------

echo "==> the FTS arm: a Chinese keyword query must hit the OCR text"
search_get "$member_jar" "query=$query_seaside_encoded" seaside
seaside_ids="$(search_item_ids "$work_dir/search_body" || true)"
seaside_count="$(printf '%s\n' "$seaside_ids" | grep -c . || true)"
printf '%s\n' "$seaside_ids" | grep -q "^$ocr_id$" \
    || fail "the 海边 query does not answer the OCR photo: $(cat "$work_dir/search_body")"
seaside_ocr_hits="$(printf '%s\n' "$seaside_ids" | grep -c "^$ocr_id$" || true)"
[ "$seaside_ocr_hits" = "1" ] \
    || fail "the 海边 query answers the OCR photo $seaside_ocr_hits times, want exactly once (the fusion's dedup): $(cat "$work_dir/search_body")"
[ "$seaside_count" = "6" ] \
    || fail "the 海边 query answers $seaside_count items, want all 6 (the semantic arm ranks the whole library, the count line the web renders): $(cat "$work_dir/search_body")"
echo "    the OCR photo answers the 海边 query exactly once (the text hit fused with its semantic rank)"

echo "==> an English substring query must hit a filename"
search_get "$member_jar" "query=$query_filename" filename
filename_ids="$(search_item_ids "$work_dir/search_body" || true)"
printf '%s\n' "$filename_ids" | grep -q "^$portrait_id$" \
    || fail "the $query_filename query does not answer the portrait (its filename): $(cat "$work_dir/search_body")"
filename_first="$(printf '%s\n' "$filename_ids" | head -1)"
[ "$filename_first" = "$portrait_id" ] \
    || fail "the $query_filename query's first item is $filename_first, want the portrait (the only text hit outranks the semantic-only hits): $(cat "$work_dir/search_body")"
echo "    the filename substring hit leads the page"

echo "==> the wildcards stay literal (the query-mode facet domain is the observable)"
search_get "$member_jar" "query=$query_percent_encoded" percent
percent_years="$(grep -o '"years":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the 50% query's years facet: $percent_years"
[ "$percent_years" = '"years":[{"assetCount":1,"year":2026}]' ] \
    || fail "the 50% query's years facet is $percent_years, want the single literal-percent hit (the OCR photo): $(cat "$work_dir/search_body")"
search_get "$member_jar" "query=$query_underscore" underscore
underscore_years="$(grep -o '"years":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the 50_ query's years facet: $underscore_years"
[ "$underscore_years" = '"years":[]' ] \
    || fail "the 50_ query's years facet is $underscore_years, want empty (no literal 50_ substring exists; the live underscore would glob IMG_M3D_508.jpg): $(cat "$work_dir/search_body")"
echo "    50% matches the literal text, 50_ matches nothing (the wildcards carry no meaning)"

# ---------------------------------------------------------------------------
# The semantic arm and the merge.
# ---------------------------------------------------------------------------

echo "==> the semantic arm: a descriptive query matching no text answers the embedded library with the subject first"
search_get "$member_jar" "query=$query_semantic_encoded" semantic
semantic_ids="$(search_item_ids "$work_dir/search_body" || true)"
semantic_count="$(printf '%s\n' "$semantic_ids" | grep -c . || true)"
[ "$semantic_count" = "6" ] \
    || fail "the 女宇航员 query answers $semantic_count items, want all 6 (the semantic arm ranks the whole embedded library): $(cat "$work_dir/search_body")"
# No text anywhere matches the query: the query mode's facet domain
# (the TEXT-match domain) is empty -- the pure-semantic observable.
semantic_years="$(grep -o '"years":\[[^]]*\]' "$work_dir/search_body" || true)"
[ "$semantic_years" = '"years":[]' ] \
    || fail "the 女宇航员 query's years facet is $semantic_years, want empty (no text matches the query -- the semantic arm serves alone): $(cat "$work_dir/search_body")"
semantic_first="$(printf '%s\n' "$semantic_ids" | head -1)"
semantic_second="$(printf '%s\n' "$semantic_ids" | sed -n '2p')"
for position_item in "first:$semantic_first" "second:$semantic_second"; do
    position="${position_item%%:*}"
    item="${position_item#*:}"
    [ "$item" = "$portrait_id" ] || [ "$item" = "$twin_id" ] \
        || fail "the 女宇航员 query's $position item is $item, want the face pair (the semantically-clustered subject set leads the ranking): $(cat "$work_dir/search_body")"
done
echo "    no text anywhere matches 女宇航员 (the empty text domain proves it); the face pair leads the embedded ranking"

echo "==> the merge: the same query twice answers byte-identical pages (determinism)"
search_get "$member_jar" "query=$query_seaside_encoded" seaside
cp "$work_dir/search_body" "$work_dir/seaside_body_first"
search_get "$member_jar" "query=$query_seaside_encoded" seaside-repeat
cmp -s "$work_dir/seaside_body_first" "$work_dir/search_body" \
    || fail "the repeated 海边 query answered a different page (determinism): $(head -c 400 "$work_dir/seaside_body_first") vs $(head -c 400 "$work_dir/search_body")"
echo "    the repeated query answers the identical page"

echo "==> the mode-tagged cursor walks the fused order (query mode, limit 4)"
search_get "$member_jar" "query=$query_seaside_encoded&limit=4" seaside-page1
page1_ids="$(search_item_ids "$work_dir/search_body" || true)"
page1_count="$(printf '%s\n' "$page1_ids" | grep -c . || true)"
[ "$page1_count" = "4" ] \
    || fail "the 海边 page 1 holds $page1_count items, want the limit 4: $(cat "$work_dir/search_body")"
rank_cursor="$(sed -n 's/.*"nextCursor":"\([^"]*\)".*/\1/p' "$work_dir/search_body")"
[ -n "$rank_cursor" ] \
    || fail "the 海边 page 1 carries no nextCursor: $(cat "$work_dir/search_body")"
# The cursor rides the URL-safe base64 alphabet raw (no percent
# escaping is needed -- the wire form is its own encoding).
search_get "$member_jar" "query=$query_seaside_encoded&limit=4&cursor=$rank_cursor" seaside-page2
page2_ids="$(search_item_ids "$work_dir/search_body" || true)"
grep -q '"nextCursor"' "$work_dir/search_body" \
    && fail "the 海边 page 2 carries a nextCursor, want the exhausted walk: $(cat "$work_dir/search_body")"
walked_ids="$(printf '%s\n' "$page1_ids" "$page2_ids")"
[ "$(printf '%s\n' "$walked_ids" | grep -c . || true)" = "6" ] \
    || fail "the walked 海边 pages hold $(printf '%s\n' "$walked_ids" | grep -c . || true) items, want 6: $(cat "$work_dir/search_body")"
seaside_order="$(printf '%s ' $seaside_ids)"
walked_order="$(printf '%s ' $walked_ids)"
[ "$walked_order" = "$seaside_order" ] \
    || fail "the cursor walk's concatenation [$walked_order] differs from the unpaged order [$seaside_order]"
echo "    the two pages concatenate to the unpaged fused order"

echo "==> a mismatched cursor mode must answer 400"
mismatch_status="$(search_get_raw "$member_jar" "cursor=$rank_cursor")"
[ "$mismatch_status" = "400" ] \
    || fail "the rank cursor in the browse mode returned status $mismatch_status, want 400 (the mode tag)"
grep -q 'VALIDATION_FAILED' "$work_dir/search_raw_body" \
    || fail "the mismatched-cursor error does not answer VALIDATION_FAILED: $(cat "$work_dir/search_raw_body")"
echo "    the rank cursor in the browse mode answers 400 VALIDATION_FAILED"

# ---------------------------------------------------------------------------
# The people face: the cluster label the person filter consumes.
# ---------------------------------------------------------------------------

echo "==> the member labels the face pair's cluster"
status="$(curl -sS -o "$work_dir/unnamed_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (member) returned status $status, want 200: $(cat "$work_dir/unnamed_body")"
cluster_id="$(sed -n 's/.*"clusterId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/unnamed_body")"
[ -n "$cluster_id" ] \
    || fail "the member's unnamed region lists no cluster: $(cat "$work_dir/unnamed_body")"
grep -q '"faceCount":2' "$work_dir/unnamed_body" \
    || fail "the member's cluster does not carry both faces of the pair: $(cat "$work_dir/unnamed_body")"
status="$(curl -sS -o "$work_dir/label_body" -b "$member_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"name":"Acceptance Person"}' \
    "$base_url/api/v1/people/$cluster_id/label")" \
    || fail "PUT /api/v1/people/$cluster_id/label request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/people/$cluster_id/label returned status $status, want 200: $(cat "$work_dir/label_body")"
echo "    the cluster ($cluster_id) is labeled Acceptance Person"

echo "==> the person filter must answer the pair's photos"
search_get "$member_jar" "personIds=$cluster_id" person-filter
person_ids="$(search_item_ids "$work_dir/search_body" || true)"
person_count="$(printf '%s\n' "$person_ids" | grep -c . || true)"
[ "$person_count" = "2" ] \
    || fail "the person filter answers $person_count items, want the pair's two: $(cat "$work_dir/search_body")"
for wanted_id in "$portrait_id" "$twin_id"; do
    printf '%s\n' "$person_ids" | grep -q "^$wanted_id$" \
        || fail "the person filter does not answer $wanted_id: $(cat "$work_dir/search_body")"
done
echo "    the person filter answers exactly the pair"

echo "==> the place filter must answer the GPS photo"
search_get "$member_jar" "placeIds=$beijing_place_id" place-filter
place_ids="$(search_item_ids "$work_dir/search_body" || true)"
place_count="$(printf '%s\n' "$place_ids" | grep -c . || true)"
[ "$place_count" = "1" ] \
    || fail "the place filter answers $place_count items, want the GPS photo alone: $(cat "$work_dir/search_body")"
printf '%s\n' "$place_ids" | grep -q "^$gps_id$" \
    || fail "the place filter does not answer the GPS photo: $(cat "$work_dir/search_body")"
echo "    the place filter answers the Beijing photo (the GeoNames id round-trips)"

# ---------------------------------------------------------------------------
# The tags domain: the idempotent create, the honest batch skips, the
# caller-scoped surfaces.
# ---------------------------------------------------------------------------

echo "==> POST /api/v1/tags must create the member's tag (201 fresh)"
status="$(curl -sS -o "$work_dir/tag_create_body" -b "$member_jar" -w '%{http_code}' \
    -X POST -H 'Content-Type: application/json' \
    -d '{"name":"精选合集"}' \
    "$base_url/api/v1/tags")" \
    || fail "POST /api/v1/tags request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/tags returned status $status, want 201: $(cat "$work_dir/tag_create_body")"
tag_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/tag_create_body")"
[ -n "$tag_id" ] \
    || fail "the created tag carries no id: $(cat "$work_dir/tag_create_body")"
echo "    the tag 精选合集 exists ($tag_id)"

echo "==> POST /api/v1/tags with the same name must answer the existing tag (200, idempotent)"
status="$(curl -sS -o "$work_dir/tag_create2_body" -b "$member_jar" -w '%{http_code}' \
    -X POST -H 'Content-Type: application/json' \
    -d '{"name":"精选合集"}' \
    "$base_url/api/v1/tags")" \
    || fail "POST /api/v1/tags (idempotent) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/tags (idempotent) returned status $status, want 200: $(cat "$work_dir/tag_create2_body")"
grep -q "\"id\":\"$tag_id\"" "$work_dir/tag_create2_body" \
    || fail "the idempotent create answered a different tag id: $(cat "$work_dir/tag_create2_body")"
echo "    the duplicate ask converges on the same row"

echo "==> POST /api/v1/tags/{id}/assets must report the honest skips (a bogus id, then the already-tagged pair)"
status="$(curl -sS -o "$work_dir/tag_assign_body" -b "$member_jar" -w '%{http_code}' \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"assetIds\":[\"$gps_id\",\"$ocr_id\",\"00000000-0000-0000-0000-000000000000\"]}" \
    "$base_url/api/v1/tags/$tag_id/assets")" \
    || fail "POST /api/v1/tags/$tag_id/assets request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/tags/$tag_id/assets returned status $status, want 200: $(cat "$work_dir/tag_assign_body")"
grep -q '"added":2' "$work_dir/tag_assign_body" \
    || fail "the first assignment did not add the two photos: $(cat "$work_dir/tag_assign_body")"
grep -q '"skipped":1' "$work_dir/tag_assign_body" \
    || fail "the bogus id did not answer the honest skip: $(cat "$work_dir/tag_assign_body")"
echo "    added 2, skipped 1 (the bogus id)"
status="$(curl -sS -o "$work_dir/tag_assign2_body" -b "$member_jar" -w '%{http_code}' \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"assetIds\":[\"$gps_id\",\"$ocr_id\"]}" \
    "$base_url/api/v1/tags/$tag_id/assets")" \
    || fail "POST /api/v1/tags/$tag_id/assets (re-assign) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/tags/$tag_id/assets (re-assign) returned status $status, want 200: $(cat "$work_dir/tag_assign2_body")"
grep -q '"added":0' "$work_dir/tag_assign2_body" \
    || fail "the re-assignment added rows, want the set semantics: $(cat "$work_dir/tag_assign2_body")"
grep -q '"skipped":2' "$work_dir/tag_assign2_body" \
    || fail "the re-assignment did not skip the already-tagged pair: $(cat "$work_dir/tag_assign2_body")"
echo "    the re-assignment answers added 0, skipped 2 (the idempotent set)"

echo "==> GET /api/v1/tags must list the vocabulary with the honest count"
status="$(curl -sS -o "$work_dir/tags_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/tags")" \
    || fail "GET /api/v1/tags request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/tags returned status $status, want 200: $(cat "$work_dir/tags_body")"
grep -q "\"id\":\"$tag_id\"" "$work_dir/tags_body" \
    || fail "the tags listing does not carry the tag: $(cat "$work_dir/tags_body")"
grep -q '"name":"精选合集"' "$work_dir/tags_body" \
    || fail "the tags listing does not carry the tag's name: $(cat "$work_dir/tags_body")"
grep -q '"assetCount":2' "$work_dir/tags_body" \
    || fail "the tag's count is not the two tagged photos: $(cat "$work_dir/tags_body")"
echo "    the vocabulary lists the tag over its two members"

echo "==> the asset detail must carry the CALLER's tags"
status="$(curl -sS -o "$work_dir/detail_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/$gps_id")" \
    || fail "GET /api/v1/assets/$gps_id (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/$gps_id (member) returned status $status, want 200: $(cat "$work_dir/detail_body")"
grep -q '"tags":\[{"id":"'"$tag_id"'","name":"精选合集"}\]' "$work_dir/detail_body" \
    || fail "the member's asset detail does not carry the caller's tag: $(cat "$work_dir/detail_body")"
echo "    the member's detail carries the tag row"

echo "==> the tag filter must answer the tagged photos"
search_get "$member_jar" "tagIds=$tag_id" tag-filter
tag_filtered_ids="$(search_item_ids "$work_dir/search_body" || true)"
tag_filtered_count="$(printf '%s\n' "$tag_filtered_ids" | grep -c . || true)"
[ "$tag_filtered_count" = "2" ] \
    || fail "the tag filter answers $tag_filtered_count items, want the two tagged: $(cat "$work_dir/search_body")"
for wanted_id in "$gps_id" "$ocr_id"; do
    printf '%s\n' "$tag_filtered_ids" | grep -q "^$wanted_id$" \
        || fail "the tag filter does not answer $wanted_id: $(cat "$work_dir/search_body")"
done
echo "    the tag filter answers the two tagged photos"

echo "==> the FTS arm joins the caller's tag vocabulary at query time"
search_get "$member_jar" "query=$query_tag_text_encoded" tag-text
tagtext_years="$(grep -o '"years":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the 合集 query's years facet: $tagtext_years"
[ "$tagtext_years" = '"years":[{"assetCount":2,"year":2026}]' ] \
    || fail "the 合集 query's years facet is $tagtext_years, want the tag's two members (the tag NAME is a text source): $(cat "$work_dir/search_body")"
echo "    the tag name feeds the text path (its two members match)"

# ---------------------------------------------------------------------------
# The share: the member opens the fixture directory as the album's
# source and grants the admin the view at create (the sharee of the
# visibility story).
# ---------------------------------------------------------------------------

echo "==> PUT /api/v1/admin/settings/mount-roots must open the e2e fixture mount (the trust boundary)"
status="$(curl -sS -o "$work_dir/mount_roots_body" -b "$admin_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d "{\"roots\":[\"$mount_root\"]}" \
    "$base_url/api/v1/admin/settings/mount-roots")" \
    || fail "PUT /api/v1/admin/settings/mount-roots request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/admin/settings/mount-roots returned status $status, want 200: $(cat "$work_dir/mount_roots_body")"
grep -q "\"roots\":\[\"$mount_root\"\]" "$work_dir/mount_roots_body" \
    || fail "the saved allowlist is not the fixture mount alone: $(cat "$work_dir/mount_roots_body")"
echo "    the fixture mount ($mount_root) is the instance's one allowed root"

echo "==> the member's path completion must list the fixture's trip directory (the user domain)"
status="$(curl -sS -o "$work_dir/completions_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/path-completions?prefix=$mount_root/tr")" \
    || fail "GET /api/v1/albums/path-completions (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/path-completions (member) returned status $status, want 200: $(cat "$work_dir/completions_body")"
grep -q "\"paths\":\[\"$source_dir\"\]" "$work_dir/completions_body" \
    || fail "the member's completion does not list the trip directory: $(cat "$work_dir/completions_body")"
echo "    the completion's domain is the opened root (every signed-in member may complete a path)"

echo "==> POST /api/v1/albums must create the member's directory album (the audience at create)"
status="$(curl -sS -o "$work_dir/album_body" -b "$member_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"M3D Acceptance Album\",\"rootPath\":\"$source_dir\",\"audience\":[\"$admin_user_id\"]}" \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/albums returned status $status, want 201: $(cat "$work_dir/album_body")"
grep -q '"callerRole":"owner"' "$work_dir/album_body" \
    || fail "the created album's card is not the owner's: $(cat "$work_dir/album_body")"
grep -q '"itemCount":0' "$work_dir/album_body" \
    || fail "the created album's count is not zero: $(cat "$work_dir/album_body")"
grep -q '"isShared":true' "$work_dir/album_body" \
    || fail "the created album's card does not carry the landed audience: $(cat "$work_dir/album_body")"
grep -q '"skipped":\[\]' "$work_dir/album_body" \
    || fail "the create's audience report skipped the admin id: $(cat "$work_dir/album_body")"
album_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/album_body")"
[ -n "$album_id" ] \
    || fail "the created album carries no id: $(cat "$work_dir/album_body")"
echo "    the album exists ($album_id), sourced at $source_dir; the admin's grant landed at create"

echo "==> POST /api/v1/albums/{id}/rescan must return 202 with the run's id"
status="$(curl -sS -o "$work_dir/rescan_body" -b "$member_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/albums/$album_id/rescan")" \
    || fail "POST /api/v1/albums/$album_id/rescan request failed"
[ "$status" = "202" ] \
    || fail "POST /api/v1/albums/$album_id/rescan returned status $status, want 202: $(cat "$work_dir/rescan_body")"
grep -q '"jobRunId":"[^"]*"' "$work_dir/rescan_body" \
    || fail "POST /api/v1/albums/$album_id/rescan carries no jobRunId: $(cat "$work_dir/rescan_body")"
echo "    the rescan run started: $(cat "$work_dir/rescan_body")"

echo "==> the album's membership must converge onto the fixture's four photos (up to ${rescan_timeout_seconds}s)"
deadline="$(( $(date +%s) + rescan_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/album_detail_body" -b "$member_jar" -w '%{http_code}' \
        "$base_url/api/v1/albums/$album_id")" \
        || fail "GET /api/v1/albums/$album_id (member, scanning) request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/albums/$album_id (member, scanning) returned status $status, want 200: $(cat "$work_dir/album_detail_body")"
    grep -q '"itemCount":4' "$work_dir/album_detail_body" && break
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the album's membership did not converge onto the fixture's four photos within ${rescan_timeout_seconds}s: $(cat "$work_dir/album_detail_body")"
    fi
    sleep 3
done
grep -q "\"rootPath\":\"$source_dir\"" "$work_dir/album_detail_body" \
    || fail "the album's detail does not carry the source path: $(cat "$work_dir/album_detail_body")"
echo "    the membership is the fixture's four photos"

echo "==> the scan's owner+checksum dedup: the four same-bytes files are honest skips, not second assets"
# The fixture's files are byte-identical to four of the member's own
# uploads, so the add arm's dedup answers each with the honest skip:
# the EXISTING rows gain the album's membership -- no second row, no
# external ingest, the asset count stays the member's six.
album_member_ids="$(grep -o '"assetId"[[:space:]]*:[[:space:]]*"[^"]*"' "$work_dir/album_detail_body" \
    | sed 's/.*"\([^"]*\)"$/\1/' | sort | tr '\n' ' ' || true)"
[ "$album_member_ids" = "$(printf '%s\n' "$gps_id" "$ocr_id" "$portrait_id" "$twin_id" | sort | tr '\n' ' ')" ] \
    || fail "the album's members are [$album_member_ids], want the member's existing four rows: $(cat "$work_dir/album_detail_body")"
asset_rows="$(psql_query 'SELECT count(*) FROM assets')"
[ "$asset_rows" = "6" ] \
    || fail "the scan created rows ($asset_rows assets, want the member's unchanged six)"
external_rows="$(psql_query "SELECT count(*) FROM assets WHERE storage_kind = 'external'")"
[ "$external_rows" = "0" ] \
    || fail "the scan ingested external rows ($external_rows, want 0 -- the content was the member's own)"
share_rows="$(psql_query "SELECT count(*) FROM album_shares WHERE album_id = '$album_id'")"
[ "$share_rows" = "1" ] \
    || fail "the create's audience landed $share_rows share rows, want 1 (the admin's grant)"
echo "    the existing four rows gained the membership; the asset count stayed six (the private photo and the decoy stay out: the album IS the shared surface)"

# ---------------------------------------------------------------------------
# The visibility (B4 and the union predicate): the sharee's search
# never answers the member's private photo from either arm.
# ---------------------------------------------------------------------------

echo "==> the sharee's search must never answer the member's private photo (both arms)"
search_get "$admin_jar" "query=$query_visibility_encoded" sharee-visibility
sharee_ids="$(search_item_ids "$work_dir/search_body" || true)"
sharee_count="$(printf '%s\n' "$sharee_ids" | grep -c . || true)"
[ "$sharee_count" = "4" ] \
    || fail "the sharee's 人脸 search answers $sharee_count items, want the four shared (the semantic arm over the union): $(cat "$work_dir/search_body")"
printf '%s\n' "$sharee_ids" | grep -q "^$private_id$" \
    && fail "the sharee's search answered the member's private photo (the text arm leaked through the union predicate): $(cat "$work_dir/search_body")"
printf '%s\n' "$sharee_ids" | grep -q "^$decoy_id$" \
    && fail "the sharee's search answered the unshared decoy: $(cat "$work_dir/search_body")"
for wanted_id in "$gps_id" "$ocr_id" "$portrait_id" "$twin_id"; do
    printf '%s\n' "$sharee_ids" | grep -q "^$wanted_id$" \
        || fail "the sharee's search does not answer the shared $wanted_id: $(cat "$work_dir/search_body")"
done
echo "    the sharee sees exactly the four shared photos -- the private photo (the query's only TEXT hit, an unguarded rank leader) never answers"

echo "==> the member's own search answers the private photo FIRST (the both-arms rank leader)"
search_get "$member_jar" "query=$query_visibility_encoded" owner-visibility
owner_ids="$(search_item_ids "$work_dir/search_body" || true)"
owner_count="$(printf '%s\n' "$owner_ids" | grep -c . || true)"
[ "$owner_count" = "6" ] \
    || fail "the member's 人脸 search answers $owner_count items, want all 6: $(cat "$work_dir/search_body")"
owner_first="$(printf '%s\n' "$owner_ids" | head -1)"
[ "$owner_first" = "$private_id" ] \
    || fail "the member's 人脸 search's first item is $owner_first, want the private photo (its text hit plus its semantic rank outrank the single-path hits): $(cat "$work_dir/search_body")"
echo "    the owner's own search answers the private photo first"

echo "==> the sharee's tag facet stays empty and the places surface drops the private city"
search_get "$admin_jar" "" sharee-browse
grep -q '"tags":\[\]' "$work_dir/search_body" \
    || fail "the sharee's tag facet is not empty (another account's tags surfaced, B4): $(cat "$work_dir/search_body")"
grep -q '"people":\[\]' "$work_dir/search_body" \
    || fail "the sharee's people facet is not empty (labels are per-user, the sharee named nobody): $(cat "$work_dir/search_body")"
sharee_places="$(grep -o '"places":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the sharee's places facet: $sharee_places"
printf '%s' "$sharee_places" | grep -q '{"assetCount":1,"country":"CN","name":"Beijing","placeId":1816670}' \
    || fail "the sharee's places facet does not carry Beijing alone: $sharee_places"
printf '%s' "$sharee_places" | grep -q "2988394" \
    && fail "the sharee's places facet carries the private photo's city: $sharee_places"
echo "    the sharee's tags and people facets are empty; the places facet carries the shared city alone"

echo "==> the sharee's asset detail carries no tags of the member's"
status="$(curl -sS -o "$work_dir/sharee_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/$gps_id")" \
    || fail "GET /api/v1/assets/$gps_id (sharee) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/$gps_id (sharee) returned status $status, want 200: $(cat "$work_dir/sharee_detail_body")"
grep -q '"tags":\[\]' "$work_dir/sharee_detail_body" \
    || fail "the sharee's asset detail carries the member's tags (B4 leak): $(cat "$work_dir/sharee_detail_body")"
echo "    the sharee's detail of the same asset carries an empty tags row (per-user state)"

# ---------------------------------------------------------------------------
# The facet counts over the member's whole library (the honest counts
# of all seven dimensions).
# ---------------------------------------------------------------------------

echo "==> the member's browse facets carry the honest counts of all seven dimensions"
search_get "$member_jar" "" member-browse
grep -q "\"people\":\[{\"assetCount\":2,\"id\":\"$cluster_id\",\"name\":\"Acceptance Person\"}\]" \
    "$work_dir/search_body" \
    || fail "the member's people facet is not the labeled pair's count: $(cat "$work_dir/search_body")"
member_places="$(grep -o '"places":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the member's places facet: $member_places"
printf '%s' "$member_places" | grep -q '{"assetCount":1,"country":"CN","name":"Beijing","placeId":1816670}' \
    || fail "the member's places facet does not carry Beijing: $member_places"
printf '%s' "$member_places" | grep -q '{"assetCount":1,"country":"FR","name":"'"$paris_name"'"' \
    || fail "the member's places facet does not carry the private photo's city: $member_places"
member_years="$(grep -o '"years":\[[^]]*\]' "$work_dir/search_body" || true)"
echo "    the member's years facet: $member_years"
printf '%s' "$member_years" | grep -q '{"assetCount":5,"year":2026}' \
    || fail "the member's years facet does not carry the five 2026 photos: $member_years"
printf '%s' "$member_years" | grep -q '{"assetCount":1,"year":2023}' \
    || fail "the member's years facet does not carry the private photo's 2023: $member_years"
grep -q '"cameras":\[{"assetCount":2,"id":"Lumemor|Golden Camera","make":"Lumemor","model":"Golden Camera"}\]' \
    "$work_dir/search_body" \
    || fail "the member's cameras facet is not the golden pair's two: $(cat "$work_dir/search_body")"
grep -q '"kinds":\[{"assetCount":6,"kind":"image"}\]' "$work_dir/search_body" \
    || fail "the member's kinds facet is not the six photos: $(cat "$work_dir/search_body")"
grep -q "\"albums\":\[{\"assetCount\":4,\"id\":\"$album_id\",\"title\":\"M3D Acceptance Album\"}\]" \
    "$work_dir/search_body" \
    || fail "the member's albums facet is not the album's four members: $(cat "$work_dir/search_body")"
grep -q "\"tags\":\[{\"assetCount\":2,\"id\":\"$tag_id\",\"name\":\"精选合集\"}\]" \
    "$work_dir/search_body" \
    || fail "the member's tags facet is not the tag's two members: $(cat "$work_dir/search_body")"
echo "    people 2, places Beijing+Paris, years 2026x5+2023x1, cameras 2, kinds 6, albums 4, tags 2 -- every dimension honest"

echo "==> the browse cursor walks the timeline order (the second mode)"
search_get "$member_jar" "limit=4" browse-page1
browse_page1_ids="$(search_item_ids "$work_dir/search_body" || true)"
browse_page1_count="$(printf '%s\n' "$browse_page1_ids" | grep -c . || true)"
[ "$browse_page1_count" = "4" ] \
    || fail "the browse page 1 holds $browse_page1_count items, want the limit 4: $(cat "$work_dir/search_body")"
browse_cursor="$(sed -n 's/.*"nextCursor":"\([^"]*\)".*/\1/p' "$work_dir/search_body")"
[ -n "$browse_cursor" ] \
    || fail "the browse page 1 carries no nextCursor: $(cat "$work_dir/search_body")"
search_get "$member_jar" "limit=4&cursor=$browse_cursor" browse-page2
browse_page2_ids="$(search_item_ids "$work_dir/search_body" || true)"
grep -q '"nextCursor"' "$work_dir/search_body" \
    && fail "the browse page 2 carries a nextCursor, want the exhausted walk: $(cat "$work_dir/search_body")"
search_get "$member_jar" "" browse-unpaged
browse_unpaged_ids="$(search_item_ids "$work_dir/search_body" || true)"
browse_walked="$(printf '%s\n' "$browse_page1_ids" "$browse_page2_ids")"
browse_walked_order="$(printf '%s ' $browse_walked)"
browse_unpaged_order="$(printf '%s ' $browse_unpaged_ids)"
[ "$browse_walked_order" = "$browse_unpaged_order" ] \
    || fail "the browse cursor walk [$browse_walked_order] differs from the unpaged browse order [$browse_unpaged_order]"
echo "    the browse pages concatenate to the unpaged time order"

# ---------------------------------------------------------------------------
# The places domain: the locale join and the caller-visible counts.
# ---------------------------------------------------------------------------

echo "==> GET /api/v1/places must answer the member's two cities with the locale names"
status="$(curl -sS -o "$work_dir/places_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/places?locale=en")" \
    || fail "GET /api/v1/places (member, en) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/places (member, en) returned status $status, want 200: $(cat "$work_dir/places_body")"
grep -q "{\"assetCount\":1,\"country\":\"CN\",\"name\":\"$beijing_name_en\",\"placeId\":$beijing_place_id,\"representativeCoverAssetIds\":\[\"$gps_id\"\]}" \
    "$work_dir/places_body" \
    || fail "the en places listing does not carry Beijing with its canonical name and the GPS cover: $(cat "$work_dir/places_body")"
grep -q "{\"assetCount\":1,\"country\":\"FR\",\"name\":\"$paris_name\",\"placeId\":2988394" \
    "$work_dir/places_body" \
    || fail "the en places listing does not carry the private photo's city: $(cat "$work_dir/places_body")"
echo "    the en listing carries both cities, the GPS photo as Beijing's cover"

status="$(curl -sS -o "$work_dir/places_zh_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/places?locale=zh-Hans")" \
    || fail "GET /api/v1/places (member, zh-Hans) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/places (member, zh-Hans) returned status $status, want 200: $(cat "$work_dir/places_zh_body")"
grep -q "{\"assetCount\":1,\"country\":\"CN\",\"name\":\"$beijing_name_zh\",\"placeId\":$beijing_place_id" \
    "$work_dir/places_zh_body" \
    || fail "the zh-Hans places listing does not carry Beijing as 北京 (the locale join): $(cat "$work_dir/places_zh_body")"
grep -q "{\"assetCount\":1,\"country\":\"FR\",\"name\":\"$paris_name\",\"placeId\":2988394" \
    "$work_dir/places_zh_body" \
    || fail "the zh-Hans listing does not keep the zh-less city's canonical fallback: $(cat "$work_dir/places_zh_body")"
echo "    the zh-Hans listing joins 北京 (the display-only locale, the canonical fallback where no zh name exists)"

echo "==> the sharee's GET /api/v1/places counts caller-visible (the private city retracts)"
status="$(curl -sS -o "$work_dir/sharee_places_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/places?locale=en")" \
    || fail "GET /api/v1/places (sharee) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/places (sharee) returned status $status, want 200: $(cat "$work_dir/sharee_places_body")"
sharee_places_rows="$(grep -c '"placeId":' "$work_dir/sharee_places_body" || true)"
[ "$sharee_places_rows" = "1" ] \
    || fail "the sharee's places listing carries $sharee_places_rows cities, want the shared city alone: $(cat "$work_dir/sharee_places_body")"
grep -q "{\"assetCount\":1,\"country\":\"CN\",\"name\":\"$beijing_name_en\",\"placeId\":$beijing_place_id" \
    "$work_dir/sharee_places_body" \
    || fail "the sharee's places listing does not carry the shared Beijing: $(cat "$work_dir/sharee_places_body")"
grep -q '2988394' "$work_dir/sharee_places_body" \
    && fail "the sharee's places listing carries the private photo's city: $(cat "$work_dir/sharee_places_body")"
echo "    the sharee sees one city (the private photo's place retracted with its visibility)"

if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the search phase (see compose logs)"
fi
teardown_stack
echo "    phase 1 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 2: the e2e stack (the M3d search story of the web app).
# ---------------------------------------------------------------------------

echo "==> phase 2/2: the e2e stack (the M3d search story of the web app)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror

echo "==> running the Playwright spec (web/e2e/m3-search.spec.ts)"
if ! (cd web && pnpm e2e e2e/m3-search.spec.ts); then
    echo "==> server logs at the time of the failure:"
    compose logs --no-color --tail 200 lumemor || true
    fail "the M3d e2e spec failed (see the report above; artifacts under web/test-results/)"
fi
teardown_stack
echo "    phase 2 passed after $(elapsed)s"

echo "==> acceptance flow completed in $(elapsed)s"
echo "M3D ACCEPTANCE PASS"
