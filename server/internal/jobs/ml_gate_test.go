// The ML model gate's tests: the three slot states the shared
// readiness seam distinguishes, and the automatic fill's seam -- the
// policy switch and the run idempotence (one download behind a burst
// of jobs). The enqueue-side tests run against an EMPTY manifest
// group, so the download worker the environment starts has nothing to
// fetch: the tests never touch a network.
package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/ml"
)

// gateLogger keeps the gate's lines out of the test output.
func gateLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// countModelDownloadRuns counts the model download rows the environment
// created.
func (e *env) countModelDownloadRuns(t *testing.T) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE kind = 'model_download'").Scan(&count); err != nil {
		t.Fatalf("count the model download runs: %v", err)
	}
	return count
}

// TestResolveMlModelUnconfiguredSlotDegrades pins the no-AI mode: an
// unfilled slot answers the registry's sentinel and the fill seam is
// never consulted -- a job never downloads anything into a slot the
// administrator has not configured.
func TestResolveMlModelUnconfiguredSlotDegrades(t *testing.T) {
	e := newEnv(t)
	calls := 0
	fill := func(context.Context) error { calls++; return nil }

	model, err := resolveMlModel(context.Background(), e.registry, fill, gateLogger(), ml.SlotFaceDetect)
	if !errors.Is(err, ml.ErrNoActiveModel) {
		t.Fatalf("error = %v, want ErrNoActiveModel", err)
	}
	if model != nil {
		t.Errorf("model = %+v, want nil", model)
	}
	if calls != 0 {
		t.Errorf("the fill seam ran %d times, want 0", calls)
	}
}

// TestResolveMlModelConfiguredButMissingJoinsTheDownload pins the
// automatic fill: an active model whose weights file was deleted is
// the configured-but-missing state -- the seam runs once and the job
// answers the retryable sentinel.
func TestResolveMlModelConfiguredButMissingJoinsTheDownload(t *testing.T) {
	e := newEnv(t)
	id := e.registerFixtureModel(t, clipFixtureSpec)
	if _, err := e.registry.Activate(ml.SlotClipEmbed, id); err != nil {
		t.Fatalf("activate the clip fixture: %v", err)
	}
	if err := os.Remove(filepath.Join(e.modelsDir, id, clipFixtureSpec.file)); err != nil {
		t.Fatalf("delete the weights: %v", err)
	}
	calls := 0
	fill := func(context.Context) error { calls++; return nil }

	_, err := resolveMlModel(context.Background(), e.registry, fill, gateLogger(), ml.SlotClipEmbed)
	if !errors.Is(err, ErrModelFilesMissing) {
		t.Fatalf("error = %v, want ErrModelFilesMissing", err)
	}
	if calls != 1 {
		t.Errorf("the fill seam ran %d times, want 1", calls)
	}
}

// TestResolveMlModelConfiguredButMissingKeepsTheFailureWhenOff pins the
// manual-only policy: with the fill refusing (the switch off), the job
// fails with the registry's own class instead of retrying forever.
func TestResolveMlModelConfiguredButMissingKeepsTheFailureWhenOff(t *testing.T) {
	e := newEnv(t)
	id := e.registerFixtureModel(t, clipFixtureSpec)
	if _, err := e.registry.Activate(ml.SlotClipEmbed, id); err != nil {
		t.Fatalf("activate the clip fixture: %v", err)
	}
	if err := os.Remove(filepath.Join(e.modelsDir, id, clipFixtureSpec.file)); err != nil {
		t.Fatalf("delete the weights: %v", err)
	}
	fill := func(context.Context) error { return errAutoDownloadOff }

	_, err := resolveMlModel(context.Background(), e.registry, fill, gateLogger(), ml.SlotClipEmbed)
	if !errors.Is(err, ml.ErrInvalidModel) {
		t.Fatalf("error = %v, want ErrInvalidModel", err)
	}
	if errors.Is(err, ErrModelFilesMissing) {
		t.Fatal("the switch-off path answered the retryable sentinel, want the failure class")
	}
}

// TestResolveMlModelCorruptManifestKeepsTheFailure pins the scope: a
// present-but-unusable model (its manifest no longer lists the
// registered file) is not a missing file, and no download is started
// for it.
func TestResolveMlModelCorruptManifestKeepsTheFailure(t *testing.T) {
	e := newEnv(t)
	id := e.registerFixtureModel(t, clipFixtureSpec)
	if _, err := e.registry.Activate(ml.SlotClipEmbed, id); err != nil {
		t.Fatalf("activate the clip fixture: %v", err)
	}
	manifest := `{"id":"` + id + `","slot":"clip-embed","version":"fixture","files":[]}`
	if err := os.WriteFile(filepath.Join(e.modelsDir, id, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("rewrite the manifest: %v", err)
	}
	calls := 0
	fill := func(context.Context) error { calls++; return nil }

	_, err := resolveMlModel(context.Background(), e.registry, fill, gateLogger(), ml.SlotClipEmbed)
	if !errors.Is(err, ml.ErrInvalidModel) {
		t.Fatalf("error = %v, want ErrInvalidModel", err)
	}
	if calls != 0 {
		t.Errorf("the fill seam ran %d times, want 0", calls)
	}
}

// TestEnsureModelDownloadJoinsOneRun pins the seam's idempotence: two
// calls behind one missing slot produce one run.
func TestEnsureModelDownloadJoinsOneRun(t *testing.T) {
	// The empty manifest group: the worker the environment starts has
	// nothing to fetch, so the run completes without touching a
	// network.
	e := newEnvManifest(t, &ml.Manifest{Revision: 1})

	first, err := e.queue.EnsureModelDownload(context.Background())
	if err != nil {
		t.Fatalf("ensure the download: %v", err)
	}
	second, err := e.queue.EnsureModelDownload(context.Background())
	if err != nil {
		t.Fatalf("ensure the download again: %v", err)
	}
	if first != second {
		t.Errorf("the second call created %s, want the joined run %s", second, first)
	}
	if count := e.countModelDownloadRuns(t); count != 1 {
		t.Errorf("model download runs = %d, want 1", count)
	}
}

// TestEnsureModelDownloadPolicyOff pins the settings switch: off means
// no run and the seam's own sentinel, so the worker keeps the manual
// policy's failure semantics.
func TestEnsureModelDownloadPolicyOff(t *testing.T) {
	e := newEnvManifest(t, &ml.Manifest{Revision: 1})
	if err := db.SaveModelSettings(context.Background(), e.queries, db.ModelSettings{AutoDownload: false}); err != nil {
		t.Fatalf("save the policy: %v", err)
	}

	err := e.queue.ensureModelDownload(context.Background())
	if !errors.Is(err, errAutoDownloadOff) {
		t.Fatalf("error = %v, want errAutoDownloadOff", err)
	}
	if count := e.countModelDownloadRuns(t); count != 0 {
		t.Errorf("model download runs = %d, want 0", count)
	}
}
