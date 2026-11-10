// The model registry: the models table is the authority
// for what is registered and which model fills each slot; the
// per-model manifest on disk is the checksum authority. The registry
// ties the two together -- activation verifies the files against their
// pinned digests before flipping any flag, and the dimension policy
// for the clip-embed slot reads the embeddings column's current
// dimension from the database catalog, never from an assumption.
//
// Disk layout (pinned by the default-manifest task, interpreted here):
// the models dir holds one directory per model id containing
// manifest.json plus the weights. models.file_path stores the model's
// primary weights file relative to the models dir (<id>/<file>); the
// manifest's first file is the primary one, and the remaining files of
// multi-file models (the OCR detector/recognition pair) resolve
// through the same manifest. Downloaded, custom-scanned and
// hand-registered rows share the layout.
package ml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// Slot is a functional slot of the ML pipeline. The values are the
// model_slot enum's database values; the manifests' hyphenated slot
// names (face-detect and friends) map onto them below.
type Slot string

// The five inference slots. The database enum also carries stt, which
// stays unclaimed until a later milestone adds a speech model.
const (
	SlotFaceDetect    Slot = "face_detect"
	SlotFaceRecognize Slot = "face_recognize"
	SlotClipEmbed     Slot = "clip_embed"
	SlotClipText      Slot = "clip_text"
	SlotOcr           Slot = "ocr"
)

// manifestSlotValues maps the manifest slot names (hyphenated, as the
// default manifest writes them) onto the Slot values the database
// enum carries.
var manifestSlotValues = map[string]Slot{
	"face-detect":    SlotFaceDetect,
	"face-recognize": SlotFaceRecognize,
	"clip-embed":     SlotClipEmbed,
	"clip-text":      SlotClipText,
	"ocr":            SlotOcr,
}

// The registry's error contract. ErrNoActiveModel is the sentinel the
// job layer's readiness seam consumes: an unfilled slot means ML jobs
// are simply not dispatched.
var (
	ErrNoActiveModel = errors.New("ml: no active model for the slot")
	ErrModelNotFound = errors.New("ml: model not found")
	ErrInvalidModel  = errors.New("ml: invalid model")
)

// manifestFileName is the per-model manifest's fixed name, sitting
// next to the weights it pins.
const manifestFileName = "manifest.json"

// Model is a registered model as the consumers see it: the database
// row's facts (identity, slot, source, dimension, params) plus the
// resolved path of the primary weights file and, when the on-disk
// manifest is readable, the pinned digest, the input geometry, the
// upstream license and the provenance notice. It carries no
// activation flag on purpose: activation state is a fact of the
// models table, and listing it is the admin surface's business.
type Model struct {
	ID      string
	Version string
	Slot    Slot
	Source  string
	Path    string
	SHA256  string
	License string
	// Notice is the manifest's provenance notice line, or nil when
	// the manifest is unreadable (the fact is unknowable). The empty
	// string is a readable manifest that declares no notice.
	Notice *string
	Width  int
	Height int
	Dim    int
	Params map[string]any
}

// Registry resolves and activates models. The pinned interface carries
// no contexts: the operations are short local-database calls and run
// under context.Background(). Construct with NewRegistry.
type Registry struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	modelsDir string

	// alterEmbeddingDim runs the dimension ALTER of the
	// cross-dimension activation inside its transaction. Production
	// leaves it nil (the ALTER below runs); tests replace it to fail
	// the step mid-transaction and pin the rollback.
	alterEmbeddingDim func(ctx context.Context, tx pgx.Tx, dim int) error
}

// NewRegistry builds a registry over the shared pool. The pool carries
// the activation transaction (including its direct DDL); the queries
// carry the generated statements; modelsDir is where the model
// directories live and what models.file_path resolves against.
func NewRegistry(pool *pgxpool.Pool, queries *dbgen.Queries, modelsDir string) *Registry {
	return &Registry{pool: pool, queries: queries, modelsDir: modelsDir}
}

