// The model source pipeline: the staging probe and the staged
// registration behind the switch dialog. The pipeline's shape is the
// four steps the admin surface walks -- stage, probe, register, swap
// -- and the first three live here:
//
//   - STAGE: a URL candidate downloads into the staging area (a
//     scratch directory that is never an active model), a local file
//     candidate arrives there through the upload handler. A staged
//     file is dead weight until the probe passes: an interrupted or
//     failed probe deletes it.
//   - PROBE: the staged bytes digest-check when the caller pinned a
//     digest, then load in the sandbox (the onnxruntime's own parse
//     and session creation; a file that is not a readable model is
//     refused) and check against the slot's contract -- the output
//     dimension the slot needs, read off the graph and named when it
//     disagrees. An undeclared host does NOT fail: the probe answers
//     its warn and the save stays the administrator's own act.
//   - REGISTER: a passing probe's bytes move into the models dir
//     under the content's own name (the model id carries the digest's
//     first eight chars -- the same bytes land in one directory no
//     matter which source they came from), the per-model manifest is
//     written beside them, and the models row registers with source
//     `custom`. The fourth step -- the swap -- is the registry's
//     Activate, owned by the admin surface.
//
// Probes are transient: their records live minutes after they settle,
// then expire with their staging. Nothing is registered by the probe
// itself, so a restart simply clears every probe and its staging --
// no durable state carries the pipeline's first three steps.
package ml

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	ort "github.com/yalue/onnxruntime_go"
)

// Probe states, the contract's enum values verbatim.
const (
	ProbeQueued  = "queued"
	ProbeRunning = "running"
	ProbeOK      = "ok"
	ProbeError   = "error"
	ProbeWarn    = "warn"
)

// ProbeTTL is how long a settled probe's record (and its staged
// bytes) survive: the dialog polls a probe within seconds, and the
// switch follows within the dialog's lifetime; the TTL keeps an
// abandoned probe's disk cost bounded.
const ProbeTTL = 15 * time.Minute

// probeStagingRoot is the staging directory the probes stage into,
// under the models dir.
const probeStagingRoot = ".staging"

// ProbeStagingRoot is the exported name of the staging root, the tests'
// handle on the directory the failed probe must leave empty.
const ProbeStagingRoot = probeStagingRoot

// probeMaxFileBytes caps a staged candidate: 2 GiB, the upload
// contract's own bound.
const probeMaxFileBytes = int64(2 * 1024 * 1024 * 1024)

// probeIDSlug strips everything but the identifier characters out of
// a candidate's display name: the registered id is the slug plus the
// digest's short form.
var probeIDSlug = regexp.MustCompile(`[^a-z0-9]+`)

// ProbeSnapshot is one probe's state at a read: the facts the listing
// endpoint maps onto the contract's probe answer, copied under the
// manager's lock. StagingPath is the manager-internal pointer the
// register step consumes; it rides the same struct.
type ProbeSnapshot struct {
	ID          string
	Slot        Slot
	State       string
	Done        int
	Total       int
	Dims        int
	Width       int
	Height      int
	SHA256      string
	Declared    bool
	Reason      string
	StagingPath string
	CreatedAt   time.Time
}

// probeHosts answers the download sources list a URL candidate's host
// is checked against; nil means every host is declared.
type probeHosts func() []string

// probeDims answers the dimension a slot's contract requires for a
// CUSTOM import (true when a requirement exists): the embed slot's
// requirement is the vector column's current dimension, read per
// probe -- the column is the database's truth, never an assumption.
// An official-list entry does not ride this check: its cross-dimension
// switch is the confirmed wipe.
type probeDims func(slot Slot) (int, bool)

// ProbeManager owns the transient probes: the records, the staging
// and the register step. One per server.
type ProbeManager struct {
	mu          sync.Mutex
	records     map[string]*ProbeSnapshot
	modelsDir   string
	client      *http.Client
	hosts       probeHosts
	requireDims probeDims
	now         func() time.Time
}

// NewProbeManager builds the manager over the models dir. hosts
// answers the allowed hosts for the declared-host check (nil admits
// every host, the unconfigured gate); requireDims answers the
// dimension a custom import must match (nil lifts the requirement);
// client fetches the URL candidates (nil falls back to a plain
// client); now drives the TTL (nil falls back to time.Now).
func NewProbeManager(modelsDir string, hosts probeHosts, requireDims probeDims, client *http.Client, now func() time.Time) *ProbeManager {
	if client == nil {
		client = &http.Client{}
	}
	if now == nil {
		now = time.Now
	}
	return &ProbeManager{
		records:     map[string]*ProbeSnapshot{},
		modelsDir:   modelsDir,
		client:      client,
		hosts:       hosts,
		requireDims: requireDims,
		now:         now,
	}
}

