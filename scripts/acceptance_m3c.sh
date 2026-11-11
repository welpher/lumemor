#!/usr/bin/env bash
# M3c acceptance: the albums and sharing milestone's end-to-end
# acceptance -- the chained M3b acceptance (which itself chains the
# M3a chain and through it the whole M1/M2 line, every script owning
# its own compose stack and tearing it down fully, including the
# database volume) runs first and must be green before the M3c phases
# start. Then two stack lifetimes of this repository's own compose
# deployment walk the sharing story, album autonomy
# included:
#
# Phase 1 (the sharing story over the API, two accounts, plus the
# autonomy faces): the stack starts fresh, the first-run admin is
# created and signed in (the session's identity face asserted: the
# login body and the session list both carry the account's userId and
# role -- the identity seam), and a MEMBER account is provisioned
# through the admin API and signs in with its own session. The member
# uploads one photo of their own (their sync stream) -- the story's
# ONLY upload: an album is a source, so the admin gathers no
# photos by upload. THE BOUNDARY: the mount-root allowlist is
# retired -- the compose binds ARE the boundary -- so no root is
# opened; instead the refusal matrix's new arms are pinned first (the
# member's create-album POST and their path-completion call both
# answer 403 AUTH_FORBIDDEN, and the member's create attempt lands no
# row), then the SYSTEM-WIDE completion is drilled (the empty prefix
# answers ["/"], the fixture mount's directories list without any
# opening, and the managed data tree NEVER lists -- the storage root's
# own directory answers its config sibling alone). The album is then
# created by naming a source directory: POST /albums carries the pinned triple (title,
# rootPath, audience) with the MEMBER in the audience at create and
# the admin's own id beside it: the owner id answers the honest skip
# in the create's audience report (the curation-era
# honest skip -- the report is per-entry, never an error), and the
# CREATION-TIME WRITE PROBE answers its fact honestly: the fixture
# bind is read-only, so the response carries sourceWritable false --
# feedback, never a failure. The rescan then fills the membership:
# POST /albums/{id}/rescan answers 202 with the run's id, the
# album_rescan job run reaches succeeded, and the two fixture photos
# are ingested BY REFERENCE (storage_kind external, file_path the
# real source paths, taken_at_source file -- zero copy, zero move;
# the read-only bind is the pin held by construction) through the
# same pipeline fan-out an upload's, converging the two faces into
# the one cluster (the per-user-state people arm's structure, now over external
# rows). The member's whole shared surface is asserted: the wall's
# shared group (the honest zero before the scan, the live count
# after), the timeline's inclusion (the union's shared arm beside the
# member's own sync stream), the read-only detail (callerRole
# viewer), the per-user favorite, the timeline-exclusion toggle with
# its retraction and re-include (the member's own arm, then the
# OWNER's own arm over the same album), and the people surface's
# sharing arm (the sharee sees the shared assets' faces as a cluster;
# labels stay per-user -- the sharee names it, the owner's unnamed
# region never hears about it). The sharee's write attempts answer
# the refusal matrix (the rescan, the audience write and the delete
# all refuse the view-only share with 403). The owner's audience
# write (PUT /albums/{id}/audience, the full empty set) then retracts
# EVERYTHING the share gave the member (the wall card, the timeline
# items, the detail read, the favorites view, the people surface --
# while the member's own photo and the label row survive). The
# closing refusal: the sync album's removal attempt answers the
# single-shape 404 (the default stream is no album resource -- the
# removal's system-level arm) and the refusal moved nothing -- the
# directory album, its membership and its now-empty share rows stay,
# and the referenced photos never leave the source.
#
# The album removal (the un-ingest, the E8 semantics, the closing
# beat of phase 1 over the SAME writable source): a second directory
# album is created over the rw root (inTimeline true, the affirm
# arm), and its rescan does the dedup's own trick -- it links the
# first album's member by the owner+checksum dedup AND ingests the
# removal's own seed (genuinely distinct bytes) fresh -- so the
# removal exercises BOTH guard arms at once: the fresh member
# (referenced by this album alone) purges whole through the
# permanent delete's own cascade (the asset row, the faces with the
# cluster sweep -- the cluster survives on the survivor's face with
# the representative re-homed, the embedding, the OCR text, the
# per-user favorite, the thumbnail/encoded trees), while the
# borrowed member survives on the first album's hold (its row, its
# derived trees, its timeline entry through the exclusionless second
# album -- all untouched). The host filesystem is the judge: BOTH
# source files stay byte for byte (the writable source is the honest
# form of the zero-contact pin -- the server could have written
# there and did not), the purged member's derivative trees are gone,
# the survivor's stay. The events announce the removal over the SSE
# stream (album.deleted plus asset.deleted for the purged member
# alone -- the guarded survivor announces nothing), the aggregated
# timeline drops the purged member, and the search drops it with
# the row.
#
# The autonomy faces (the closing section, over a WRITABLE source
# fixture): the repo's compose file also binds
# web/e2e/fixtures/m3-source-rw into the container WITHOUT :ro (the
# seed file is staged by this script before the album references it
# -- genuinely distinct bytes, see the fixture's README), and the
# same administrator walks the autonomy chain's every arm: the
# writable creation carries the two switches IN THE REQUEST
# (allowAppDeletion true, inTimeline false -- the album is created
# kept out of its creator's timeline, the creator's own preference
# row landed in the creation itself) and answers sourceWritable
# TRUE; the rescan ingests the file by reference; the owner's
# aggregated timeline holds the read-only album's pair and NEVER the
# excluded writable album's file (the create-time exclusion's
# convergence); the owner's detail carries the deletability
# faces (allowAppDeletion on, the derived deletable true); the
# PATCH of the deletability switch flips the derived face (the
# setting gates the affordance, pure data -- the operating system
# stays the execution's authority); the trash then MOVES the file
# into the managed trash area (the row carries the trash path, the
# source directory loses the file), and the restore moves it BACK
# (the row live again, the file returned -- the fixture healed for
# the next run).
#
# Phase 2 (the e2e stack): a fresh stack whose models ride the SAME
# storage bind mount; the Playwright spec web/e2e/m3-albums.spec.ts
# walks the albums-as-sources story of the web app through a real
# browser -- the sign-in, the create dialog's system-wide completion
# (the managed tree never listed) with its autonomy block (the
# read-only pick holds no deletability row, the writable pick's row
# renders checked on), the read-only creation's honest
# report face and the writable creation's direct close (both
# switches riding their POST bodies, the exclusion flipped at
# create), the rescans' convergence, the edit dialog's audience
# grant and autonomy block (the lock retired -- the exclusion
# toggle and, for a writable source, the deletability row), the
# create-time exclusion's aggregated-timeline convergence, the
# member's read-only perspective with the creation guard's
# refusal matrix, and the deletable wall's delete-only selection
# with its trash/restore round trip (the e2e spec seeds and heals the
# writable fixture itself). The removal story rides the same spec's
# closing scenes: the share and the naming stage the removal's
# people-convergence witness, the member's 403 refuses the sharee's
# attempt, the confirm's pinned semantics drive the DELETE, every
# held-open surface (the member's timeline, the search wall, the
# people page) converges on the events without a reload, the source
# files are byte for byte the seeds on the host, and the aftermath's
# fresh answers pin the 404s (the repeat delete, the sync album's
# attempt), the empty search and the surviving read-only cluster.
# The e2e spec bootstraps itself (the first-run
# admin and the member account through the APIs, the model group's
# download through the admin API), exactly the way every prior e2e
# spec owns its stack's bootstrap.
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
# still runs. A passing run prints "M3C ACCEPTANCE PASS" as its final
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
rescan_timeout_seconds=180
sse_timeout_seconds=60

admin_password="correct-horse-battery"
member_password="correct-horse-battery"
member_photo="$repo_root/server/internal/ml/testdata/ocr-text.png"

# The source fixtures (read-only binds, both documented in their
# READMEs): the repository's compose file binds the read-only fixture
# tree (web/e2e/fixtures/m3-source) into the container at
# /mnt/e2e-source (:ro -- the zero-copy pin held by construction, and
# the write probe's honest false fact), and the writable sibling
# (web/e2e/fixtures/m3-source-rw) WITHOUT :ro at /mnt/e2e-source-rw
# (the deletion chain's true arm -- its seed file is staged below,
# before the album references it). The story's album references
# coastal-week under the read-only root -- the two face-portrait
# JPEGs the upload stories seed, arriving through the source path
# this time (referenced in place, never uploaded). The read-only bind
# is the zero-copy pin held by construction: a scan that tried to
# write into the source would fail loudly against it.
mount_root="/mnt/e2e-source"
source_dir="/mnt/e2e-source/coastal-week"
fixture_photo_a="IMG_E4_0001.jpg"
fixture_photo_b="IMG_E4_0002.jpg"

# The writable fixture's arrangement (see web/e2e/fixtures/
# m3-source-rw/README.md): the seeded file carries genuinely distinct
# bytes (the cross-person portrait -- never a byte-identical copy of
# the read-only fixture's pair or the member's upload, so the rescan's
# owner+checksum dedup never links the stories' rows), and the trash
# + restore round trip at the section's end returns it, so the tree
# self-heals for the next run.
rw_mount_root="/mnt/e2e-source-rw"
rw_source_host_dir="$repo_root/web/e2e/fixtures/m3-source-rw"
rw_seed_template="$repo_root/server/internal/ml/testdata/face-portrait-other.jpg"
rw_fixture_photo="IMG_E6_0001.jpg"

