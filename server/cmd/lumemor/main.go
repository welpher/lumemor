// Command lumemor runs the lumemor server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/welpher/lumemor/server/internal/api"
	"github.com/welpher/lumemor/server/internal/buildinfo"
	"github.com/welpher/lumemor/server/internal/config"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

const (
	// shutdownTimeout bounds how long in-flight requests may finish after a
	// termination signal before the server exits anyway.
	shutdownTimeout = 10 * time.Second

	// dbConnectWait bounds how long startup waits for the database to
	// become reachable, tolerating the race against a freshly launched
	// database container.
	dbConnectWait = 60 * time.Second

	// eventRetention is the activity ring window pinned by the API
	// contract: GET /api/v1/events replays this much history on
	// reconnect.
	eventRetention = 5 * time.Minute
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// run executes the server lifecycle: load config, connect to the database
// and apply migrations, start the HTTP server, then shut down gracefully on
// SIGINT/SIGTERM.
func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL, dbConnectWait)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	logger.Info("schema migrations applied")

	// The first-run predicate mirrors how NewRouter builds its internal
	// server struct: the sqlc queries over the pool. It derives from the
	// users table, so a restarted server routes the same way; an
	// unreachable database falls back to the wizard, whose own endpoints
	// surface the real errors.
	queries := dbgen.New(pool)

	// The event bus is the SSE activity stream's backbone and the
	// plugin seam: handlers publish domain events (session lifecycle
	// in M2a) and every connected stream follows them.
	bus := events.NewBus(eventRetention, time.Now)

	srv := &http.Server{
		Addr: cfg.Listen,
		Handler: api.NewRouter(&api.Deps{
			Pool:    pool,
			Logger:  logger,
			Version: buildinfo.Version,
			Events:  bus,
			Initialized: func(ctx context.Context) bool {
				admins, err := queries.CountAdmins(ctx)
				if err != nil {
					logger.Error("count admins for root routing", "error", err)
					return false
				}
				return admins > 0
			},
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server stopped")
	return nil
}
