// Typed accessors of the instance_settings key-value table: the raw
// jsonb values leave the sqlc layer as bytes, and the interpretation
// -- which key exists, what type it carries, what the default is --
// belongs to a typed getter next to its consumers, not to every
// call site. The trash retention (M2c Task 3) is the first consumer:
// the trash listing's countdown and the expiry job's cutoff both
// read the same value through this function. The settings feature
// adds the trash retention's writer (the admin surface's save), the
// storage template's read/write seam, and the video transcoding
// profile's read/write pair (the transcode and thumbnail jobs'
// preset source). The albums-as-sources milestone (E4) adds the
// mount-root allowlist's read/write pair: the directory-album trust
// boundary the path completion and the album creation share.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// TrashRetentionKey is the instance_settings key holding the
	// trash retention in whole days.
	TrashRetentionKey = "trash_retention_days"

	// TrashRetentionDefault is the retention when the setting row is
	// absent (the design spec's default value for the retention).
	TrashRetentionDefault = 30

	// TrashRetentionMax bounds the retention the admin surface
	// accepts: a year is the honest ceiling of a recovery view.
	TrashRetentionMax = 365

	// StorageTemplateKey is the instance_settings key holding the
	// storage template: the path grammar every NEW upload renders its
	// file placement from (existing assets never move).
	StorageTemplateKey = "storage_template"

	// StorageTemplateMaxRunes bounds the stored template's length --
	// the same bound the template engine's compiler enforces.
	StorageTemplateMaxRunes = 512

	// TranscodingSettingsKey is the instance_settings key holding
	// the video transcoding profile (the preset, the concurrency
	// bound, the poster-frame switch, the failure policy) as one
	// JSON object.
	TranscodingSettingsKey = "transcoding_settings"

	// AllowedMountRootsKey is the instance_settings key holding the
	// mount-root allowlist: the absolute directory paths a directory
	// album's source may live at-or-under. The stored value is a
	// JSON array of strings; the default is the empty array, the
	// "closed" posture (no directory album can be created until an
	// administrator opens at least one root).
	AllowedMountRootsKey = "allowed_mount_roots"

	// MountRootsMaxEntries bounds the allowlist the admin surface
	// accepts: a family instance has a handful of photo mounts, and
	// the bound keeps the completion's domain read bounded too.
	MountRootsMaxEntries = 32

	// MountRootMaxBytes bounds one configured root path: the
	// filesystem's own path ceiling, so a stored root can always be
	// handed to the operating system verbatim.
	MountRootMaxBytes = 4096
)

// TranscodingProfile is the instance's video transcoding settings:
// the preset the transcode job re-encodes new ingests at, how many
// re-encodes may run at once, whether the video thumbnail job
// captures a poster frame, and the failed-re-encode policy. It is
// the typed form of the instance_settings row the admin surface
// reads and writes; the zero value is never used (the accessor
// always answers the documented defaults). The JSON tags match the
// contract's TranscodingSettings field names, so the stored row's
// shape and the API's shape are one shape.
type TranscodingProfile struct {
	Preset         string `json:"preset"`
	Concurrency    int    `json:"concurrency"`
	ThumbnailFrame bool   `json:"thumbnailFrame"`
	KeepOriginal   bool   `json:"keepOriginal"`
}

// TranscodingDefault is the profile when the setting row is absent
// (the design spec's default posture): keep the source resolution,
// two concurrent re-encodes, poster frames captured, and a failed
// re-encode keeps the original in place.
var TranscodingDefault = TranscodingProfile{
	Preset:         "auto",
	Concurrency:    2,
	ThumbnailFrame: true,
	KeepOriginal:   true,
}

// transcodingPresets are the preset values the profile admits; the
// contract's TranscodePreset enum is the same list.
var transcodingPresets = map[string]bool{
	"auto": true, "1080p": true, "720p": true, "480p": true, "off": true,
}

// transcodingConcurrency values the profile admits; the contract's
// enum is the same list.
var transcodingConcurrency = map[int]bool{1: true, 2: true, 4: true}

// ValidTranscodingPreset reports whether preset is a member of the
// contract's TranscodePreset enum.
func ValidTranscodingPreset(preset string) bool {
	return transcodingPresets[preset]
}

// ValidTranscodingConcurrency reports whether concurrency is a
// member of the contract's concurrency enum.
func ValidTranscodingConcurrency(concurrency int) bool {
	return transcodingConcurrency[concurrency]
}

