package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed the zoneinfo database: the device time zone fallback must resolve everywhere, including runtimes without /usr/share/zoneinfo

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/storage"
)

// MetadataArgs identifies the asset whose file the metadata job
// extracts.
type MetadataArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind is River's stable job kind identifier.
func (MetadataArgs) Kind() string { return "metadata" }

// MetadataWorker runs the metadata job: exiftool over the asset's
// content-addressed file, the exif sidecar row filled from the output,
// the asset row refined (taken-at, dimensions, duration), and the
// pipeline completion recorded -- the completion that empties the
// asset's pending-jobs counter publishes asset.ready.
type MetadataWorker struct {
	river.WorkerDefaults[MetadataArgs]

	pool    *pgxpool.Pool
	bus     *events.Bus
	dataDir string
	logger  *slog.Logger
}

// Work executes one metadata job. Every step runs before the database
// transaction opens except the writes themselves: the exiftool
// subprocess (up to its 30s budget) never holds a connection.
func (w *MetadataWorker) Work(ctx context.Context, job *river.Job[MetadataArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The asset is gone (deleted while the job waited in the
		// queue): there is nothing to refine and retrying cannot
		// change that, so the job completes rather than burning its
		// attempts.
		w.logger.Info("metadata job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}

	raw, err := extractMetadata(ctx, path)
	if err != nil {
		return fmt.Errorf("extract metadata of asset %s: %w", assetID, err)
	}
	rawObject, tags, err := parseExiftoolJSON(raw)
	if err != nil {
		return fmt.Errorf("parse exiftool output of asset %s: %w", assetID, err)
	}

	// The taken-at chain: EXIF DateTimeOriginal resolved with an
	// offset (the EXIF OffsetTime tag, else the per-tag
	// OffsetTimeOriginal, else the device-declared time zone from the
	// upload sidecar) refines taken_at; without a resolvable zone the
	// upload's value stands (see resolveExifTime).
	exifTime, ok := resolveExifTime(tags, deviceTimeZone(row.Exif))

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin metadata transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	if err := queries.UpdateExifFromMetadata(ctx, exifParams(pgUUID(assetID), tags, rawObject, exifTime)); err != nil {
		return fmt.Errorf("write exif row: %w", err)
	}
	if err := queries.RefineAssetFromMetadata(ctx, refineParams(pgUUID(assetID), tags, exifTime, ok)); err != nil {
		return fmt.Errorf("refine asset row: %w", err)
	}
	completion, err := queries.CompleteAssetJob(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The guarded decrement found nothing to decrement: either the
		// pipeline already completed (a re-run: idempotency -- no
		// double decrement, no second event), or the row counts no
		// pipeline of its own (a Live Photo component video), or the
		// asset vanished mid-job. All of these are successful no-ops.
		completion = dbgen.CompleteAssetJobRow{BecameReady: false}
	} else if err != nil {
		return fmt.Errorf("complete pipeline job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit metadata transaction: %w", err)
	}

	if completion.BecameReady {
		// Published after the commit so a subscriber that turns
		// around and reads the database sees the final state. The
		// payload discipline mirrors asset.uploaded: the resource id
		// and its media kind, nothing more.
		w.bus.Publish("asset.ready", map[string]string{
			"assetId": assetID.String(),
			"kind":    string(completion.MediaKind),
		})
	}
	return nil
}

// parseExiftoolJSON decodes the -json output of an invocation over one
// file: an array holding exactly one object. It returns the object's
// verbatim bytes -- the jsonb payload: exiftool's array is its
// multi-file envelope, and one invocation here always covers one file
// -- together with the decoded tag map.
func parseExiftoolJSON(raw []byte) ([]byte, map[string]any, error) {
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, nil, fmt.Errorf("decode json: %w", err)
	}
	if len(rows) != 1 {
		return nil, nil, fmt.Errorf("expected one object in output, got %d", len(rows))
	}
	var tags map[string]any
	if err := json.Unmarshal(rows[0], &tags); err != nil {
		return nil, nil, fmt.Errorf("decode tags: %w", err)
	}
	return rows[0], tags, nil
}

// devicePlaceholder is the upload's sidecar payload that the metadata
// job consumes: the raw client-declared fields captured before the
// handler normalized them away.
type devicePlaceholder struct {
	DeviceCreatedAt string `json:"deviceCreatedAt"`
	DeviceTimeZone  string `json:"deviceTimeZone"`
}

// deviceTimeZone reads the device-declared IANA time zone from the
// exif jsonb the job found on the asset (the upload's placeholder on
// the first run; the raw exiftool output after it, which carries no
// device fields and leaves the return empty).
func deviceTimeZone(exifJSON []byte) string {
	var placeholder devicePlaceholder
	if len(exifJSON) == 0 {
		return ""
	}
	if err := json.Unmarshal(exifJSON, &placeholder); err != nil {
		return ""
	}
	return strings.TrimSpace(placeholder.DeviceTimeZone)
}

// resolvedTime is the EXIF capture time resolved to an instant,
// together with the offset that resolved it.
type resolvedTime struct {
	when     time.Time
	tzOffset string
}

// resolveExifTime turns the EXIF DateTimeOriginal wall clock into an
// instant. The zone resolution chain: the EXIF OffsetTime tag, the
// per-tag OffsetTimeOriginal (many cameras write only that one), then
// the device-declared IANA time zone of the upload. If none yields
// a zone, the time is NOT refined: an EXIF wall clock interpreted in
// a guessed zone can be twelve hours wrong, which is strictly worse
// than the device-reported instant the upload already recorded, so
// the upload fallback chain stays intact instead (the job's caller
// then leaves taken_at and taken_at_source exactly as they are).
func resolveExifTime(tags map[string]any, deviceZone string) (resolvedTime, bool) {
	raw, ok := tagString(tags, "DateTimeOriginal")
	if !ok {
		return resolvedTime{}, false
	}
	wall, err := time.Parse("2006:01:02 15:04:05", raw)
	if err != nil {
		wall, err = time.Parse("2006:01:02 15:04:05.999999999", raw)
		if err != nil {
			return resolvedTime{}, false
		}
	}

	if offset, ok := tagString(tags, "OffsetTime"); ok {
		if seconds, ok := parseOffset(offset); ok {
			return resolveWithFixedZone(wall, seconds), true
		}
	}
	// Many cameras write only the per-tag offset next to the capture
	// time, never the composite OffsetTime, so it is the next link of
	// the chain before the device-declared zone.
	if offset, ok := tagString(tags, "OffsetTimeOriginal"); ok {
		if seconds, ok := parseOffset(offset); ok {
			return resolveWithFixedZone(wall, seconds), true
		}
	}
	if deviceZone != "" {
		if loc, err := time.LoadLocation(deviceZone); err == nil {
			// Re-read the wall clock in the device zone (not a
			// conversion: time.Date interprets the wall clock in loc,
			// applying the zone's rules -- DST included -- at that
			// date).
			when := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), loc)
			_, seconds := when.Zone()
			return resolveWithFixedZone(wall, seconds), true
		}
	}
	return resolvedTime{}, false
}

