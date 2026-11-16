package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// THE SINGLE-FILE MOTION PHOTO: the most common Live Photo a family
// carries is not two files -- it is one camera file whose tail (or
// whose embedded tag) holds a short video, stitched on by an Android
// camera. Recognition is SILENT: it happens inside the metadata job,
// writes a component video row beside the still, links it, and queues
// that row's transcode. There is no "recognizing" state, no user
// action, and no error the family ever sees -- a still whose motion
// cannot be extracted is simply a still, and the job's log line is the
// only trace.
//
// THE FOUR CONTAINERS the cameras write, read from the extraction's
// own tags:
//
//  1. Google's MotionPhoto container (HEIC and JPEG): the XMP block
//     says MotionPhoto and lists a ContainerDirectory; the entry whose
//     Semantic is "MotionPhoto" carries the video's Length and its
//     Padding, and the bytes sit at the very end of the file
//     (size - length - padding).
//  2. The older MicroVideo shape: MicroVideo plus MicroVideoOffset --
//     the same tail, read as size - offset.
//  3. Samsung's HEIC: a MotionPhotoVideo binary tag holds the whole
//     video (the frame plus the still's HEIC container; the transcode
//     job makes it playable).
//  4. Samsung/Xiaomi JPEG: EmbeddedVideoType = MotionPhoto_Data and an
//     EmbeddedVideoFile binary tag.
//
// The binary tags are read through exiftool's -b arm (a fixed tag
// name per family, never a caller's string); the tail shapes are one
// ReadAt apiece. Either way the bytes are hashed (sha1) and a motion
// already stored on disk is REUSED -- the same couple of megabytes
// never land twice -- before a fresh row is written.

// motionVideoPlan answers how to reach one file's embedded motion,
// from the extraction's tags alone. Exactly one of the two shapes is
// set: a tail window (offset and length into the file) or a binary
// tag's name.
type motionVideoPlan struct {
	// tail is true when the motion is the file's trailing bytes.
	tail bool
	// offset and length delimit the tail window.
	offset int64
	length int64
	// binaryTag is the exiftool binary tag holding the motion.
	binaryTag string
}

// ok reports whether the plan addresses any bytes at all.
func (p motionVideoPlan) ok() bool {
	if p.binaryTag != "" {
		return true
	}
	return p.tail && p.length > 0
}

// motionVideoPlanOf reads the four containers' tags into a plan. The
// order matters where the families overlap: the binary tags win over
// the tail shapes, because a HEIC that carries a MotionPhotoVideo tag
// also carries a MotionPhoto directory whose Length addresses the
// container's tail, not a playable video.
func motionVideoPlanOf(tags map[string]any) motionVideoPlan {
	// The Samsung HEIC arm: the whole video as one binary tag.
	if _, ok := tagString(tags, "MotionPhotoVideo"); ok {
		return motionVideoPlan{binaryTag: "MotionPhotoVideo"}
	}
	// The JPEG arm: the flag names the file's shape and the second
	// tag carries the bytes.
	if kind, ok := tagString(tags, "EmbeddedVideoType"); ok && kind == "MotionPhoto_Data" {
		if _, ok := tagString(tags, "EmbeddedVideoFile"); ok {
			return motionVideoPlan{binaryTag: "EmbeddedVideoFile"}
		}
	}

	// The Google container: the motion entry's Length governs, and its
	// Padding (the gap between the video and the end of the file)
	// subtracts from the tail's offset.
	if tagTruthy(tags, "MotionPhoto") {
		length, padding := containerDirectoryMotion(tags)
		if length > 0 {
			return motionVideoPlan{tail: true, offset: length + padding, length: length}
		}
	}
	// The older MicroVideo shape: the offset alone addresses the tail.
	if tagTruthy(tags, "MicroVideo") {
		if offset, ok := tagFloat(tags, "MicroVideoOffset"); ok && offset > 0 {
			length := int64(offset)
			return motionVideoPlan{tail: true, offset: length, length: length}
		}
	}
	return motionVideoPlan{}
}

// containerDirectoryMotion answers the MotionPhoto entry's Length and
// Padding from the XMP ContainerDirectory. The structure's depth
// depends on the camera: a Google HEIC lists its items directly, a
// JPEG nests them one directory deeper. The walk therefore searches
// the whole tree for the item whose Semantic names the motion --
// exactly one item does -- instead of pinning one camera's depth.
func containerDirectoryMotion(tags map[string]any) (int64, int64) {
	raw, ok := tags["ContainerDirectory"]
	if !ok {
		return 0, 0
	}
	length, padding, found := motionContainerItem(raw)
	if !found {
		return 0, 0
	}
	return length, padding
}

