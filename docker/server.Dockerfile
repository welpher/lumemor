# Multi-stage image for the lumemor server.
#
# Builder stage: compile a pure-Go static binary on the official golang
# image. CGO is disabled because M1 has no native dependencies; the
# M2/M3 media and ML pipeline (libvips, onnxruntime) will switch this
# stage to a CGO build and the runtime below to a Debian base carrying
# those libraries together with ffmpeg and exiftool.
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

WORKDIR /src

# Cache module downloads separately from the source tree so that
# source-only changes reuse the module layer.
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w -X github.com/welpher/lumemor/server/internal/buildinfo.Version=${VERSION}" \
        -o /out/lumemor \
        ./cmd/lumemor

# Runtime stage: distroless static is a shell-less, package-manager-less
# base that fits the pure-Go binary exactly. Revisit together with the
# CGO switch above.
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/lumemor /lumemor

# The reserved unprivileged account (uid 65532) ships with every
# distroless image. The server needs no privileges: it binds port 8244
# and keeps all state in PostgreSQL.
USER nonroot:nonroot

EXPOSE 8244

ENTRYPOINT ["/lumemor"]