# The removal beat's own seed (the E8 semantics): genuinely distinct
# bytes (the second person's SECOND portrait -- the first is the rw
# fixture's own seeded photo), so the removal album's rescan ingests
# it fresh while the dedup links the first album's member into the
# same album -- the two guard arms ride one removal. The next run's
# sweeps (this chain's seeder, the e2e spec's bootstrap) remove the
# file again; the removal itself must never touch it, which is the
# pin.
removal_seed_template="$repo_root/server/internal/ml/testdata/face-portrait-other-second.jpg"
removal_fixture_photo="IMG_E6_9001.jpg"

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
    # bind sources must be writable by it before the server starts.
    # Docker only creates a missing source as root:root, so the
    # directory is created by the host user first and then chowned
    # through the image itself running as root -- the fresh-install
    # contract without sudo. The chown rides the app image, so it must
    # exist: build it here (a cached no-op when the image is current;
    # `up -d --build` below re-checks it).
    #
    # The TWO binds get DIFFERENT treatments, and the difference is
    # load-bearing:
    # - /var/lib/lumemor (the data tree) is CHOWNED to 10001 -- the
    #   container is its only writer; the host never touches it again.
    # - /mnt/e2e-source-rw (the writable source fixture) is
    #   CHMOD-only (a+rwX, ownership untouched). On a real Linux bind
    #   a chown would propagate to the host directory and lock the
    #   HOST user out of the sweep/seed steps that follow
    #   (seed_rw_fixture runs as the host user -- the first CI run
    #   hit exactly that: cp: Permission denied). With chmod the
    #   directory stays the host user's (the sweep and the seed keep
    #   working) while the container's uid 10001 writes through the
    #   "other" bits: the probe's create/remove, the trash move's
    #   rename (a rename needs the DIRECTORY's write bit, never the
    #   file's), and the restore's return. On Darwin's osxfs the
    #   chown never propagated anyway -- which is why the local runs
    #   stayed green while CI died: the local green was masking the
    #   Linux semantics, and the chmod model makes both hosts behave
    #   the same.
    mkdir -p lumemor
    compose build || fail "docker compose build failed"
    compose run --rm --no-deps --entrypoint /bin/sh -u 0 lumemor \
        -c 'chown -R 10001:10001 /var/lib/lumemor; chmod -R a+rwX /mnt/e2e-source-rw' \
        || fail "the data chown and the writable-fixture chmod through the image failed"
}

# seed_rw_fixture stages the writable source fixture's photo: the
# committed tree is the README alone, the runs copy the seed in, and
# the deletion chain's restore returns it. The bind is live, so the
# container sees the file the moment it lands; the directory's
# ownership (prepare_bind_mount above) is what the probe and the
# trash move exercise. The sweep first is the determinism: the
# directory persists across every run on the host (this chain's one
# seed, the e2e spec's two, a probe wart), and a stale photo would
# leave the scan's membership counting leftovers -- everything that
# is not the README goes.
seed_rw_fixture() {
    [ -f "$rw_seed_template" ] \
        || fail "fixture missing: $rw_seed_template -- the writable source's seed template"
    mkdir -p "$rw_source_host_dir"
    find "$rw_source_host_dir" -mindepth 1 -maxdepth 1 -type f ! -name 'README.md' -delete
    cp "$rw_seed_template" "$rw_source_host_dir/$rw_fixture_photo"
    echo "    the writable source fixture holds $rw_mount_root/$rw_fixture_photo (alone)"
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
    entries="$(grep -o '"userId"' "$work_dir/member_sessions_body" | wc -l | tr -d ' ' || true)"
    with_identity="$(grep -o "\"userId\":\"$member_user_id\"" "$work_dir/member_sessions_body" | wc -l | tr -d ' ' || true)"
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

echo "==> running the prior acceptance (the M3b chain owns its stacks)"
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

echo "==> the member's people surface stays empty (they own no face and nothing is shared yet)"
status="$(curl -sS -o "$work_dir/member_unnamed_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (member, pre-share) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (member, pre-share) returned status $status, want 200: $(cat "$work_dir/member_unnamed_body")"
grep -q '"items":\[\]' "$work_dir/member_unnamed_body" \
    || fail "the member's unnamed region is not empty before the share: $(cat "$work_dir/member_unnamed_body")"
echo "    the member sees no cluster yet"

echo "==> the member's POST /api/v1/albums must answer 403 (the creation guard)"
# The refusal matrix's new arm:
# mounting an existing photo directory is a configuration act, so the
# creation is the administrator's alone -- the member's attempt is
# refused with the admin surfaces' own envelope, before any request
# part is considered, and no row lands.
status="$(curl -sS -o "$work_dir/member_create_refusal_body" -b "$member_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"Member attempt\",\"rootPath\":\"$source_dir\",\"audience\":[]}" \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums (member) request failed"
[ "$status" = "403" ] \
    || fail "POST /api/v1/albums (member) returned status $status, want 403 (the creation is the administrator's): $(cat "$work_dir/member_create_refusal_body")"
grep -q 'AUTH_FORBIDDEN' "$work_dir/member_create_refusal_body" \
    || fail "the member's create refusal does not answer AUTH_FORBIDDEN: $(cat "$work_dir/member_create_refusal_body")"
directory_album_rows="$(psql_query "SELECT count(*) FROM albums WHERE type = 'directory'")"
[ "$directory_album_rows" = "0" ] \
    || fail "the member's refused create landed $directory_album_rows directory album rows, want 0"
echo "    the member's create answers 403 AUTH_FORBIDDEN and lands no row"

echo "==> the member's GET /api/v1/albums/path-completions must answer 403 (the completion is the admin's domain)"
status="$(curl -sS -o "$work_dir/member_completion_refusal_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/path-completions?prefix=$mount_root")" \
    || fail "GET /api/v1/albums/path-completions (member) request failed"
[ "$status" = "403" ] \
    || fail "GET /api/v1/albums/path-completions (member) returned status $status, want 403 (the completion browses the filesystem for the configuration act): $(cat "$work_dir/member_completion_refusal_body")"
grep -q 'AUTH_FORBIDDEN' "$work_dir/member_completion_refusal_body" \
    || fail "the member's completion refusal does not answer AUTH_FORBIDDEN: $(cat "$work_dir/member_completion_refusal_body")"
echo "    the member's completion answers 403 AUTH_FORBIDDEN (the members' contribution path is the upload stream)"

echo "==> GET /api/v1/albums/path-completions must drill the container's real filesystem (the boundary)"
# The system-wide domain (the allowlist retired): the
# completion's rows are the honest reflection of what the operating
# system lets the server read, and the compose mounts ARE the
# boundary -- no root was opened for anything below to list.
status="$(curl -sS -o "$work_dir/completions_root_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/path-completions")" \
    || fail "GET /api/v1/albums/path-completions (empty prefix) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/path-completions (empty prefix) returned status $status, want 200: $(cat "$work_dir/completions_root_body")"
grep -q '"completions":\[{"path":"/","writable":false}\]' "$work_dir/completions_root_body" \
    || fail "the empty prefix does not answer the filesystem root itself with its write fact: $(cat "$work_dir/completions_root_body")"
echo "    the empty prefix answers the root row with its write fact (\"/\" refuses the server's write probe) -- the combobox's start-here row"

status="$(curl -sS -o "$work_dir/completions_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/path-completions?prefix=$mount_root/")" \
    || fail "GET /api/v1/albums/path-completions request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/path-completions returned status $status, want 200: $(cat "$work_dir/completions_body")"
grep -q "\"completions\":\[{\"path\":\"$mount_root/coastal-week\",\"writable\":false},{\"path\":\"$mount_root/trip-2026\",\"writable\":false}\]" "$work_dir/completions_body" \
    || fail "the completion does not list the fixture's directories with their write facts: $(cat "$work_dir/completions_body")"
echo "    the fixture mount lists its directories with their write facts (both rows read-only) with no root opened (the bind is the boundary)"

status="$(curl -sS -o "$work_dir/completions_managed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/path-completions?prefix=/var/lib/lumemor/")" \
    || fail "GET /api/v1/albums/path-completions (managed tree) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/path-completions (managed tree) returned status $status, want 200: $(cat "$work_dir/completions_managed_body")"
grep -q '"/var/lib/lumemor/config"' "$work_dir/completions_managed_body" \
    || fail "the storage root's own directory does not list its config sibling: $(cat "$work_dir/completions_managed_body")"
grep -q '"/var/lib/lumemor/data"' "$work_dir/completions_managed_body" \
    && fail "the completion listed the managed data tree: $(cat "$work_dir/completions_managed_body")"
echo "    the managed data tree never lists (the one exclusion that stayed: the config sibling answers alone)"

