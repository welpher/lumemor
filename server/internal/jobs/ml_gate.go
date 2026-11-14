// The ML jobs' model gate: the shared readiness seam every inference
// worker consults before it decodes anything.
//
// Three outcomes, one per state of the slot:
//
//   - The slot has an active model whose files are on disk: the worker
//     proceeds with it.
//   - The slot is unconfigured: the degraded no-AI mode. The worker
//     completes its job silently (the dispatch seam would not have
//     enqueued it, so this is a stale or mis-routed job).
//   - The slot is CONFIGURED (an active model row exists) but its
//     weights are gone -- a model store wiped below a configured
//     instance. This is the one state the automatic fill covers: with
//     the policy switch on, the gate joins or starts the model
//     group's download and the worker answers a retryable error, so
//     the job waits for the run and resumes against the landed files
//     when River retries it. The first-run download and the settings
//     face's button remain the administrator's active path: an
//     unconfigured slot is never filled by a job.
//
// The gate is deliberately not a place for new failure semantics: a
// failed download fails the run exactly as it always did, the job's
// retries are River's, and a slot whose files are present but whose
// manifest is corrupt or whose bytes drifted keeps failing the job as
// it always did -- the fill answers missing files only.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/welpher/lumemor/server/internal/ml"
)

// ErrModelFilesMissing marks the configured-but-missing outcome: the
// download was joined or started, and this attempt cannot proceed.
// The returned error makes River retry the job; by the time the retry
// runs, the download has usually landed the files.
var ErrModelFilesMissing = errors.New("jobs: the model weights are missing and the download is in progress")

// errAutoDownloadOff is the seam's answer when the policy switch is
// off: the gate falls back to the slot's own error, so the job keeps
// the semantics an administrator who chose manual downloads expects.
var errAutoDownloadOff = errors.New("jobs: the automatic model download is switched off")

// errModelDownloadFailed is the seam's answer when the most recent run
// failed: the failure stays the administrator's to retry, and the job
// answers the ordinary missing-weights failure instead of waiting for
// a run that is not coming.
var errModelDownloadFailed = errors.New("jobs: the most recent model download failed")

// missingWeightsError is the configured-but-missing failure when no
// fill happens (no queue wired, the switch off, or a failing enqueue).
// It carries the registry's own sentinel, so the job's failure is the
// same class a model with unusable files always produced.
func missingWeightsError(slot ml.Slot) error {
	return fmt.Errorf("%w: slot %s has no weights on disk", ml.ErrInvalidModel, slot)
}

// resolveMlModel is the ML workers' readiness gate. fill is the
// automatic download seam (nil when no queue is wired): it joins or
// starts the group's download and answers errAutoDownloadOff when the
// policy switch is off.
func resolveMlModel(ctx context.Context, registry *ml.Registry, fill func(context.Context) error, logger *slog.Logger, slot ml.Slot) (*ml.Model, error) {
	model, err := registry.Active(slot)
	// The row read alone cannot see a deleted weights file (the
	// manifest beside it is the digest authority, and the session
	// load is where a missing file surfaced before this gate): the
	// presence probe is what makes the configured-but-missing state
	// visible here.
	missing := registry.ActiveFilesMissing(slot)

	if err == nil && !missing {
		return model, nil
	}
	if errors.Is(err, ml.ErrNoActiveModel) {
		// Unfilled slot: the caller degrades silently.
		return nil, err
	}
	if !missing {
		// A readable row whose manifest is corrupt or whose path is
		// unlisted: a present-but-unusable model, which no download
		// repairs.
		return nil, err
	}
	if err != nil && !errors.Is(err, ml.ErrInvalidModel) {
		// A database failure: not the fill's case.
		return nil, err
	}

	// Configured but missing from here on.
	if fill == nil {
		return nil, missingWeightsError(slot)
	}
	if fillErr := fill(ctx); fillErr != nil {
		if !errors.Is(fillErr, errAutoDownloadOff) && !errors.Is(fillErr, errModelDownloadFailed) {
			logger.Warn("start the model download for a missing model", "error", fillErr, "slot", string(slot))
		}
		return nil, missingWeightsError(slot)
	}
	logger.Info("model weights missing; the download is running and this job will retry", "slot", string(slot))
	return nil, fmt.Errorf("%w: slot %s", ErrModelFilesMissing, slot)
}

// ensureModelDownload is the queue's automatic fill seam: the policy
// switch decides whether anything happens, and the enqueue joins a
// run already in flight or the most recent one -- the download
// endpoint's idempotence, so a burst of jobs behind one missing slot
// starts one download, not one per job.
func (q *Queue) ensureModelDownload(ctx context.Context) error {
	settings, err := q.modelSettings(ctx)
	if err != nil {
		// The accessor answered the documented default (on); the
		// line points at the row to fix.
		q.logger.Warn("read the model download policy; answering the default", "error", err)
	}
	if !settings.AutoDownload {
		return errAutoDownloadOff
	}
	return q.joinOrStartModelDownload(ctx)
}
