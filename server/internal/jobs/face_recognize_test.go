// The face recognition job tests: the worker wiring
// through the landmark-redetection architecture -- the active detect
// model's landmarks re-run over the stored original, matched against
// the asset's stored face rows by box overlap, every match aligned
// and embedded with the active recognize model's stamp -- and the
// clustering step's diff-apply: new clusters born with their
// highest-score representative, cluster IDENTITY preserved across
// converging re-clusters (the user's labels never move -- Review
// Focus #1's cluster side), vacated clusters dissolved with their
// labels cascading, and foreign-model clusters left intact. The
// chaining restore is pinned through the real queue: a detection
// completion with the recognize slot active and faces on the asset
// enqueues the recognition job; zero faces or an empty slot enqueues
// nothing. The face-deletion surfaces (the detection's replace step,
// the permanent-delete endpoint, the trash expiry) carry the
// representative sweep: a cluster never points at a deleted face.
//
// The inference-bearing tests skip on hosts without the onnxruntime
// shared library (and the decode-bearing ones without libvips): the
// containerized `make docker-test` gate runs everything for real.
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
	"github.com/welpher/lumemor/server/internal/trash"
)

// sfaceFixtureSpec is the recognize-slot fixture: the committed
// const128 graph (a dynamic-geometry image-in/vector-out model whose
// output is a constant non-unit vector), registered and activated
// the way a downloaded model lands.
var sfaceFixtureSpec = fixtureModelSpec{
	id:        "sface-fixture",
	slot:      ml.SlotFaceRecognize,
	file:      "const128.onnx",
	sha256:    "e5d4940db6bddc912dee319434394700cf82fd8950cc32e609435ae7f9597263",
	inputSize: ml.ManifestSize{Width: 112, Height: 112},
	dim:       128,
	params:    map[string]any{"eps": 0.55, "minPts": 2},
}

// activateRecognition activates the detect and recognize fixtures
// both, the chain's two slots, and returns the recognize model's id.
func (e *env) activateRecognition(t *testing.T) string {
	t.Helper()
	e.activateFixtureModel(t, yunetFixtureSpec)
	return e.activateFixtureModel(t, sfaceFixtureSpec)
}

// faceRecognizeWorker builds the worker over the environment for a
// direct Work invocation (the River machinery has its own tests).
func (e *env) faceRecognizeWorker() *FaceRecognizeWorker {
	return &FaceRecognizeWorker{
		pool:     e.pool,
		dataDir:  e.dataDir,
		registry: e.registry,
		bus:      e.bus,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// runFaceRecognize drives one direct Work invocation for the asset.
func runFaceRecognize(t *testing.T, e *env, assetID uuid.UUID) error {
	t.Helper()
	return e.faceRecognizeWorker().Work(context.Background(), &river.Job[FaceRecognizeArgs]{
		Args: FaceRecognizeArgs{AssetID: assetID},
	})
}

// clusterRow is one face_clusters row as the assertions read it.
type clusterRow struct {
	id             uuid.UUID
	representative uuid.UUID
}

// clustersOf lists every cluster row, in id order.
func (e *env) clustersOf(t *testing.T) []clusterRow {
	t.Helper()
	rows, err := e.pool.Query(context.Background(),
		`SELECT id, representative_face_id FROM face_clusters ORDER BY id`)
	if err != nil {
		t.Fatalf("query face clusters: %v", err)
	}
	defer rows.Close()
	var out []clusterRow
	for rows.Next() {
		var id, rep pgtype.UUID
		if err := rows.Scan(&id, &rep); err != nil {
			t.Fatalf("scan cluster row: %v", err)
		}
		out = append(out, clusterRow{id: uuid.UUID(id.Bytes), representative: uuid.UUID(rep.Bytes)})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate clusters: %v", err)
	}
	return out
}

// faceLabelRow is one user_face_labels row as the assertions read it.
type faceLabelRow struct {
	clusterID uuid.UUID
	name      string
}

// faceLabelsOf lists every user label, in cluster order.
func (e *env) faceLabelsOf(t *testing.T) []faceLabelRow {
	t.Helper()
	rows, err := e.pool.Query(context.Background(),
		`SELECT cluster_id, name FROM user_face_labels ORDER BY cluster_id`)
	if err != nil {
		t.Fatalf("query face labels: %v", err)
	}
	defer rows.Close()
	var out []faceLabelRow
	for rows.Next() {
		var cluster pgtype.UUID
		var row faceLabelRow
		if err := rows.Scan(&cluster, &row.name); err != nil {
			t.Fatalf("scan label row: %v", err)
		}
		row.clusterID = uuid.UUID(cluster.Bytes)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate labels: %v", err)
	}
	return out
}

// eventPayloadsOf replays the bus's retained ring and decodes every
// payload of the named event, in publish order.
func (e *env) eventPayloadsOf(t *testing.T, name string) []map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	subscription := e.bus.Subscribe(ctx, 0)
	var out []map[string]string
	for {
		event, ok := subscription.Next()
		if !ok {
			return out
		}
		if event.Name != name {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode the %s event payload: %v", name, err)
		}
		out = append(out, payload)
	}
}

// seedFace stores one face row directly: a stored detection of the
// given model without the detection job having run.
func (e *env) seedFace(t *testing.T, assetID uuid.UUID, modelID string, bbox [4]int32, score float32) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new face id: %v", err)
	}
	if err := e.queries.InsertFace(context.Background(), dbgen.InsertFaceParams{
		ID:      pgUUID(id),
		AssetID: pgUUID(assetID),
		BboxX:   bbox[0], BboxY: bbox[1], BboxW: bbox[2], BboxH: bbox[3],
		Score:   score,
		ModelID: modelID,
	}); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	return id
}

