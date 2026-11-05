# Root Makefile for lumemor development tasks.

# sqlc is pinned so that regenerating the committed code under
# server/internal/db/gen is deterministic; bump deliberately.
SQLC_VERSION ?= v1.31.1

# API toolchain versions are pinned for the same reason: regenerating the
# committed artifacts under server/internal/api/gen and openapi/generated
# must be byte-identical; bump deliberately. The openapi-generator (Java)
# release is pinned separately by openapitools.json in the repo root.
OAPI_CODEGEN_VERSION ?= v2.8.1
OPENAPI_TYPESCRIPT_VERSION ?= 7.13.0
OPENAPI_GENERATOR_CLI_VERSION ?= 2.42.0
SPECTRAL_VERSION ?= 6.17.0

# Regenerate the sqlc code (committed under server/internal/db/gen) from
# server/sqlc.yaml.
.PHONY: sqlc-gen
sqlc-gen:
	cd server && go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

# Lint the API contract (openapi/lumemor.yaml) against the .spectral.yaml
# ruleset.
.PHONY: api-lint
api-lint:
	npx --yes @stoplight/spectral-cli@$(SPECTRAL_VERSION) lint openapi/lumemor.yaml

# Regenerate every API artifact from openapi/lumemor.yaml:
# - Go server interface + types (server/internal/api/gen, config in
#   openapi/codegen/go.yaml)
# - TypeScript types (openapi/generated/web/api.d.ts)
# - Dart client package (openapi/generated/mobile, config in
#   openapi/codegen/mobile.yaml)
#
# The artifacts are committed; run from the repository root.
.PHONY: api-gen
api-gen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) \
		-config openapi/codegen/go.yaml openapi/lumemor.yaml
	cd server && go mod tidy
	npx --yes openapi-typescript@$(OPENAPI_TYPESCRIPT_VERSION) \
		openapi/lumemor.yaml -o openapi/generated/web/api.d.ts
	rm -rf openapi/generated/mobile
	npx --yes @openapitools/openapi-generator-cli@$(OPENAPI_GENERATOR_CLI_VERSION) generate \
		-g dart-dio -i openapi/lumemor.yaml -o openapi/generated/mobile \
		-c openapi/codegen/mobile.yaml

# Verify that the committed generated artifacts match a fresh regeneration.
.PHONY: api-check
api-check: api-gen
	git diff --exit-code
	@if git status --porcelain -- openapi/generated server/internal/api/gen | grep -q .; then \
		echo "generated artifacts drifted: untracked or changed files"; \
		exit 1; \
	fi

# --- Web app (SvelteKit) ----------------------------------------------------
#
# The web client is a pure SvelteKit single-page app under web/:
# adapter-static builds it into web/build (the index.html app shell
# plus the content-hashed _app/ assets), and web-build then stages that
# output into server/internal/web/dist, where the Go embed directive
# picks it up. The staged tree is never committed: a Go build without a
# prior web-build fails at the embed directive ("no matching files
# found") -- that loud failure is intentional and pins the build order,
# so CI runs the web build before the Go build.

# Install the web toolchain and its dependencies with pnpm. The
# lockfile is committed; pnpm freezes it automatically in CI.
.PHONY: web-install
web-install:
	cd web && pnpm install

# Type-check and unit-test the web app: svelte-check plus vitest. The
# CI gate for web changes.
.PHONY: web-check
web-check:
	cd web && pnpm run check
	cd web && pnpm run test

# Build the web app and stage the output for the Go embed. Must run
# before any step that compiles the server (go build, go test,
# docker-test, docker-lint, the compose image build). The docker-test
# and docker-lint targets below declare that dependency outright:
# their containers bind-mount this repository and compile the server
# inside, so a missing dist would fail them at the embed directive
# and -- worse -- a stale one would silently pass. The declared
# dependency keeps both loud and fresh; the compose image build needs
# no such line because its own Node stage builds the app hermetically.
.PHONY: web-build
web-build:
	cd web && pnpm run build
	rm -rf server/internal/web/dist
	cp -R web/build server/internal/web/dist

