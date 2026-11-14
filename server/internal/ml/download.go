// The model download service: it lands the manifest's
// models into the models dir, production driving the embedded default
// group (pure links: upstream URLs plus pinned digests, the repository
// committing no weights). Every file is fetched from its source URL --
// falling back to the model's secondary URL when the primary fails --
// streamed into a staging file under the model's directory, verified
// against the pinned sha256 and byte count, and only then atomically
// renamed into place: a half file never registers, and a crashed run's
// leftovers are replaced, not trusted, by the next attempt. A model
// whose every file verifies gets its manifest copy, a models row with
// source 'downloaded', and -- when its slot has no active model yet --
// the activation the registry verifies internally.
//
// Restricted networks reach the same artifacts through the HF domain
// mirror: a non-empty HFMirror replaces huggingface.co hostnames
// wholesale (LUMEMOR_MODELS_HF_MIRROR in production) and leaves every
// other host untouched.
//
// The progress callback reports cumulative pinned bytes: the first and
// the final reports are mandatory, every intermediate one is throttled
// to at most one per second, so a caller that publishes an event per
// report stays bounded on a large download.
package ml

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

//go:embed manifests/default.json
var defaultManifestJSON []byte

var (
	defaultManifestOnce sync.Once
	defaultManifest     *Manifest
	defaultManifestErr  error
)

// DefaultManifest parses the embedded default group's manifest. It is
// the manifest the model download job lands in production; tests point
// the service at their own manifests.
func DefaultManifest() (*Manifest, error) {
	defaultManifestOnce.Do(func() {
		var manifest Manifest
		if err := json.Unmarshal(defaultManifestJSON, &manifest); err != nil {
			defaultManifestErr = fmt.Errorf("ml: parse the embedded default manifest: %w", err)
			return
		}
		defaultManifest = &manifest
	})
	return defaultManifest, defaultManifestErr
}

// PinnedBytes sums the pinned byte counts of every file of every model:
// the total the download's progress reports and the job_runs row
// carries.
func (m *Manifest) PinnedBytes() int {
	total := 0
	for i := range m.Models {
		for _, file := range m.Models[i].Files {
			total += int(file.Bytes)
		}
	}
	return total
}

// RunDownloadDeps carries the dependencies of one download run: the
// worker passes the queue's wiring, tests pass local sources.
type RunDownloadDeps struct {
	// Manifest is the model group to land.
	Manifest *Manifest
	// ModelsDir is the model store root: each model lands in
	// <ModelsDir>/<id>/.
	ModelsDir string
	// Client fetches the source URLs. Nil falls back to a plain client
	// without a timeout -- the caller's context bounds the transfer (a
	// client-level timeout would abort large legitimate downloads).
	Client *http.Client
	// Registry registers landed models and activates empty slots.
	Registry *Registry
	// Queries writes the models rows.
	Queries *dbgen.Queries
	// HFMirror replaces huggingface.co hostnames wholesale when
	// non-empty; every other host stays untouched.
	HFMirror string
	// AllowedHosts gates every hop of every candidate URL: a URL
	// (or a redirect hop) whose host is outside the list fails with
	// the host named, and the walk moves to the next candidate. The
	// empty list is the unconfigured gate: every host is admitted
	// and the sha256 alone guards the bytes.
	AllowedHosts []string
	// Now drives the progress throttle; nil falls back to time.Now.
	Now func() time.Time
}

// DownloadReport summarizes one download run.
type DownloadReport struct {
	// BytesTotal is the pinned byte count of the whole group.
	BytesTotal int
	// BytesDone is the pinned byte count verified in place after the
	// run: landed this run, or skipped as already verified.
	BytesDone int
	// ModelsDone is the number of models fully landed, registered and
	// (when their slot was empty) activated.
	ModelsDone int
	// ModelsFailed is the number of models that could not be
	// completed.
	ModelsFailed int
	// FailedModels lists the failed models' ids, for the run's logs.
	FailedModels []string
}

// readChunkSize is the download's read buffer: large enough to keep
// the transfer efficient, small enough that a multi-megabyte file
// yields many progress attempts -- the density the throttle collapses.
const readChunkSize = 64 * 1024