// seedCluster stores one cluster row directly with the given
// representative face.
func (e *env) seedCluster(t *testing.T, representativeFaceID uuid.UUID) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new cluster id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO face_clusters (id, representative_face_id) VALUES ($1, $2)`,
		pgUUID(id), pgUUID(representativeFaceID)); err != nil {
		t.Fatalf("seed cluster: %v", err)
	}
	return id
}

// attachFaceToCluster points one face row at one cluster row: the
// state a previous clustering pass left behind.
func (e *env) attachFaceToCluster(t *testing.T, faceID, clusterID uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE faces SET cluster_id = $2 WHERE id = $1`, pgUUID(faceID), pgUUID(clusterID)); err != nil {
		t.Fatalf("attach face to cluster: %v", err)
	}
}

// seedFaceLabel stores the owner's name for one cluster: the user's
// label whose survival the identity preservation pins.
func (e *env) seedFaceLabel(t *testing.T, clusterID uuid.UUID, name string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO user_face_labels (user_id, cluster_id, name) VALUES ($1, $2, $3)`,
		pgUUID(e.ownerID), pgUUID(clusterID), name); err != nil {
		t.Fatalf("seed face label: %v", err)
	}
}

// seedForeignEmbedding stamps one face row with a foreign recognize
// model's embedding: the coexistence state after a model switch.
func (e *env) seedForeignEmbedding(t *testing.T, faceID uuid.UUID, embedModelID string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE faces SET embedding = '[1, 2, 3]'::vector, embed_model_id = $2 WHERE id = $1`,
		pgUUID(faceID), embedModelID); err != nil {
		t.Fatalf("seed foreign embedding: %v", err)
	}
}

// registryDetectModelID reads the active detect model's id from the
// models table (the registered id a seeded faces row must carry to
// satisfy the model foreign key).
func (e *env) registryDetectModelID(t *testing.T) string {
	t.Helper()
	var id pgtype.Text
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id FROM models WHERE slot = 'face_detect' AND is_active`).Scan(&id); err != nil {
		t.Fatalf("read the active detect model id: %v", err)
	}
	return id.String
}

// faceAssetOf reads the asset one face row belongs to.
func (e *env) faceAssetOf(t *testing.T, faceID uuid.UUID) uuid.UUID {
	t.Helper()
	var asset pgtype.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT asset_id FROM faces WHERE id = $1`, pgUUID(faceID)).Scan(&asset); err != nil {
		t.Fatalf("read the face's asset: %v", err)
	}
	return uuid.UUID(asset.Bytes)
}