# --- Development/test container --------------------------------------------
#
# The media toolchain (libvips, exiftool, ffmpeg) lives in the dev
# container, not on the host. docker-test builds the image from
# docker/dev.Dockerfile and runs the full Go test suite inside it with
# the repository bind-mounted at /src, so the image itself carries
# nothing environment-specific.
#
# Mechanics:
# - gocache / gomodcache named volumes keep the build and module caches
#   across runs, so repeat invocations compile incrementally;
# - /var/run/docker.sock is mounted so testcontainers (and the Docker
#   CLI) can start throwaway PostgreSQL containers through the host
#   daemon -- they run as siblings of this container;
# - GOPROXY is read from the HOST's `go env` at run time and passed
#   with -e, keeping the image free of proxy configuration;
# - Networking hint: if a sibling container's published port is not
#   reachable from inside this container (testcontainers derives the
#   host address from the Docker daemon), make the connection go
#   through the host's loopback instead:
#       make docker-test 'TEST_ENV=-e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal'
#   OrbStack resolves host.docker.internal inside containers. Extend
#   TEST_ENV with further docker run flags as needed.
DEV_IMAGE ?= lumemor-dev
# Extra docker run flags for docker-test, empty by default. Quote the
# whole assignment on the make command line (see the hint above) so the
# shell of the recipe receives the flags as one expansion.
TEST_ENV ?=
# Optional apt mirror for the dev image build, empty by default
# (upstream Debian). Pass a mirror for networks where deb.debian.org is
# slow or unreachable, e.g.:
#     make docker-test APT_MIRROR=mirrors.tuna.tsinghua.edu.cn
# Like GOPROXY, it is passed in at build time; the committed image
# carries no mirror configuration.
APT_MIRROR ?=
# Optional override of the onnxruntime release URL the dev image build
# fetches the pinned shared library from (restricted networks can
# point it at a mirror of the same bytes). The per-architecture sha256
# stays pinned in the Dockerfile either way, so anything but the
# release bytes fails the build. Empty keeps the GitHub release URL.
ONNXRUNTIME_URL ?=
# Optional proxy for that download, passed as the predefined
# https_proxy build arg. The value must be reachable from the build
# container -- on a desktop runtime that is
# http://host.docker.internal:7890, not 127.0.0.1. Empty keeps direct
# downloads.
ONNXRUNTIME_PROXY ?=

# The containerized test gate: builds the dev image and runs the full
# Go test suite inside it with the repository bind-mounted at /src.
# Depends on web-build: the embedded SPA is a compile-time input of
# the server (the embed directive), so a clean checkout builds the web
# app first and a stale staged dist can never silently serve the
# containerized suite.
.PHONY: docker-test
docker-test: web-build
	docker build -f docker/dev.Dockerfile \
		--build-arg APT_MIRROR=$(APT_MIRROR) \
		--build-arg GOPROXY=$(shell go env GOPROXY) \
		--build-arg ONNXRUNTIME_URL=$(ONNXRUNTIME_URL) \
		--build-arg https_proxy=$(ONNXRUNTIME_PROXY) \
		-t $(DEV_IMAGE) .
	docker run --rm \
		-v $(CURDIR):/src \
		-w /src/server \
		-v gocache:/root/.cache/go-build \
		-v gomodcache:/go/pkg/mod \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e GOPROXY=$(shell go env GOPROXY) \
		$(TEST_ENV) \
		$(DEV_IMAGE) \
		go test -p 1 ./...

# Lint the server inside the dev container: the media package is CGO
# (govips/libvips), so the lint toolchain must run where libvips
# headers exist -- a host without them cannot even compile the package
# being linted. The container carries golangci-lint (pinned to the CI
# version), and the run covers the full Go gate set: gofmt -l (any
# output fails), go vet, then golangci-lint. Depends on web-build for
# the same reason as docker-test (the embed directive is compiled by
# the linter too).
.PHONY: docker-lint
docker-lint: web-build
	docker build -f docker/dev.Dockerfile \
		--build-arg APT_MIRROR=$(APT_MIRROR) \
		--build-arg GOPROXY=$(shell go env GOPROXY) \
		--build-arg ONNXRUNTIME_URL=$(ONNXRUNTIME_URL) \
		--build-arg https_proxy=$(ONNXRUNTIME_PROXY) \
		-t $(DEV_IMAGE) .
	docker run --rm \
		-v $(CURDIR):/src \
		-w /src/server \
		-v gocache:/root/.cache/go-build \
		-v gomodcache:/go/pkg/mod \
		-e GOPROXY=$(shell go env GOPROXY) \
		$(TEST_ENV) \
		$(DEV_IMAGE) \
		sh -c 'unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "gofmt needed on:" $$unformatted; exit 1; fi && go vet ./... && golangci-lint run'