// Start stages and probes a URL candidate in the background and
// answers the probe's identifier immediately. The record stays
// running until the download and the checks settle.
func (pm *ProbeManager) Start(slot Slot, rawURL, pinnedSHA string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return "", errors.New("ml: the model URL must be an absolute http(s) URL")
	}
	id, err := newProbeID()
	if err != nil {
		return "", err
	}
	declared := true
	if pm.hosts != nil {
		declared = HostAllowed(pm.hosts(), parsed.Hostname())
	}
	record := &ProbeSnapshot{
		ID:       id,
		Slot:     slot,
		State:    ProbeRunning,
		Declared: declared,
		Total:    1, // the honest unknown until the Content-Length names it
	}
	pm.put(record)
	go pm.runURL(record, rawURL, pinnedSHA)
	return id, nil
}

// StartFile stages an uploaded candidate from the reader and probes
// it synchronously -- the upload itself is the long part, and the
// checks are seconds. The returned id answers the same GET.
func (pm *ProbeManager) StartFile(slot Slot, src io.Reader, pinnedSHA string) (string, error) {
	id, err := newProbeID()
	if err != nil {
		return "", err
	}
	// A local file has no host to declare or refuse: the check does
	// not apply, and the honest answer is declared.
	record := &ProbeSnapshot{ID: id, Slot: slot, Declared: true}
	pm.put(record)
	if err := pm.stageFile(record, src); err != nil {
		pm.fail(record, err.Error())
		return id, nil
	}
	pm.inspect(record)
	return id, nil
}

// Get answers one probe's state copy, expiring the stale ones first.
// A missing or expired id answers false.
func (pm *ProbeManager) Get(id string) (ProbeSnapshot, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.expireLocked()
	record, ok := pm.records[id]
	if !ok {
		return ProbeSnapshot{}, false
	}
	return *record, true
}

// RegisterStaged moves a passing probe's staged bytes into the models
// dir under their content's own name and registers the row with
// source custom. The caller does the swap (the registry's Activate)
// and owns the semantics around it. A probe that did not finish ok is
// refused -- the pipeline has no path from a failed probe to the
// models dir -- and an unknown or expired probe answers
// ErrProbeNotFound.
func (pm *ProbeManager) RegisterStaged(ctx context.Context, id string, registry *Registry) (*ProbeSnapshot, error) {
	pm.mu.Lock()
	pm.expireLocked()
	record, ok := pm.records[id]
	pm.mu.Unlock()
	if !ok {
		return nil, ErrProbeNotFound
	}
	if record.State != ProbeOK && record.State != ProbeWarn {
		return nil, ErrProbeNotPassed
	}
	if _, err := registerStagedModel(ctx, registry, record); err != nil {
		return nil, err
	}
	pm.remove(id)
	return &ProbeSnapshot{
		ID:        record.ID,
		Slot:      record.Slot,
		State:     record.State,
		Dims:      record.Dims,
		Width:     record.Width,
		Height:    record.Height,
		SHA256:    record.SHA256,
		Declared:  record.Declared,
		CreatedAt: record.CreatedAt,
	}, nil
}

// ErrProbeNotFound is the answer for a probeId the manager does not
// know: it expired, or it was never issued.
var ErrProbeNotFound = errors.New("ml: no probe with this identifier")

// ErrProbeNotPassed is the register step's refusal: the probe named
// by the switch did not finish ok.
var ErrProbeNotPassed = errors.New("ml: the probe did not finish ok")