// faceIDOf resolves the face row's id through its asset and score.
func (e *env) faceIDOf(t *testing.T, assetID uuid.UUID, score float32) uuid.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id FROM faces WHERE asset_id = $1 AND score = $2`,
		pgUUID(assetID), score).Scan(&id); err != nil {
		t.Fatalf("resolve the face id: %v", err)
	}
	return uuid.UUID(id.Bytes)
}

// TestFaceRecognizeJobEmbedsMatchedFaces pins the worker's happy
// path: the portrait's stored face row gains the embedding and the
// active recognize model's stamp, and a lone face is noise (minPts 2
// unmet), so no cluster forms -- the event still fires with the
// honest zero counts.
func TestFaceRecognizeJobEmbedsMatchedFaces(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	recognizeID := e.activateRecognition(t)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("detect: %v", err)
	}
	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}

	faces := e.facesOf(t, id)
	if len(faces) != 1 {
		t.Fatalf("faces rows = %d, want 1", len(faces))
	}
	face := faces[0]
	if !face.embeddingSet {
		t.Error("face embedding is NULL, want the aligned crop's embedding")
	}
	if !face.embedModelID.Valid || face.embedModelID.String != recognizeID {
		t.Errorf("face embed_model_id = %v, want the active recognize model %q", face.embedModelID, recognizeID)
	}
	if face.clusterID.Valid {
		t.Errorf("face cluster_id = %v, want NULL (a lone face is noise at minPts 2)", face.clusterID)
	}
	if got := len(e.clustersOf(t)); got != 0 {
		t.Errorf("cluster rows = %d, want 0", got)
	}

	payloads := e.eventPayloadsOf(t, "face.clustered")
	if len(payloads) != 1 {
		t.Fatalf("face.clustered events = %d, want exactly 1 per run", len(payloads))
	}
	if payloads[0]["totalClusters"] != "0" || payloads[0]["labeledClusters"] != "0" {
		t.Errorf("face.clustered payload = %v, want totalClusters 0 and labeledClusters 0 (string values, camelCase keys)", payloads[0])
	}
}

// TestFaceRecognizeJobIoUMatching pins the landmark-redetection
// architecture's matching rule: the worker re-runs the active detect
// model over the stored original, and a stored row joins the
// embedding step only when its box overlaps the redetected face by at
// least half the union -- the near box matches, the corner box does
// not, and the unmatched row keeps its NULL embedding untouched.
func TestFaceRecognizeJobIoUMatching(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	detectID := e.activateFixtureModel(t, yunetFixtureSpec)
	recognizeID := e.activateFixtureModel(t, sfaceFixtureSpec)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	// The portrait's face sits at [387 214 203 245] per-mille: one
	// stored row shifted by 20 per-mille (overlap well above half the
	// union) and one in the opposite corner (overlap zero).
	e.seedFace(t, id, detectID, [4]int32{407, 234, 203, 245}, 0.8)
	e.seedFace(t, id, detectID, [4]int32{0, 0, 100, 100}, 0.7)

	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}

	faces := e.facesOf(t, id)
	if len(faces) != 2 {
		t.Fatalf("faces rows = %d, want the two seeded rows", len(faces))
	}
	embedded := map[float32]bool{}
	for _, face := range faces {
		if face.modelID != detectID {
			t.Fatalf("face model_id = %q, want %q", face.modelID, detectID)
		}
		embedded[face.score] = face.embeddingSet
	}
	if !embedded[0.8] {
		t.Error("the overlapping row's embedding is NULL, want it embedded (IoU >= 0.5 matched the redetection)")
	}
	if embedded[0.7] {
		t.Error("the corner row's embedding is set, want NULL (IoU 0 never matches)")
	}
	for _, face := range faces {
		if face.score == 0.8 && (!face.embedModelID.Valid || face.embedModelID.String != recognizeID) {
			t.Errorf("matched row embed_model_id = %v, want the active recognize model %q", face.embedModelID, recognizeID)
		}
		if face.score == 0.7 {
			if face.embedModelID.Valid {
				t.Errorf("unmatched row embed_model_id = %v, want NULL", face.embedModelID)
			}
			if face.clusterID.Valid {
				t.Errorf("unmatched row cluster_id = %v, want NULL (clustering only sees embedded faces)", face.clusterID)
			}
		}
	}
}

// TestFaceRecognizeJobClusterIdentityPreserved pins the cluster-identity
// rule: two portraits of the same face (the upright original
// and its orientation-6 twin) embed identically, form ONE cluster,
// and a converging re-cluster keeps the cluster's IDENTITY -- the row
// is reused, the user's label stays put, the members stay members.
func TestFaceRecognizeJobClusterIdentityPreserved(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	upright := e.insertFaceAsset(t, "originals/face/upright.jpg", portraitFixture(t, "face-portrait.jpg"))
	oriented := e.insertFaceAsset(t, "originals/face/oriented.jpg", portraitFixture(t, "face-portrait-orientation6.jpg"))
	for _, asset := range []uuid.UUID{upright, oriented} {
		if err := runFaceDetect(t, e, asset); err != nil {
			t.Fatalf("detect: %v", err)
		}
	}

	// Round one: each run embeds its asset's face and re-clusters the
	// library; after both, the two identical embeddings share one
	// cluster whose representative is the higher-scored member.
	if err := runFaceRecognize(t, e, upright); err != nil {
		t.Fatalf("recognize(upright) error = %v, want nil", err)
	}
	if err := runFaceRecognize(t, e, oriented); err != nil {
		t.Fatalf("recognize(oriented) error = %v, want nil", err)
	}
	clusters := e.clustersOf(t)
	if len(clusters) != 1 {
		t.Fatalf("cluster rows after round one = %d, want 1", len(clusters))
	}
	cluster := clusters[0]

	faceUpright := e.facesOf(t, upright)[0]
	faceOriented := e.facesOf(t, oriented)[0]
	idUpright := e.faceIDOf(t, upright, faceUpright.score)
	idOriented := e.faceIDOf(t, oriented, faceOriented.score)
	for i, face := range []faceRow{faceUpright, faceOriented} {
		if !face.embeddingSet {
			t.Fatalf("face %d embedding is NULL, want the recognition to have embedded it", i)
		}
		if !face.clusterID.Valid || uuid.UUID(face.clusterID.Bytes) != cluster.id {
			t.Errorf("face %d cluster_id = %v, want the cluster %s", i, face.clusterID, cluster.id)
		}
	}
	// The representative is the cluster's highest-score member; an
	// exact score tie goes to the smaller id, the listing's order.
	best, bestID := faceUpright, idUpright
	if faceOriented.score > faceUpright.score ||
		(faceOriented.score == faceUpright.score && idOriented.String() < idUpright.String()) {
		best, bestID = faceOriented, idOriented
	}
	if cluster.representative != bestID {
		t.Errorf("cluster representative = %s, want the highest-score member %s (score %v)",
			cluster.representative, bestID, best.score)
	}

	e.seedFaceLabel(t, cluster.id, "Grandma")

	// Round two: a converging re-cluster over the same faces must
	// reuse the row -- the ID, the label and the membership all stand.
	if err := runFaceRecognize(t, e, upright); err != nil {
		t.Fatalf("recognize(upright, round two) error = %v, want nil", err)
	}
	clusters = e.clustersOf(t)
	if len(clusters) != 1 || clusters[0].id != cluster.id {
		t.Fatalf("cluster rows after round two = %v, want the SAME single row %s (identity preserved)",
			clusters, cluster.id)
	}
	labels := e.faceLabelsOf(t)
	if len(labels) != 1 || labels[0].clusterID != cluster.id || labels[0].name != "Grandma" {
		t.Fatalf("user labels after round two = %v, want the untouched label on %s", labels, cluster.id)
	}
	for _, asset := range []uuid.UUID{upright, oriented} {
		face := e.facesOf(t, asset)[0]
		if !face.clusterID.Valid || uuid.UUID(face.clusterID.Bytes) != cluster.id {
			t.Errorf("face of %s left the cluster across the re-cluster, want membership preserved", asset)
		}
	}

	payloads := e.eventPayloadsOf(t, "face.clustered")
	if len(payloads) != 3 {
		t.Errorf("face.clustered events = %d, want one per recognition run (3)", len(payloads))
	}
	if last := payloads[len(payloads)-1]; last["totalClusters"] != "1" || last["labeledClusters"] != "1" {
		t.Errorf("final face.clustered payload = %v, want totalClusters 1 and labeledClusters 1", last)
	}
}

// TestFaceRecognizeJobDissolvesVacatedClusters pins the empty-cluster
// semantics: a cluster whose members the new clustering no longer
// populates (here: its only member became noise) loses its row, and
// the user's labels go with it -- a person without faces is a person
// without a name, the honest semantics.
func TestFaceRecognizeJobDissolvesVacatedClusters(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("detect: %v", err)
	}
	face := e.facesOf(t, id)[0]
	faceID := e.faceIDOf(t, id, face.score)

	// The state a previous clustering pass left: the face a member of
	// a labeled cluster.
	cluster := e.seedCluster(t, faceID)
	e.attachFaceToCluster(t, faceID, cluster)
	e.seedFaceLabel(t, cluster, "Grandma")

	// The single face re-embeds to noise: the cluster loses its last
	// member and dissolves, the label cascading away with it.
	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}
	if got := len(e.clustersOf(t)); got != 0 {
		t.Errorf("cluster rows = %d, want 0 (the vacated cluster dissolved)", got)
	}
	if got := len(e.faceLabelsOf(t)); got != 0 {
		t.Errorf("user labels = %d, want 0 (the label cascaded with the cluster)", got)
	}
	after := e.facesOf(t, id)[0]
	if !after.embeddingSet {
		t.Error("face embedding is NULL, want the run to have embedded it")
	}
	if after.clusterID.Valid {
		t.Errorf("face cluster_id = %v, want NULL (a lone face is noise)", after.clusterID)
	}
}

// TestFaceRecognizeJobKeepsForeignModelClusters pins the
// model-switch coexistence on the cluster side: a cluster whose
// members are another recognize model's embeddings is invisible to
// the active model's clustering and survives untouched -- switching
// back revives it, labels and all.
func TestFaceRecognizeJobKeepsForeignModelClusters(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("detect: %v", err)
	}

	// A foreign-model face holding a labeled cluster: the state after
	// a recognize-model switch, before the new model's rerun. The row
	// belongs to the active detect model (a registered id -- the
	// foreign half is the EMBED model, a registered-but-inactive
	// second recognize model), with the old model's embedding and
	// membership still attached.
	detectID := e.registryDetectModelID(t)
	otherRecognize := e.registerFixtureModel(t, fixtureModelSpec{
		id:        "sface-other",
		slot:      ml.SlotFaceRecognize,
		file:      sfaceFixtureSpec.file,
		sha256:    sfaceFixtureSpec.sha256,
		inputSize: sfaceFixtureSpec.inputSize,
		dim:       sfaceFixtureSpec.dim,
		params:    sfaceFixtureSpec.params,
	})
	foreign := e.seedFace(t, id, detectID, [4]int32{100, 100, 200, 200}, 0.9)
	e.seedForeignEmbedding(t, foreign, otherRecognize)
	cluster := e.seedCluster(t, foreign)
	e.attachFaceToCluster(t, foreign, cluster)
	e.seedFaceLabel(t, cluster, "Grandma")

	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}

	clusters := e.clustersOf(t)
	if len(clusters) != 1 || clusters[0].id != cluster {
		t.Fatalf("cluster rows = %v, want the foreign model's cluster untouched", clusters)
	}
	labels := e.faceLabelsOf(t)
	if len(labels) != 1 || labels[0].name != "Grandma" {
		t.Fatalf("user labels = %v, want the foreign cluster's label intact", labels)
	}
	// The foreign face keeps its own embedding, model and membership.
	var embedModel pgtype.Text
	var embeddingSet bool
	var clusterID pgtype.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT embed_model_id, embedding IS NOT NULL, cluster_id FROM faces WHERE id = $1`,
		pgUUID(foreign)).Scan(&embedModel, &embeddingSet, &clusterID); err != nil {
		t.Fatalf("read the foreign face: %v", err)
	}
	if !embeddingSet || !embedModel.Valid || embedModel.String != otherRecognize {
		t.Errorf("foreign face embedding state = (%v, %v), want the old model's embedding kept", embeddingSet, embedModel)
	}
	if !clusterID.Valid || uuid.UUID(clusterID.Bytes) != cluster {
		t.Errorf("foreign face cluster_id = %v, want its cluster kept", clusterID)
	}
}