// motionContainerItem walks one container node (a list of directories,
// a directory, or one item) for the motion's own descriptor.
func motionContainerItem(node any) (int64, int64, bool) {
	switch typed := node.(type) {
	case []any:
		for _, entry := range typed {
			if length, padding, found := motionContainerItem(entry); found {
				return length, padding, true
			}
		}
	case map[string]any:
		if semantic, ok := tagString(typed, "Semantic"); ok && semantic == "MotionPhoto" {
			length, _ := tagFloat(typed, "Length")
			padding, _ := tagFloat(typed, "Padding")
			if length > 0 {
				return int64(length), int64(padding), true
			}
			return 0, 0, false
		}
		for _, value := range typed {
			if length, padding, found := motionContainerItem(value); found {
				return length, padding, true
			}
		}
	}
	return 0, 0, false
}

// tagTruthy answers a flag tag that may arrive as a number (the -n
// arm), a boolean, or a string.
func tagTruthy(tags map[string]any, name string) bool {
	value, ok := tags[name]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case string:
		trimmed := strings.TrimSpace(typed)
		return trimmed != "" && trimmed != "0" && !strings.EqualFold(trimmed, "false")
	}
	return false
}

// readMotionBytes answers one plan's bytes: the tail window through
// one ReadAt, or the binary tag through exiftool's -b arm.
func readMotionBytes(ctx context.Context, path string, size int64, plan motionVideoPlan) ([]byte, error) {
	if plan.binaryTag != "" {
		return extractBinaryTag(ctx, path, plan.binaryTag)
	}
	if !plan.tail {
		return nil, errors.New("motion photo plan addresses no bytes")
	}
	if plan.offset < 0 || plan.offset > size || plan.length <= 0 || plan.length > plan.offset {
		return nil, fmt.Errorf("motion photo window %d+%d leaves a %d byte file", plan.offset, plan.length, size)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	buffer := make([]byte, plan.length)
	if _, err := file.ReadAt(buffer, size-plan.offset); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buffer, nil
}

// extractBinaryTag reads one exiftool binary tag's bytes: the fixed
// -b arm over a tag name this file's own switch names (never a
// caller's string).
func extractBinaryTag(ctx context.Context, path, tag string) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, exiftoolTimeout)
	defer cancel()

	cmd := exiftoolBinaryCommand(runCtx, path, tag)
	cmd.SysProcAttr = procAttr()
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return killProcessGroup(cmd.Process.Pid)
	}
	cmd.WaitDelay = 5 * time.Second

	out, err := cmd.Output()
	if runCtx.Err() != nil {
		return nil, fmt.Errorf("exiftool exceeded its %s budget and its process group was killed: %w", exiftoolTimeout, runCtx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("read binary tag %s: %w: %s", tag, err, stderrOf(err))
	}
	return out, nil
}

// applyMotionPhoto runs the extraction for one still inside the
// metadata job. It never fails the job: the still is the asset, and a
// motion that cannot be read leaves it exactly that. The steps follow
// the chain's order -- extract, dedupe by checksum, write the bytes,
// link, queue the motion's own pipeline --
// and every write is idempotent, so a re-run converges.
func (w *MetadataWorker) applyMotionPhoto(ctx context.Context, assetID uuid.UUID, tags map[string]any, filePath string, size int64) {
	plan := motionVideoPlanOf(tags)
	if !plan.ok() {
		return
	}

	video, err := readMotionBytes(ctx, filePath, size, plan)
	if err != nil {
		w.logger.Warn("motion photo extraction failed; the still stands alone",
			"asset_id", assetID, "error", err)
		return
	}
	if len(video) == 0 {
		return
	}

	still, err := dbgen.New(w.pool).GetAssetForMotionPhoto(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		w.logger.Error("read the still for the motion photo step", "asset_id", assetID, "error", err)
		return
	}
	// A watched directory's still: the extraction has nowhere honest
	// to land -- the family's source tree never takes our writes, and
	// the blob tree's placement rides the upload template's own
	// machinery. The still stands alone (no error line); the paired
	// upload remains the live path for it.
	if still.StorageKind == "external" {
		return
	}

	motionID, fresh, err := w.placeMotion(ctx, still, video)
	if err != nil {
		w.logger.Error("store the extracted motion", "asset_id", assetID, "error", err)
		return
	}

	if err := w.linkMotion(ctx, still, motionID); err != nil {
		w.logger.Error("link the extracted motion", "asset_id", assetID, "error", err)
		return
	}

	// The motion's own pipeline: its metadata row (duration and
	// dimensions) and the transcode that makes it playable -- exactly
	// the pair the upload path's component receives. Only a fresh row
	// needs the enqueue; a reused row already ran (or is running) both.
	if fresh && w.motionDispatch != nil {
		w.motionDispatch(ctx, motionID)
	}
}