// Active returns the slot's active model. An unfilled slot answers
// ErrNoActiveModel -- the sentinel the job layer's readiness seam
// consumes to skip ML dispatch without an error. A model whose
// manifest is unreadable or does not list the registered file answers
// ErrInvalidModel: handing out an unverified session is worse than
// failing.
func (r *Registry) Active(slot Slot) (*Model, error) {
	ctx := context.Background()
	row, err := r.queries.GetActiveModelBySlot(ctx, dbgen.ModelSlot(slot))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrNoActiveModel, slot)
	}
	if err != nil {
		return nil, fmt.Errorf("ml: load active model of slot %s: %w", slot, err)
	}
	return r.modelFromRow(&row, true)
}

// List returns every registered model, grouped by slot. It is the
// admin listing: a model whose manifest has gone missing still lists
// (with its database facts and a zero digest) instead of failing the
// whole listing -- the admin surface needs to see the row to clean it
// up.
func (r *Registry) List() ([]Model, error) {
	ctx := context.Background()
	rows, err := r.queries.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("ml: list models: %w", err)
	}
	models := make([]Model, 0, len(rows))
	for i := range rows {
		model, err := r.modelFromRow(&rows[i], false)
		if err != nil {
			return nil, err
		}
		models = append(models, *model)
	}
	return models, nil
}

// ActivationReport is what one activation did. The zero value is the
// same-dimension switch's report: nothing but the flag flip -- the
// old model's products survive untouched. A cross-dimension switch of
// the clip-embed slot fills in the dimension change and the wipe, so
// the admin surface can report the destructive arm exactly.
type ActivationReport struct {
	// DimensionChanged is true when the switch re-typed the
	// clip_embeddings column.
	DimensionChanged bool
	// FromDim and ToDim are the column's dimensions around the
	// change.
	FromDim int
	ToDim   int
	// WipedEmbeddings is the number of embedding rows the change
	// deleted inside its transaction.
	WipedEmbeddings int
}