// TestFaceRecognizeJobNoRecognizeModelSkipsCleanly pins the degraded
// mode: a stale or mis-routed job against an empty recognize slot
// completes as a no-op (no error, no rows, no events).
func TestFaceRecognizeJobNoRecognizeModelSkipsCleanly(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	detectID := e.activateFixtureModel(t, yunetFixtureSpec)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	e.seedFace(t, id, detectID, [4]int32{387, 214, 203, 245}, 0.9)

	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the clean skip", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 1 || faces[0].embeddingSet {
		t.Errorf("faces = %v, want the one untouched row (no embedding without a recognize model)", faces)
	}
	if got := len(e.eventPayloadsOf(t, "face.clustered")); got != 0 {
		t.Errorf("face.clustered events = %d, want 0", got)
	}
}

// TestFaceRecognizeJobNoDetectModelSkipsCleanly pins the other half
// of the slot pair: the recognition needs the active detect model's
// landmarks (the redetection architecture), so an empty detect slot
// is also a clean no-op even with a recognize model active.
func TestFaceRecognizeJobNoDetectModelSkipsCleanly(t *testing.T) {
	e := newEnv(t)
	e.activateFixtureModel(t, sfaceFixtureSpec)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the clean skip", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0", len(faces))
	}
	if got := len(e.eventPayloadsOf(t, "face.clustered")); got != 0 {
		t.Errorf("face.clustered events = %d, want 0", got)
	}
}