// progressMinInterval is the throttle window of the progress callback:
// at most one intermediate report per second, plus the mandatory first
// and final ones.
const progressMinInterval = time.Second

// RunDownload lands the manifest's models into the models dir. A
// failing model fails the run (the error carries every model's cause)
// but never the other models: each one lands independently, so a
// partial network failure leaves the healthy models in place and the
// failed ones simply absent -- never registered, their slots stay
// empty. The returned report carries the counts the job layer's row
// and events publish; progress receives the throttled cumulative
// pinned-byte reports.
func RunDownload(ctx context.Context, dst RunDownloadDeps, progress func(done, total int)) (DownloadReport, error) {
	if dst.Manifest == nil {
		return DownloadReport{}, errors.New("ml: download requires a manifest")
	}
	if dst.ModelsDir == "" {
		return DownloadReport{}, errors.New("ml: download requires a models dir")
	}
	if dst.Registry == nil || dst.Queries == nil {
		return DownloadReport{}, errors.New("ml: download requires a registry and queries")
	}
	client := dst.Client
	if client == nil {
		client = &http.Client{}
	}
	if len(dst.AllowedHosts) > 0 {
		// The redirect gate wraps the caller's transport once per
		// run: the walk validates each candidate's first hop, and
		// the policy validates every hop a redirect reaches.
		client = clientWithHostGate(client, dst.AllowedHosts)
	}
	now := dst.Now
	if now == nil {
		now = time.Now
	}

	report := DownloadReport{BytesTotal: dst.Manifest.PinnedBytes()}
	emit := &progressEmitter{progress: progress, now: now}
	emit.report(0, report.BytesTotal, true)

	var failures []error
	for i := range dst.Manifest.Models {
		model := &dst.Manifest.Models[i]
		if err := runDownloadModel(ctx, dst, client, model, emit, &report); err != nil {
			report.ModelsFailed++
			report.FailedModels = append(report.FailedModels, model.ID)
			failures = append(failures, err)
			continue
		}
		report.ModelsDone++
	}

	emit.report(report.BytesDone, report.BytesTotal, true)

	if len(failures) > 0 {
		return report, fmt.Errorf("ml: download failed for %d model(s): %w", len(failures), errors.Join(failures...))
	}
	return report, nil
}

// runDownloadModel lands one model: every file fetched (or skipped as
// already verified), the manifest copy written, the models row
// upserted, and the empty slot activated. A failing file fails the
// model -- the model's remaining files stop, because a partially
// landed model is never registered -- but never the whole run.
func runDownloadModel(ctx context.Context, dst RunDownloadDeps, client *http.Client, model *ManifestModel, emit *progressEmitter, report *DownloadReport) error {
	slot, ok := manifestSlotValues[model.Slot]
	if !ok {
		return fmt.Errorf("model %s fills unknown slot %q", model.ID, model.Slot)
	}
	if len(model.Files) == 0 {
		return fmt.Errorf("model %s lists no files", model.ID)
	}

	modelDir := filepath.Join(dst.ModelsDir, model.ID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return fmt.Errorf("create the model dir of %s: %w", model.ID, err)
	}

	for _, file := range model.Files {
		target := filepath.Join(modelDir, file.Name)
		if verifiedFile(target, file.SHA256) {
			// Already landed by an earlier run: skip the fetch, count
			// the pinned bytes, keep the run idempotent.
			report.BytesDone += int(file.Bytes)
			emit.report(report.BytesDone, report.BytesTotal, false)
			continue
		}
		if err := fetchModelFile(ctx, dst, client, model, file, modelDir, target, emit, report); err != nil {
			return fmt.Errorf("land %s/%s: %w", model.ID, file.Name, err)
		}
	}

	// The per-model manifest copy: the registry's checksum authority,
	// sitting next to the weights it pins.
	if err := writeManifestCopy(modelDir, model); err != nil {
		return err
	}

	// The models row: the upsert makes re-runs converge on the same
	// row while leaving its activation state alone.
	params := []byte("{}")
	if model.Params != nil {
		blob, err := json.Marshal(model.Params)
		if err != nil {
			return fmt.Errorf("marshal the params of %s: %w", model.ID, err)
		}
		params = blob
	}
	if err := dst.Queries.InsertModel(ctx, dbgen.InsertModelParams{
		ID:       model.ID,
		Slot:     dbgen.ModelSlot(slot),
		Version:  model.Version,
		Source:   dbgen.ModelSourceDownloaded,
		FilePath: model.ID + "/" + model.Files[0].Name,
		InputSize: pgtype.Int4{
			Int32: int32(model.InputSize.Width),
			Valid: model.InputSize.Width > 0,
		},
		Dim: pgtype.Int4{
			Int32: int32(model.Dim),
			Valid: model.Dim > 0,
		},
		Params: params,
	}); err != nil {
		return fmt.Errorf("register %s: %w", model.ID, err)
	}

	// An empty slot auto-activates; a filled one -- even one whose
	// active model has gone bad -- keeps its model: switching is the
	// admin surface's business, not the download's.
	if _, err := dst.Registry.Active(slot); errors.Is(err, ErrNoActiveModel) {
		if _, err := dst.Registry.Activate(slot, model.ID); err != nil {
			return fmt.Errorf("activate %s into slot %s: %w", model.ID, slot, err)
		}
	}
	return nil
}

