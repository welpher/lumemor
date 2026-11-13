// The album removal tests (the un-ingest): the directory arm of
// DELETE /albums/{albumId} against the real surfaces. The guard's
// three arms are pinned here -- the unreferenced external
// member purges, the member another album holds survives on that
// reference (the dedup-borrow shape included: an uploaded blob row
// the album linked keeps its sync album's hold), and the blob rows
// survive on the sync album's permanent membership -- beside the
// file half (the source files the family owns stay on the mount,
// the generated trees go, the trashed member's trash copy goes with
// its row), the face-cluster sweep, the per-user states, the event
// family, and the refusal matrix. The memory arm's own pins stay
// in albums_write_test.go.
package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// insertRemovalMember writes one source file under the given source
// directory and inserts the external asset row plus its membership
// in the GIVEN album -- the removal tests control the album shape
// directly (several members over one album, several albums over one
// member), which the external lifecycle fixture's one-album-per-row
// shape cannot express.
func (e *assetEnv) insertRemovalMember(t *testing.T, album, owner uuid.UUID, sourceDir, name string, content []byte, mtime time.Time) uuid.UUID {
	t.Helper()
	path := filepath.Join(sourceDir, name)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write the source file %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("stamp the source mtime: %v", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the asset id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs, file_mtime
		) VALUES (
			$1, $2, 'external', $3, $4, $5, $6, 'image/jpeg', 'image', 0, 0,
			$7, 'file', 0, $7
		)`,
		pgUUID(id), pgUUID(owner), path, sha256Of(content), name,
		len(content), mtime); err != nil {
		t.Fatalf("insert the external member row: %v", err)
	}
	e.addTimelineItem(t, album, id, owner)
	return id
}

// insertRemovalDirectory inserts one directory album row with its
// resolved source root (the removal reads nothing from it, but the
// row the endpoint removes is the production shape).
func (e *assetEnv) insertRemovalDirectory(t *testing.T, id, owner uuid.UUID, root string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO albums (id, owner_id, title, type, root_path)
		VALUES ($1, $2, 'Family archive', 'directory', $3)`,
		pgUUID(id), pgUUID(owner), root); err != nil {
		t.Fatalf("insert the fixture directory album: %v", err)
	}
}

// removalDerivedPaths answers the two derived trees of one asset:
// the thumbnail and the transcode outputs the pipeline lands under
// the data root, keyed by the asset id alone.
func removalDerivedPaths(dataDir string, id uuid.UUID) []string {
	return []string{
		filepath.Join(dataDir, "storage", "thumbs", id.String(), "tile.avif"),
		filepath.Join(dataDir, "storage", "encoded", id.String(), "video.avif"),
	}
}

// requireSourceFileIntact asserts the family's source file survived
// the removal byte for byte -- the zero-contact pin's host-side
// check.
func requireSourceFileIntact(t *testing.T, path string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the source file %s after the removal: %v", path, err)
	}
	if string(data) != string(want) {
		t.Errorf("the source file %s changed under the removal", path)
	}
}

// requireDerivedFilesGone asserts both derived trees of one asset
// are empty of files (the physical cleanup of the generated files).
func requireDerivedFilesGone(t *testing.T, dataDir string, id uuid.UUID) {
	t.Helper()
	for _, path := range removalDerivedPaths(dataDir, id) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("derived file %s survived the purge, want gone", path)
		}
	}
}