// TestFaceRecognizeJobNonImageIsNoOp pins the kind gate: a video row
// (mis-routed job) completes as a no-op instead of burning retries.
func TestFaceRecognizeJobNonImageIsNoOp(t *testing.T) {
	e := newEnv(t)
	e.activateRecognition(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/face/video.mp4", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{}`, dbgen.MediaKindVideo, "video/mp4", 0)
	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0", len(faces))
	}
}

// TestFaceRecognizeJobMissingAssetCompletes pins the gone-asset path.
func TestFaceRecognizeJobMissingAssetCompletes(t *testing.T) {
	e := newEnv(t)
	e.activateRecognition(t)

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if err := runFaceRecognize(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
}

// TestFaceDetectChainFiresRecognize pins THE CHAINING RESTORE (the
// pinned handover landing): through the real queue, a detection
// completion with the recognize slot active and faces on the asset
// enqueues exactly one recognition job, which runs for real -- the
// face gains its embedding, and the asset's river jobs are the
// pipeline's two plus the ML pair.
func TestFaceDetectChainFiresRecognize(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	recognizeID := e.activateRecognition(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}

	eventually(t, jobWaitTimeout, "the chained recognition to land the embedding", func() bool {
		faces := e.facesOf(t, id)
		return len(faces) == 1 && faces[0].embeddingSet
	})

	faces := e.facesOf(t, id)
	if len(faces) != 1 {
		t.Fatalf("faces rows = %d, want 1", len(faces))
	}
	if !faces[0].embedModelID.Valid || faces[0].embedModelID.String != recognizeID {
		t.Errorf("face embed_model_id = %v, want the active recognize model %q", faces[0].embedModelID, recognizeID)
	}

	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["face_detect"] != 1 || kinds["face_recognize"] != 1 {
		t.Errorf("river jobs = %v, want exactly one metadata, one face_detect and the chained face_recognize", kinds)
	}
	if got := e.riverJobTotal(t, id); got != 3 {
		t.Errorf("river jobs total = %d, want 3", got)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
	if got := len(e.eventPayloadsOf(t, "face.clustered")); got != 1 {
		t.Errorf("face.clustered events = %d, want 1 (the chained run)", got)
	}
}

// TestFaceDetectChainHoldsWithoutFaces pins the chain's gate: a
// faceless image completes its detection with zero rows, and zero
// faces means zero continuation -- no recognition job exists for the
// asset.
func TestFaceDetectChainHoldsWithoutFaces(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/none.jpg", facelessJPEG(t), takenAt, `{}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}

	// The faceless asset's detection may not have been dispatched yet
	// when the first poll runs: the poll waits for the river row to
	// appear AND complete.
	eventually(t, jobWaitTimeout, "the faceless detection to complete", func() bool {
		var state string
		err := e.pool.QueryRow(context.Background(),
			`SELECT state FROM river_job WHERE kind = $1 AND args->>'assetId' = $2`,
			FaceDetectKind, id.String()).Scan(&state)
		return err == nil && state == "completed"
	})

	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["face_detect"] != 1 || e.riverJobTotal(t, id) != 2 {
		t.Errorf("river jobs = %v, want exactly the pipeline's two (no continuation without faces)", kinds)
	}
	if got := len(e.eventPayloadsOf(t, "face.clustered")); got != 0 {
		t.Errorf("face.clustered events = %d, want 0 (no recognition ran)", got)
	}
}

// TestFaceDetectRerunAfterRecognition pins the replace step's cluster
// sweep: after recognition clustered the two portraits' faces, a
// re-detection replaces its asset's face rows -- the doomed face is
// the cluster's representative, so the sweep re-homes the
// representative to the surviving member (the cluster and its label
// stand), and the recognition of the replacement face re-converges
// the library onto the SAME cluster row.
func TestFaceDetectRerunAfterRecognition(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	upright := e.insertFaceAsset(t, "originals/face/upright.jpg", portraitFixture(t, "face-portrait.jpg"))
	oriented := e.insertFaceAsset(t, "originals/face/oriented.jpg", portraitFixture(t, "face-portrait-orientation6.jpg"))
	for _, asset := range []uuid.UUID{upright, oriented} {
		if err := runFaceDetect(t, e, asset); err != nil {
			t.Fatalf("detect: %v", err)
		}
		if err := runFaceRecognize(t, e, asset); err != nil {
			t.Fatalf("recognize: %v", err)
		}
	}
	clusters := e.clustersOf(t)
	if len(clusters) != 1 {
		t.Fatalf("cluster rows = %d, want 1", len(clusters))
	}
	cluster := clusters[0]
	e.seedFaceLabel(t, cluster.id, "Grandma")

	// The re-detection of the representative's asset: the replace
	// step deletes the representative face itself.
	rerunAsset := e.faceAssetOf(t, cluster.representative)
	survivorAsset := oriented
	if rerunAsset == oriented {
		survivorAsset = upright
	}
	if err := runFaceDetect(t, e, rerunAsset); err != nil {
		t.Fatalf("re-detect over the clustered asset error = %v, want nil (the sweep re-homes the representative)", err)
	}

	clusters = e.clustersOf(t)
	if len(clusters) != 1 || clusters[0].id != cluster.id {
		t.Fatalf("cluster rows after the replace = %v, want the row %s alive", clusters, cluster.id)
	}
	labels := e.faceLabelsOf(t)
	if len(labels) != 1 || labels[0].name != "Grandma" {
		t.Fatalf("user labels after the replace = %v, want the label intact", labels)
	}
	// The representative moved to the surviving member.
	survivor := e.facesOf(t, survivorAsset)[0]
	if clusters[0].representative != e.faceIDOf(t, survivorAsset, survivor.score) {
		t.Errorf("re-homed representative = %s, want the surviving member of the re-detected asset", clusters[0].representative)
	}

	// The replacement face re-embeds and the library re-converges onto
	// the SAME cluster row.
	if err := runFaceRecognize(t, e, rerunAsset); err != nil {
		t.Fatalf("recognize the replacement face: %v", err)
	}
	clusters = e.clustersOf(t)
	if len(clusters) != 1 || clusters[0].id != cluster.id {
		t.Fatalf("cluster rows after the re-recognition = %v, want the SAME row %s", clusters, cluster.id)
	}
	for _, asset := range []uuid.UUID{upright, oriented} {
		face := e.facesOf(t, asset)[0]
		if !face.embeddingSet {
			t.Errorf("face of %s has no embedding, want the re-converged recognition", asset)
		}
		if !face.clusterID.Valid || uuid.UUID(face.clusterID.Bytes) != cluster.id {
			t.Errorf("face of %s cluster_id = %v, want the preserved cluster", asset, face.clusterID)
		}
	}
}

// TestRunExpiryPurgesClusteredAssets pins the trash expiry's half of
// the deletion sweep: purging soft-deleted assets whose faces hold a
// cluster dissolves the cluster (labels cascading) instead of
// tripping the representative foreign key, and the asset rows go.
func TestRunExpiryPurgesClusteredAssets(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	upright := e.insertFaceAsset(t, "originals/face/upright.jpg", portraitFixture(t, "face-portrait.jpg"))
	oriented := e.insertFaceAsset(t, "originals/face/oriented.jpg", portraitFixture(t, "face-portrait-orientation6.jpg"))
	for _, asset := range []uuid.UUID{upright, oriented} {
		if err := runFaceDetect(t, e, asset); err != nil {
			t.Fatalf("detect: %v", err)
		}
		if err := runFaceRecognize(t, e, asset); err != nil {
			t.Fatalf("recognize: %v", err)
		}
	}
	clusters := e.clustersOf(t)
	if len(clusters) != 1 {
		t.Fatalf("cluster rows = %d, want 1", len(clusters))
	}
	e.seedFaceLabel(t, clusters[0].id, "Grandma")

	if _, err := e.pool.Exec(context.Background(),
		`UPDATE assets SET deleted_at = now() - interval '40 days' WHERE id IN ($1, $2)`,
		pgUUID(upright), pgUUID(oriented)); err != nil {
		t.Fatalf("soft-delete the assets: %v", err)
	}

	counts, err := trash.RunExpiry(context.Background(), e.pool, e.dataDir, nil)
	if err != nil {
		t.Fatalf("RunExpiry error = %v, want nil (the sweep must precede the cascade)", err)
	}
	if counts.Expired != 2 {
		t.Errorf("expired = %d, want 2", counts.Expired)
	}
	if got := len(e.clustersOf(t)); got != 0 {
		t.Errorf("cluster rows = %d, want 0 (both members purged)", got)
	}
	if got := len(e.faceLabelsOf(t)); got != 0 {
		t.Errorf("user labels = %d, want 0 (the label cascaded)", got)
	}
	for _, asset := range []uuid.UUID{upright, oriented} {
		var count int
		if err := e.pool.QueryRow(context.Background(),
			`SELECT count(*) FROM faces WHERE asset_id = $1`, pgUUID(asset)).Scan(&count); err != nil {
			t.Fatalf("count faces of the purged asset: %v", err)
		}
		if count != 0 {
			t.Errorf("faces of the purged asset = %d, want 0", count)
		}
	}
}

// TestDeleteAssetEndpointPurgesClusteredFaces pins the permanent
// delete's half of the deletion sweep: the endpoint purges an asset
// whose face is a cluster's representative -- the cluster re-homes
// onto its surviving member and keeps its label -- and the LAST
// member's purge dissolves the cluster with the label cascade.
func TestDeleteAssetEndpointPurgesClusteredFaces(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateRecognition(t)

	upright := e.insertFaceAsset(t, "originals/face/upright.jpg", portraitFixture(t, "face-portrait.jpg"))
	oriented := e.insertFaceAsset(t, "originals/face/oriented.jpg", portraitFixture(t, "face-portrait-orientation6.jpg"))
	for _, asset := range []uuid.UUID{upright, oriented} {
		if err := runFaceDetect(t, e, asset); err != nil {
			t.Fatalf("detect: %v", err)
		}
		if err := runFaceRecognize(t, e, asset); err != nil {
			t.Fatalf("recognize: %v", err)
		}
	}
	clusters := e.clustersOf(t)
	if len(clusters) != 1 {
		t.Fatalf("cluster rows = %d, want 1", len(clusters))
	}
	cluster := clusters[0]
	e.seedFaceLabel(t, cluster.id, "Grandma")

	deleteAsset := func(id uuid.UUID) {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/assets/"+id.String(), nil)
		req.AddCookie(e.cookie)
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE /assets/%s status = %d, want 204 (body %q)", id, rec.Code, rec.Body.String())
		}
	}

	// The representative's asset goes first: the cluster re-homes and
	// survives with its label.
	repAsset := e.faceAssetOf(t, cluster.representative)
	survivorAsset := oriented
	if repAsset == oriented {
		survivorAsset = upright
	}
	deleteAsset(repAsset)
	clusters = e.clustersOf(t)
	if len(clusters) != 1 || clusters[0].id != cluster.id {
		t.Fatalf("cluster rows after the first purge = %v, want the re-homed row %s", clusters, cluster.id)
	}
	if labels := e.faceLabelsOf(t); len(labels) != 1 || labels[0].name != "Grandma" {
		t.Fatalf("user labels after the first purge = %v, want the label intact", labels)
	}

	// The last member's purge dissolves the cluster.
	deleteAsset(survivorAsset)
	if got := len(e.clustersOf(t)); got != 0 {
		t.Errorf("cluster rows = %d, want 0 (the last member's purge dissolved it)", got)
	}
	if got := len(e.faceLabelsOf(t)); got != 0 {
		t.Errorf("user labels = %d, want 0 (cascaded with the cluster)", got)
	}
}