// fetchModelFile lands one file: the model's ordered candidate chain,
// each URL rewritten by the mirror first and tried in order (a
// timeout, a 404, a 5xx, a refused host or a digest mismatch all move
// the walk to the next candidate -- the failure is named for the
// report, but the chain is not abandoned early). The bytes stream
// into a staging file next to the target, verify against the pinned
// digest and size, and rename atomically into place.
func fetchModelFile(ctx context.Context, dst RunDownloadDeps, client *http.Client, model *ManifestModel, file ManifestFile, modelDir, target string, emit *progressEmitter, report *DownloadReport) error {
	var failures []error
	for _, candidate := range model.CandidateURLs() {
		rewritten, err := mirrorURL(candidate, dst.HFMirror)
		if err != nil {
			// A malformed pinned URL is a manifest bug, not a source
			// failure: no fallback can repair it.
			return err
		}
		fileURL, err := modelFileURL(rewritten, file.Name)
		if err != nil {
			return err
		}
		host, err := URLHost(fileURL)
		if err != nil {
			return err
		}
		if !HostAllowed(dst.AllowedHosts, host) {
			failures = append(failures, fmt.Errorf("host %s is not in the allowed list", host))
			continue
		}
		err = downloadToTarget(ctx, dst, client, fileURL, file, modelDir, target, emit, report)
		if err == nil {
			return nil
		}
		failures = append(failures, err)
	}
	if len(failures) == 0 {
		return fmt.Errorf("model %s lists no download URL for %s", model.ID, file.Name)
	}
	return fmt.Errorf("every source failed for %s (last: %w)", file.Name, failures[len(failures)-1])
}