// TestRemoveDirectoryAlbum pins the removal's arms over ONE removal
// of ONE album whose members cover every case: the fixtures are
// arranged, the DELETE fires once, and the subtests assert each arm
// of the aftermath.
func TestRemoveDirectoryAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "removal-grace@example.com")

	root := env.provisionMountRoot(t)
	sourceDir := filepath.Join(root, "family")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	mtime := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

	album := fixtureWriteAlbumID("c1")
	env.insertRemovalDirectory(t, album, adminID, sourceDir)
	holder := fixtureWriteAlbumID("c2")
	env.insertRemovalDirectory(t, holder, adminID, filepath.Join(root, "other"))
	env.shareWallAlbum(t, album, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	content := func(name string) []byte { return []byte("source bytes of " + name) }

	// Two members the removal owns alone: both purge.
	solo1 := env.insertRemovalMember(t, album, adminID, sourceDir, "solo1.jpg", content("solo1"), mtime)
	solo2 := env.insertRemovalMember(t, album, adminID, sourceDir, "solo2.jpg", content("solo2"), mtime)
	// The shared-reference member: album holder holds it too -- it
	// survives on that reference when album goes.
	sharedRef := env.insertRemovalMember(t, album, adminID, sourceDir, "shared.jpg", content("shared"), mtime)
	env.addTimelineItem(t, holder, sharedRef, adminID)
	// The trashed member: the owner trashed it earlier (the move
	// semantics hold its file in the server's trash area); the
	// removal takes the row and the trash copy together.
	trashed := env.insertRemovalMember(t, album, adminID, sourceDir, "trashed.jpg", content("trashed"), mtime)
	report := env.batchAssets(t, env.cookie, string(apigen.Trash), trashed)
	if report.Affected != 1 {
		t.Fatalf("trash report = %+v, want 1 affected", report)
	}
	if _, err := os.Stat(storage.TrashHome(env.dataDir, trashed.String())); err != nil {
		t.Fatalf("the trash move never landed: %v", err)
	}
	// The borrowed blob: a real upload lives in the owner's sync
	// album; the album's membership row is the borrow. It survives
	// on the sync album's permanent hold.
	payload := jpegFixture(t)
	uploaded := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "borrow-1",
		"fileName":      "IMG_BORROW.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_BORROW.jpg", content: payload}}))
	borrowed := uuid.UUID(uploaded.Id)
	env.addTimelineItem(t, album, borrowed, adminID)

	// The generated files of every member, present before the
	// removal (the pipeline's outputs, stood in for directly).
	for _, id := range []uuid.UUID{solo1, solo2, trashed, sharedRef, borrowed} {
		for _, path := range removalDerivedPaths(env.dataDir, id) {
			env.writeDerivedFile(t, path, []byte("derived of "+id.String()))
		}
	}

	// The per-user states and the face clusters the sweep must
	// answer: solo1's face shares a cluster with the surviving
	// sharedRef's face (the representative re-homes), solo2's face
	// is its cluster's only member (the cluster dissolves with the
	// caller's label).
	env.insertPeopleModel(t)
	faceShared := fixtureAssetID("c1")
	faceSolo1 := fixtureAssetID("c2")
	faceSolo2 := fixtureAssetID("c3")
	clusterRehomed := fixtureAlbumID("c1")
	clusterDissolved := fixtureAlbumID("c2")
	env.insertFace(t, faceShared, sharedRef, 0.91)
	env.insertFace(t, faceSolo1, solo1, 0.95)
	env.insertFace(t, faceSolo2, solo2, 0.9)
	env.seedPersonCluster(t, clusterRehomed, faceSolo1, faceShared)
	env.seedPersonCluster(t, clusterDissolved, faceSolo2)
	env.labelPersonCluster(t, adminID, clusterDissolved, "Gone person")
	env.labelPersonCluster(t, adminID, clusterRehomed, "Staying person")
	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2), ($1, $3)",
		pgUUID(adminID), pgUUID(solo1), pgUUID(sharedRef))
	env.exec(t, `INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'family')`, pgUUID(fixtureAlbumID("c3")), pgUUID(adminID))
	env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2), ($3, $2)",
		pgUUID(solo1), pgUUID(fixtureAlbumID("c3")), pgUUID(sharedRef))
	env.setTimelinePreference(t, graceCookie, album, false)

	rec := env.deleteAlbumRaw(t, env.cookie, album)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /albums/%s status = %d, want %d (body %q)",
			album, rec.Code, http.StatusNoContent, rec.Body.String())
	}

	t.Run("AlbumAndItsRowsAreGone", func(t *testing.T) {
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(album)); got != 0 {
			t.Errorf("album rows = %d, want 0", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 0 {
			t.Errorf("membership rows = %d, want 0 (cascaded)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(album)); got != 0 {
			t.Errorf("share rows = %d, want 0 (cascaded)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE album_id = $1", pgUUID(album)); got != 0 {
			t.Errorf("preference rows = %d, want 0 (cascaded)", got)
		}
		// The second delete finds nothing: the removal is not
		// repeatable.
		requireAlbumNotFound(t, env.deleteAlbumRaw(t, env.cookie, album))
	})

	t.Run("UnreferencedMembersPurgeHard", func(t *testing.T) {
		// The rows die the HARD death: gone, not trashed -- the
		// library data cleans while the sources stay.
		for _, id := range []uuid.UUID{solo1, solo2, trashed} {
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(id)); got != 0 {
				t.Errorf("asset rows of the purged member %s = %d, want 0", id, got)
			}
			if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE asset_id = $1", pgUUID(id)); got != 0 {
				t.Errorf("membership rows of the purged member %s = %d, want 0", id, got)
			}
		}
		// The per-user states die with the row (the schema's
		// cascades), and the exif/detail companions go with it.
		if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE asset_id = $1", pgUUID(solo1)); got != 0 {
			t.Errorf("favorites of the purged member = %d, want 0", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE asset_id = $1", pgUUID(solo1)); got != 0 {
			t.Errorf("tag assignments of the purged member = %d, want 0", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE asset_id = ANY($1)",
			pgUUIDs([]uuid.UUID{solo1, solo2})); got != 0 {
			t.Errorf("faces of the purged members = %d, want 0 (cascaded)", got)
		}
	})

	t.Run("GeneratedFilesGoAndSourcesStay", func(t *testing.T) {
		// The generated trees of the purged members are physically
		// gone; the trash copy of the trashed member went with its
		// row.
		for _, id := range []uuid.UUID{solo1, solo2, trashed} {
			requireDerivedFilesGone(t, env.dataDir, id)
		}
		if _, err := os.Stat(storage.TrashHome(env.dataDir, trashed.String())); !os.IsNotExist(err) {
			t.Errorf("the trashed member's trash home survived the removal (stat err %v)", err)
		}
		// THE ZERO-CONTACT PIN: every LIVE member's source file is
		// still on the mount, byte for byte -- the purged members'
		// included (the trash interplay is different by design: the
		// trashed member's file was never at the original location
		// anymore -- the trash move owns it -- and its trash copy
		// went with the row above).
		for _, member := range []struct {
			path string
			want []byte
		}{
			{filepath.Join(sourceDir, "solo1.jpg"), content("solo1")},
			{filepath.Join(sourceDir, "solo2.jpg"), content("solo2")},
			{filepath.Join(sourceDir, "shared.jpg"), content("shared")},
		} {
			requireSourceFileIntact(t, member.path, member.want)
		}
		if _, err := os.Stat(filepath.Join(sourceDir, "trashed.jpg")); !os.IsNotExist(err) {
			t.Errorf("the trashed member's original location = %v, want empty (the trash move owns the file)", err)
		}
	})

	t.Run("GuardedMembersSurvive", func(t *testing.T) {
		// The shared-reference member: its row, its per-user states,
		// its derived trees and its source file all survive -- only
		// the removed album's reference died.
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(sharedRef)); got != 1 {
			t.Errorf("shared-reference member rows = %d, want 1 (another album holds it)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE asset_id = $1 AND album_id = $2",
			pgUUID(sharedRef), pgUUID(holder)); got != 1 {
			t.Errorf("the holding album's membership = %d, want 1", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE asset_id = $1", pgUUID(sharedRef)); got != 1 {
			t.Errorf("favorites of the survivor = %d, want 1 (untouched)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE asset_id = $1", pgUUID(sharedRef)); got != 1 {
			t.Errorf("tag assignments of the survivor = %d, want 1 (untouched)", got)
		}
		for _, path := range removalDerivedPaths(env.dataDir, sharedRef) {
			if _, err := os.Stat(path); err != nil {
				t.Errorf("derived file %s of the survivor = %v, want present", path, err)
			}
		}
		// The borrowed blob: the sync album's permanent hold is the
		// guard's own blob arm -- the row, its file in the managed
		// tree and its derived trees survive.
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(borrowed)); got != 1 {
			t.Errorf("borrowed blob rows = %d, want 1 (the sync album holds it)", got)
		}
		var blobPath string
		if err := env.pool.QueryRow(context.Background(),
			"SELECT file_path FROM assets WHERE id = $1", pgUUID(borrowed)).Scan(&blobPath); err != nil {
			t.Fatalf("read the borrowed blob's path: %v", err)
		}
		if _, err := os.Stat(filepath.Join(env.dataDir, filepath.FromSlash(blobPath))); err != nil {
			t.Errorf("the borrowed blob's file = %v, want present", err)
		}
		for _, path := range removalDerivedPaths(env.dataDir, borrowed) {
			if _, err := os.Stat(path); err != nil {
				t.Errorf("derived file %s of the borrowed blob = %v, want present", path, err)
			}
		}
		// The holding album never lost anything.
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(holder)); got != 1 {
			t.Errorf("holding album rows = %d, want 1", got)
		}
	})

	t.Run("FaceClustersSweep", func(t *testing.T) {
		// The re-home arm: the cluster shared with the survivor lost
		// its purged face but keeps its row, its label, and a
		// representative that is now the surviving face.
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(clusterRehomed)); got != 1 {
			t.Fatalf("the shared cluster rows = %d, want 1", got)
		}
		var representative uuid.UUID
		if err := env.pool.QueryRow(context.Background(),
			"SELECT representative_face_id FROM face_clusters WHERE id = $1",
			pgUUID(clusterRehomed)).Scan(&representative); err != nil {
			t.Fatalf("read the rehomed representative: %v", err)
		}
		if representative != faceShared {
			t.Errorf("representative = %s, want the surviving face %s", representative, faceShared)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(clusterRehomed)); got != 1 {
			t.Errorf("the shared cluster's labels = %d, want 1 (it survived)", got)
		}
		// The dissolve arm: the cluster whose only face died is gone
		// with the caller's label (the person-without-faces
		// semantics, applied where the faces die).
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(clusterDissolved)); got != 0 {
			t.Errorf("the vacated cluster rows = %d, want 0 (dissolved)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(clusterDissolved)); got != 0 {
			t.Errorf("the vacated cluster's labels = %d, want 0 (cascaded)", got)
		}
	})

	t.Run("EventsAnnounceTheRemoval", func(t *testing.T) {
		albumPayloads := busEventsOf(t, env.bus, "album.deleted")
		if len(albumPayloads) != 1 {
			t.Fatalf("album.deleted events = %d, want exactly 1", len(albumPayloads))
		}
		if albumPayloads[0]["albumId"] != album.String() || albumPayloads[0]["userId"] != adminID.String() {
			t.Errorf("album.deleted payload = %v, want the album and the acting caller", albumPayloads[0])
		}
		if len(albumPayloads[0]) != 2 {
			t.Errorf("album.deleted payload = %v, want exactly the albumId/userId pair", albumPayloads[0])
		}

		// One asset.deleted per purged non-component member, the
		// minimal id-plus-kind payload; the guarded survivors and
		// the borrowed blob announce nothing.
		want := map[string]bool{solo1.String(): true, solo2.String(): true, trashed.String(): true}
		for _, payload := range busEventsOf(t, env.bus, "asset.deleted") {
			id := payload["assetId"]
			if !want[id] {
				t.Errorf("asset.deleted for %q, want only the purged members %v", id, want)
				continue
			}
			delete(want, id)
			if payload["kind"] != "image" {
				t.Errorf("asset.deleted kind = %q, want image", payload["kind"])
			}
			if len(payload) != 2 {
				t.Errorf("asset.deleted payload = %v, want exactly the assetId/kind pair", payload)
			}
		}
		for id := range want {
			t.Errorf("asset.deleted for the purged member %s never fired", id)
		}
	})
}

