package jobs

import (
	"context"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// requireThumbnails skips the thumbnail pipeline tests when this build
// carries no libvips: they are exercised by `make docker-test` and CI,
// which link the CGO media package.
func requireThumbnails(t *testing.T) {
	t.Helper()
	if !media.ThumbnailsSupported {
		t.Skip("this build carries no libvips (CGO disabled); run the suite inside the dev container (make docker-test)")
	}
}

// requirePosterFFmpeg skips the poster flow when ffmpeg is missing.
func requirePosterFFmpeg(t *testing.T) {
	t.Helper()
	requireThumbnails(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skipf("ffmpeg is not installed on this host; run the suite inside the dev container (make docker-test): %v", err)
	}
}

// blurhashOf reads the blurhash column of an asset row.
func (e *env) blurhashOf(t *testing.T, id uuid.UUID) pgtype.Text {
	t.Helper()
	var hash pgtype.Text
	err := e.pool.QueryRow(context.Background(),
		"SELECT blurhash FROM assets WHERE id = $1", id).Scan(&hash)
	if err != nil {
		t.Fatalf("scan blurhash of asset %s: %v", id, err)
	}
	return hash
}

// thumbFiles lists the derived files of an asset's thumbnail directory
// (empty when the directory does not exist).
func (e *env) thumbFiles(t *testing.T, id uuid.UUID) []string {
	t.Helper()
	dir := filepath.Dir(storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("list thumbnail directory of %s: %v", id, err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

// requireJobThumbDimensions pins a derived file's dimensions through
// the media package's decoder.
func requireJobThumbDimensions(t *testing.T, id uuid.UUID, size, ext string, wantWidth, wantHeight int) {
	t.Helper()
	path := storage.ThumbPath(id.String(), size, ext)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("thumbnail %s missing: %v", path, err)
	}
	width, height, err := media.ImageDimensions(path)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if width != wantWidth || height != wantHeight {
		t.Errorf("%s = %dx%d, want %dx%d", path, width, height, wantWidth, wantHeight)
	}
}

// enableFullSizeCopy writes the image settings the full-size tests
// run under: the shipped sizes and encoding with the full-resolution
// copy on. The shipped default posture (the copy OFF) is pinned by
// TestThumbnailJobDefaultSettingsSkipTheOriginal; these tests pin the
// sizes' correctness, which the copy carries.
func enableFullSizeCopy(t *testing.T, e *env) {
	t.Helper()
	if err := db.SaveImageSettings(t.Context(), e.queries, db.ImageSettings{
		TileSize:    250,
		PreviewSize: 1080,
		Format:      media.FormatAvif,
		FullSize:    true,
		RawEmbedded: true,
	}); err != nil {
		t.Fatalf("save the full-size image settings: %v", err)
	}
}

// TestThumbnailJobGoldenJPEG is the mandated golden flow: the golden
// JPEG (64x64, real EXIF) produces its three AVIF serving copies with
// the correct dimensions -- tile within the 250px box, view at the
// source size (the original is smaller than 1080), original at the
// source dimensions -- and the blurhash lands on the asset row. The
// pipeline completes: pending_jobs empties, ready_at is stamped,
// asset.ready fires once, for the image kind.
func TestThumbnailJobGoldenJPEG(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	enableFullSizeCopy(t, e)

	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/golden/golden.jpg", goldenFixture(t), deviceTakenAt, `{}`)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the golden asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 64, 64)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 64, 64)
	requireJobThumbDimensions(t, id, media.SizeOriginal, media.FormatAvif, 64, 64)

	hash := e.blurhashOf(t, id)
	if !hash.Valid || len(hash.String) < 6 {
		t.Errorf("blurhash = %v, want the stored non-empty hash", hash)
	}

	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0", state.pending)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != id.String() {
		t.Errorf("asset.ready assetId = %q, want %s", payload["assetId"], id)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.ready kind = %q, want image", payload["kind"])
	}
}

// TestThumbnailJobSizes pins the downscaling math on a 1600x800
// source: the tile fits the 250px box as 250x125, the view caps the
// long edge at 1080 as 1080x540, the original keeps 1600x800.
func TestThumbnailJobSizes(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	enableFullSizeCopy(t, e)

	large, err := os.ReadFile(filepath.Join("..", "media", "testdata", "golden-large.jpg"))
	if err != nil {
		t.Fatalf("read large fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/large/large.jpg", large, deviceTakenAt, `{}`)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the large asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 250, 125)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 1080, 540)
	requireJobThumbDimensions(t, id, media.SizeOriginal, media.FormatAvif, 1600, 800)
}

// TestThumbnailJobHEIC is the mandated HEIC flow: a real HEIC decodes
// through libheif and derives the same sizes as its JPEG twin.
func TestThumbnailJobHEIC(t *testing.T) {
	requireThumbnails(t)
	if !media.HEICDecodeSupported() {
		t.Skip("this libvips build carries no HEIC decoder (libheif); run the suite inside the dev container (make docker-test)")
	}
	e := newEnv(t)
	enableFullSizeCopy(t, e)

	heic, err := os.ReadFile(filepath.Join("..", "media", "testdata", "golden-large.heic"))
	if err != nil {
		t.Fatalf("read HEIC fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/heic/large.heic", heic, deviceTakenAt, `{}`)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the HEIC asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 250, 125)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 1080, 540)
	requireJobThumbDimensions(t, id, media.SizeOriginal, media.FormatAvif, 1600, 800)

	hash := e.blurhashOf(t, id)
	if !hash.Valid || len(hash.String) < 6 {
		t.Errorf("blurhash = %v, want the stored non-empty hash", hash)
	}
}