// placeMotion answers the motion row for one still's extracted bytes:
// the row already holding them (reused, and marked as a component if
// it was not one) or a freshly written one. It reports whether the
// row is fresh, which decides the enqueue.
func (w *MetadataWorker) placeMotion(ctx context.Context, still dbgen.GetAssetForMotionPhotoRow, video []byte) (uuid.UUID, bool, error) {
	sum := sha1.Sum(video)
	checksum := hex.EncodeToString(sum[:])

	// The sha1 reuse: the same couple of megabytes may already sit in
	// the library (a re-extraction of the same still, a burst, or the
	// same camera clip arriving twice). The bytes are already on disk,
	// so the row is reused rather than written again -- and marking it
	// as a component is what makes it the pair's (never listed). The
	// lookup is blob-domain: a directory source's position row is the
	// family's own file at its own path, and marking THAT as the
	// still's component would make the pair's deletion reach a file the
	// server never owned.
	if existing, err := dbgen.New(w.pool).GetBlobAssetByOwnerChecksum(ctx, dbgen.GetBlobAssetByOwnerChecksumParams{
		OwnerID:  still.OwnerID,
		Checksum: checksum,
	}); err == nil {
		// A TRASHED row is not a reuse: the pair must point at bytes the
		// library will keep, and a trashed row's checksum is already
		// released for a fresh upload to claim.
		if !existing.DeletedAt.Valid {
			if err := dbgen.New(w.pool).MarkAssetAsLivePhotoMotion(ctx, existing.ID); err != nil {
				return uuid.Nil, false, fmt.Errorf("mark the stored motion: %w", err)
			}
			return uuid.UUID(existing.ID.Bytes), false, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, fmt.Errorf("look up the stored motion: %w", err)
	}

	// The new row's file lands beside the still, under the still's own
	// stem: the pair shares one folder (the storage layout rendered
	// from the still's date), and the motion's name says which still
	// it belongs to. The placement is collision-proof: a name already
	// taken gets a numbered suffix.
	relPath, err := placeMotionFile(w.dataDir, still, video)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("place the motion file: %w", err)
	}

	motionID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, false, err
	}
	row, err := dbgen.New(w.pool).CreateMotionAsset(ctx, dbgen.CreateMotionAssetParams{
		ID:               pgUUID(motionID),
		OwnerID:          still.OwnerID,
		FilePath:         relPath,
		Checksum:         checksum,
		OriginalFilename: motionFileName(still.OriginalFilename, relPath),
		FileSize:         int64(len(video)),
		TakenAt:          still.TakenAt,
		TakenAtSource:    still.TakenAtSource,
		TakenAtZone:      still.TakenAtZone,
	})
	if err != nil {
		// The file this attempt placed references no row (the insert
		// lost); it is removed so a retry does not accumulate copies.
		removePlacedMotion(w.dataDir, relPath)
		return uuid.Nil, false, fmt.Errorf("insert the motion row: %w", err)
	}

	// The component joins the owner's sync album for the same reason
	// the paired upload's does: the caller-visible union reads album
	// membership, so the motion stays reachable BY ITS ID (the detail,
	// the play stream, the archive) while every listing filters
	// component rows out.
	if err := w.addToSyncAlbum(ctx, still.OwnerID, motionID); err != nil {
		w.logger.Error("add the motion to the sync album", "motion_id", motionID, "error", err)
	}
	return uuid.UUID(row.ID.Bytes), true, nil
}