// runURL stages a URL candidate: the download streams into the
// record's staging file with progress, then the inspect runs. The
// digest, when the caller pinned one, gates before the sandbox load.
func (pm *ProbeManager) runURL(record *ProbeSnapshot, rawURL, pinnedSHA string) {
	stagingDir, err := pm.stagingDir(record.ID)
	if err != nil {
		pm.fail(record, err.Error())
		return
	}
	name := sanitizeFileName(filepath.Base(rawURL))
	if name == "" {
		name = "model"
	}
	target := filepath.Join(stagingDir, name+".onnx")

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		pm.fail(record, "build the request: "+err.Error())
		return
	}
	resp, err := pm.client.Do(req)
	if err != nil {
		pm.fail(record, "fetch: "+err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		pm.fail(record, fmt.Sprintf("the source answered status %d", resp.StatusCode))
		return
	}
	if size := resp.ContentLength; size > 0 {
		pm.setTotals(record, int(size))
	}
	file, err := os.Create(target)
	if err != nil {
		pm.fail(record, "create the staging file: "+err.Error())
		return
	}
	digest := sha256.New()
	var written int
	buf := make([]byte, 64*1024)
	overflow := func() bool {
		// The size cap keeps a runaway source from filling the disk;
		// the check rides the copy loop's own counter.
		return written > int(probeMaxFileBytes)
	}
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			written += n
			if overflow() {
				_ = file.Close()
				_ = os.Remove(target)
				pm.fail(record, "the candidate is larger than the 2 GiB bound")
				return
			}
			if _, werr := file.Write(buf[:n]); werr != nil {
				_ = file.Close()
				_ = os.Remove(target)
				pm.fail(record, "write the staging file: "+werr.Error())
				return
			}
			_, _ = digest.Write(buf[:n])
			pm.setProgress(record, written)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = file.Close()
			_ = os.Remove(target)
			pm.fail(record, "read the response body: "+readErr.Error())
			return
		}
	}
	if err := file.Close(); err != nil {
		pm.fail(record, "close the staging file: "+err.Error())
		return
	}
	got := hex.EncodeToString(digest.Sum(nil))
	pm.setDigest(record, got, target)
	if pinnedSHA != "" && !strings.EqualFold(pinnedSHA, got) {
		_ = os.Remove(target)
		pm.fail(record, fmt.Sprintf("the bytes' sha256 is %s, want the pinned %s", got, pinnedSHA))
		return
	}
	pm.inspect(record)
}

// stageFile streams an uploaded candidate into the staging area.
func (pm *ProbeManager) stageFile(record *ProbeSnapshot, src io.Reader) error {
	stagingDir, err := pm.stagingDir(record.ID)
	if err != nil {
		return err
	}
	target := filepath.Join(stagingDir, "model.onnx")
	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create the staging file: %w", err)
	}
	digest := sha256.New()
	limited := io.LimitReader(src, probeMaxFileBytes+1)
	written, err := io.Copy(io.MultiWriter(file, digest), limited)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("read the upload: %w", err)
	}
	if written > probeMaxFileBytes {
		_ = os.Remove(target)
		return errors.New("the candidate is larger than the 2 GiB bound")
	}
	got := hex.EncodeToString(digest.Sum(nil))
	pm.setDigest(record, got, target)
	pm.setProgress(record, int(written))
	return nil
}

// inspect runs the sandbox load and the slot contract's checks over
// the staged bytes, moving the record to ok, warn or error. The
// computed digest is the record's sha256 either way (the pin, when
// the caller gave one, already matched in the staging step).
func (pm *ProbeManager) inspect(record *ProbeSnapshot) {
	pm.mu.Lock()
	path := record.StagingPath
	sha := record.SHA256
	pm.mu.Unlock()
	if sha == "" {
		got, err := fileDigest(path)
		if err != nil {
			pm.fail(record, err.Error())
			return
		}
		sha = got
		pm.setDigest(record, sha, path)
	}
	if err := EnsureRuntime(); err != nil {
		pm.fail(record, "the onnxruntime is not available: "+err.Error())
		return
	}
	inputs, outputs, err := ort.GetInputOutputInfo(path)
	if err != nil {
		pm.fail(record, "the file is not a readable ONNX model")
		return
	}
	// The graph must load, not just parse: an opset the runtime
	// cannot execute surfaces here, before anything registers.
	sess, err := ort.NewDynamicAdvancedSession(path, metadataNames(inputs), metadataNames(outputs), nil)
	if err != nil {
		pm.fail(record, "the model does not load in the onnxruntime")
		return
	}
	_ = sess.Destroy()

	dims, width, height, reason := slotContractFacts(record.Slot, inputs, outputs)
	if reason != "" {
		pm.fail(record, reason)
		return
	}
	// The custom import's dimension requirement: the slot contract
	// is hard for a source nobody validated -- a model that would
	// change the column's dimension is the confirmed builtin switch's
	// business, never a custom import's.
	if pm.requireDims != nil && dims > 0 {
		if need, required := pm.requireDims(record.Slot); required && dims != need {
			pm.fail(record, fmt.Sprintf("the model emits %d dimensions, this slot needs %d", dims, need))
			return
		}
	}
	pm.mu.Lock()
	record.Dims = dims
	record.Width = width
	record.Height = height
	record.State = ProbeOK
	if !record.Declared {
		record.State = ProbeWarn
	}
	pm.mu.Unlock()
}