// resolveWithFixedZone re-resolves the wall clock in a fixed zone of
// the given offset: the wall clock read in that zone is the instant,
// and tzOffset records the offset actually used. (A plain .In(zone)
// would convert the already-parsed instant instead of re-reading the
// wall clock in the zone.)
func resolveWithFixedZone(wall time.Time, offsetSeconds int) resolvedTime {
	zone := time.FixedZone("", offsetSeconds)
	return resolvedTime{
		when:     time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), zone),
		tzOffset: formatOffset(offsetSeconds),
	}
}

// parseOffset reads an EXIF offset ("+05:30", "-08:00").
func parseOffset(s string) (int, bool) {
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	hours, err := strconv.Atoi(s[1:3])
	if err != nil || hours < 0 || hours > 23 {
		return 0, false
	}
	minutes, err := strconv.Atoi(s[4:6])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, false
	}
	seconds := hours*3600 + minutes*60
	if s[0] == '-' {
		seconds = -seconds
	}
	return seconds, true
}

// formatOffset renders an offset in seconds as the EXIF "±HH:MM".
func formatOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	return fmt.Sprintf("%s%02d:%02d", sign, seconds/3600, (seconds%3600)/60)
}

// exifParams maps one extraction onto the exif row upsert. The jsonb
// is the raw exiftool output verbatim; the hot fields are flattened
// from it. Absent tags leave the columns NULL -- the row mirrors the
// file's actual metadata, not a merge of successive extractions.
func exifParams(assetID pgtype.UUID, tags map[string]any, raw []byte, exifTime resolvedTime) dbgen.UpdateExifFromMetadataParams {
	return dbgen.UpdateExifFromMetadataParams{
		AssetID:     assetID,
		Exif:        raw,
		CameraMake:  tagText(tags, "Make"),
		CameraModel: tagText(tags, "Model"),
		Lens:        firstTagText(tags, "LensModel", "LensID", "Lens"),
		Iso:         tagInt4(tags, "ISO"),
		Aperture:    tagFloat4(tags, "FNumber"),
		Shutter:     shutterText(tags),
		Focal:       tagFloat4(tags, "FocalLength"),
		Lat:         coordFloat8(tags, "GPSLatitude", 90),
		Lon:         coordFloat8(tags, "GPSLongitude", 180),
		Alt:         tagFloat4(tags, "GPSAltitude"),
		TzOffset:    textOrNil(exifTime.tzOffset),
	}
}