// TrashRetentionDays reads the instance's trash retention in days.
// A missing setting row answers the default (an instance that never
// wrote the setting runs on the spec's default). A value that is not
// a non-negative integer (the row exists but holds garbage) ALSO
// answers the default, together with the error describing it: the
// callers log the error and proceed on the returned value, because
// refusing to run over a corrupt setting would silently stop every
// trash expiry on the instance -- the default keeps the lifecycle
// honest while the log line points at the row to fix. The value 0 is
// valid (no retention: everything trashed expires on the next run);
// negative values are treated as corrupt, since they would read as
// "never expire" under a naive cutoff computation.
func TrashRetentionDays(ctx context.Context, q *dbgen.Queries) (int, error) {
	raw, err := q.GetInstanceSetting(ctx, TrashRetentionKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return TrashRetentionDefault, nil
	}
	if err != nil {
		return TrashRetentionDefault, fmt.Errorf("db: read instance setting %s: %w", TrashRetentionKey, err)
	}
	var days int
	if err := json.Unmarshal(raw, &days); err != nil || days < 0 {
		return TrashRetentionDefault, fmt.Errorf("db: instance setting %s = %s is not a non-negative integer", TrashRetentionKey, raw)
	}
	return days, nil
}

// StorageTemplateDefault is the template when the setting row is
// absent: the year/month organization of the camera roll. It is
// deliberately a value of the db package (not a call into the
// template engine) so the accessor stays a leaf every caller can
// reach; the api layer compiles it through the engine.
const StorageTemplateDefault = "{{y}}/{{MM}}/{{filename}}"

// StorageTemplate reads the instance's storage template. A missing
// setting row answers the default (an instance that never wrote the
// setting runs on the default layout). A value that is not a
// non-empty string within the length bound ALSO answers the default,
// together with the error describing it: the upload path calls this
// on every request and must never fail over a corrupt setting row --
// the admin surface validates a template before it saves, so a bad
// row is a corruption the caller logs, not an outage it honors. The
// returned text is NOT grammar-checked here; compiling it is the
// caller's step (the engine lives in the storage package, above this
// leaf).
func StorageTemplate(ctx context.Context, q *dbgen.Queries) (string, error) {
	raw, err := q.GetInstanceSetting(ctx, StorageTemplateKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return StorageTemplateDefault, nil
	}
	if err != nil {
		return StorageTemplateDefault, fmt.Errorf("db: read instance setting %s: %w", StorageTemplateKey, err)
	}
	var template string
	if err := json.Unmarshal(raw, &template); err != nil {
		return StorageTemplateDefault, fmt.Errorf("db: instance setting %s = %s is not a JSON string", StorageTemplateKey, raw)
	}
	if utf8.RuneCountInString(template) == 0 || utf8.RuneCountInString(template) > StorageTemplateMaxRunes {
		return StorageTemplateDefault, fmt.Errorf("db: instance setting %s is empty or longer than %d runes", StorageTemplateKey, StorageTemplateMaxRunes)
	}
	return template, nil
}

// SaveTrashRetentionDays writes the instance's trash retention. The
// value lands as a jsonb NUMBER (the raw writer), because the read
// side unmarshals an integer -- a text-cast write would store the
// JSON string "30" and every reader would fall back to the default.
// The caller validates the 0-365 bound; this leaf only writes.
func SaveTrashRetentionDays(ctx context.Context, q *dbgen.Queries, days int) error {
	if err := q.SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   TrashRetentionKey,
		Value: []byte(strconv.Itoa(days)),
	}); err != nil {
		return fmt.Errorf("db: write instance setting %s: %w", TrashRetentionKey, err)
	}
	return nil
}

