// The model params' dispatch read point (E41): the ML jobs read the
// admin's two steering values ONCE at their own start -- an in-flight
// job keeps what it read, the next dispatched job reads the fresh
// ones -- the admin value overrides the active model's manifest entry
// for the knob, and a corrupt settings row answers the pipeline
// default through the accessor. The defaults themselves are pinned
// against the ML wrapper's own constants, so the db package's copies
// and the pipeline's truth cannot drift apart.
package jobs

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
	mlface "github.com/welpher/lumemor/server/internal/ml/face"
)

// TestTheParamsDefaultsPinThePipeline pins the db package's shipped
// params against the ML wrapper's own constants: the two packages
// carry the same numbers on purpose (the db leaf imports nothing
// above it), and this test is the pin that keeps them honest.
func TestTheParamsDefaultsPinThePipeline(t *testing.T) {
	if db.MlModelParamsDefault.FaceDetectMinScore != mlface.DefaultScoreThreshold {
		t.Errorf("db default faceDetectMinScore = %v, want the detector's own %v",
			db.MlModelParamsDefault.FaceDetectMinScore, mlface.DefaultScoreThreshold)
	}
	if db.MlModelParamsDefault.FaceRecognizeThreshold != mlface.DefaultClusterEps {
		t.Errorf("db default faceRecognizeThreshold = %v, want the clusterer's own %v",
			db.MlModelParamsDefault.FaceRecognizeThreshold, mlface.DefaultClusterEps)
	}
}

// TestFaceDetectDecodeParamsReadTheAdminMinScore pins the detection
// job's dispatch read point: the admin's minimum score overrides the
// model's manifest entry for the knob, the model's other params
// survive the merge, a corrupt settings row answers the pipeline
// default, and a read after a save picks the fresh value up (each job
// reads at its own start -- nothing is pinned in the worker).
func TestFaceDetectDecodeParamsReadTheAdminMinScore(t *testing.T) {
	e := newEnv(t)
	w := &FaceDetectWorker{
		pool:   e.pool,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx := context.Background()
	model := &ml.Model{Params: map[string]any{"scoreThreshold": 0.4, "nmsThreshold": 0.3}}

	// The fresh instance: the accessor answers the pipeline default,
	// which still overrides the model's own manifest entry (the admin
	// surface owns the knob).
	merged := w.decodeParams(ctx, model)
	if merged["scoreThreshold"] != db.MlModelParamsDefault.FaceDetectMinScore {
		t.Errorf("merged scoreThreshold = %v, want the pipeline default %v (the admin surface owns the knob)",
			merged["scoreThreshold"], db.MlModelParamsDefault.FaceDetectMinScore)
	}
	if merged["nmsThreshold"] != 0.3 {
		t.Errorf("merged nmsThreshold = %v, want the model's own 0.3 (only the score is the admin's)", merged["nmsThreshold"])
	}

	// The save lands: the NEXT dispatched job reads the fresh value.
	if err := db.SaveModelSettings(ctx, dbgen.New(e.pool), db.ModelSettings{
		AutoDownload:  true,
		AllowedHosts:  []string{"huggingface.co"},
		MirrorBaseUrl: "",
		Params:        db.MlModelParams{FaceDetectMinScore: 0.9, FaceRecognizeThreshold: 0.55},
	}); err != nil {
		t.Fatalf("save the params: %v", err)
	}
	merged = w.decodeParams(ctx, model)
	if merged["scoreThreshold"] != 0.9 {
		t.Errorf("merged scoreThreshold after the save = %v, want 0.9 (the next dispatch reads the fresh value)", merged["scoreThreshold"])
	}

	// A corrupt row degrades to the pipeline default, the read never
	// fails the job: the drifted value (out of the 0-1 domain) is
	// normalized back to the knob's own default.
	if err := dbgen.New(e.pool).SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   db.ModelSettingsKey,
		Value: []byte(`{"autoDownload":true,"allowedHosts":[],"mirrorBaseUrl":"","params":{"faceDetectMinScore":7,"faceRecognizeThreshold":0.55}}`),
	}); err != nil {
		t.Fatalf("corrupt the params row: %v", err)
	}
	merged = w.decodeParams(ctx, model)
	if merged["scoreThreshold"] != db.MlModelParamsDefault.FaceDetectMinScore {
		t.Errorf("corrupt-row scoreThreshold = %v, want the pipeline default %v",
			merged["scoreThreshold"], db.MlModelParamsDefault.FaceDetectMinScore)
	}
}

// TestFaceRecognizeClusteringEpsReadsTheAdminThreshold pins the
// recognition job's dispatch read point: the admin's clustering
// threshold rides out of the same settings document, a corrupt row
// answers the pipeline default.
func TestFaceRecognizeClusteringEpsReadsTheAdminThreshold(t *testing.T) {
	e := newEnv(t)
	w := &FaceRecognizeWorker{
		pool:   e.pool,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx := context.Background()

	if got := w.clusteringEps(ctx); got != db.MlModelParamsDefault.FaceRecognizeThreshold {
		t.Errorf("fresh-instance eps = %v, want the pipeline default %v", got, db.MlModelParamsDefault.FaceRecognizeThreshold)
	}
	if err := db.SaveModelSettings(ctx, dbgen.New(e.pool), db.ModelSettings{
		AutoDownload:  true,
		AllowedHosts:  []string{"huggingface.co"},
		MirrorBaseUrl: "",
		Params:        db.MlModelParams{FaceDetectMinScore: 0.6, FaceRecognizeThreshold: 0.35},
	}); err != nil {
		t.Fatalf("save the params: %v", err)
	}
	if got := w.clusteringEps(ctx); got != 0.35 {
		t.Errorf("eps after the save = %v, want 0.35 (the next dispatch reads the fresh value)", got)
	}
}
