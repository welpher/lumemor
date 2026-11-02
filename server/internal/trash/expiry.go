// Package trash runs the expiry side of the trash lifecycle (M2c
// Task 3): the physical deletion of soft-deleted assets whose
// retention window closed. It closes the M2b hard-gate handoff #2,
// the shared-blob rule: content addressing stores identical bytes of
// any owner under one physical file, so before a file is removed the
// reference count over the assets table decides -- another row on
// the same file_path (another owner's upload of the same photo, or a
// live row) keeps the file, and only the row goes. The derived copies
// (thumbs, encoded) are keyed by the asset id alone and always go.
//
// The package is deliberately free of the job queue and the media
// pipeline: RunExpiry is the pure database-and-filesystem operation
// that the boot wiring, the River periodic worker, and the tests all
// share, and keeping it here lets the HTTP-layer tests exercise the
// real expiry without pulling the CGO media dependency in.
package trash

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/storage"
)

// ExpiryBatchLimit caps one expiry run's row deletions. A bounded
// loop per run keeps every hourly job short no matter how large the
// expired backlog is; the remainder waits for the next run (an hour
// at most). 500 rows is far beyond a family instance's daily delete
// volume, so in practice the bound never bites.
const ExpiryBatchLimit = 500

// ExpiryCounts reports one expiry run: the rows it deleted and the
// original files it removed (announcements are not counted -- every
// non-component purge publishes one asset.deleted, component purges
// none, so the rows are the honest number). A shared blob
// (another row still referencing the file) keeps the file, so the
// row and file counts diverge by design; the counts are the single
// log line each run emits.
type ExpiryCounts struct {
	Expired      int
	FilesRemoved int
}

// RunExpiry purges the expired trash rows: rows whose deleted_at
// fell past the instance's retention window (instance settings key
// trash_retention_days, default 30). The retention window is read
// once per run and parameterizes both the candidate listing and the
// guarded delete, and each statement computes the deadline itself
// on the database's own clock -- the same clock that stamped the
// deletion marks, so an application clock drifting against the
// database's (a separate container in the compose deployment, or
// the test host against the throwaway database container) can never
// shift the boundary. A corrupt setting row degrades to the default
// (logged, never fatal) so one bad row cannot silently stop the
// lifecycle.
//
// Per candidate: the shared-blob reference count is taken first, the
// guarded delete runs, and only a row the delete actually removed
// has its files removed -- the guard (still trashed, still past the
// cutoff, inside the deleting statement itself) is what makes a
// restore racing the run safe: the restored row matches nothing and
// its files are never touched. The file removal follows the row
// delete's commit deliberately: a failure in between is a storage
// leak, never data loss, so it is logged and the run continues.
//
// The bus, when wired, receives one asset.deleted event per purged
// row -- the trash view of a connected client follows the expiry the
// same way it follows every other lifecycle transition. A Live
// Photo's component row (M2d) is the one exception: its purge stays
// silent, because the trash view never listed components (the pair
// is represented by its photo), so an announcement for one would be
// noise no client can act on -- the row still goes, only the event
// is suppressed. A nil bus publishes nothing.
func RunExpiry(ctx context.Context, pool *pgxpool.Pool, dataDir string, bus *events.Bus) (ExpiryCounts, error) {
	queries := dbgen.New(pool)

	retention, err := db.TrashRetentionDays(ctx, queries)
	if err != nil {
		// A corrupt setting row: the accessor already fell back to
		// the default, and the run proceeds on it -- one bad settings
		// row must not stop the lifecycle. The condition is logged
		// for the operator; the counts and the deadline stay honest.
		slog.Error("trash expiry: retention setting is corrupt, using the default", "error", err)
	}

	rows, err := queries.ListExpiredAssets(ctx, dbgen.ListExpiredAssetsParams{
		RetentionDays: int32(retention),
		LimitCount:    ExpiryBatchLimit,
	})
	if err != nil {
		return ExpiryCounts{}, fmt.Errorf("trash expiry: list expired assets: %w", err)
	}

	counts := ExpiryCounts{}
	for _, row := range rows {
		// The shared-blob reference count runs BEFORE the row delete
		// and excludes the row itself by id, so it answers the same
		// whichever side of the delete it runs on. Every OTHER row
		// counts, trashed rows included: a trashed row can still be
		// restored, and restoring it must find its file.
		refs, err := queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
			FilePath: row.FilePath,
			ID:       row.ID,
		})
		if err != nil {
			return counts, fmt.Errorf("trash expiry: count file references of asset %s: %w", uuid.UUID(row.ID.Bytes), err)
		}

		// The guarded delete: a row restored (or already purged by a
		// concurrent run) since the listing matches nothing, and the
		// run skips its files -- the restore-vs-expiry race guard,
		// in the statement itself.
		deleted, err := queries.ExpireAsset(ctx, dbgen.ExpireAssetParams{
			ID:            row.ID,
			RetentionDays: int32(retention),
		})
		if err != nil {
			return counts, fmt.Errorf("trash expiry: expire asset %s: %w", uuid.UUID(row.ID.Bytes), err)
		}
		if deleted == 0 {
			continue
		}
		counts.Expired++

		assetID := uuid.UUID(row.ID.Bytes)
		if err := storage.RemoveAssetFiles(dataDir, row.FilePath, assetID.String(), refs); err != nil {
			// Best-effort by design: the row is already deleted, so
			// the failure is a leak, not data loss, and failing the
			// run here would only retry a no-op.
			slog.Error("trash expiry: remove files of expired asset", "error", err, "asset_id", assetID)
		}
		if refs == 0 {
			counts.FilesRemoved++
		}
		if bus != nil && !row.IsComponent {
			// Components stay silent (see the function comment): the
			// row is purged above like any other, but no client view
			// ever listed it, so there is nothing to announce.
			bus.Publish("asset.deleted", map[string]string{
				"assetId": assetID.String(),
				"kind":    string(row.MediaKind),
			})
		}
	}
	return counts, nil
}
