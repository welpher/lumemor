# Development and test image for the lumemor server.
#
# The host stays free of media dependencies: everything the server and
# its tests need lives here -- the Go toolchain, libvips and libheif
# headers for the CGO thumbnail pipeline, exiftool and ffmpeg for the
# metadata pipeline, and the Docker CLI so testcontainers can start
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

# Media and container tooling:
# - libvips-dev / libheif-dev: headers for the CGO thumbnail pipeline;
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
        exiftool \
        ffmpeg \
        docker-cli \
    && rm -rf /var/lib/apt/lists/*

# The default working directory matches the repository layout mounted
# at /src by `make docker-test`: the Go module lives in server/.
WORKDIR /src/server