echo "==> POST /api/v1/albums must create the admin's directory album (the audience at create, the owner id honestly skipped)"
# The read-only creation carries the exclusion switch's AFFIRM arm
# explicitly (inTimeline true -- the feed's default, no preference
# row written); the writable section below carries the EXCLUDE arm,
# so the two create bodies pin both arms of the switch.
status="$(curl -sS -o "$work_dir/album_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"M3c Acceptance Album\",\"rootPath\":\"$source_dir\",\"audience\":[\"$member_user_id\",\"$admin_user_id\"],\"inTimeline\":true}" \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/albums returned status $status, want 201: $(cat "$work_dir/album_body")"
grep -q '"callerRole":"owner"' "$work_dir/album_body" \
    || fail "the created album's card is not the owner's: $(cat "$work_dir/album_body")"
grep -q '"type":"directory"' "$work_dir/album_body" \
    || fail "the created album is not the directory type: $(cat "$work_dir/album_body")"
grep -q '"itemCount":0' "$work_dir/album_body" \
    || fail "the created album's count is not zero: $(cat "$work_dir/album_body")"
grep -q '"isShared":true' "$work_dir/album_body" \
    || fail "the created album's card does not carry the landed audience: $(cat "$work_dir/album_body")"
# The write probe's honest fact: the fixture bind is
# read-only, so the creation answers sourceWritable FALSE -- a fact
# about the source, never a failure (the album is legitimate; its
# photos just cannot be deleted through the application).
grep -q '"sourceWritable":false' "$work_dir/album_body" \
    || fail "the read-only creation does not carry sourceWritable false: $(cat "$work_dir/album_body")"
# The response card carries the exclusion switch's landed state (the
# affirm's default -- the album feeds its creator's timeline).
grep -q '"inTimeline":true' "$work_dir/album_body" \
    || fail "the affirm-armed creation's card does not carry inTimeline true: $(cat "$work_dir/album_body")"
# The curation-era honest skip: the audience report
# carries each unusable id with its reason, never an error -- the
# owner sees the album by construction, so the own id is the skip.
grep -q "\"userId\":\"$admin_user_id\"" "$work_dir/album_body" \
    || fail "the create's audience report does not name the skipped owner id: $(cat "$work_dir/album_body")"
grep -q '"reason":"owner"' "$work_dir/album_body" \
    || fail "the create's audience report does not carry the owner reason: $(cat "$work_dir/album_body")"
album_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/album_body")"
[ -n "$album_id" ] \
    || fail "the created album carries no id: $(cat "$work_dir/album_body")"
echo "    the album exists ($album_id), sourced at $source_dir; the member's grant landed, the owner id answered the honest skip, the probe answered the read-only fact"

echo "==> the member's wall must list the album in the shared group with the honest zero count"
status="$(curl -sS -o "$work_dir/member_wall_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums")" \
    || fail "GET /api/v1/albums (member, pre-scan) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums (member, pre-scan) returned status $status, want 200: $(cat "$work_dir/member_wall_body")"
grep -q "\"$album_id\"" "$work_dir/member_wall_body" \
    || fail "the member's wall does not list the shared album: $(cat "$work_dir/member_wall_body")"
grep -q '"callerRole":"viewer"' "$work_dir/member_wall_body" \
    || fail "the member's wall card is not the viewer's: $(cat "$work_dir/member_wall_body")"
grep -q '"itemCount":0' "$work_dir/member_wall_body" \
    || fail "the member's wall card does not carry the honest zero count: $(cat "$work_dir/member_wall_body")"
echo "    the shared card lists with the viewer role and the creation-time zero (creation references, it does not scan)"

echo "==> POST /api/v1/albums/{id}/rescan must return 202 with the run's id"
status="$(curl -sS -o "$work_dir/rescan_body" -b "$admin_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/albums/$album_id/rescan")" \
    || fail "POST /api/v1/albums/$album_id/rescan request failed"
[ "$status" = "202" ] \
    || fail "POST /api/v1/albums/$album_id/rescan returned status $status, want 202: $(cat "$work_dir/rescan_body")"
grep -q '"jobRunId":"[^"]*"' "$work_dir/rescan_body" \
    || fail "POST /api/v1/albums/$album_id/rescan carries no jobRunId: $(cat "$work_dir/rescan_body")"
echo "    the rescan run started: $(cat "$work_dir/rescan_body")"

echo "==> the album_rescan job run must reach succeeded (up to ${rescan_timeout_seconds}s)"
rescan_row() {
    grep -o '"kind":"album_rescan"[^}]*' "$work_dir/jobs_body" | head -1
}
deadline="$(( $(date +%s) + rescan_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/jobs_body" -b "$admin_jar" -w '%{http_code}' \
        "$base_url/api/v1/admin/jobs")" \
        || fail "GET /api/v1/admin/jobs request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/admin/jobs returned status $status, want 200: $(cat "$work_dir/jobs_body")"
    row="$(rescan_row)"
    state="$(printf '%s\n' "$row" | sed -n 's/.*"state":"\([a-z]*\)".*/\1/p')"
    if [ "$state" = "succeeded" ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the album_rescan run did not succeed within ${rescan_timeout_seconds}s (last row: $row)"
    fi
    if [ "$state" = "failed" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the album_rescan run failed: $row (see the server logs above)"
    fi
    sleep 3
done
echo "    the rescan run succeeded after $(elapsed)s: $row"

echo "==> the album's detail must carry the scanned membership and the source's own names"
status="$(curl -sS -o "$work_dir/album_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$album_id")" \
    || fail "GET /api/v1/albums/$album_id (admin, scanned) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/$album_id (admin, scanned) returned status $status, want 200: $(cat "$work_dir/album_detail_body")"
grep -q "\"rootPath\":\"$source_dir\"" "$work_dir/album_detail_body" \
    || fail "the album's detail does not carry the source path: $(cat "$work_dir/album_detail_body")"
grep -q '"itemCount":2' "$work_dir/album_detail_body" \
    || fail "the scanned album's count is not the fixture's two: $(cat "$work_dir/album_detail_body")"
grep -q "\"originalFilename\":\"$fixture_photo_a\"" "$work_dir/album_detail_body" \
    || fail "the album's items do not carry $fixture_photo_a (the source tree's own name): $(cat "$work_dir/album_detail_body")"
grep -q "\"originalFilename\":\"$fixture_photo_b\"" "$work_dir/album_detail_body" \
    || fail "the album's items do not carry $fixture_photo_b (the source tree's own name): $(cat "$work_dir/album_detail_body")"
grep -q "\"userId\":\"$member_user_id\"" "$work_dir/album_detail_body" \
    || fail "the owner's detail does not list the member in the shares: $(cat "$work_dir/album_detail_body")"
echo "    the detail carries the source path, the two items and the audience row"

echo "==> the scan must ingest the fixture photos by reference (zero copy, zero move)"
asset_rows="$(psql_query 'SELECT count(*) FROM assets')"
[ "$asset_rows" = "3" ] \
    || fail "the instance holds $asset_rows assets, want 3 (the member's upload and the two by-reference ingests)"
external_rows="$(psql_query "SELECT count(*) FROM assets WHERE storage_kind = 'external' AND deleted_at IS NULL")"
[ "$external_rows" = "2" ] \
    || fail "the scan did not land two live external rows: $external_rows"
taken_file_rows="$(psql_query "SELECT count(*) FROM assets WHERE taken_at_source = 'file'")"
[ "$taken_file_rows" = "2" ] \
    || fail "the external rows do not carry the file taken-at source: $taken_file_rows"
source_paths="$(psql_query "SELECT file_path FROM assets WHERE storage_kind = 'external' ORDER BY file_path" | tr '\n' ' ')"
[ "$source_paths" = "$source_dir/$fixture_photo_a $source_dir/$fixture_photo_b " ] \
    || fail "the external rows do not point at the real source paths: [$source_paths]"
echo "    two external rows at the real source paths (the read-only bind holds: nothing was copied)"

echo "==> the scan's pipeline must converge the two source photos to ready (up to ${ml_timeout_seconds}s)"
deadline="$(( $(date +%s) + ml_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/album_timeline_body" -b "$admin_jar" -w '%{http_code}' \
        "$base_url/api/v1/assets/timeline?albumId=$album_id")" \
        || fail "GET /api/v1/assets/timeline (album, scanned) request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/assets/timeline (album, scanned) returned status $status, want 200: $(cat "$work_dir/album_timeline_body")"
    ready_count="$(grep -o '"isReady":true' "$work_dir/album_timeline_body" | wc -l | tr -d ' ' || true)"
    [ "$ready_count" = "2" ] && break
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the source photos did not become ready within ${ml_timeout_seconds}s (ready: $ready_count)"
    fi
    sleep 3
done
shared_ids="$(item_ids "$work_dir/album_timeline_body")"
shared_a_id="$(printf '%s\n' "$shared_ids" | sed -n 1p)"
shared_b_id="$(printf '%s\n' "$shared_ids" | sed -n 2p)"
[ -n "$shared_a_id" ] && [ -n "$shared_b_id" ] \
    || fail "the album's timeline listing does not carry the two source photos: $(cat "$work_dir/album_timeline_body")"
echo "    both source photos are ready ($shared_a_id, $shared_b_id) -- the source ingest runs the upload's own fan-out"

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

echo "==> the member's wall card must carry the live count"
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
for wanted_id in "$member_photo_id" "$shared_a_id" "$shared_b_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/member_timeline_body" \
        || fail "the member's timeline does not list $wanted_id: $(cat "$work_dir/member_timeline_body")"
done
# The pair's own order is the files': the source ingest's taken-at
# is the source file's mtime (taken_at_source file) -- the working
# tree's creation time, newer than the member's mid-2026 upload in
# any honest run, so the pair leads and the member's own photo
# closes the timeline. The pair's INTERNAL order rides the id
# tiebreak (the two fixture files share one mtime second), so the
# leading two are asserted as a set.
member_timeline_ids="$(item_ids "$work_dir/member_timeline_body")"
member_timeline_count="$(printf '%s\n' "$member_timeline_ids" | grep -c . || true)"
[ "$member_timeline_count" = "3" ] \
    || fail "the member's timeline holds $member_timeline_count items, want 3: $(cat "$work_dir/member_timeline_body")"
member_timeline_last="$(printf '%s\n' "$member_timeline_ids" | tail -1)"
[ "$member_timeline_last" = "$member_photo_id" ] \
    || fail "the member's timeline's last item is $member_timeline_last, want the member's own photo (the source pair's file mtimes are the newer times): $(cat "$work_dir/member_timeline_body")"
pair_leading="$(printf '%s\n' "$member_timeline_ids" | head -2 | sort | tr '\n' ' ')"
pair_sorted="$(printf '%s\n' "$shared_a_id" "$shared_b_id" | sort | tr '\n' ' ')"
[ "$pair_leading" = "$pair_sorted" ] \
    || fail "the member's timeline's leading pair is [$pair_leading], want the shared source pair [$pair_sorted]: $(cat "$work_dir/member_timeline_body")"
echo "    three items: the member's own photo and the two shared ones (the union's shared arm; the source pair leads, the member's photo closes)"

echo "==> the member's album detail must read as viewer"
status="$(curl -sS -o "$work_dir/member_detail_body" -b "$member_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$album_id")" \
    || fail "GET /api/v1/albums/$album_id (member) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/$album_id (member) returned status $status, want 200: $(cat "$work_dir/member_detail_body")"
grep -q '"callerRole":"viewer"' "$work_dir/member_detail_body" \
    || fail "the member's detail is not the viewer's: $(cat "$work_dir/member_detail_body")"
grep -q "\"$shared_a_id\"" "$work_dir/member_detail_body" \
    || fail "the member's detail does not list the shared items: $(cat "$work_dir/member_detail_body")"
grep -q '"inTimeline":true' "$work_dir/member_detail_body" \
    || fail "the member's detail does not carry the excludable inTimeline field: $(cat "$work_dir/member_detail_body")"
echo "    the detail answers the viewer role with the items page"

echo "==> the member favorites a shared asset (per-user state)"
status="$(curl -sS -o "$work_dir/member_favorite_body" -b "$member_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"favorite\",\"ids\":[\"$shared_a_id\"]}" \
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
grep -q "\"$shared_a_id\"" "$work_dir/member_favorites_body" \
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
for wanted_id in "$member_photo_id" "$shared_a_id" "$shared_b_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/member_timeline_body" \
        || fail "the re-included timeline does not list $wanted_id: $(cat "$work_dir/member_timeline_body")"
done
echo "    the shared items returned with the re-include"

echo "==> the owner's own timeline-exclusion toggle must retract and re-include the album's photos (the owner's arm)"
# The same per-user preference, the OWNER's own row this time (the
# edit dialog's second toggle): the admin's aggregated timeline drops
# the album's photos while the preference holds, and the album's own
# detail never filters for either caller whatever the switch says.
status="$(curl -sS -o "$work_dir/admin_pref_body" -b "$admin_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"inTimeline":false}' \
    "$base_url/api/v1/albums/$album_id/timeline-preference")" \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (owner) request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (owner) returned status $status, want 200: $(cat "$work_dir/admin_pref_body")"
