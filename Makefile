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