// Activate makes the model fill its slot. The switch runs in one
// transaction: the slot's previous active model is deactivated, the
// new one activated, and -- only for the clip-embed slot, only when
// the incoming model's dimension differs from the embedding column's
// current one (read from the database catalog) -- clip_embeddings is
// cleared and the column re-typed to vector(N) first, so the wiped
// rows, the new dimension and the activation change land together or
// not at all. A same-dimension switch flips the flags only: old
// embedding rows survive, and switching back revives them untouched.
//
// The model's files are verified against their pinned digests before
// the transaction starts: activation never turns a model without
// loadable files into the active one. The wipe and the ALTER are
// executed directly over the transaction (never through sqlc: DDL and
// the pg_attribute read are beyond the schema-driven generator).
func (r *Registry) Activate(slot Slot, id string) (ActivationReport, error) {
	ctx := context.Background()
	var report ActivationReport

	row, err := r.queries.ModelByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return report, fmt.Errorf("%w: %s", ErrModelNotFound, id)
	}
	if err != nil {
		return report, fmt.Errorf("ml: load model %s: %w", id, err)
	}
	if Slot(row.Slot) != slot {
		return report, fmt.Errorf("%w: %s fills slot %s, not %s", ErrInvalidModel, id, row.Slot, slot)
	}
	if err := r.verifyModelFiles(&row); err != nil {
		return report, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return report, fmt.Errorf("ml: begin activation of %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed

	qtx := r.queries.WithTx(tx)

	if Slot(row.Slot) == SlotClipEmbed && row.Dim.Valid && row.Dim.Int32 > 0 {
		current, err := EmbeddingColumnDim(ctx, tx)
		if err != nil {
			return report, err
		}
		if current != int(row.Dim.Int32) {
			tag, err := tx.Exec(ctx, "DELETE FROM clip_embeddings")
			if err != nil {
				return report, fmt.Errorf("ml: clear clip_embeddings for the dimension switch: %w", err)
			}
			alter := r.alterEmbeddingDim
			if alter == nil {
				alter = alterEmbeddingColumn
			}
			if err := alter(ctx, tx, int(row.Dim.Int32)); err != nil {
				return report, fmt.Errorf("ml: re-dimension clip_embeddings to vector(%d): %w", row.Dim.Int32, err)
			}
			report.DimensionChanged = true
			report.FromDim = current
			report.ToDim = int(row.Dim.Int32)
			report.WipedEmbeddings = int(tag.RowsAffected())
		}
	}

	if err := qtx.DeactivateSlot(ctx, dbgen.ModelSlot(slot)); err != nil {
		return report, fmt.Errorf("ml: deactivate previous %s model: %w", slot, err)
	}
	if err := qtx.ActivateModel(ctx, id); err != nil {
		return report, fmt.Errorf("ml: activate %s: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return report, fmt.Errorf("ml: commit activation of %s: %w", id, err)
	}
	return report, nil
}

// Scan registers the custom models found under dir: every
// subdirectory carrying manifest.json becomes a models row with source
// custom, its files verified against the pinned digests first (a
// drifted or missing file fails the scan with ErrInvalidModel and
// registers nothing). Directories without a manifest are skipped --
// the models dir also holds download staging, not every entry is a
// model. Scanning registers, it never activates. Re-scanning
// converges: the insert is an upsert on the descriptive fields and
// leaves every row's activation state alone. The returned models are
// the ones the scan verified and registered, in scan order.
//
// The scanned models must live under the registry's models dir: the
// registered file_path is relative to it, and a model outside it is
// refused with ErrInvalidModel.
func (r *Registry) Scan(dir string) ([]Model, error) {
	ctx := context.Background()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ml: scan %s: %w", dir, err)
	}
	var models []Model
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modelDir := filepath.Join(dir, entry.Name())
		manifestPath := filepath.Join(modelDir, manifestFileName)
		if _, err := os.Stat(manifestPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("ml: stat %s: %w", manifestPath, err)
		}
		model, err := r.scanModelDir(ctx, modelDir, entry.Name())
		if err != nil {
			return nil, err
		}
		models = append(models, *model)
	}
	return models, nil
}

// scanModelDir verifies and registers one model directory, returning
// the registered model.
func (r *Registry) scanModelDir(ctx context.Context, modelDir, id string) (*Model, error) {
	manifest, err := loadManifestModel(modelDir)
	if err != nil {
		return nil, err
	}
	if manifest.ID != id {
		return nil, fmt.Errorf("%w: %s declares id %q, want the directory name %q",
			ErrInvalidModel, filepath.Join(modelDir, manifestFileName), manifest.ID, id)
	}
	slot, ok := manifestSlotValues[manifest.Slot]
	if !ok {
		return nil, fmt.Errorf("%w: %s fills unknown slot %q", ErrInvalidModel, id, manifest.Slot)
	}
	if len(manifest.Files) == 0 {
		return nil, fmt.Errorf("%w: %s lists no files", ErrInvalidModel, id)
	}
	if err := verifyManifestFiles(modelDir, manifest); err != nil {
		return nil, err
	}

	// The registered path is the primary file, relative to the models
	// dir; a model outside it cannot be addressed.
	primary := manifest.Files[0].Name
	weights := filepath.Join(modelDir, primary)
	rel, err := filepath.Rel(r.modelsDir, weights)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("%w: %s is outside the models dir", ErrInvalidModel, weights)
	}

	params := []byte("{}")
	if manifest.Params != nil {
		params, err = json.Marshal(manifest.Params)
		if err != nil {
			return nil, fmt.Errorf("%w: marshal params of %s: %w", ErrInvalidModel, id, err)
		}
	}

	if err := r.queries.InsertModel(ctx, dbgen.InsertModelParams{
		ID:       id,
		Slot:     dbgen.ModelSlot(slot),
		Version:  manifest.Version,
		Source:   dbgen.ModelSourceCustom,
		FilePath: rel,
		InputSize: pgtype.Int4{
			Int32: int32(manifest.InputSize.Width),
			Valid: manifest.InputSize.Width > 0,
		},
		Dim: pgtype.Int4{
			Int32: int32(manifest.Dim),
			Valid: manifest.Dim > 0,
		},
		Params: params,
	}); err != nil {
		return nil, fmt.Errorf("ml: register %s: %w", id, err)
	}

	model := &Model{
		ID:      id,
		Version: manifest.Version,
		Slot:    slot,
		Source:  string(dbgen.ModelSourceCustom),
		Path:    weights,
		License: manifest.License,
		Width:   manifest.InputSize.Width,
		Height:  manifest.InputSize.Height,
		Params:  manifest.Params,
	}
	notice := manifest.Notice
	model.Notice = &notice
	if manifest.Dim > 0 {
		model.Dim = manifest.Dim
	}
	for _, file := range manifest.Files {
		if file.Name == primary {
			model.SHA256 = file.SHA256
			break
		}
	}
	return model, nil
}