grep -q '"inTimeline":false' "$work_dir/admin_pref_body" \
    || fail "the owner's preference response does not carry the stored switch: $(cat "$work_dir/admin_pref_body")"
status="$(curl -sS -o "$work_dir/admin_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (owner, excluded) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (owner, excluded) returned status $status, want 200: $(cat "$work_dir/admin_timeline_body")"
# The owner's timeline holds NOTHING after the exclusion (the album
# is the admin's only source in this stack), so the extraction must
# tolerate the empty answer -- grep exits 1 on no matches and the
# script runs under pipefail.
admin_timeline_ids="$(item_ids "$work_dir/admin_timeline_body" | tr '\n' ' ' || true)"
[ "$admin_timeline_ids" = "" ] \
    || fail "the owner's excluded timeline holds [$admin_timeline_ids], want nothing (the album is the owner's only source): $(cat "$work_dir/admin_timeline_body")"
status="$(curl -sS -o "$work_dir/admin_pref_body" -b "$admin_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"inTimeline":true}' \
    "$base_url/api/v1/albums/$album_id/timeline-preference")" \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (owner, re-include) request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/albums/$album_id/timeline-preference (owner, re-include) returned status $status, want 200: $(cat "$work_dir/admin_pref_body")"
status="$(curl -sS -o "$work_dir/admin_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (owner, re-included) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (owner, re-included) returned status $status, want 200: $(cat "$work_dir/admin_timeline_body")"
for wanted_id in "$shared_a_id" "$shared_b_id"; do
    grep -q "\"$wanted_id\"" "$work_dir/admin_timeline_body" \
        || fail "the owner's re-included timeline does not list $wanted_id: $(cat "$work_dir/admin_timeline_body")"
done
echo "    the owner's own exclusion retracted the album's photos and the re-include returned them"

echo "==> the people surface's sharing arm: the sharee sees the shared faces, labels stay per-user"
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

echo "==> the sharee's write attempts must answer the refusal matrix (the read-only share)"
status="$(curl -sS -o "$work_dir/member_rescan_refusal_body" -b "$member_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/albums/$album_id/rescan")" \
    || fail "POST /api/v1/albums/$album_id/rescan (member) request failed"
[ "$status" = "403" ] \
    || fail "the member's rescan returned status $status, want 403 (the share is view-only): $(cat "$work_dir/member_rescan_refusal_body")"
grep -q 'AUTH_FORBIDDEN' "$work_dir/member_rescan_refusal_body" \
    || fail "the member's rescan refusal does not answer AUTH_FORBIDDEN: $(cat "$work_dir/member_rescan_refusal_body")"
status="$(curl -sS -o "$work_dir/member_audience_refusal_body" -b "$member_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d "{\"audience\":[\"$member_user_id\"]}" \
    "$base_url/api/v1/albums/$album_id/audience")" \
    || fail "PUT /api/v1/albums/$album_id/audience (member) request failed"
[ "$status" = "403" ] \
    || fail "the member's audience write returned status $status, want 403 (the share is view-only): $(cat "$work_dir/member_audience_refusal_body")"
grep -q 'AUTH_FORBIDDEN' "$work_dir/member_audience_refusal_body" \
    || fail "the member's audience refusal does not answer AUTH_FORBIDDEN: $(cat "$work_dir/member_audience_refusal_body")"
status="$(curl -sS -o "$work_dir/member_delete_refusal_body" -b "$member_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$album_id")" \
    || fail "DELETE /api/v1/albums/$album_id (member) request failed"
[ "$status" = "403" ] \
    || fail "the member's album delete returned status $status, want 403 (the share is view-only): $(cat "$work_dir/member_delete_refusal_body")"
grep -q 'AUTH_FORBIDDEN' "$work_dir/member_delete_refusal_body" \
    || fail "the member's delete refusal does not answer AUTH_FORBIDDEN: $(cat "$work_dir/member_delete_refusal_body")"
echo "    the rescan, the audience write and the delete all refuse the view-only share"

echo "==> the owner's audience write must retract everything the share gave the member (the full empty set)"
status="$(curl -sS -o "$work_dir/audience_body" -b "$admin_jar" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' \
    -d '{"audience":[]}' \
    "$base_url/api/v1/albums/$album_id/audience")" \
    || fail "PUT /api/v1/albums/$album_id/audience request failed"
[ "$status" = "200" ] \
    || fail "PUT /api/v1/albums/$album_id/audience returned status $status, want 200: $(cat "$work_dir/audience_body")"
grep -q "\"revoked\":\[\"$member_user_id\"\]" "$work_dir/audience_body" \
    || fail "the audience write's report does not name the revoked member: $(cat "$work_dir/audience_body")"
grep -q '"granted":\[\]' "$work_dir/audience_body" \
    || fail "the retraction granted rows: $(cat "$work_dir/audience_body")"
grep -q '"audience":\[\]' "$work_dir/audience_body" \
    || fail "the resulting audience is not empty: $(cat "$work_dir/audience_body")"
grep -q '"skipped":\[\]' "$work_dir/audience_body" \
    || fail "the retraction skipped ids: $(cat "$work_dir/audience_body")"
echo "    the full empty set revoked the member (the honest report: one revoked, nothing granted, nothing skipped)"

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

