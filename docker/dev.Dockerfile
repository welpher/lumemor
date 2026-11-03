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
# - curl: the onnxruntime fetch below (the base image ships none).
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
        curl \
    && rm -rf /var/lib/apt/lists/* \
    && GOPROXY="${GOPROXY}" GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_LINT_VERSION

# The onnxruntime shared library for the in-process ML inference: the
# CGO binding (yalue/onnxruntime_go, pinned in server/go.mod) loads it
# with dlopen at run time -- nothing links against it at build time,
# so installing it here is all the tests need. The version pairing is
# pinned: binding v1.38.0 bundles the onnxruntime 1.31.0 C API headers
# (ORT_API_VERSION 31), so the library is the 1.31.0 release -- keep
# the two in lockstep when either is bumped.
#
# The tgz from the GitHub release carries only the versioned
# libonnxruntime.so.1.31.0; it goes to /usr/local/lib and ldconfig,
# where the ml package's library probe finds it. ONNXRUNTIME_URL
# overrides the default release URL for restricted networks (a mirror
# of the same bytes; the per-architecture sha256 stays pinned either
# way, so anything but the release bytes fails the build). The
# download itself can go through a proxy by passing the predefined
# https_proxy build arg; its value must be reachable from the build
# container -- http://host.docker.internal:7890 on a desktop runtime,
# not 127.0.0.1 (the Makefile passes it as ONNXRUNTIME_PROXY).
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
    mv "/tmp/onnxruntime-linux-${ort_arch}-1.31.0/lib/libonnxruntime.so.1.31.0" /usr/local/lib/; \
    rm -rf /tmp/onnxruntime.tgz "/tmp/onnxruntime-linux-${ort_arch}-1.31.0"; \
    ldconfig

# The default working directory matches the repository layout mounted
# at /src by `make docker-test`: the Go module lives in server/.
WORKDIR /src/server