// TestThumbnailJobVideoPoster is the mandated video flow: a real (tiny)
// MP4 produces poster.jpg -- the first frame scaled into the 250px
// tile box, a JPEG, NOT AVIF thumbnails (the thumbnail directory holds
// the poster alone) -- and its blurhash lands on the row. The pipeline
// completes for the video kind.
func TestThumbnailJobVideoPoster(t *testing.T) {
	requirePosterFFmpeg(t)
	e := newEnv(t)

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/video/tiny.mp4", video, deviceTakenAt, `{}`,
		dbgen.MediaKindVideo, "video/mp4", 1)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the video asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	files := e.thumbFiles(t, id)
	if len(files) != 1 || files[0] != "poster.jpg" {
		t.Fatalf("thumbnail files of the video = %v, want exactly [poster.jpg]", files)
	}

	posterPath := storage.PosterPath(id.String())
	file, err := os.Open(posterPath)
	if err != nil {
		t.Fatalf("open poster %s: %v", posterPath, err)
	}
	poster, err := jpeg.Decode(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("decode poster %s: %v", posterPath, err)
	}
	if closeErr != nil {
		t.Fatalf("close poster %s: %v", posterPath, closeErr)
	}
	bounds := poster.Bounds()
	if bounds.Dx() != 250 || bounds.Dy() != 125 {
		t.Errorf("poster = %dx%d, want the aspect-preserving 250x125 of the 1600x800 source frame", bounds.Dx(), bounds.Dy())
	}

	hash := e.blurhashOf(t, id)
	if !hash.Valid || len(hash.String) < 6 {
		t.Errorf("blurhash = %v, want the stored non-empty hash", hash)
	}

	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["kind"] != "video" {
		t.Errorf("asset.ready kind = %q, want video", payload["kind"])
	}
}