echo "==> the sync album's removal attempt must answer the single-shape 404 (the system-level refusal)"
# The removal's refusal matrix, the system arm: the sync album is no
# album resource -- the owner's own default stream folds into the
# single-shape 404 at the gate, exactly like an unknown id. The
# sharee arm already answered above (the member's 403); the
# directory album's removal is the closing beat's own story.
sync_album_id="$(psql_query "SELECT id FROM albums WHERE type = 'sync' LIMIT 1")"
[ -n "$sync_album_id" ] \
    || fail "the instance holds no sync album row to attempt the removal against"
status="$(curl -sS -o "$work_dir/admin_delete_body" -b "$admin_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$sync_album_id")" \
    || fail "DELETE /api/v1/albums/$sync_album_id (admin) request failed"
[ "$status" = "404" ] \
    || fail "DELETE /api/v1/albums/$sync_album_id (admin) returned status $status, want 404 (the sync album is system-managed, the single-shape refusal): $(cat "$work_dir/admin_delete_body")"
grep -q 'ALBUM_NOT_FOUND' "$work_dir/admin_delete_body" \
    || fail "the sync album's removal refusal does not answer ALBUM_NOT_FOUND: $(cat "$work_dir/admin_delete_body")"
# The refusal moved nothing: the sync row stays, and the phase's
# directory album -- its membership rows, its now-empty share rows
# and its two live photos -- stands untouched beside it (the
# audience write above is the only face that ever touched the share
# rows, and the referenced photos never leave the source).
album_rows="$(psql_query "SELECT count(*) FROM albums WHERE id = '$album_id'")"
[ "$album_rows" = "1" ] \
    || fail "the refusal touched the directory album's row ($album_rows rows, want 1)"
sync_rows="$(psql_query "SELECT count(*) FROM albums WHERE id = '$sync_album_id' AND type = 'sync'")"
[ "$sync_rows" = "1" ] \
    || fail "the sync refusal touched the sync row ($sync_rows rows, want 1)"
item_rows="$(psql_query "SELECT count(*) FROM album_items WHERE album_id = '$album_id'")"
[ "$item_rows" = "2" ] \
    || fail "the refusal touched the membership rows ($item_rows rows, want 2)"
share_rows="$(psql_query "SELECT count(*) FROM album_shares WHERE album_id = '$album_id'")"
[ "$share_rows" = "0" ] \
    || fail "the audience write left share rows behind ($share_rows rows, want 0)"
external_live="$(psql_query "SELECT count(*) FROM assets WHERE storage_kind = 'external' AND deleted_at IS NULL")"
[ "$external_live" = "2" ] \
    || fail "the refusal touched the photos ($external_live of 2 live)"
echo "    the sync refusal answers the single-shape 404 and moved nothing: both album rows, the membership and the photos stay"

