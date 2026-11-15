package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/storage"
)

// THE LIVE PHOTO MAINTENANCE: the pair's two halves can fall out of
// step with the disk, and nothing in the family's surfaces can say so
// (a component row is invisible by construction). Two shapes exist:
//
//   - an ORPHAN: a component the pair no longer references (its still
//     was permanently deleted, or a re-extraction swapped the link).
//     Nothing will ever serve it, so it goes -- row, original and
//     derived trees.
//   - a BROKEN pair: a component whose file vanished (a manual delete,
//     a half-synced disk). The still cannot play a motion it no longer
//     has, so the component row goes too: the still's link nulls out
//     through the schema and it degrades to a plain photo -- the type
//     mark disappears, and nothing is reported to the family.
//
// The run is bounded and hourly, mirrors the trash expiry's discipline,
// and only looks at LIVE component rows: a trashed one is the expiry's
// business, and one younger than the grace window may be mid-extraction
// (the row is created and linked inside one metadata job), so it is
// never mistaken for an orphan.

// LivePhotoSweepInterval is the cadence of the pair maintenance. One
// hour matches the other maintenance runs and keeps the scan off the
// hot path.
const LivePhotoSweepInterval = time.Hour

// livePhotoSweepPeriodicJobID identifies the registration in River's
// periodic bundle, so it is assertable by RemoveByID.
const livePhotoSweepPeriodicJobID = "live_photo_sweep"

const (
	// livePhotoSweepGraceSeconds is how old a component row must be
	// before the run may judge it: one metadata job's lifetime is
	// seconds, so an hour is generous and still bounded.
	livePhotoSweepGraceSeconds = 3600
	// livePhotoSweepBatch caps one run, like the trash expiry's own
	// bound: a backlog drains over several hours instead of one
	// unbounded walk.
	livePhotoSweepBatch = 200
)

// LivePhotoSweepCounts reports one run: the orphans and the broken
// pairs it removed. The two numbers are the single log line.
type LivePhotoSweepCounts struct {
	Orphans int
	Broken  int
}

// RunLivePhotoSweep walks the live component rows older than the grace
// window and removes the ones nothing can use: a row whose file is
// gone takes the still's link with it (the pair degrades), and a row no
// still references goes with its bytes.
func RunLivePhotoSweep(ctx context.Context, pool *pgxpool.Pool, dataDir string, bus *events.Bus, logger *slog.Logger) (LivePhotoSweepCounts, error) {
	queries := dbgen.New(pool)
	rows, err := queries.ListLivePhotoComponents(ctx, dbgen.ListLivePhotoComponentsParams{
		GraceSeconds: livePhotoSweepGraceSeconds,
		LimitCount:   livePhotoSweepBatch,
	})
	if err != nil {
		return LivePhotoSweepCounts{}, fmt.Errorf("live photo sweep: list components: %w", err)
	}

	counts := LivePhotoSweepCounts{}
	for _, row := range rows {
		assetID := uuid.UUID(row.ID.Bytes)

		stills, err := queries.ListLivePhotoStillsOfMotion(ctx, row.ID)
		if err != nil {
			return counts, fmt.Errorf("live photo sweep: list the stills of %s: %w", assetID, err)
		}

		// The broken arm first: a missing file is decided by the disk,
		// not the links.
		missing := false
		if row.StorageKind != dbgen.StorageKindExternal {
			if path, err := storage.ResolveStoredPath(dataDir, row.FilePath); err != nil {
				logger.Error("live photo sweep: resolve the motion's path", "asset_id", assetID, "error", err)
				continue
			} else if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
				missing = true
			}
		}
		if !missing && len(stills) > 0 {
			// A pair in good order: nothing to do.
			continue
		}

		if err := removeComponent(ctx, pool, dataDir, row); err != nil {
			logger.Error("live photo sweep: remove the component", "asset_id", assetID, "error", err)
			continue
		}
		if missing {
			counts.Broken++
		} else {
			counts.Orphans++
		}

		// A still that lost its motion changed shape: the announcement
		// lets an open wall drop the type mark without a reload. A
		// trashed still has no surface to converge (the trash view
		// renders no mark).
		if bus != nil {
			for _, still := range stills {
				if still.DeletedAt.Valid {
					continue
				}
				bus.Publish("asset.updated", map[string]string{
					"assetId": uuid.UUID(still.ID.Bytes).String(),
					"kind":    string(dbgen.MediaKindImage),
				})
			}
		}
	}
	return counts, nil
}

// removeComponent deletes one component row and its files (the shared
// blob rule decides the original; the derived trees are keyed by the
// id alone and always go).
func removeComponent(ctx context.Context, pool *pgxpool.Pool, dataDir string, row dbgen.ListLivePhotoComponentsRow) error {
	queries := dbgen.New(pool)
	refs, err := queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
		FilePath: row.FilePath,
		ID:       row.ID,
	})
	if err != nil {
		return fmt.Errorf("count the motion's file references: %w", err)
	}
	if _, err := queries.DeleteAssetPermanently(ctx, row.ID); err != nil {
		return fmt.Errorf("delete the motion row: %w", err)
	}
	if err := storage.RemoveAssetFiles(dataDir, row.FilePath, uuid.UUID(row.ID.Bytes).String(), refs); err != nil {
		// Best-effort by design: the row is gone, so a failure is a
		// leak, not data loss.
		return nil
	}
	return nil
}

// LivePhotoSweepArgs is the periodic job: it carries no payload,
// because the run reads its candidates from the database.
type LivePhotoSweepArgs struct{}

// Kind is River's stable job kind identifier.
func (LivePhotoSweepArgs) Kind() string { return "live_photo_sweep" }

// LivePhotoSweepWorker runs the periodic maintenance over the queue's
// own pool and data root.
type LivePhotoSweepWorker struct {
	river.WorkerDefaults[LivePhotoSweepArgs]

	pool    *pgxpool.Pool
	dataDir string
	bus     *events.Bus
	logger  *slog.Logger
}

// Work executes one run; the counts ride the single log line.
func (w *LivePhotoSweepWorker) Work(ctx context.Context, job *river.Job[LivePhotoSweepArgs]) error {
	counts, err := RunLivePhotoSweep(ctx, w.pool, w.dataDir, w.bus, w.logger)
	if err != nil {
		w.logger.Error("live photo sweep failed", "error", err, "orphans", counts.Orphans, "broken", counts.Broken)
		return fmt.Errorf("live photo sweep: %w", err)
	}
	w.logger.Info("live photo sweep complete", "orphans", counts.Orphans, "broken", counts.Broken)
	return nil
}

// livePhotoSweepPeriodicJobs returns the run's periodic registration:
// hourly, RunOnStart off (the other maintenance runs behave the same;
// a boot run would double the first tick).
func livePhotoSweepPeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(LivePhotoSweepInterval), livePhotoSweepJobConstructor,
			&river.PeriodicJobOpts{ID: livePhotoSweepPeriodicJobID}),
	}
}

// livePhotoSweepJobConstructor is the periodic insert payload.
func livePhotoSweepJobConstructor() (river.JobArgs, *river.InsertOpts) {
	return LivePhotoSweepArgs{}, nil
}
