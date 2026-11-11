// The album removal (the un-ingest): the database half of
// DELETE /albums/{albumId}'s directory arm. A directory album is a
// source -- its members entered the library by reference, the files
// never moved -- so removing it is the ingest's reverse: the album
// row, its membership, its shares and its per-user preference rows
// go in ONE transaction, and every member asset that no other album
// references is purged from the library in the same transaction.
// The purge is the permanent delete's own cascade (the face cluster
// sweep rides it: a representative must re-home or the cluster
// dissolve before the asset's faces go), applied member by member
// behind the reference guard -- the sync album's permanent
// membership covers every uploaded row, a member shared with a
// second album survives on that reference, and the dedup-borrowed
// member (an existing upload row that linked into the album) is
// just that case. The SOURCE files are never part of the cascade:
// the album only referenced them where they lie, and whatever the
// purge takes from the database, the family's own files stay on the
// mount.
//
// The file half runs AFTER the commit, best-effort: rows first,
// files after is the crash-safety order -- a crash between them
// leaves inert garbage (files no row names) rather than a row that
// names missing files. The trash interplay rides the same file
// half: an external member already in the trash (its file in the
// server's trash area) loses the row and the trash copy together,
// the trash-expiry cleanup's own shape.
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// purgedMember carries one member asset's post-purge facts: the
// event pair (the id and kind the asset events publish, the
// component flag that keeps a Live Photo component's purge silent)
// and the file half's inputs (the storage kind that splits the
// removal helpers, the row's paths, the shared-blob reference count
// a blob arm took before its row went).
type purgedMember struct {
	id        uuid.UUID
	kind      dbgen.MediaKind
	component bool
	external  bool
	filePath  string
	trashPath string
	fileRefs  int64
}

// removeDirectoryAlbum runs the removal's database half in ONE
// transaction: the member read (the membership rows die with the
// album row, so the members are listed before it goes), the album
// row's guarded delete (the cascades take the membership, share and
// preference rows with it), then each member's guard-and-purge. The
// guard counts the album_items rows that still name the asset --
// inside this transaction the removed album's own rows are already
// gone, so a count above zero is another album's live hold and the
// member survives, while zero sends the member through the
// permanent delete's cascade. The transaction answers the purged
// members' facts for the post-commit file half and the events; ok
// is false when a concurrent removal deleted the album row first
// (the caller answers the gone-album 404, and the rollback returns
// everything the transaction touched).
func (s *server) removeDirectoryAlbum(ctx context.Context, albumID uuid.UUID) (purged []purgedMember, ok bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin the album removal transaction: %w", err)
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	members, err := queries.ListAlbumMemberAssets(ctx, pgUUID(albumID))
	if err != nil {
		return nil, false, fmt.Errorf("list the album's members: %w", err)
	}

	deleted, err := queries.DeleteDirectoryAlbum(ctx, pgUUID(albumID))
	if err != nil {
		return nil, false, fmt.Errorf("delete the album row: %w", err)
	}
	if deleted == 0 {
		return nil, false, nil
	}

	for _, member := range members {
		assetID := uuid.UUID(member.ID.Bytes)
		refs, err := queries.CountAlbumReferencesOfAsset(ctx, member.ID)
		if err != nil {
			return nil, false, fmt.Errorf("count the album references of asset %s: %w", assetID, err)
		}
		if refs > 0 {
			// The guard's survivor: another album holds a membership
			// row (the sync album's own hold covers every blob row),
			// so only THIS album's reference dies with it.
			continue
		}

		// The shared-blob reference count for a blob row's file half:
		// taken before the row delete (the count excludes the row
		// itself, so it would answer the same either side), spent
		// after the commit. An external row counts nothing -- its
		// only file of ours is the trash copy keyed by the id.
		var fileRefs int64
		if member.StorageKind != dbgen.StorageKindExternal {
			if fileRefs, err = queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
				FilePath: member.FilePath,
				ID:       member.ID,
			}); err != nil {
				return nil, false, fmt.Errorf("count the file references of asset %s: %w", assetID, err)
			}
		}

		// The permanent delete's own sweep, in its own order: the
		// cluster representatives re-home (a representative face
		// must move to a surviving member before the cascade
		// removes its row), the clusters left without members
		// dissolve (their per-user labels cascade), then the row
		// goes and the schema's cascades take the faces, the
		// favorites, the tag assignments, the embedding and the OCR
		// text with it.
		if err := queries.RehomeClusterRepresentatives(ctx, dbgen.RehomeClusterRepresentativesParams{
			AssetID: member.ID,
		}); err != nil {
			return nil, false, fmt.Errorf("rehome the clusters of asset %s's faces: %w", assetID, err)
		}
		if err := queries.DissolveVacatedClusters(ctx, dbgen.DissolveVacatedClustersParams{
			AssetID: member.ID,
		}); err != nil {
			return nil, false, fmt.Errorf("dissolve the clusters asset %s's faces held: %w", assetID, err)
		}
		if _, err := queries.DeleteAssetPermanently(ctx, member.ID); err != nil {
			return nil, false, fmt.Errorf("delete the asset row %s: %w", assetID, err)
		}

		purged = append(purged, purgedMember{
			id:        assetID,
			kind:      member.MediaKind,
			component: member.IsComponent,
			external:  member.StorageKind == dbgen.StorageKindExternal,
			filePath:  member.FilePath,
			trashPath: member.TrashPath.String,
			fileRefs:  fileRefs,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit the album removal: %w", err)
	}
	return purged, true, nil
}

// removePurgedMemberFiles runs one purged member's file half after
// the commit, split on the storage kind exactly like the expiry's:
// an external row's only files of ours are the derived trees keyed
// by the asset id and the trash copy the trash move landed (the
// containment check inside the removal enforces that even against a
// corrupted row; an orphan mark's NULL trash path cleans the trees
// alone) -- the SOURCE file the row referenced is the family's own
// and is never touched. A blob row's original goes under the
// shared-file rule (the pre-counted references decide) beside the
// same derived trees. Every failure is logged and swallowed: the
// rows are committed gone, so a missed file is inert garbage the
// disk keeps and no view serves -- a leak, never data loss.
func (s *server) removePurgedMemberFiles(r *http.Request, member purgedMember) {
	if member.external {
		if err := storage.RemoveExternalAssetFiles(s.dataDir, member.trashPath, member.id.String()); err != nil {
			s.logger.Error("remove files of a removed album's purged external member",
				"error", err, "asset_id", member.id, "method", r.Method, "path", r.URL.Path)
		}
		return
	}
	if err := storage.RemoveAssetFiles(s.dataDir, member.filePath, member.id.String(), member.fileRefs); err != nil {
		s.logger.Error("remove files of a removed album's purged member",
			"error", err, "asset_id", member.id, "method", r.Method, "path", r.URL.Path)
	}
}
