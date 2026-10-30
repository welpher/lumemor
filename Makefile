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

.PHONY: docker-test
docker-test:
	docker build -f docker/dev.Dockerfile \
		--build-arg APT_MIRROR=$(APT_MIRROR) \
		--build-arg GOPROXY=$(shell go env GOPROXY) \
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
		go test ./...

# Lint the server inside the dev container: the media package is CGO
# (govips/libvips), so the lint toolchain must run where libvips
# headers exist -- a host without them cannot even compile the package
# being linted. The container carries golangci-lint (pinned to the CI
# version), and the run covers the full Go gate set: gofmt -l (any
# output fails), go vet, then golangci-lint.
.PHONY: docker-lint
docker-lint:
	docker build -f docker/dev.Dockerfile \
		--build-arg APT_MIRROR=$(APT_MIRROR) \
		--build-arg GOPROXY=$(shell go env GOPROXY) \
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