// TranscodingSettings reads the instance's video transcoding
// profile. A missing setting row answers the documented default (an
// instance that never wrote the setting runs on the default
// posture). A row that is not a structurally valid profile -- a
// broken JSON object, a preset or concurrency outside its enum, a
// missing field -- ALSO answers the default, together with the
// error describing it: the transcode job reads this on every video
// ingest and must never fail over a corrupt settings row (the admin
// surface's save validates before it writes, so a bad row is a
// corruption the caller logs while the pipeline keeps running on
// the default, exactly like the trash retention's corrupt row).
// Partial corruption degrades to the default per FIELD, so a row
// that lost one field keeps the honest rest.
func TranscodingSettings(ctx context.Context, q *dbgen.Queries) (TranscodingProfile, error) {
	profile := TranscodingDefault
	raw, err := q.GetInstanceSetting(ctx, TranscodingSettingsKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile, nil
	}
	if err != nil {
		return profile, fmt.Errorf("db: read instance setting %s: %w", TranscodingSettingsKey, err)
	}
	var stored map[string]json.RawMessage
	if err := json.Unmarshal(raw, &stored); err != nil {
		return profile, fmt.Errorf("db: instance setting %s = %s is not a JSON object", TranscodingSettingsKey, raw)
	}
	if value, ok := stored["preset"]; ok {
		var preset string
		if err := json.Unmarshal(value, &preset); err != nil || !transcodingPresets[preset] {
			return profile, fmt.Errorf("db: instance setting %s preset = %s is not a member of the enum", TranscodingSettingsKey, value)
		}
		profile.Preset = preset
	}
	if value, ok := stored["concurrency"]; ok {
		var concurrency int
		if err := json.Unmarshal(value, &concurrency); err != nil || !transcodingConcurrency[concurrency] {
			return profile, fmt.Errorf("db: instance setting %s concurrency = %s is not a member of the enum", TranscodingSettingsKey, value)
		}
		profile.Concurrency = concurrency
	}
	for field, into := range map[string]*bool{
		"thumbnailFrame": &profile.ThumbnailFrame,
		"keepOriginal":   &profile.KeepOriginal,
	} {
		if value, ok := stored[field]; ok {
			if err := json.Unmarshal(value, into); err != nil {
				return profile, fmt.Errorf("db: instance setting %s %s = %s is not a JSON boolean", TranscodingSettingsKey, field, value)
			}
		}
	}
	return profile, nil
}

// SaveTranscodingSettings writes the instance's video transcoding
// profile as one JSON object (the raw writer). The caller validates
// every field against its enum; this leaf only writes.
func SaveTranscodingSettings(ctx context.Context, q *dbgen.Queries, profile TranscodingProfile) error {
	value, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("db: encode instance setting %s: %w", TranscodingSettingsKey, err)
	}
	if err := q.SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   TranscodingSettingsKey,
		Value: value,
	}); err != nil {
		return fmt.Errorf("db: write instance setting %s: %w", TranscodingSettingsKey, err)
	}
	return nil
}

// AllowedMountRoots reads the instance's mount-root allowlist, the
// directory-album trust boundary. A missing setting row answers the
// empty list (the default posture: directory album creation closed).
// A row that is not a JSON array of strings -- a broken array, a
// non-string element, an entry past the length or count bounds --
// ALSO answers the empty list together with the error describing
// it: the completion and the creation validation read this on every
// request and must never fail over a corrupt settings row (the admin
// surface validates before it saves, so a bad row is a corruption
// the caller logs while the boundary stays closed -- the corrupt
// state is the safe state, exactly like the trash retention's and
// the transcoding profile's corrupt-row discipline).
//
// The answer is the STORED list verbatim; whether each root actually
// exists, is a directory, and sits outside the managed tree is the
// caller's domain computation (api), not this leaf's.
func AllowedMountRoots(ctx context.Context, q *dbgen.Queries) ([]string, error) {
	raw, err := q.GetInstanceSetting(ctx, AllowedMountRootsKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: read instance setting %s: %w", AllowedMountRootsKey, err)
	}
	var stored []string
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, fmt.Errorf("db: instance setting %s = %s is not a JSON array of strings", AllowedMountRootsKey, raw)
	}
	if len(stored) > MountRootsMaxEntries {
		return nil, fmt.Errorf("db: instance setting %s holds %d entries, over the bound of %d",
			AllowedMountRootsKey, len(stored), MountRootsMaxEntries)
	}
	for _, root := range stored {
		if root == "" || len(root) > MountRootMaxBytes {
			return nil, fmt.Errorf("db: instance setting %s holds an empty entry or one past the %d-byte bound",
				AllowedMountRootsKey, MountRootMaxBytes)
		}
	}
	return stored, nil
}

// SaveAllowedMountRoots writes the instance's mount-root allowlist
// as one JSON array of strings (the raw writer). The caller
// validates the list's shape (absolute paths, the count and length
// bounds, the collapse of duplicates); this leaf only writes.
func SaveAllowedMountRoots(ctx context.Context, q *dbgen.Queries, roots []string) error {
	value, err := json.Marshal(roots)
	if err != nil {
		return fmt.Errorf("db: encode instance setting %s: %w", AllowedMountRootsKey, err)
	}
	if err := q.SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   AllowedMountRootsKey,
		Value: value,
	}); err != nil {
		return fmt.Errorf("db: write instance setting %s: %w", AllowedMountRootsKey, err)
	}
	return nil
}