// TestRemoveDirectoryAlbumRefusals pins the removal's gate: the
// sharee's honest 403 (the share is view-only), the stranger's and
// the sync album's single-shape 404, and the unknown id's -- and
// every refusal leaves the album exactly as it stood.
func TestRemoveDirectoryAlbumRefusals(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "removal-refusal-grace@example.com")

	root := env.provisionMountRoot(t)
	album := fixtureWriteAlbumID("d1")
	env.insertRemovalDirectory(t, album, adminID, filepath.Join(root, "shared-source"))
	env.shareWallAlbum(t, album, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	member := env.insertRemovalMember(t, album, adminID, filepath.Join(root, "shared-source"),
		"member.jpg", []byte("refusal source"), time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, graceCookie, album)
		requireAlbumWriteForbidden(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(album)); got != 1 {
			t.Errorf("album rows after the sharee's attempt = %d, want 1", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(member)); got != 1 {
			t.Errorf("member rows after the sharee's attempt = %d, want 1", got)
		}
	})

	t.Run("SyncAlbumIsNotFound", func(t *testing.T) {
		ownSync := env.syncAlbumIDOf(t, adminID)
		requireAlbumNotFound(t, env.deleteAlbumRaw(t, env.cookie, ownSync))
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(ownSync)); got != 1 {
			t.Errorf("the sync album rows = %d, want 1 (system album)", got)
		}
	})

	t.Run("StrangerIsNotFound", func(t *testing.T) {
		requireAlbumNotFound(t, env.deleteAlbumRaw(t, graceCookie, fixtureWriteAlbumID("d2")))
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		requireAlbumNotFound(t, env.deleteAlbumRaw(t, env.cookie, fixtureWallAlbumID("ff")))
	})
}