status="$(curl -sS -o "$work_dir/admin_unnamed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (admin, after the refusal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (admin, after the refusal) returned status $status, want 200: $(cat "$work_dir/admin_unnamed_body")"
grep -q "\"$member_cluster_id\"" "$work_dir/admin_unnamed_body" \
    || fail "the owner's unnamed region lost the cluster (the photos never left): $(cat "$work_dir/admin_unnamed_body")"
echo "    the owner's people surface keeps the faces (the photos never left)"

# ---------------------------------------------------------------------------
# The album autonomy faces, over a WRITABLE source:
# the probe's true arm, the two create switches' request beats, the
# create-time exclusion's convergence, the deletability setting's
# derived face, and the trash/restore round trip that moves the
# source file itself.
# ---------------------------------------------------------------------------

echo "==> seeding the writable source fixture (the deletion chain's true arm)"
seed_rw_fixture

echo "==> POST /api/v1/albums must carry the two switches and answer the write probe's TRUE fact over the writable source"
# The two switches ride the creation request, exactly as the create
# dialog's drafts hold them: the deletability switch on (the writable
# pick's completion row rendered it) and the timeline exclusion
# CHECKED -- inTimeline false lands the creator's own preference row
# in the creation itself, so the album is born out of its creator's
# aggregated timeline (the audience's own feeds untouched).
status="$(curl -sS -o "$work_dir/rw_album_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"M3c Writable Album\",\"rootPath\":\"$rw_mount_root\",\"audience\":[],\"allowAppDeletion\":true,\"inTimeline\":false}" \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums (writable) request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/albums (writable) returned status $status, want 201: $(cat "$work_dir/rw_album_body")"
grep -q '"sourceWritable":true' "$work_dir/rw_album_body" \
    || fail "the writable creation does not carry sourceWritable true: $(cat "$work_dir/rw_album_body")"
# The response card carries the landed exclusion.
grep -q '"inTimeline":false' "$work_dir/rw_album_body" \
    || fail "the exclude-armed creation's card does not carry the landed exclusion: $(cat "$work_dir/rw_album_body")"
rw_album_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/rw_album_body")"
[ -n "$rw_album_id" ] \
    || fail "the writable album carries no id: $(cat "$work_dir/rw_album_body")"
rw_pref_rows="$(psql_query "SELECT count(*) FROM user_album_prefs WHERE album_id = '$rw_album_id' AND in_timeline = false")"
[ "$rw_pref_rows" = "1" ] \
    || fail "the exclude-armed creation landed $rw_pref_rows creator preference rows, want exactly 1 (in_timeline false)"
echo "    the writable album exists ($rw_album_id) at $rw_mount_root: both switches rode the request, the creator's exclusion row landed, the probe answered true"

echo "==> POST /api/v1/albums/{id}/rescan must return 202 and the run must reach succeeded"
status="$(curl -sS -o "$work_dir/rw_rescan_body" -b "$admin_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/albums/$rw_album_id/rescan")" \
    || fail "POST /api/v1/albums/$rw_album_id/rescan request failed"
[ "$status" = "202" ] \
    || fail "POST /api/v1/albums/$rw_album_id/rescan returned status $status, want 202: $(cat "$work_dir/rw_rescan_body")"
grep -q '"jobRunId":"[^"]*"' "$work_dir/rw_rescan_body" \
    || fail "POST /api/v1/albums/$rw_album_id/rescan carries no jobRunId: $(cat "$work_dir/rw_rescan_body")"
deadline="$(( $(date +%s) + rescan_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/jobs_body" -b "$admin_jar" -w '%{http_code}' \
        "$base_url/api/v1/admin/jobs")" \
        || fail "GET /api/v1/admin/jobs request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/admin/jobs returned status $status, want 200: $(cat "$work_dir/jobs_body")"
    row="$(rescan_row)"
    state="$(printf '%s\n' "$row" | sed -n 's/.*"state":"\([a-z]*\)".*/\1/p')"
    if [ "$state" = "succeeded" ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the writable album's rescan run did not succeed within ${rescan_timeout_seconds}s (last row: $row)"
    fi
    if [ "$state" = "failed" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the writable album's rescan run failed: $row (see the server logs above)"
    fi
    sleep 3
done
echo "    the writable album's rescan run succeeded: $row"

echo "==> the writable album's detail must carry the deletability faces"
status="$(curl -sS -o "$work_dir/rw_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$rw_album_id")" \
    || fail "GET /api/v1/albums/$rw_album_id (writable) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/albums/$rw_album_id (writable) returned status $status, want 200: $(cat "$work_dir/rw_detail_body")"
grep -q '"itemCount":1' "$work_dir/rw_detail_body" \
    || fail "the writable album's count is not the seeded one: $(cat "$work_dir/rw_detail_body")"
grep -q '"sourceWritable":true' "$work_dir/rw_detail_body" \
    || fail "the writable album's detail does not carry the probe's true fact: $(cat "$work_dir/rw_detail_body")"
grep -q '"allowAppDeletion":true' "$work_dir/rw_detail_body" \
    || fail "the writable album's deletability switch is not on by default: $(cat "$work_dir/rw_detail_body")"
grep -q '"deletable":true' "$work_dir/rw_detail_body" \
    || fail "the writable album's derived deletable face is not true (owner AND writable AND switch): $(cat "$work_dir/rw_detail_body")"
rw_asset_id="$(sed -n 's/.*"assetId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/rw_detail_body")"
[ -n "$rw_asset_id" ] \
    || fail "the writable album's detail lists no member asset id: $(cat "$work_dir/rw_detail_body")"
echo "    the detail carries the probe fact, the switch on, and the derived deletable face true ($rw_asset_id)"

echo "==> the owner's aggregated timeline must hold the read-only pair and NEVER the excluded writable album's file"
# The create-time exclusion's convergence (the creator's own arm):
# the writable album was born excluded, so its file never feeds the
# owner's aggregated timeline -- while the read-only album (created
# with the affirm's default) feeds both its photos. The exclusion
# scopes the AGGREGATE only: the writable album's own detail above
# still lists its file.
status="$(curl -sS -o "$work_dir/admin_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (owner, after the writable rescan) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (owner, after the writable rescan) returned status $status, want 200: $(cat "$work_dir/admin_timeline_body")"
admin_timeline_ids="$(item_ids "$work_dir/admin_timeline_body" | sort | tr '\n' ' ')"
ro_pair_sorted="$(printf '%s\n' "$shared_a_id" "$shared_b_id" | sort | tr '\n' ' ')"
[ "$admin_timeline_ids" = "$ro_pair_sorted" ] \
    || fail "the owner's timeline holds [$admin_timeline_ids], want exactly the read-only pair [$ro_pair_sorted] -- the writable album's file must stay excluded: $(cat "$work_dir/admin_timeline_body")"
echo "    the timeline holds exactly the read-only pair; the excluded writable album's file never entered it"

echo "==> PATCH allowAppDeletion must gate the derived face (the setting drives the affordance)"
status="$(curl -sS -o "$work_dir/rw_patch_body" -b "$admin_jar" -w '%{http_code}' \
    -X PATCH -H 'Content-Type: application/json' \
    -d '{"allowAppDeletion":false}' \
    "$base_url/api/v1/albums/$rw_album_id")" \
    || fail "PATCH /api/v1/albums/$rw_album_id (deletable off) request failed"
[ "$status" = "200" ] \
    || fail "PATCH /api/v1/albums/$rw_album_id (deletable off) returned status $status, want 200: $(cat "$work_dir/rw_patch_body")"
status="$(curl -sS -o "$work_dir/rw_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$rw_album_id")" \
    || fail "GET /api/v1/albums/$rw_album_id (deletable off) request failed"
grep -q '"allowAppDeletion":false' "$work_dir/rw_detail_body" \
    || fail "the switch's off state did not land: $(cat "$work_dir/rw_detail_body")"
grep -q '"deletable":false' "$work_dir/rw_detail_body" \
    || fail "the derived face stayed true with the switch off (the setting must drive the affordance): $(cat "$work_dir/rw_detail_body")"
status="$(curl -sS -o "$work_dir/rw_patch_body" -b "$admin_jar" -w '%{http_code}' \
    -X PATCH -H 'Content-Type: application/json' \
    -d '{"allowAppDeletion":true}' \
    "$base_url/api/v1/albums/$rw_album_id")" \
    || fail "PATCH /api/v1/albums/$rw_album_id (deletable on) request failed"
[ "$status" = "200" ] \
    || fail "PATCH /api/v1/albums/$rw_album_id (deletable on) returned status $status, want 200: $(cat "$work_dir/rw_patch_body")"
status="$(curl -sS -o "$work_dir/rw_detail_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/albums/$rw_album_id")" \
    || fail "GET /api/v1/albums/$rw_album_id (deletable on) request failed"
grep -q '"deletable":true' "$work_dir/rw_detail_body" \
    || fail "the derived face did not return with the switch back on: $(cat "$work_dir/rw_detail_body")"
echo "    the switch off killed the derived face; the switch on returned it (the idempotent round trip)"

echo "==> POST /api/v1/assets/batch (trash) must MOVE the file (the OS's authority over a writable source)"
# The two-layer deletion story's execution arm: the setting above
# gates the AFFORDANCE only -- the trash itself answers to ownership
# and to the operating system's authority over the writable source
# (the switch stayed on through the PATCH round trip). No album
# state can freeze this: the trash soft-deletes the row with its
# trash path and physically relocates the file -- the source
# directory loses it, the managed trash home holds it.
status="$(curl -sS -o "$work_dir/rw_trash_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"trash\",\"ids\":[\"$rw_asset_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (trash) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (trash) returned status $status, want 200: $(cat "$work_dir/rw_trash_body")"
grep -q '"affected":1' "$work_dir/rw_trash_body" \
    || fail "the trash did not affect the one member: $(cat "$work_dir/rw_trash_body")"
grep -q '"missing":\[\]' "$work_dir/rw_trash_body" \
    || fail "the trash reported missing ids: $(cat "$work_dir/rw_trash_body")"
rw_trashed="$(psql_query "SELECT count(*) FROM assets WHERE id = '$rw_asset_id' AND deleted_at IS NOT NULL AND trash_path IS NOT NULL")"
[ "$rw_trashed" = "1" ] \
    || fail "the trashed row does not carry the soft delete and the trash path: $rw_trashed"
[ ! -f "$rw_source_host_dir/$rw_fixture_photo" ] \
    || fail "the source file did not leave the writable fixture (the trash move relocates it)"
[ -f "lumemor/data/trash/$rw_asset_id/$rw_fixture_photo" ] \
    || fail "the moved file did not land in the managed trash home"
echo "    the trash moved the file: the row carries the trash path, the source lost the file, the trash home holds it"

echo "==> POST /api/v1/trash/{id}/restore must move the file BACK and heal the fixture"
status="$(curl -sS -o "$work_dir/rw_restore_body" -b "$admin_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/trash/$rw_asset_id/restore")" \
    || fail "POST /api/v1/trash/$rw_asset_id/restore request failed"
[ "$status" = "204" ] \
    || fail "POST /api/v1/trash/$rw_asset_id/restore returned status $status, want 204: $(cat "$work_dir/rw_restore_body")"
rw_restored="$(psql_query "SELECT count(*) FROM assets WHERE id = '$rw_asset_id' AND deleted_at IS NULL")"
[ "$rw_restored" = "1" ] \
    || fail "the restore did not return the row to live: $rw_restored"
[ -f "$rw_source_host_dir/$rw_fixture_photo" ] \
    || fail "the restore did not move the file back to the source (the fixture must heal)"
echo "    the restore returned the row to live and the file to the source (the fixture healed)"

# ---------------------------------------------------------------------------
# The album removal (the un-ingest): the E8 semantics over the SAME
# writable source. A second directory album over the rw root makes
# its rescan link the first album's member (the owner+checksum dedup)
# AND ingest the removal's own seed fresh, so one removal exercises
# BOTH guard arms -- the fresh member purges whole, the borrowed
# member survives on the first album's hold. The source files stay
# byte for byte; the purged member's library data and derivative
# trees go; the events announce the removal.
# ---------------------------------------------------------------------------

echo "==> seeding the removal's own source photo (genuinely distinct bytes)"
[ -f "$removal_seed_template" ] \
    || fail "fixture missing: $removal_seed_template -- the removal beat's seed template"
cp "$removal_seed_template" "$rw_source_host_dir/$removal_fixture_photo"
echo "    the writable source fixture holds $rw_mount_root/$removal_fixture_photo beside the rw album's own"

echo "==> POST /api/v1/albums must create the removal's directory album over the same writable root"
# The affirm arm again (inTimeline true): the album feeds its
# creator's aggregated timeline, so the removal's library-empty pin
# has a timeline tile to lose.
status="$(curl -sS -o "$work_dir/removal_album_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"M3c Removal Album\",\"rootPath\":\"$rw_mount_root\",\"audience\":[],\"inTimeline\":true}" \
    "$base_url/api/v1/albums")" \
    || fail "POST /api/v1/albums (removal) request failed"
[ "$status" = "201" ] \
    || fail "POST /api/v1/albums (removal) returned status $status, want 201: $(cat "$work_dir/removal_album_body")"
grep -q '"sourceWritable":true' "$work_dir/removal_album_body" \
    || fail "the removal album's creation does not carry sourceWritable true: $(cat "$work_dir/removal_album_body")"
removal_album_id="$(sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$work_dir/removal_album_body")"
[ -n "$removal_album_id" ] \
    || fail "the removal album carries no id: $(cat "$work_dir/removal_album_body")"
echo "    the removal album exists ($removal_album_id) over $rw_mount_root"

echo "==> POST /api/v1/albums/{id}/rescan must converge the removal album's membership (both files)"
status="$(curl -sS -o "$work_dir/removal_rescan_body" -b "$admin_jar" -w '%{http_code}' \
    -X POST "$base_url/api/v1/albums/$removal_album_id/rescan")" \
    || fail "POST /api/v1/albums/$removal_album_id/rescan request failed"
[ "$status" = "202" ] \
    || fail "POST /api/v1/albums/$removal_album_id/rescan returned status $status, want 202: $(cat "$work_dir/removal_rescan_body")"
