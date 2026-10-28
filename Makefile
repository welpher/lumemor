# Root Makefile for lumemor development tasks.

# sqlc is pinned so that regenerating the committed code under
# server/internal/db/gen is deterministic; bump deliberately.
SQLC_VERSION ?= v1.31.1

# Regenerate the sqlc code (committed under server/internal/db/gen) from
# server/sqlc.yaml.
.PHONY: sqlc-gen
sqlc-gen:
	cd server && go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate
