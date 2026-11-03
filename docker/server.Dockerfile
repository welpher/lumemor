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
# correct even if the base image ever drops it).
RUN if [ -n "$APT_MIRROR" ]; then \
        sed -i "s|deb.debian.org|$APT_MIRROR|g" /etc/apt/sources.list.d/debian.sources; \
    fi \
    && apt-get update && apt-get install -y --no-install-recommends \
        libvips-dev \
        libheif-dev \
        pkg-config \
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
