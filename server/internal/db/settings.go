// Typed accessors of the instance_settings key-value table: the raw
// jsonb values leave the sqlc layer as bytes, and the interpretation
// -- which key exists, what type it carries, what the default is --
// belongs to a typed getter next to its consumers, not to every
// call site. The trash retention (M2c Task 3) is the first consumer:
// the trash listing's countdown and the expiry job's cutoff both
// read the same value through this function.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// StorageTemplateKey is the instance_settings key holding the
	// storage template: the path grammar every NEW upload renders its
	// file placement from (existing assets never move).
	StorageTemplateKey = "storage_template"

	// StorageTemplateMaxRunes bounds the stored template's length --
	// the same bound the template engine's compiler enforces.
	StorageTemplateMaxRunes = 512
)

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
