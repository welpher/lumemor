// Package migrations embeds the goose SQL migrations that make up the
// lumemor schema. The files are applied at startup by internal/db.
package migrations

import "embed"

// FS holds all versioned migration files.
//
//go:embed *.sql
var FS embed.FS
