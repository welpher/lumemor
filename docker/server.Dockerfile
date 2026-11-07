# Multi-stage image for the lumemor server.
#
# Web builder stage: build the SvelteKit SPA the Go binary embeds. A
# clean checkout carries no staged server/internal/web/dist (it is
# git-ignored by design), so the image builds the app itself instead
# of trusting whatever the checkout happens to carry -- `docker
# compose up --build` on a fresh clone works, and no stale host build
# can silently ride into the image. The Go builder stage below stages
# the output into the embed path and compiles the binary on the
# official golang image. The media pipeline (govips) links libvips and
# libheif through cgo, so that stage carries their dev headers and
# builds with CGO enabled. The runtime stage runs on the same Debian
# release as the Go builder (golang:1.26 is a trixie image), so the
# shared libraries the binary resolves at load time are the exact
# versions it was compiled against -- bookworm's libvips/libheif
# major-minor would differ.
FROM node:22 AS webbuilder

# Optional npm registry override (a URL such as
# https://registry.npmmirror.com) for networks where the default
# registry.npmjs.org is slow or unreachable -- the same pattern as the
# GOPROXY/APT_MIRROR args of the Go stage below. Empty keeps upstream
# npm; the value is passed at build time, never baked into the
# committed image. When set it also points corepack's own pnpm
# download at the mirror.
ARG PNPM_REGISTRY=""

# The app pins pnpm through web/package.json's packageManager field
# (pnpm@9.15.0); the corepack shim that ships with node:22 activates
# exactly that version, so the image never picks a drifting pnpm.
WORKDIR /web

# The build's input surface beyond web/ itself: the shared translation
# table at the repository root (read through the relative
# import.meta.glob of src/lib/i18n) and the generated TypeScript
# contract types. Both live one level above web/ in the repository,
# and the relative paths resolve the same way inside the image.
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable \
    && if [ -n "$PNPM_REGISTRY" ]; then \
        export COREPACK_NPM_REGISTRY="$PNPM_REGISTRY"; \
        pnpm install --frozen-lockfile --registry "$PNPM_REGISTRY"; \
    else \
        pnpm install --frozen-lockfile; \
    fi

COPY web/ ./
COPY i18n/ /i18n/
COPY openapi/generated/web/ /openapi/generated/web/
RUN pnpm run build

FROM golang:1.26 AS builder

# Version stamped into the binary through the linker; release builds
# pass their tag, everything else keeps the dev placeholder.
ARG VERSION=dev

# GOPROXY forwards the module proxy of the build environment into the
# container; the empty default leaves the go toolchain default in place.
ARG GOPROXY=""
ENV GOPROXY=${GOPROXY}

# Never download another toolchain inside the image: the go directive in
# server/go.mod is satisfied by the golang:1.26 image itself.
ENV GOTOOLCHAIN=local

# Optional apt mirror override (a host name such as
# mirrors.tuna.tsinghua.edu.cn) for networks where the default
# deb.debian.org is slow or unreachable -- the same pattern as
# docker/dev.Dockerfile. Empty keeps upstream Debian; the value is
# passed at build time, never baked into the committed image.
# The value is a bare host name, never a URL with a scheme: the sed swaps the deb.debian.org host inside URIs that already carry http://.
ARG APT_MIRROR=""

# The cgo build needs the headers of the libraries the runtime stage
# carries: libvips-dev for the thumbnail pipeline, libheif-dev for the
# HEIC decode and AVIF encode, and pkg-config to locate them (the
# golang image already ships pkg-config; naming it keeps the line
# correct even if the base image ever drops it). curl and unzip join
# the list for the onnxruntime and GeoNames fetches below.
RUN if [ -n "$APT_MIRROR" ]; then \
        sed -i "s|deb.debian.org|$APT_MIRROR|g" /etc/apt/sources.list.d/debian.sources; \
    fi \
    && apt-get update && apt-get install -y --no-install-recommends \
        libvips-dev \
        libheif-dev \
        pkg-config \
        curl \
        unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Cache module downloads separately from the source tree so that
# source-only changes reuse the module layer.
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
# The SPA from the web builder stage, staged exactly where the embed
# directive of server/internal/web picks it up: the staged tree is the
# image's own build, never the checkout's -- the checkout's copy (if
# any) was excluded by .dockerignore before COPY server/ ran.
COPY --from=webbuilder /web/build ./internal/web/dist
RUN CGO_ENABLED=1 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w -X github.com/welpher/lumemor/server/internal/buildinfo.Version=${VERSION}" \
        -o /out/lumemor \
        ./cmd/lumemor

# The onnxruntime shared library, fetched and verified here so the
# runtime stage below receives exactly the bytes this build checked.
# The version pairing is pinned: the CGO binding (yalue/onnxruntime_go,
# pinned in server/go.mod) v1.38.0 bundles the onnxruntime 1.31.0 C API
# headers (ORT_API_VERSION 31), so the library is the 1.31.0 release --
# keep the two in lockstep when either is bumped. Nothing links against
# it at build time (the binding dlopens it at run time); the fetch rides
# the builder stage only because it already carries curl.
#
# ONNXRUNTIME_URL overrides the default GitHub release URL for
# restricted networks (a mirror of the same bytes; the
# per-architecture sha256 stays pinned either way, so anything but the
# release bytes fails the build). The download can go through a proxy
# via the predefined https_proxy build arg.
ARG ONNXRUNTIME_URL=""
ARG TARGETARCH
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) ort_arch="x64"; ort_sha256="cc5c72baf5ae5c8238a6841f0897227be2d02826b9cf98eaf02fdefaeeb03a57";; \
        arm64) ort_arch="aarch64"; ort_sha256="c5b8b3cca31f3d643a3b313b8f42e2d0f16a4fcefd02cda343d97bd5306afaab";; \
        *) echo "unsupported architecture for libonnxruntime: ${TARGETARCH}" >&2; exit 1;; \
    esac; \
    ort_url="${ONNXRUNTIME_URL:-https://github.com/microsoft/onnxruntime/releases/download/v1.31.0/onnxruntime-linux-${ort_arch}-1.31.0.tgz}"; \
    curl -fsSL --retry 3 -o /tmp/onnxruntime.tgz "$ort_url"; \
    echo "${ort_sha256}  /tmp/onnxruntime.tgz" | sha256sum -c -; \
    tar -xzf /tmp/onnxruntime.tgz -C /tmp; \
    mv "/tmp/onnxruntime-linux-${ort_arch}-1.31.0/lib/libonnxruntime.so.1.31.0" /out/; \
    rm -rf /tmp/onnxruntime.tgz "/tmp/onnxruntime-linux-${ort_arch}-1.31.0"

# The GeoNames reference data (M3d places), fetched and verified here
# so the runtime stage below receives exactly the bytes this build
# checked -- the places distribution discipline: build-time bake,
# sha256-pinned, ZERO runtime network. The three sources:
#
# - cities500.zip: every populated place of population >= 500
#   (~237k rows, the resolution grid's centroids);
# - admin1CodesASCII.txt: the first-level administrative divisions
#   (the admin1_code column of places and the locale names of the
#   admin1 entities);
# - alternateNamesV2.zip: the multilingual names. Only the zh rows
#   of the known geoname ids survive the bake (the en display names
#   come from the canonical names by the data model's uniform rule,
#   so the en alternates would be dead weight; the loader re-applies
#   both filters, so the staged subset parses to exactly the dataset
#   the unfiltered dump would).
#
# The staged result under /out/geo is what the loader reads: the
# server's startup import replaces the places/place_names tables with
# it when its data-version marker (the sha256 digest of the three
# staged files) changes, and the in-memory resolution grid builds
# from the imported rows -- no runtime download, no external call,
# the zero-cloud promise kept. The dumps update upstream over time:
# a changed dump fails the pinned digest on the next build (drift
# detection, same as the model pins), and re-pinning is the
# maintainer's deliberate act. GeoNames data is CC BY 4.0
# (attribution: geonames.org).
#
# GEONAMES_MIRROR overrides the base URL (https://download.geonames
# .org/export/dump) for restricted networks -- a mirror of the same
# byte paths; the per-file sha256 stays pinned either way, so
# anything but the pinned bytes fails the build. The download can go
# through a proxy via the predefined https_proxy build arg (same as
# the onnxruntime fetch above).
ARG GEONAMES_MIRROR=""
RUN set -eux; \
    geo_base="https://download.geonames.org/export/dump"; \
    if [ -n "$GEONAMES_MIRROR" ]; then geo_base="${GEONAMES_MIRROR}"; fi; \
    mkdir -p /out/geo /tmp/geo; \
    curl -fsSL --retry 3 -o /tmp/geo/cities500.zip "${geo_base}/cities500.zip"; \
    echo "bf2516f404ceefd24e099e48c8cd91730c9b1ab85158dd77eb9d943275d6b072  /tmp/geo/cities500.zip" | sha256sum -c -; \
    curl -fsSL --retry 3 -o /out/geo/admin1CodesASCII.txt "${geo_base}/admin1CodesASCII.txt"; \
    echo "2e4867f5397acb056ea26377601b57af37cb4c81399770225018e9e439e91ad3  /out/geo/admin1CodesASCII.txt" | sha256sum -c -; \
    curl -fsSL --retry 3 -o /tmp/geo/alternateNamesV2.zip "${geo_base}/alternateNamesV2.zip"; \
    alt_bytes=$(stat -c%s /tmp/geo/alternateNamesV2.zip); \
    if [ "$alt_bytes" -lt 100000000 ] || [ "$alt_bytes" -gt 400000000 ]; then \
        echo "alternateNamesV2.zip size $alt_bytes outside the sanity bound (100-400 MB)" >&2; exit 1; \
    fi; \
    unzip -tq /tmp/geo/alternateNamesV2.zip > /dev/null; \
    unzip -o -q /tmp/geo/cities500.zip -d /out/geo; \
    unzip -o -q /tmp/geo/alternateNamesV2.zip -d /tmp/geo; \
    awk -F'\t' '{print $1}' /out/geo/cities500.txt > /tmp/geo/ids.txt; \
    awk -F'\t' 'NF >= 4 && $4 ~ /^[0-9]+$/ {print $4}' /out/geo/admin1CodesASCII.txt >> /tmp/geo/ids.txt; \
    sort -u /tmp/geo/ids.txt -o /tmp/geo/ids.txt; \
    awk -F'\t' 'NR == FNR {ids[$1]; next} $3 == "zh" && ($2 in ids)' /tmp/geo/ids.txt /tmp/geo/alternateNamesV2.txt > /out/geo/alternateNames-zh.txt; \
    rm -rf /tmp/geo