// modelFromRow builds the Model value from a database row. The
// database carries identity, slot, source, dimension and params; the
// on-disk manifest carries the pinned digest and the input geometry.
// Strict callers (Active) fail on a manifest that is unreadable or
// does not list the registered file; lenient callers (List) keep the
// row's facts and leave the manifest fields zero.
func (r *Registry) modelFromRow(row *dbgen.Model, strict bool) (*Model, error) {
	model := &Model{
		ID:      row.ID,
		Version: row.Version,
		Slot:    Slot(row.Slot),
		Source:  string(row.Source),
		Path:    r.resolvePath(row.FilePath),
	}
	if row.Dim.Valid {
		model.Dim = int(row.Dim.Int32)
	}
	if len(row.Params) > 0 {
		if err := json.Unmarshal(row.Params, &model.Params); err != nil {
			return nil, fmt.Errorf("%w: params of %s: %w", ErrInvalidModel, row.ID, err)
		}
	}

	manifest, err := loadManifestModel(filepath.Dir(model.Path))
	if err != nil {
		if strict {
			return nil, err
		}
		return model, nil
	}
	model.Width = manifest.InputSize.Width
	model.Height = manifest.InputSize.Height
	model.License = manifest.License
	notice := manifest.Notice
	model.Notice = &notice
	for _, file := range manifest.Files {
		if file.Name == filepath.Base(model.Path) {
			model.SHA256 = file.SHA256
			break
		}
	}
	if strict && model.SHA256 == "" {
		return nil, fmt.Errorf("%w: %s is not listed in the model manifest", ErrInvalidModel, model.Path)
	}
	return model, nil
}

// verifyModelFiles verifies a registered row against its on-disk
// manifest: the manifest must be readable, every file must match its
// pinned digest, and the registered path must be one of the listed
// files.
func (r *Registry) verifyModelFiles(row *dbgen.Model) error {
	path := r.resolvePath(row.FilePath)
	dir := filepath.Dir(path)
	manifest, err := loadManifestModel(dir)
	if err != nil {
		return err
	}
	if err := verifyManifestFiles(dir, manifest); err != nil {
		return err
	}
	for _, file := range manifest.Files {
		if file.Name == filepath.Base(path) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s is not listed in the model manifest", ErrInvalidModel, path)
}

// verifyManifestFiles checks that every file the manifest pins exists
// and matches its pinned sha256.
func verifyManifestFiles(dir string, manifest *ManifestModel) error {
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		if file.Name == "" {
			return fmt.Errorf("%w: manifest in %s lists a file with an empty name", ErrInvalidModel, dir)
		}
		if seen[file.Name] {
			return fmt.Errorf("%w: manifest in %s lists file %q twice", ErrInvalidModel, dir, file.Name)
		}
		seen[file.Name] = true
		if err := verifyModelDigest(filepath.Join(dir, file.Name), file.SHA256); err != nil {
			return err
		}
	}
	return nil
}