// TestRemoveDirectoryAlbumSharedBlob pins the blob arm's file rule:
// a borrowed blob whose ORIGINAL is content-addressed and shared
// with another owner's row keeps the file when the album removal
// purges the row -- the shared-file rule spans both delete paths
// (the pre-template era's shape, where identical bytes of any owner
// share one physical file).
func TestRemoveDirectoryAlbumSharedBlob(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "removal-shared-blob-grace@example.com")

	// The legacy shared-file state: both owners' rows name ONE
	// content-addressed path.
	sharedPath := "originals/legacy/shared.jpg"
	if err := os.MkdirAll(filepath.Join(env.dataDir, "originals", "legacy"), 0o755); err != nil {
		t.Fatalf("create the legacy originals directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(env.dataDir, filepath.FromSlash(sharedPath)), []byte("shared bytes"), 0o644); err != nil {
		t.Fatalf("write the shared original: %v", err)
	}
	album := fixtureWriteAlbumID("e1")
	env.insertRemovalDirectory(t, album, adminID, "unused")
	rowA, rowB := fixtureWallAssetID("e1"), fixtureWallAssetID("e2")
	for _, member := range []struct {
		id    uuid.UUID
		owner uuid.UUID
	}{{rowA, adminID}, {rowB, graceID}} {
		env.exec(t, `
			INSERT INTO assets (
				id, owner_id, storage_kind, file_path, checksum, original_filename,
				file_size, mime_type, media_kind, width, height, taken_at,
				taken_at_source, pending_jobs
			) VALUES (
				$1, $2, 'blob', $3, $4, 'shared.jpg', 12, 'image/jpeg', 'image', 0, 0,
				$5, 'import', 0
			)`,
			pgUUID(member.id), pgUUID(member.owner), sharedPath, "shared-checksum",
			time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	}
	env.addTimelineItem(t, album, rowA, adminID)

	rec := env.deleteAlbumRaw(t, env.cookie, album)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /albums/%s status = %d, want %d (body %q)", album, rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(rowA)); got != 0 {
		t.Errorf("the purged blob's rows = %d, want 0", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(rowB)); got != 1 {
		t.Errorf("the other owner's rows = %d, want 1 (untouched)", got)
	}
	if _, err := os.Stat(filepath.Join(env.dataDir, filepath.FromSlash(sharedPath))); err != nil {
		t.Errorf("the shared file = %v, want present (the other row references it)", err)
	}
}