# Runtime stage: Debian slim of the same release as the builder,
# carrying the full media toolchain the server drives:
#
# - libvips42t64: the shared library of the cgo thumbnail pipeline
#   (trixie's 64-bit time_t transition renamed bookworm's libvips42);
# - libheif1 with its codec plugins: aomenc encodes the AVIF serving
#   copies (only a Recommends of libheif1, so --no-install-recommends
#   drops it unless named -- without it every encode fails with
#   "heifsave: Unsupported compression"); libde265 decodes HEIC
#   originals (a hard Depends today, named explicitly so the line
#   stays correct if that relaxes); x265 encodes HEVC and rides
#   along for codec parity with the dev/CI toolchain;
# - ffmpeg: the video poster extraction;
# - exiftool: the metadata pipeline's subprocess;
# - util-linux: prlimit(1), which the metadata and poster jobs shell
#   out to for the subprocess memory caps. It is a Priority: required
#   package (present in the base image already); it is named
#   explicitly because the jobs fail without it;
# - ca-certificates: the M1 distroless/static runtime shipped a CA
#   bundle; keeping one preserves the image's outbound-TLS capability
#   (a database with sslmode != disable, future model downloads).
FROM debian:trixie-slim

# Same optional mirror override as the builder stage.
# The value is a bare host name, never a URL with a scheme: the sed swaps the deb.debian.org host inside URIs that already carry http://.
ARG APT_MIRROR=""

RUN if [ -n "$APT_MIRROR" ]; then \
        sed -i "s|deb.debian.org|$APT_MIRROR|g" /etc/apt/sources.list.d/debian.sources; \
    fi \
    && apt-get update && apt-get install -y --no-install-recommends \
        libvips42t64 \
        libheif1 \
        libheif-plugin-aomenc \
        libheif-plugin-x265 \
        libheif-plugin-libde265 \
        ffmpeg \
        exiftool \
        util-linux \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/lumemor /lumemor

# The onnxruntime shared library from the builder stage: the exact
# bytes the builder verified (same release version as the binding's
# pinned headers -- see the builder stage's fetch block). The ml
# package's library probe finds it under /usr/local/lib after
# ldconfig.
COPY --from=builder /out/libonnxruntime.so.1.31.0 /usr/local/lib/libonnxruntime.so.1.31.0
RUN ldconfig

# The GeoNames reference data from the builder stage: the exact bytes
# the builder verified and filtered (cities500, admin1 codes, the zh
# alternate names of the known ids -- see the builder stage's bake
# block). LUMEMOR_GEODATA_DIR points the server's startup load at it:
# the idempotent import lands the places/place_names rows and the
# in-memory resolution grid builds over them. The local development
# default of the variable is empty (the feature honestly off); this
# image always carries the data, so the runtime always resolves.
COPY --from=builder /out/geo /usr/local/share/lumemor/geo
ENV LUMEMOR_GEODATA_DIR=/usr/local/share/lumemor/geo

# Fixed unprivileged uid/gid 10001. The fixed (not auto-allocated) id
# is what makes the documented host-side preparation of the bind
# mount stable across hosts and rebuilds: docker-compose.yml pins the
# fresh-install one-liner (mkdir plus chown 10001:10001). The home
# directory matches the storage root; the account gets no login shell.
RUN useradd --uid 10001 --user-group --home-dir /var/lib/lumemor --no-create-home --shell /usr/sbin/nologin lumemor

# The server needs no privileges: it binds port 8244, keeps all state
# under /var/lib/lumemor and talks to PostgreSQL over the network.
USER lumemor:lumemor

EXPOSE 8244

ENTRYPOINT ["/lumemor"]
