package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/welpher/lumemor/server/migrations"
)

// Migrate applies every embedded goose migration to the database behind
// pool. It is idempotent: migrations already recorded in goose_db_version
// are skipped, so it is safe to run on every startup.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// Closing the database/sql wrapper returns its connections to the pool
	// without closing the pool itself.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
