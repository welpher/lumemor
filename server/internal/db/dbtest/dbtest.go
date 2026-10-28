// Package dbtest provisions throwaway PostgreSQL instances for tests: it
// starts a pgvector container, waits for readiness and hands back a
// connected pool. Migrations are deliberately the caller's concern so
// tests of the migration code itself stay honest.
//
// This package must not import the parent db package: the db package's
// own tests import dbtest, and an internal test file cannot import a
// package that imports the code under test (import cycle not allowed in
// test). That is also why the ping loop below resembles db.Connect
// instead of reusing it.
package dbtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moby/moby/client"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// pgImage is the PostgreSQL image used for the test instances. It bundles
// the pgvector extension that the schema depends on.
const pgImage = "pgvector/pgvector:pg17"

const (
	// startupTimeout bounds container start and the first connection.
	startupTimeout = 5 * time.Minute

	// pingInterval is the delay between connection attempts while
	// waiting for the first successful ping.
	pingInterval = time.Second

	// terminateTimeout bounds the container teardown.
	terminateTimeout = 30 * time.Second
)

// New starts a fresh PostgreSQL container for the test and returns a pool
// connected to it. The container and the pool are cleaned up automatically
// when the test ends. The test is skipped when no container runtime is
// reachable; a container that fails to start is fatal. The database is
// empty: callers apply migrations themselves.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	ctr, err := tcpostgres.Run(ctx, pgImage,
		tcpostgres.WithDatabase("lumemor"),
		tcpostgres.WithUsername("lumemor"),
		tcpostgres.WithPassword("lumemor"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		termCtx, termCancel := context.WithTimeout(context.Background(), terminateTimeout)
		defer termCancel()
		if err := ctr.Terminate(termCtx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("container connection string: %v", err)
	}

	pool, err := connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// connect builds a pool and waits for the first successful ping. The
// container's wait strategies already gate on database readiness; the
// bounded retry only absorbs the short window where the pool's first
// connections are still dialing.
func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	for {
		pingCtx, cancel := context.WithTimeout(ctx, pingInterval)
		err = pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("database not reachable: %w", ctx.Err())
		case <-time.After(pingInterval):
		}
	}
}

// skipIfNoDocker skips the test when the Docker daemon is unreachable, for
// example on a machine without a local container runtime.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer cli.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(pingCtx, client.PingOptions{}); err != nil {
		t.Skipf("docker daemon unreachable, skipping: %v", err)
	}
}