// slotContractFacts reads the slot contract's facts off the graph
// metadata and names the check that failed, if any. The embed slot is
// the strict one (its vector column is dimension-typed): exactly one
// output, a fixed embedding dimension. The recognition slot demands a
// single embedding-shaped output and reports its dimension (the face
// embedding column is dimension-free, a new dimension is a new
// generation); the detector and the OCR pair are procedural -- the
// load itself is their check.
func slotContractFacts(slot Slot, inputs, outputs []ort.InputOutputInfo) (dims, width, height int, reason string) {
	outputDim := func(info ort.InputOutputInfo) int {
		dims := info.Dimensions
		if len(dims) == 0 {
			return 0
		}
		last := dims[len(dims)-1]
		if last <= 0 {
			return 0
		}
		return int(last)
	}
	inputGeometry := func() (int, int) {
		if len(inputs) == 0 || len(inputs[0].Dimensions) < 4 {
			return 0, 0
		}
		dims := inputs[0].Dimensions
		return int(dims[3]), int(dims[2])
	}
	switch slot {
	case SlotClipEmbed:
		if len(outputs) != 1 {
			return 0, 0, 0, fmt.Sprintf("the graph carries %d outputs, this slot needs exactly one embedding", len(outputs))
		}
		if dims = outputDim(outputs[0]); dims == 0 {
			return 0, 0, 0, "the graph's output dimension is dynamic; this slot needs a fixed embedding dimension"
		}
		width, height = inputGeometry()
		return dims, width, height, ""
	case SlotFaceRecognize:
		if len(outputs) != 1 {
			return 0, 0, 0, fmt.Sprintf("the graph carries %d outputs, this slot needs exactly one embedding", len(outputs))
		}
		dims = outputDim(outputs[0])
		width, height = inputGeometry()
		return dims, width, height, ""
	default:
		width, height = inputGeometry()
		return 0, width, height, ""
	}
}

// registerStagedModel moves a passing probe's staged bytes into the
// models dir under their content's own name and registers the row.
// The id is the display slug plus the digest's first eight chars: the
// same bytes land in one directory no matter which source they came
// from, and a re-import of the same bytes is an upsert, not a
// duplicate.
func registerStagedModel(ctx context.Context, registry *Registry, record *ProbeSnapshot) (*Model, error) {
	if record.StagingPath == "" {
		return nil, fmt.Errorf("%w: %s has no staged file", ErrInvalidModel, record.ID)
	}
	sha := record.SHA256
	if sha == "" {
		digest, err := fileDigest(record.StagingPath)
		if err != nil {
			return nil, err
		}
		sha = digest
	}
	name := displayNameOf(record.StagingPath)
	if name == "" {
		name = "imported"
	}
	id := name + "@" + sha[:8]
	modelDir := filepath.Join(registry.modelsDir, id)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return nil, fmt.Errorf("ml: create the model dir of %s: %w", id, err)
	}
	primary := "model.onnx"
	target := filepath.Join(modelDir, primary)
	if err := os.Rename(record.StagingPath, target); err != nil {
		return nil, fmt.Errorf("ml: move the staged bytes of %s: %w", id, err)
	}
	model := &ManifestModel{
		ID:      id,
		Slot:    manifestSlotName(record.Slot),
		Version: sha[:8],
		License: "unknown",
		Name:    name,
		Notice:  "Imported by the administrator" + declaredSuffix(record),
		InputSize: ManifestSize{
			Width:  record.Width,
			Height: record.Height,
		},
		Dim:    record.Dims,
		Params: map[string]any{},
		Files: []ManifestFile{{
			Name:   primary,
			SHA256: sha,
			Bytes:  fileSize(target),
		}},
	}
	if err := writeManifestCopy(modelDir, model); err != nil {
		return nil, err
	}
	if err := registry.InsertCustom(ctx, model); err != nil {
		return nil, err
	}
	return &Model{
		ID:      id,
		Version: model.Version,
		Slot:    record.Slot,
		Source:  "custom",
		Path:    target,
		SHA256:  sha,
		License: model.License,
		Width:   record.Width,
		Height:  record.Height,
		Dim:     record.Dims,
	}, nil
}