// TestThumbnailJobPosterFrameOffSkipsCapture pins the transcoding
// settings' poster-frame switch (thumbnailFrame=false): the
// thumbnail job of a video completes without capturing a poster --
// no derived files, no blurhash -- and the pipeline still drains,
// exactly like the transcode job's own `off` preset.
func TestThumbnailJobPosterFrameOffSkipsCapture(t *testing.T) {
	requirePosterFFmpeg(t)
	e := newEnv(t)

	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO instance_settings (key, value) VALUES ('transcoding_settings', '{"thumbnailFrame":false}'::jsonb)`); err != nil {
		t.Fatalf("seed the poster-frame-off profile: %v", err)
	}

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	deviceTakenAt := time.Date(2022, 2, 2, 2, 2, 2, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/video/poster-off.mp4", video, deviceTakenAt, `{}`,
		dbgen.MediaKindVideo, "video/mp4", 1)

	worker := &ThumbnailWorker{
		pool:    e.pool,
		bus:     e.bus,
		dataDir: e.dataDir,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	job := &river.Job[ThumbnailArgs]{Args: ThumbnailArgs{AssetID: id}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Work under poster-frame-off = %v, want nil", err)
	}

	if files := e.thumbFiles(t, id); len(files) != 0 {
		t.Errorf("thumbnail files under poster-frame-off = %v, want none", files)
	}
	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d under poster-frame-off, want the drained 0", state.pending)
	}
	if !state.readyAt.Valid {
		t.Error("ready_at is not stamped under poster-frame-off, want the completed pipeline")
	}
	hash := e.blurhashOf(t, id)
	if hash.Valid {
		t.Errorf("blurhash = %q under poster-frame-off, want none stored", hash.String)
	}
}

// TestThumbnailJobOrientation is the mandated orientation flow: a
// 200x100 sensor JPEG carrying EXIF orientation 6 renders upright --
// the tile is 100x200, the sensor dimensions swapped at render.
func TestThumbnailJobOrientation(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	enableFullSizeCopy(t, e)

	portrait, err := os.ReadFile(filepath.Join("..", "media", "testdata", "portrait-orientation6.jpg"))
	if err != nil {
		t.Fatalf("read portrait fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/portrait/portrait.jpg", portrait, deviceTakenAt, `{}`)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the portrait asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 100, 200)
	requireJobThumbDimensions(t, id, media.SizeOriginal, media.FormatAvif, 100, 200)
}

// TestThumbnailJobRerunIsIdempotent is the mandated re-run contract: a
// second run of the job on an already-completed asset rewrites exactly
// the same file set at the same paths with the same dimensions, the
// blurhash converges to the same value, the readiness state is
// untouched, and no second asset.ready is published.
func TestThumbnailJobRerunIsIdempotent(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	enableFullSizeCopy(t, e)

	large, err := os.ReadFile(filepath.Join("..", "media", "testdata", "golden-large.jpg"))
	if err != nil {
		t.Fatalf("read large fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/rerun/large.jpg", large, deviceTakenAt, `{}`)

	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the first run to complete", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	firstFiles := e.thumbFiles(t, id)
	firstHash := e.blurhashOf(t, id)
	firstReadyAt := e.assetOf(t, id).readyAt

	// The manual re-enqueue: the review-focus rerun must be safe.
	if err := e.queue.EnqueueThumbnail(context.Background(), id); err != nil {
		t.Fatalf("re-enqueue thumbnail job: %v", err)
	}
	// The re-run's blurhash update moves updated_at; that is the
	// observable completion of the second pass.
	eventually(t, jobWaitTimeout, "the re-run to complete", func() bool {
		var updatedAt time.Time
		if err := e.pool.QueryRow(context.Background(),
			"SELECT updated_at FROM assets WHERE id = $1", id).Scan(&updatedAt); err != nil {
			return false
		}
		return updatedAt.After(e.assetOf(t, id).readyAt.Time)
	})
	// The event, if one were wrongly published, follows the commit
	// synchronously; a short settle makes the count deterministic.
	time.Sleep(300 * time.Millisecond)

	secondFiles := e.thumbFiles(t, id)
	if len(secondFiles) != len(firstFiles) {
		t.Fatalf("thumbnail files after the re-run = %v, want the same set as before %v", secondFiles, firstFiles)
	}
	for i := range firstFiles {
		if firstFiles[i] != secondFiles[i] {
			t.Errorf("thumbnail files after the re-run = %v, want the same set as before %v", secondFiles, firstFiles)
		}
	}
	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 250, 125)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 1080, 540)
	requireJobThumbDimensions(t, id, media.SizeOriginal, media.FormatAvif, 1600, 800)

	secondHash := e.blurhashOf(t, id)
	if secondHash != firstHash {
		t.Errorf("blurhash changed on re-run: first %v, second %v", firstHash, secondHash)
	}

	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 after the re-run", state.pending)
	}
	if !state.readyAt.Time.Equal(firstReadyAt.Time) {
		t.Errorf("ready_at = %s, want the unchanged %s", state.readyAt.Time, firstReadyAt.Time)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want the single original one", got)
	}
}

// TestThumbnailJobMissingAsset pins the gone-asset path: a job whose
// asset row disappeared completes as a no-op (no retry burn, no
// event).
func TestThumbnailJobMissingAsset(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	gone, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	worker := &ThumbnailWorker{
		pool:    e.pool,
		bus:     e.bus,
		dataDir: e.dataDir,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	// Invoked directly: the assertion is Work's own behavior, not
	// River's retry machinery around it.
	job := &river.Job[ThumbnailArgs]{Args: ThumbnailArgs{AssetID: gone}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Work of a missing asset = %v, want nil", err)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 0 {
		t.Errorf("asset.ready events = %d, want 0", got)
	}
}

// TestThumbnailJobSettingsHonorTheDocument pins the settings-reading
// discipline: the job derives at the image settings' document --
// here, a 320 tile at WebP with the full-size copy on -- and a row
// saved before the ingest binds at that derivation.
func TestThumbnailJobSettingsHonorTheDocument(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	if err := db.SaveImageSettings(t.Context(), e.queries, db.ImageSettings{
		TileSize:    320,
		PreviewSize: 1440,
		Format:      media.FormatWebp,
		FullSize:    true,
		RawEmbedded: true,
	}); err != nil {
		t.Fatalf("save the image settings: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("..", "jobs", "testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	id := e.insertAsset(t, "originals/depth/settings-honor.jpg", content, time.Now(), `{}`)
	if err := e.queue.EnqueueThumbnail(t.Context(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the settings-honoring asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	// The sizes and encoding follow the document: a 320-box tile and
	// a 1440-long-edge view, both WebP, plus the full-size copy.
	for _, tt := range []struct {
		size         string
		wantW, wantH int
	}{
		// The 64x64 golden source never upscales: the boxes bind
		// only, the source size stays.
		{media.SizeTile, 64, 64},
		{media.SizeView, 64, 64},
		{media.SizeOriginal, 64, 64},
	} {
		path := storage.ThumbPath(id.String(), tt.size, media.FormatWebp)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("derivative %s missing: %v", path, err)
		}
		width, height, err := media.ImageDimensions(path)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if width != tt.wantW || height != tt.wantH {
			t.Errorf("%s = %dx%d, want %dx%d", path, width, height, tt.wantW, tt.wantH)
		}
	}
	// The shipped encoding's paths are NOT written: the document's
	// encoding is the one the file carries.
	avifPath := storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)
	if _, err := os.Stat(avifPath); !os.IsNotExist(err) {
		t.Errorf("AVIF tile stat err = %v, want NotExist when the format is WebP", err)
	}
}

// TestThumbnailJobDefaultSettingsSkipTheOriginal pins the shipped
// posture: the settings row absent, the job derives the tile and the
// view and NOTHING else -- the full-resolution copy is disk the
// default declines to spend.
func TestThumbnailJobDefaultSettingsSkipTheOriginal(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	content, err := os.ReadFile(filepath.Join("..", "jobs", "testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	id := e.insertAsset(t, "originals/depth/default-posture.jpg", content, time.Now(), `{}`)
	if err := e.queue.EnqueueThumbnail(t.Context(), id); err != nil {
		t.Fatalf("enqueue thumbnail job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the default-posture asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	// The 64x64 golden source: tile and view keep the source size
	// (never upscaled), and no original exists.
	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 64, 64)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 64, 64)
	original := storage.ThumbPath(id.String(), media.SizeOriginal, media.FormatAvif)
	if _, err := os.Stat(original); !os.IsNotExist(err) {
		t.Errorf("original derivative stat err = %v, want NotExist under the shipped posture", err)
	}
}