deadline="$(( $(date +%s) + rescan_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/jobs_body" -b "$admin_jar" -w '%{http_code}' \
        "$base_url/api/v1/admin/jobs")" \
        || fail "GET /api/v1/admin/jobs request failed"
    [ "$status" = "200" ] \
        || fail "GET /api/v1/admin/jobs returned status $status, want 200: $(cat "$work_dir/jobs_body")"
    row="$(rescan_row)"
    state="$(printf '%s\n' "$row" | sed -n 's/.*"state":"\([a-z]*\)".*/\1/p')"
    if [ "$state" = "succeeded" ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the removal album's rescan run did not succeed within ${rescan_timeout_seconds}s (last row: $row)"
    fi
    if [ "$state" = "failed" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the removal album's rescan run failed: $row (see the server logs above)"
    fi
    sleep 3
done
echo "    the removal album's rescan run succeeded: $row"

echo "==> the removal album's membership must carry both files: the dedup link and the fresh ingest"
status="$(curl -sS -o "$work_dir/removal_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline?albumId=$removal_album_id")" \
    || fail "GET /api/v1/assets/timeline (removal album) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (removal album) returned status $status, want 200: $(cat "$work_dir/removal_timeline_body")"
removal_album_ids="$(item_ids "$work_dir/removal_timeline_body" | sort | tr '\n' ' ')"
rw_in_removal="$(item_ids "$work_dir/removal_timeline_body" | grep -c "^$rw_asset_id$" || true)"
[ "$rw_in_removal" = "1" ] \
    || fail "the removal album's membership does not carry the rw album's member (the owner+checksum dedup must link the existing row): [$removal_album_ids]"
removal_asset_id="$(item_ids "$work_dir/removal_timeline_body" | grep -v "^$rw_asset_id$" | head -1)"
[ -n "$removal_asset_id" ] \
    || fail "the removal album's membership carries no fresh member: [$removal_album_ids]"
echo "    both members present: the borrowed $rw_asset_id (the dedup link) and the fresh $removal_asset_id"

echo "==> the fresh member's pipeline must run to ready (up to ${ml_timeout_seconds}s)"
deadline="$(( $(date +%s) + ml_timeout_seconds ))"
while :; do
    status="$(curl -sS -o "$work_dir/removal_timeline_body" -b "$admin_jar" -w '%{http_code}' \
        "$base_url/api/v1/assets/timeline?albumId=$removal_album_id")" \
        || fail "GET /api/v1/assets/timeline (removal album, ready) request failed"
    ready_count="$(grep -o '"isReady":true' "$work_dir/removal_timeline_body" | wc -l | tr -d ' ' || true)"
    if [ "$ready_count" = "2" ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the removal album's members did not become ready within ${ml_timeout_seconds}s (ready: $ready_count)"
    fi
    sleep 3
done

echo "==> the fresh member's face must cluster with the survivor's (up to ${ml_timeout_seconds}s)"
deadline="$(( $(date +%s) + ml_timeout_seconds ))"
while :; do
    shared_cluster="$(psql_query "SELECT count(*) FROM faces a JOIN faces b ON a.cluster_id = b.cluster_id WHERE a.asset_id = '$removal_asset_id' AND b.asset_id = '$rw_asset_id'")"
    if [ "$shared_cluster" -ge 1 ]; then
        break
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
        compose logs --no-color --tail 200 lumemor || true
        fail "the fresh member's face never clustered with the survivor's within ${ml_timeout_seconds}s"
    fi
    sleep 3
done
removal_cluster_id="$(psql_query "SELECT cluster_id FROM faces WHERE asset_id = '$removal_asset_id' AND cluster_id IS NOT NULL" | head -1)"
[ -n "$removal_cluster_id" ] \
    || fail "the fresh member's face carries no cluster id"
removal_faces_before="$(psql_query "SELECT count(*) FROM faces WHERE asset_id = '$removal_asset_id'")"
removal_embeds_before="$(psql_query "SELECT count(*) FROM clip_embeddings WHERE asset_id = '$removal_asset_id'")"
[ "$removal_faces_before" -ge 1 ] && [ "$removal_embeds_before" -ge 1 ] \
    || fail "the fresh member's recognition did not land ($removal_faces_before faces, $removal_embeds_before embeddings)"
echo "    both members ready; the fresh member's face shares cluster $removal_cluster_id with the survivor's ($removal_faces_before face rows, $removal_embeds_before embeddings)"

echo "==> the pre-removal faces must be pinned: the timeline tile, the search hit, the favorite, the derivative tree"
status="$(curl -sS -o "$work_dir/favorite_body" -b "$admin_jar" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"action\":\"favorite\",\"ids\":[\"$removal_asset_id\"]}" \
    "$base_url/api/v1/assets/batch")" \
    || fail "POST /api/v1/assets/batch (favorite) request failed"
[ "$status" = "200" ] \
    || fail "POST /api/v1/assets/batch (favorite) returned status $status, want 200: $(cat "$work_dir/favorite_body")"
grep -q '"affected":1' "$work_dir/favorite_body" \
    || fail "the favorite did not affect the one member: $(cat "$work_dir/favorite_body")"
favorite_rows="$(psql_query "SELECT count(*) FROM user_favorites WHERE asset_id = '$removal_asset_id'")"
[ "$favorite_rows" = "1" ] \
    || fail "the favorite did not land a per-user row ($favorite_rows rows, want 1)"
[ -d "lumemor/data/storage/thumbs/$removal_asset_id" ] \
    || fail "the fresh member's thumbnail tree is absent before the removal (the pipeline must have made it)"
removal_encoded_before=0
[ -d "lumemor/data/storage/encoded/$removal_asset_id" ] && removal_encoded_before=1
[ -d "lumemor/data/storage/thumbs/$rw_asset_id" ] \
    || fail "the survivor's thumbnail tree is absent before the removal"
status="$(curl -sS -o "$work_dir/admin_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (owner, pre-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (owner, pre-removal) returned status $status, want 200: $(cat "$work_dir/admin_timeline_body")"
pre_removal_count="$(item_ids "$work_dir/admin_timeline_body" | wc -l | tr -d ' ')"
[ "$pre_removal_count" = "4" ] \
    || fail "the owner's pre-removal timeline holds $pre_removal_count items, want 4 (the read-only pair, the borrowed member, the fresh member)"
item_ids "$work_dir/admin_timeline_body" | grep -q "^$removal_asset_id$" \
    || fail "the owner's pre-removal timeline does not carry the fresh member's tile"
status="$(curl -sS -o "$work_dir/pre_removal_search_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/search?q=$removal_fixture_photo")" \
    || fail "GET /api/v1/search (pre-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/search (pre-removal) returned status $status, want 200: $(cat "$work_dir/pre_removal_search_body")"
item_ids "$work_dir/pre_removal_search_body" | grep -q "^$removal_asset_id$" \
    || fail "the pre-removal search does not answer the fresh member's row: $(cat "$work_dir/pre_removal_search_body")"
status="$(curl -sS -o "$work_dir/pre_removal_unnamed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (pre-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (pre-removal) returned status $status, want 200: $(cat "$work_dir/pre_removal_unnamed_body")"
grep -q "\"clusterId\":\"$removal_cluster_id\"" "$work_dir/pre_removal_unnamed_body" \
    || fail "the admin's unnamed listing does not carry the fresh member's cluster"
echo "    the tile feeds the timeline, the search answers, the favorite row stands, the derivative tree exists (cluster $removal_cluster_id listed)"

echo "==> GET /api/v1/events must announce the removal (album.deleted + asset.deleted, the survivor silent)"
# The cookie-authenticated stream runs in the background; the
# retention ring's replay lands bytes immediately (the connectivity
# proof), then the removal fires and the poll waits for the event
# pair. The survivor's silence is the events pin's other half: the
# guard kept its row, so nothing announces it.
curl -sS -N --max-time 120 -b "$admin_jar" \
    "$base_url/api/v1/events" >"$work_dir/removal_sse_stream" 2>"$work_dir/removal_sse_err" &
removal_sse_pid=$!
sse_deadline="$(( $(date +%s) + 30 ))"
until [ -s "$work_dir/removal_sse_stream" ]; do
    if [ "$(date +%s)" -ge "$sse_deadline" ]; then
        kill "$removal_sse_pid" 2>/dev/null || true
        wait "$removal_sse_pid" 2>/dev/null || true
        removal_sse_pid=""
        fail "the event stream stayed empty for 30s (the retention ring's replay never landed)"
    fi
    sleep 1
done
status="$(curl -sS -o "$work_dir/removal_delete_body" -b "$admin_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$removal_album_id")" \
    || fail "DELETE /api/v1/albums/$removal_album_id (admin) request failed"
[ "$status" = "204" ] \
    || fail "DELETE /api/v1/albums/$removal_album_id (admin) returned status $status, want 204 (the un-ingest): $(cat "$work_dir/removal_delete_body")"
deadline="$(( $(date +%s) + sse_timeout_seconds ))"
until awk -v album="$removal_album_id" -v asset="$removal_asset_id" '
    $0 == "event: album.deleted" {
        getline data
        if (data ~ "\"albumId\":\"" album "\"") found_album = 1
    }
    $0 == "event: asset.deleted" {
        getline data
        if (data ~ "\"assetId\":\"" asset "\"") found_asset = 1
    }
    END { exit(found_album && found_asset ? 0 : 1) }
' "$work_dir/removal_sse_stream"; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        kill "$removal_sse_pid" 2>/dev/null || true
        wait "$removal_sse_pid" 2>/dev/null || true
        removal_sse_pid=""
        fail "the removal's event pair was not delivered within ${sse_timeout_seconds}s; stream so far: $(cat "$work_dir/removal_sse_stream")"
    fi
    if ! kill -0 "$removal_sse_pid" 2>/dev/null; then
        kill "$removal_sse_pid" 2>/dev/null || true
        wait "$removal_sse_pid" 2>/dev/null || true
        removal_sse_pid=""
        fail "GET /api/v1/events curl exited before the removal's events arrived: $(cat "$work_dir/removal_sse_err") stream: $(cat "$work_dir/removal_sse_stream")"
    fi
    sleep 1
done
if awk -v survivor="$rw_asset_id" '
    $0 == "event: asset.deleted" {
        getline data
        if (data ~ "\"assetId\":\"" survivor "\"") found = 1
    }
    END { exit(found ? 0 : 1) }
' "$work_dir/removal_sse_stream"; then
    kill "$removal_sse_pid" 2>/dev/null || true
    wait "$removal_sse_pid" 2>/dev/null || true
    removal_sse_pid=""
    fail "the guarded survivor announced asset.deleted (the guard kept its row; nothing may announce it)"
fi
kill "$removal_sse_pid" 2>/dev/null || true
wait "$removal_sse_pid" 2>/dev/null || true
removal_sse_pid=""
echo "    album.deleted + asset.deleted delivered; the guarded survivor announced nothing"

echo "==> the removal must answer the single-shape 404 to the repeat delete"
status="$(curl -sS -o "$work_dir/removal_repeat_body" -b "$admin_jar" -w '%{http_code}' \
    -X DELETE "$base_url/api/v1/albums/$removal_album_id")" \
    || fail "DELETE /api/v1/albums/$removal_album_id (repeat) request failed"
[ "$status" = "404" ] \
    || fail "the repeat delete returned status $status, want 404 (the album is gone exactly like an unknown id): $(cat "$work_dir/removal_repeat_body")"
grep -q 'ALBUM_NOT_FOUND' "$work_dir/removal_repeat_body" \
    || fail "the repeat delete's refusal does not answer ALBUM_NOT_FOUND: $(cat "$work_dir/removal_repeat_body")"
status="$(curl -sS -o /dev/null -b "$admin_jar" -w '%{http_code}' "$base_url/api/v1/albums/$removal_album_id")" \
    || fail "GET /api/v1/albums/$removal_album_id (removed) request failed"
[ "$status" = "404" ] \
    || fail "the removed album's detail returned status $status, want 404"
echo "    the repeat delete and the detail both answer the single-shape 404"

echo "==> the database must hold zero rows of the purge (the cascade) and every row of the survivors"
removal_album_rows="$(psql_query "SELECT count(*) FROM albums WHERE id = '$removal_album_id'")"
[ "$removal_album_rows" = "0" ] || fail "the removal album's row survived ($removal_album_rows rows, want 0)"
removal_item_rows="$(psql_query "SELECT count(*) FROM album_items WHERE album_id = '$removal_album_id'")"
[ "$removal_item_rows" = "0" ] || fail "the removal album's membership rows survived ($removal_item_rows rows, want 0)"
removal_asset_rows="$(psql_query "SELECT count(*) FROM assets WHERE id = '$removal_asset_id'")"
[ "$removal_asset_rows" = "0" ] || fail "the purged member's asset row survived ($removal_asset_rows rows, want 0)"
removal_faces="$(psql_query "SELECT count(*) FROM faces WHERE asset_id = '$removal_asset_id'")"
[ "$removal_faces" = "0" ] || fail "the purged member's faces survived ($removal_faces rows, want 0)"
removal_embeds="$(psql_query "SELECT count(*) FROM clip_embeddings WHERE asset_id = '$removal_asset_id'")"
[ "$removal_embeds" = "0" ] || fail "the purged member's embedding survived ($removal_embeds rows, want 0)"
removal_ocr="$(psql_query "SELECT count(*) FROM ocr_texts WHERE asset_id = '$removal_asset_id'")"
[ "$removal_ocr" = "0" ] || fail "the purged member's OCR text survived ($removal_ocr rows, want 0)"
removal_favorites="$(psql_query "SELECT count(*) FROM user_favorites WHERE asset_id = '$removal_asset_id'")"
[ "$removal_favorites" = "0" ] || fail "the purged member's favorite row survived ($removal_favorites rows, want 0)"
cluster_faces="$(psql_query "SELECT count(*) FROM faces WHERE cluster_id = '$removal_cluster_id'")"
[ "$cluster_faces" = "1" ] \
    || fail "the shared cluster did not survive on the survivor's face ($cluster_faces faces, want exactly 1)"
purged_in_cluster="$(psql_query "SELECT count(*) FROM faces WHERE cluster_id = '$removal_cluster_id' AND asset_id = '$removal_asset_id'")"
[ "$purged_in_cluster" = "0" ] \
    || fail "the purged member's face still names the surviving cluster ($purged_in_cluster rows, want 0)"
rw_row="$(psql_query "SELECT count(*) FROM assets WHERE id = '$rw_asset_id' AND deleted_at IS NULL")"
[ "$rw_row" = "1" ] || fail "the guarded survivor's asset row did not survive ($rw_row rows, want 1)"
rw_hold="$(psql_query "SELECT count(*) FROM album_items WHERE album_id = '$rw_album_id' AND asset_id = '$rw_asset_id'")"
[ "$rw_hold" = "1" ] || fail "the first album's hold on its member did not survive ($rw_hold rows, want 1)"
echo "    the purge took the album, its membership, the asset row, the faces, the embedding, the OCR text and the favorite"
echo "    the cluster survives on the survivor's face (the re-home arm), the survivor's row and the first album's hold stand"

echo "==> the host filesystem must be the judge: the sources stay byte for byte, the purged derivative trees are gone"
cmp -s "$removal_seed_template" "$rw_source_host_dir/$removal_fixture_photo" \
    || fail "the removal touched the purged member's source file (the zero-contact pin: the bytes must be identical)"
cmp -s "$rw_seed_template" "$rw_source_host_dir/$rw_fixture_photo" \
    || fail "the removal touched the survivor's source file (the zero-contact pin: the bytes must be identical)"
[ ! -d "lumemor/data/storage/thumbs/$removal_asset_id" ] \
    || fail "the purged member's thumbnail tree survived on the host"
if [ "$removal_encoded_before" = "1" ]; then
    [ ! -d "lumemor/data/storage/encoded/$removal_asset_id" ] \
        || fail "the purged member's encoded tree survived on the host"
fi
[ -d "lumemor/data/storage/thumbs/$rw_asset_id" ] \
    || fail "the survivor's thumbnail tree did not survive (the guard keeps the files the row still names)"
echo "    both source files byte-identical; the purged derivative trees gone, the survivor's trees stand"

echo "==> the library's fresh answers must drop the purged member and keep the survivor"
status="$(curl -sS -o "$work_dir/admin_timeline_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/assets/timeline")" \
    || fail "GET /api/v1/assets/timeline (owner, post-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/assets/timeline (owner, post-removal) returned status $status, want 200: $(cat "$work_dir/admin_timeline_body")"
post_removal_ids="$(item_ids "$work_dir/admin_timeline_body" | sort | tr '\n' ' ')"
ro_pair_sorted="$(printf '%s\n' "$shared_a_id" "$shared_b_id" | sort | tr '\n' ' ')"
[ "$post_removal_ids" = "$ro_pair_sorted" ] \
    || fail "the owner's post-removal timeline holds [$post_removal_ids], want exactly the read-only pair [$ro_pair_sorted] (the purged member's tile must fall, the survivor's must stay out through the exclusion)"
status="$(curl -sS -o "$work_dir/post_removal_search_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/search?q=$removal_fixture_photo")" \
    || fail "GET /api/v1/search (post-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/search (post-removal) returned status $status, want 200: $(cat "$work_dir/post_removal_search_body")"
post_search_count="$(item_ids "$work_dir/post_removal_search_body" | wc -l | tr -d ' ')"
[ "$post_search_count" = "0" ] \
    || fail "the post-removal search still answers the purged member's name ($post_search_count hits, want 0)"
status="$(curl -sS -o "$work_dir/post_removal_unnamed_body" -b "$admin_jar" -w '%{http_code}' \
    "$base_url/api/v1/people/unnamed")" \
    || fail "GET /api/v1/people/unnamed (post-removal) request failed"
[ "$status" = "200" ] \
    || fail "GET /api/v1/people/unnamed (post-removal) returned status $status, want 200: $(cat "$work_dir/post_removal_unnamed_body")"
grep -q "\"clusterId\":\"$removal_cluster_id\"" "$work_dir/post_removal_unnamed_body" \
    || fail "the surviving cluster fell out of the unnamed region (the purge took the faces with it -- the survivor's face must keep the cluster listed)"
echo "    the timeline holds exactly the read-only pair, the search dropped the purged row, the surviving cluster still lists"

if compose logs --no-color lumemor 2>/dev/null | grep -q 'panic'; then
    fail "the server panicked during the sharing phase (see compose logs)"
fi
teardown_stack
echo "    phase 1 passed after $(elapsed)s"

# ---------------------------------------------------------------------------
# Phase 2: the e2e stack (the sharing story of the web app).
# ---------------------------------------------------------------------------

echo "==> phase 2/2: the e2e stack (the sharing story of the web app)"
reset_stack_state
prepare_bind_mount
start_stack
probe_offline_mirror

echo "==> running the Playwright spec (web/e2e/m3-albums.spec.ts)"
if ! (cd web && pnpm e2e e2e/m3-albums.spec.ts); then
    echo "==> server logs at the time of the failure:"
    compose logs --no-color --tail 200 lumemor || true
    fail "the sharing e2e spec failed (see the report above; artifacts under web/test-results/)"
fi
teardown_stack
echo "    phase 2 passed after $(elapsed)s"

echo "==> acceptance flow completed in $(elapsed)s"
echo "M3C ACCEPTANCE PASS"