// declaredSuffix names the import's host discipline in the
// registered model's notice: an undeclared host is the
// administrator's own act, and the notice is where the record lives.
func declaredSuffix(record *ProbeSnapshot) string {
	if record.Declared {
		return "."
	}
	return " from a host outside the download sources list (logged)."
}

// manifestSlotName answers the manifest slot name of a Slot value.
func manifestSlotName(slot Slot) string {
	for name, value := range manifestSlotValues {
		if value == slot {
			return name
		}
	}
	return string(slot)
}

// displayNameOf derives the display name from the staged file's name:
// the extension is dropped, the rest is the name the dialog showed.
func displayNameOf(stagingPath string) string {
	base := filepath.Base(stagingPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return sanitizeFileName(base)
}

// sanitizeFileName slugs a name into the identifier shape the models
// dir and the models table accept: lowercase, alphanumerics and
// dashes, never empty.
func sanitizeFileName(raw string) string {
	slug := probeIDSlug.ReplaceAllString(strings.ToLower(raw), "-")
	return strings.Trim(slug, "-")
}

// fileDigest computes a file's sha256.
func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("ml: read the staged bytes: %w", err)
	}
	defer func() { _ = file.Close() }()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("ml: digest the staged bytes: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// fileSize answers a file's size, 0 when it cannot be read.
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// metadataNames flattens a metadata list into the name list the
// dynamic session constructor wants.
func metadataNames(infos []ort.InputOutputInfo) []string {
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name
	}
	return names
}

// stagingDir builds (and answers) one probe's staging directory.
func (pm *ProbeManager) stagingDir(id string) (string, error) {
	dir := filepath.Join(pm.modelsDir, probeStagingRoot, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("ml: create the staging dir: %w", err)
	}
	return dir, nil
}

// The record mutators hold the manager's lock: the polling GET copies
// the record under it, so every field store is serialized. The
// progress fields are advisory (the dialog's spinner), the settled
// fields are the contract's facts.

func (pm *ProbeManager) setProgress(record *ProbeSnapshot, done int) {
	pm.mu.Lock()
	record.Done = done
	pm.mu.Unlock()
}

func (pm *ProbeManager) setTotals(record *ProbeSnapshot, total int) {
	pm.mu.Lock()
	record.Total = total
	pm.mu.Unlock()
}

func (pm *ProbeManager) setDigest(record *ProbeSnapshot, sha, stagingPath string) {
	pm.mu.Lock()
	record.SHA256 = sha
	record.StagingPath = stagingPath
	pm.mu.Unlock()
}

// fail moves a record to error with the reason, and removes its
// staging (a failed probe leaves nothing behind).
func (pm *ProbeManager) fail(record *ProbeSnapshot, reason string) {
	pm.mu.Lock()
	record.State = ProbeError
	record.Reason = reason
	if record.StagingPath != "" {
		_ = os.Remove(record.StagingPath)
		record.StagingPath = ""
	}
	pm.mu.Unlock()
}

// Discard drops one probe and its staged bytes: the same-bytes reuse
// consumed the probe without a registration.
func (pm *ProbeManager) Discard(id string) {
	pm.mu.Lock()
	record, ok := pm.records[id]
	if ok {
		if record.StagingPath != "" {
			_ = os.Remove(record.StagingPath)
		}
		delete(pm.records, id)
	}
	pm.mu.Unlock()
}

func (pm *ProbeManager) put(record *ProbeSnapshot) {
	record.CreatedAt = pm.now()
	pm.mu.Lock()
	pm.records[record.ID] = record
	pm.mu.Unlock()
}

func (pm *ProbeManager) remove(id string) {
	pm.mu.Lock()
	delete(pm.records, id)
	pm.mu.Unlock()
}

func (pm *ProbeManager) expireLocked() {
	now := pm.now()
	for id, record := range pm.records {
		if now.Sub(record.CreatedAt) > ProbeTTL {
			if record.StagingPath != "" {
				_ = os.Remove(record.StagingPath)
			}
			delete(pm.records, id)
		}
	}
}

// newProbeID mints a random identifier.
func newProbeID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("ml: mint a probe id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
