# Development and test image for the lumemor server.
#
# The host stays free of media dependencies: everything the server and
# its tests need lives here -- the Go toolchain, libvips and libheif
# headers for the CGO thumbnail pipeline, exiftool and ffmpeg for the
# metadata pipeline, golangci-lint so the lint gate runs where the CGO
# package compiles, and the Docker CLI so testcontainers can start
# throwaway PostgreSQL containers through the daemon socket mounted
# from the host.
#
# The image is intentionally generic: no lumemor sources are baked in.
# `make docker-test` bind-mounts the repository over /src and runs the
# test suite inside, so the image only carries the toolchain and system
# packages, and nothing environment-specific is committed.
FROM golang:1.26

# Never download another toolchain inside the image: its Go version
# satisfies the go directive in server/go.mod.
ENV GOTOOLCHAIN=local

# Optional apt mirror override (a host name such as
# mirrors.tuna.tsinghua.edu.cn) for networks where the default
# deb.debian.org is slow or unreachable. Empty keeps upstream Debian;
# the value is passed at build time by make docker-test, never baked
# into the committed image.
ARG APT_MIRROR=""

# golangci-lint is pinned to the version the CI workflow runs, so the
# containerized lint gate and CI agree. go install builds it from the
# module proxy (the image has no curl); the GOPROXY build arg is passed
# at build time from the host's go env, like the Makefile's docker run
# flags -- the committed image itself carries no proxy configuration.
ARG GOLANGCI_LINT_VERSION=v2.15.0
ARG GOPROXY=""

# Media and container tooling:
# - libvips-dev / libheif-dev: headers for the CGO thumbnail pipeline;
# - libheif-plugin-aomenc / libheif-plugin-x265: the libheif ENCODER
#   plugins (AVIF for the serving copies, HEVC to regenerate the HEIC
#   test fixture). They are only Recommends of libheif1, so
#   --no-install-recommends excludes them unless named here, and
#   without them every AVIF encode fails with "heifsave: Unsupported
#   compression". The decoder plugin libheif-plugin-libde265 rides a
#   hard Depends of libheif1 today but is named explicitly anyway --
#   the install line then mirrors the CI workflow's and stays correct
#   if that dependency ever relaxes (dav1d, the AVIF decoder, stays a
#   hard Depends alternative);
# - exiftool / ffmpeg: the metadata and video pipelines' subprocesses;
# - docker-cli: the Docker CLI. (Debian trixie split the CLI out of
#   docker.io, which now ships only the engine; the engine stays on
#   the host, and the socket mount is how the CLI and testcontainers
#   reach it.)
RUN if [ -n "$APT_MIRROR" ]; then \
        sed -i "s|deb.debian.org|$APT_MIRROR|g" /etc/apt/sources.list.d/debian.sources; \
    fi \
    && apt-get update && apt-get install -y --no-install-recommends \
        libvips-dev \
        libheif-dev \
        libheif-plugin-aomenc \
        libheif-plugin-x265 \
        libheif-plugin-libde265 \
        exiftool \
        ffmpeg \
        docker-cli \
    && rm -rf /var/lib/apt/lists/* \
    && GOPROXY="${GOPROXY}" GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_LINT_VERSION

# The default working directory matches the repository layout mounted
# at /src by `make docker-test`: the Go module lives in server/.
WORKDIR /src/server