// linkMotion points the still at its motion and reports the change on
// the activity stream. A pair already linked to the same row is left
// untouched (no write, no event); a pair whose motion changed -- the
// same still re-extracted after its file changed -- drops the old
// motion when nothing else references it.
func (w *MetadataWorker) linkMotion(ctx context.Context, still dbgen.GetAssetForMotionPhotoRow, motionID uuid.UUID) error {
	current := uuid.Nil
	if still.LivePhotoVideoID.Valid {
		current = uuid.UUID(still.LivePhotoVideoID.Bytes)
	}
	if current == motionID {
		return nil
	}

	queries := dbgen.New(w.pool)
	if err := queries.SetAssetLivePhotoVideo(ctx, dbgen.SetAssetLivePhotoVideoParams{
		ID:      still.ID,
		VideoID: pgUUID(motionID),
	}); err != nil {
		return err
	}
	if current != uuid.Nil {
		// The replaced motion: it goes only when no still still needs
		// it, and its file follows its row.
		if err := w.retireMotion(ctx, current); err != nil {
			w.logger.Error("retire the replaced motion", "motion_id", current, "error", err)
		}
	}

	// The still's shape changed: the walls' tiles show the type mark
	// the moment this converges, so the event carries the new state.
	w.bus.Publish("asset.updated", map[string]string{
		"assetId": uuid.UUID(still.ID.Bytes).String(),
		"kind":    string(dbgen.MediaKindImage),
	})
	return nil
}

// retireMotion removes one motion row nothing references any more,
// with its file and its derived trees.
func (w *MetadataWorker) retireMotion(ctx context.Context, motionID uuid.UUID) error {
	queries := dbgen.New(w.pool)
	references, err := queries.CountLivePhotoReferences(ctx, pgUUID(motionID))
	if err != nil {
		return err
	}
	if references > 0 {
		return nil
	}
	motion, err := queries.GetAssetByID(ctx, pgUUID(motionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := queries.DeleteAssetPermanently(ctx, motion.ID); err != nil {
		return err
	}
	if err := storage.RemoveAssetFiles(w.dataDir, motion.FilePath, motionID.String(), 0); err != nil {
		w.logger.Warn("remove the replaced motion's files", "motion_id", motionID, "error", err)
	}
	return nil
}

// addToSyncAlbum links one row into the owner's sync album (the
// stream every timeline aggregates): the ensure arm recovers an
// account whose sync album predates the wiring, and the upsert keeps
// a concurrent recovery from failing.
func (w *MetadataWorker) addToSyncAlbum(ctx context.Context, owner pgtype.UUID, assetID uuid.UUID) error {
	queries := dbgen.New(w.pool)
	albumID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	syncID, err := queries.EnsureSyncAlbum(ctx, dbgen.EnsureSyncAlbumParams{
		ID:      pgUUID(albumID),
		OwnerID: owner,
	})
	if err != nil {
		return err
	}
	return queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
		AlbumID: syncID,
		AssetID: pgUUID(assetID),
		AddedBy: owner,
	})
}

// motionFileName answers the motion row's display name: the still's
// stem with the .mp4 extension the extracted bytes carry. The stored
// path may have gained a collision suffix; the DISPLAY name keeps the
// still's stem so the pair reads as one capture.
func motionFileName(stillName, relPath string) string {
	stem := strings.TrimSuffix(path.Base(relPath), path.Ext(relPath))
	if stem == "" {
		stem = strings.TrimSuffix(stillName, filepath.Ext(stillName))
	}
	return stem + ".mp4"
}

// placeMotionFile writes one motion beside its still and answers the
// stored relative path. The name is the still's stem, numbered on a
// collision (two bursts' motions, both named after one still): the
// placement never overwrites a file.
func placeMotionFile(dataDir string, still dbgen.GetAssetForMotionPhotoRow, video []byte) (string, error) {
	dir := path.Dir(still.FilePath)
	stem := strings.TrimSuffix(path.Base(still.FilePath), path.Ext(still.FilePath))
	if stem == "" {
		stem = "motion"
	}
	for suffix := 0; suffix < 100; suffix++ {
		name := stem + ".mp4"
		if suffix > 0 {
			name = fmt.Sprintf("%s (%d).mp4", stem, suffix)
		}
		relPath := path.Join(dir, name)
		absolute, err := storage.ResolveStoredPath(dataDir, relPath)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(absolute, video, 0o644); err != nil {
			return "", err
		}
		return relPath, nil
	}
	return "", fmt.Errorf("no free motion file name beside %s", still.FilePath)
}

// removePlacedMotion removes one placed file, best-effort.
func removePlacedMotion(dataDir, relPath string) {
	absolute, err := storage.ResolveStoredPath(dataDir, relPath)
	if err != nil {
		return
	}
	_ = os.Remove(absolute)
}