// loadManifestModel reads and parses the per-model manifest of a model
// directory.
func loadManifestModel(dir string) (*ManifestModel, error) {
	path := filepath.Join(dir, manifestFileName)
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrInvalidModel, path, err)
	}
	var manifest ManifestModel
	if err := json.Unmarshal(blob, &manifest); err != nil {
		return nil, fmt.Errorf("%w: parse %s: %w", ErrInvalidModel, path, err)
	}
	return &manifest, nil
}

// FilesPresent probes a model's files: true when the model's on-disk
// manifest is readable and every file it pins exists. It is a
// PRESENCE probe only -- deliberately cheap for listings -- while the
// sha256 verification itself stays the activation's and the session
// loader's business: a drifted file shows present here and still
// fails to activate. A model whose manifest is unreadable answers
// false: without the manifest the file set is unknowable and the
// model is not usable.
func FilesPresent(model Model) bool {
	dir := filepath.Dir(model.Path)
	manifest, err := loadManifestModel(dir)
	if err != nil {
		return false
	}
	for _, file := range manifest.Files {
		if _, err := os.Stat(filepath.Join(dir, file.Name)); err != nil {
			return false
		}
	}
	return true
}

// ModelFile is one file of a registered model, resolved from its
// on-disk manifest: the pinned name, the absolute path and the pinned
// sha256 digest.
type ModelFile struct {
	Name   string
	Path   string
	SHA256 string
}

// ModelFiles answers a registered model's pinned files in the
// manifest's order (the primary first -- the file the models row's
// file_path registers), with resolved paths and digests. The
// multi-file consumers resolve their graphs this way: the OCR model's
// directory pins the detection and recognition pair, and the OCR job
// opens one session over each.
func ModelFiles(model *Model) ([]ModelFile, error) {
	dir := filepath.Dir(model.Path)
	manifest, err := loadManifestModel(dir)
	if err != nil {
		return nil, err
	}
	if len(manifest.Files) == 0 {
		return nil, fmt.Errorf("%w: %s lists no files", ErrInvalidModel, model.ID)
	}
	files := make([]ModelFile, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.Name == "" {
			return nil, fmt.Errorf("%w: the manifest of %s lists a file with an empty name", ErrInvalidModel, model.ID)
		}
		files = append(files, ModelFile{
			Name:   file.Name,
			Path:   filepath.Join(dir, file.Name),
			SHA256: file.SHA256,
		})
	}
	return files, nil
}

// resolvePath resolves a models.file_path value: absolute paths are
// taken as they are, anything else is relative to the models dir.
func (r *Registry) resolvePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(r.modelsDir, filePath)
}

// rowQuerier is the one method the catalog read needs; the pool, every
// pool transaction and every explicit transaction carry it.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// EmbeddingColumnDim reads the clip_embeddings embedding column's
// current dimension from the database catalog (pgvector stores the
// dimension in the attribute's typmod). Two consumers share it: the
// activation transaction's dimension switch (the cross-dimension
// clear-and-ALTER of the registry) and the CLIP embedding job's INSERT
// GUARD -- the column's current dimension is the
// database's truth, never an assumption, so an embedding whose length
// disagrees with it is refused before any insert.
func EmbeddingColumnDim(ctx context.Context, db rowQuerier) (int, error) {
	var dim int
	err := db.QueryRow(ctx, `
		SELECT atttypmod FROM pg_attribute
		WHERE attrelid = 'clip_embeddings'::regclass AND attname = 'embedding'`).Scan(&dim)
	if err != nil {
		return 0, fmt.Errorf("ml: read embedding column dimension: %w", err)
	}
	return dim, nil
}

// alterEmbeddingColumn re-types the embedding column inside the
// activation transaction. The table is empty at this point (the wipe
// ran in the same transaction), so pgvector rebuilds the HNSW index on
// it instantly. dim comes from a validated database integer, never
// from user input.
func alterEmbeddingColumn(ctx context.Context, tx pgx.Tx, dim int) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(
		"ALTER TABLE clip_embeddings ALTER COLUMN embedding TYPE vector(%d)", dim))
	return err
}