// refineParams maps one extraction onto the asset row refinement.
// taken_at is set only when the EXIF time resolved (ok); the SQL then
// moves the value and flips taken_at_source to 'exif', and a NULL
// leaves the upload's chain untouched.
//
// Dimensions are stored in sensor order, deliberately NOT
// orientation-corrected: the file reports them that way, and the
// serving copies handle orientation themselves at render time
// (libvips auto-rotates when it derives the thumbnails).
func refineParams(assetID pgtype.UUID, tags map[string]any, exifTime resolvedTime, ok bool) dbgen.RefineAssetFromMetadataParams {
	params := dbgen.RefineAssetFromMetadataParams{
		ID:         assetID,
		Width:      firstTagInt4(tags, "ImageWidth", "ExifImageWidth"),
		Height:     firstTagInt4(tags, "ImageHeight", "ExifImageHeight"),
		DurationMs: videoDuration(tags),
	}
	if ok {
		params.TakenAt = pgtype.Timestamptz{Time: exifTime.when.UTC(), Valid: true}
	}
	return params
}

// videoDuration flattens the container duration of a video file into
// milliseconds (exiftool -n renders it as seconds). Image extractions
// report none and keep the column NULL.
func videoDuration(tags map[string]any) pgtype.Int4 {
	for _, name := range []string{"Duration", "TrackDuration", "MediaDuration"} {
		if seconds, ok := tagFloat(tags, name); ok && seconds > 0 {
			millis := seconds * 1000
			if millis >= math.MaxInt32 {
				return pgtype.Int4{}
			}
			return pgtype.Int4{Int32: int32(millis + 0.5), Valid: true}
		}
	}
	return pgtype.Int4{}
}

// shutterText renders the exposure time (exiftool -n renders it as
// seconds) in the notation photo interfaces expect: "1/250" when the
// reciprocal is a whole number, seconds otherwise ("2.5s").
func shutterText(tags map[string]any) pgtype.Text {
	seconds, ok := tagFloat(tags, "ExposureTime")
	if !ok || seconds <= 0 {
		return pgtype.Text{}
	}
	reciprocal := 1 / seconds
	if reciprocal >= 1 && math.Abs(reciprocal-math.Round(reciprocal)) < 1e-6 {
		return pgtype.Text{String: fmt.Sprintf("1/%d", int(math.Round(reciprocal))), Valid: true}
	}
	return pgtype.Text{String: strconv.FormatFloat(seconds, 'g', -1, 64) + "s", Valid: true}
}

// tagValue returns the tag's value when present and non-null.
func tagValue(tags map[string]any, name string) (any, bool) {
	value, ok := tags[name]
	if !ok || value == nil {
		return nil, false
	}
	return value, true
}

// tagString returns a text tag; exiftool emits strings for text tags
// (numbers arrive as JSON numbers and do not masquerade as text).
func tagString(tags map[string]any, name string) (string, bool) {
	value, ok := tagValue(tags, name)
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

// tagText wraps tagString as the nullable column value.
func tagText(tags map[string]any, name string) pgtype.Text {
	if text, ok := tagString(tags, name); ok {
		return pgtype.Text{String: text, Valid: true}
	}
	return pgtype.Text{}
}

// firstTagText returns the first present text tag of the names.
func firstTagText(tags map[string]any, names ...string) pgtype.Text {
	for _, name := range names {
		if text, ok := tagString(tags, name); ok {
			return pgtype.Text{String: text, Valid: true}
		}
	}
	return pgtype.Text{}
}

// tagFloat returns a numeric tag. exiftool -n renders numerics as JSON
// numbers; numeric strings are still accepted for tags that predate
// the coercion.
func tagFloat(tags map[string]any, name string) (float64, bool) {
	value, ok := tagValue(tags, name)
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, false
		}
		return typed, true
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

// tagInt4 flattens a numeric tag into the nullable int column.
func tagInt4(tags map[string]any, name string) pgtype.Int4 {
	if number, ok := tagFloat(tags, name); ok && number >= math.MinInt32 && number <= math.MaxInt32 {
		return pgtype.Int4{Int32: int32(math.Round(number)), Valid: true}
	}
	return pgtype.Int4{}
}

// firstTagInt4 flattens the first present numeric tag of the names.
func firstTagInt4(tags map[string]any, names ...string) pgtype.Int4 {
	for _, name := range names {
		if params := tagInt4(tags, name); params.Valid {
			return params
		}
	}
	return pgtype.Int4{}
}

// tagFloat4 flattens a numeric tag into the nullable real column.
func tagFloat4(tags map[string]any, name string) pgtype.Float4 {
	if number, ok := tagFloat(tags, name); ok && math.Abs(number) <= math.MaxFloat32 {
		return pgtype.Float4{Float32: float32(number), Valid: true}
	}
	return pgtype.Float4{}
}

// coordFloat8 flattens a GPS coordinate, refusing values outside the
// column's CHECK bounds (a corrupted extraction must not fail the
// whole job).
func coordFloat8(tags map[string]any, name string, bound float64) pgtype.Float8 {
	if number, ok := tagFloat(tags, name); ok && math.Abs(number) <= bound {
		return pgtype.Float8{Float64: number, Valid: true}
	}
	return pgtype.Float8{}
}

// textOrNil wraps a possibly empty string as the nullable text column.
func textOrNil(text string) pgtype.Text {
	if text == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: text, Valid: true}
}

// pgUUID converts a uuid.UUID into the pgx scan/param type.
func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
