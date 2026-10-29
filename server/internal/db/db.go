// Package db manages the PostgreSQL connection pool and applies schema
// migrations at startup.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DefaultConnectWait is how long Connect waits for the database to
	// become reachable when the caller passes a non-positive wait.
	DefaultConnectWait = 60 * time.Second

	// pingInterval is the delay between connection attempts.
	pingInterval = time.Second

	// pingTimeout bounds a single connection attempt so a hanging dial
	// cannot stall the whole wait budget.
	pingTimeout = time.Second
)

// Connect creates a connection pool for url and waits until the database
// accepts connections, retrying every pingInterval for up to wait. This
// tolerates the startup race against a freshly launched database container.
// A non-positive wait falls back to DefaultConnectWait. The caller owns the
// returned pool and must close it.
func Connect(ctx context.Context, url string, wait time.Duration) (*pgxpool.Pool, error) {
	if wait <= 0 {
		wait = DefaultConnectWait
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	deadline := time.After(wait)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err = pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			pool.Close()
			return nil, fmt.Errorf("connect database: %w", ctxErr)
		}

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("connect database: %w", ctx.Err())
		case <-deadline:
			pool.Close()
			return nil, fmt.Errorf("database not reachable within %s: %w", wait, err)
		case <-time.After(pingInterval):
		}
	}
}