// downloadToTarget streams one source URL into the file's staging
// name and, once the bytes verify, renames it onto the target. The
// read loop caps the transfer at the pinned byte count (a source that
// serves more is a drifted source), digests while streaming, and
// reports the throttled cumulative progress.
//
// The per-hop host gate rides the client's redirect policy: the first
// hop's host was validated by the chain walk, and every redirect hop
// is validated the same way before the client follows it -- a CDN 302
// to an admitted host passes, a redirect inward or to an unnamed host
// stops the transfer with the host named. The gate only binds when
// the run carries a list (the unconfigured gate admits every hop).
func downloadToTarget(ctx context.Context, dst RunDownloadDeps, client *http.Client, fileURL string, file ManifestFile, modelDir, target string, emit *progressEmitter, report *DownloadReport) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("build the request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch: status %d", resp.StatusCode)
	}

	// The staging name sits in the model's own directory: the final
	// rename stays inside one filesystem, hence atomic. Creating it
	// truncates whatever a previous interrupted attempt left behind.
	staging := filepath.Join(modelDir, ".staging-"+file.Name)
	out, err := os.Create(staging)
	if err != nil {
		return fmt.Errorf("create the staging file: %w", err)
	}

	base := report.BytesDone // the bytes verified before this file
	counted := int64(0)
	digest := sha256.New()
	buf := make([]byte, readChunkSize)
	streamErr := func() error {
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				counted += int64(n)
				if counted > file.Bytes {
					return fmt.Errorf("served %d bytes, more than the pinned %d", counted, file.Bytes)
				}
				if _, writeErr := out.Write(buf[:n]); writeErr != nil {
					return fmt.Errorf("write the staging file: %w", writeErr)
				}
				_, _ = digest.Write(buf[:n])
				emit.report(base+int(counted), report.BytesTotal, false)
			}
			if readErr == io.EOF {
				return nil
			}
			if readErr != nil {
				return fmt.Errorf("read the response body: %w", readErr)
			}
		}
	}()
	if streamErr == nil {
		if err := out.Sync(); err != nil {
			streamErr = fmt.Errorf("flush the staging file: %w", err)
		}
	}
	if err := out.Close(); streamErr == nil && err != nil {
		streamErr = fmt.Errorf("close the staging file: %w", err)
	}
	if streamErr == nil && counted != file.Bytes {
		streamErr = fmt.Errorf("served %d bytes, want the pinned %d", counted, file.Bytes)
	}
	if streamErr == nil && hex.EncodeToString(digest.Sum(nil)) != file.SHA256 {
		streamErr = errors.New("sha256 mismatch: the served bytes drifted from the pinned digest")
	}
	if streamErr != nil {
		_ = os.Remove(staging)
		return streamErr
	}
	if err := os.Rename(staging, target); err != nil {
		_ = os.Remove(staging)
		return fmt.Errorf("rename into place: %w", err)
	}
	report.BytesDone += int(file.Bytes)
	emit.report(report.BytesDone, report.BytesTotal, false)
	return nil
}

// verifiedFile reports whether the file at path exists and matches its
// pinned digest: the skip test of the idempotent re-run. It streams
// (the real weights run to hundreds of megabytes), and an unpinned
// digest is never trusted.
func verifiedFile(path, sha256Hex string) bool {
	if sha256Hex == "" {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return false
	}
	return hex.EncodeToString(digest.Sum(nil)) == sha256Hex
}

// writeManifestCopy writes the model's manifest next to its weights:
// the registry reads this copy as the checksum authority of the
// registered model.
func writeManifestCopy(modelDir string, model *ManifestModel) error {
	blob, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("marshal the manifest copy of %s: %w", model.ID, err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, manifestFileName), blob, 0o644); err != nil {
		return fmt.Errorf("write the manifest copy of %s: %w", model.ID, err)
	}
	return nil
}

// modelFileURL derives one file's URL from the model's URL: the
// upstream artifacts of a model live in one directory, so the file's
// name replaces the URL's last segment (which the manifest pins as
// the primary file's name).
func modelFileURL(modelURL, fileName string) (string, error) {
	parsed, err := url.Parse(modelURL)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", modelURL, err)
	}
	parsed.Path = path.Join(path.Dir(parsed.Path), fileName)
	return parsed.String(), nil
}

// mirrorURL applies the HF domain mirror rewrite: a non-empty mirror
// replaces the huggingface.co hostname wholesale (host and, when
// present, port); every other host -- and an empty mirror -- stays
// exactly as pinned.
func mirrorURL(raw, mirror string) (string, error) {
	if mirror == "" {
		return raw, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse %s for the mirror rewrite: %w", raw, err)
	}
	if strings.EqualFold(parsed.Hostname(), "huggingface.co") {
		parsed.Host = mirror
	}
	return parsed.String(), nil
}

// progressEmitter throttles the progress callback: the first and the
// final reports are mandatory (force), every intermediate one waits
// out the interval, so dense per-chunk attempts collapse to a bounded
// number of callbacks.
type progressEmitter struct {
	progress func(done, total int)
	now      func() time.Time
	last     time.Time
}

// report emits one progress callback unless the throttle holds it
// back. force bypasses the interval (the mandatory first and final
// reports).
func (e *progressEmitter) report(done, total int, force bool) {
	if e.progress == nil {
		return
	}
	at := e.now()
	if !force && !e.last.IsZero() && at.Sub(e.last) < progressMinInterval {
		return
	}
	e.last = at
	e.progress(done, total)
}
