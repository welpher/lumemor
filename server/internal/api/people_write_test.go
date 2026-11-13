// People surface write tests: the label, birth date, avatar,
// merge, face-move, from-face creation and face-detach endpoints,
// with the pinned semantics under load -- labels are per-caller
// upserts and deletes, duplicate names are allowed, the person
// entity's attributes (birth date, avatar) are one value per cluster
// for every account, and every structural cluster change re-points
// representatives and dissolves vacated clusters (source labels die
// with them -- the honest documented cost). Fixtures insert models,
// faces, clusters and assets directly in the database and drive every
// write through the API, exactly like the read-side tests before
// them.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// fixtureWriteAssetID builds a deterministic fixture asset id for the
// write tests, in a namespace disjoint from the read-side families.
func fixtureWriteAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("40000000-0000-0000-0003-0000000000" + suffix)
}

// fixtureWriteClusterID builds a deterministic fixture cluster id for
// the write tests.
func fixtureWriteClusterID(suffix string) uuid.UUID {
	return uuid.MustParse("20000000-0000-0000-0003-0000000000" + suffix)
}

// fixtureWriteFaceID builds a deterministic fixture face id for the
// write tests.
func fixtureWriteFaceID(suffix string) uuid.UUID {
	return uuid.MustParse("30000000-0000-0000-0003-0000000000" + suffix)
}

// writeFaceSpec is one face of a write-test cluster fixture: its id
// and its detection score.
type writeFaceSpec struct {
	id    uuid.UUID
	score float32
}

// seedWriteCluster seeds one asset of the owner carrying the given
// faces, wired into one cluster whose representative is the FIRST
// face listed (the write endpoints re-point representatives
// themselves; fixtures never need to guess the best member).
func (e *assetEnv) seedWriteCluster(t *testing.T, assetID, ownerID uuid.UUID, taken time.Time, clusterID uuid.UUID, faces ...writeFaceSpec) {
	t.Helper()
	e.insertOwnedAsset(t, assetID, ownerID, taken, timelineAssetOpts{})
	ids := make([]uuid.UUID, len(faces))
	for i, face := range faces {
		ids[i] = face.id
		e.insertFace(t, face.id, assetID, face.score)
	}
	e.seedPersonCluster(t, clusterID, ids...)
}

// putPersonLabel fires PUT /people/{id}/label with the raw body (nil
// cookie sends it unauthenticated) and returns the recorder.
func (e *assetEnv) putPersonLabel(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPut, "/api/v1/people/"+clusterID.String()+"/label", body)
}

// deletePersonLabel fires DELETE /people/{clusterId}/label.
func (e *assetEnv) deletePersonLabel(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodDelete, "/api/v1/people/"+clusterID.String()+"/label", "")
}

// putPersonBirthdate fires PUT /people/{clusterId}/birthdate with
// the raw body.
func (e *assetEnv) putPersonBirthdate(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPut, "/api/v1/people/"+clusterID.String()+"/birthdate", body)
}

// putPersonAvatar fires PUT /people/{clusterId}/avatar with the raw
// body.
func (e *assetEnv) putPersonAvatar(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPut, "/api/v1/people/"+clusterID.String()+"/avatar", body)
}

// postPeopleMerge fires POST /people/merge with the raw body.
func (e *assetEnv) postPeopleMerge(t *testing.T, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPost, "/api/v1/people/merge", body)
}

// postFaceMove fires POST /faces/{faceId}/move with the raw body.
func (e *assetEnv) postFaceMove(t *testing.T, cookie *http.Cookie, faceID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPost, "/api/v1/faces/"+faceID.String()+"/move", body)
}

// postFaceCluster fires POST /faces/{faceId}/cluster: the from-face
// person creation carries no body at all.
func (e *assetEnv) postFaceCluster(t *testing.T, cookie *http.Cookie, faceID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPost, "/api/v1/faces/"+faceID.String()+"/cluster", "")
}

// deleteFace fires DELETE /faces/{faceId}: the single-face detach.
func (e *assetEnv) deleteFace(t *testing.T, cookie *http.Cookie, faceID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodDelete, "/api/v1/faces/"+faceID.String(), "")
}

// getAssetFaces reads one asset's detail and returns its faces
// array: the face-level tests' read-side proof that a released face
// still lists on its asset, now carrying no cluster.
func (e *assetEnv) getAssetFaces(t *testing.T, cookie *http.Cookie, assetID uuid.UUID) []apigen.AssetFaceSummary {
	t.Helper()
	rec := e.writePeopleRaw(t, cookie, http.MethodGet, "/api/v1/assets/"+assetID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET asset detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode asset detail %q: %v", rec.Body.String(), err)
	}
	return detail.Faces
}

// requireSoleMemberAndRepresentative pins the from-face creation's
// core invariant: the cluster holds exactly the seeding face, and
// that face is the cluster's representative (a sole member is the
// strongest member by definition).
func (e *assetEnv) requireSoleMemberAndRepresentative(t *testing.T, clusterID, faceID uuid.UUID) {
	t.Helper()
	if got := e.queryInt(t, "SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(clusterID)); got != 1 {
		t.Errorf("cluster %s holds %d faces, want exactly the seeding one", clusterID, got)
	}
	if got := e.queryInt(t,
		"SELECT count(*) FROM face_clusters WHERE id = $1 AND representative_face_id = $2",
		pgUUID(clusterID), pgUUID(faceID)); got != 1 {
		t.Errorf("cluster %s points at another representative than the seeding face %s", clusterID, faceID)
	}
}

// requirePayloadIDs asserts the person event discipline: the payload
// carries exactly the given ids plus the acting user -- ids only,
// never any account's private state.
func requirePayloadIDs(t *testing.T, payload map[string]string, what string, want map[string]string) {
	t.Helper()
	for key, value := range want {
		if payload[key] != value {
			t.Errorf("%s payload[%q] = %q, want %q", what, key, payload[key], value)
		}
	}
	if len(payload) != len(want) {
		t.Errorf("%s payload = %v, want exactly the keys %v (ids only)", what, payload, want)
	}
}

// writePeopleRaw is the shared write dispatcher: an empty body means
// no request content at all, anything else goes out as JSON.
func (e *assetEnv) writePeopleRaw(t *testing.T, cookie *http.Cookie, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decodePersonLabel decodes a 200 label-state body.
func decodePersonLabel(t *testing.T, rec *httptest.ResponseRecorder) apigen.PersonLabelResponse {
	t.Helper()
	var state apigen.PersonLabelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode label state %q: %v", rec.Body.String(), err)
	}
	return state
}

// requirePersonNotFound asserts the anti-enumeration 404: the same
// code and message every people read answers for an unknown id.
func requirePersonNotFound(t *testing.T, rec *httptest.ResponseRecorder, what string) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("%s status = %d, want %d (body %q)", what, rec.Code, http.StatusNotFound, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.PERSONNOTFOUND {
		t.Errorf("%s code = %q, want %q", what, apiErr.Code, apigen.PERSONNOTFOUND)
	}
	if apiErr.Message == nil || *apiErr.Message != "no such person for this caller" {
		t.Errorf("%s message = %v, want the pinned person-not-found text", what, apiErr.Message)
	}
}

// requireValidationError asserts a 400 VALIDATION_FAILED envelope and
// returns its details map (nil when the envelope carries none).
func requireValidationError(t *testing.T, rec *httptest.ResponseRecorder, what string) (apigen.Error, map[string]interface{}) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%s status = %d, want %d (body %q)", what, rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("%s code = %q, want %q", what, apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if apiErr.Details == nil {
		return apiErr, nil
	}
	return apiErr, *apiErr.Details
}

// TestPeopleLabelWrite walks the label lifecycle through the API: the
// upsert lands the cluster in the caller's named region, a re-PUT
// renames, the DELETE returns it to the unnamed region, the 1-64
// character bound rejects the edges, duplicate names coexist, and
// unknown or invisible clusters answer the anti-enumeration 404.
func TestPeopleLabelWrite(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.createSecondUser(t, "label-write-grace@example.com")
	env.insertPeopleModel(t)

	// The caller's cluster: two faces on one live asset.
	c1 := fixtureWriteClusterID("01")
	env.seedWriteCluster(t, fixtureWriteAssetID("01"), env.sessionUserID(t),
		time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), c1,
		writeFaceSpec{fixtureWriteFaceID("01"), 0.90},
		writeFaceSpec{fixtureWriteFaceID("02"), 0.80})

	// A second cluster whose faces sit only on Grace's asset: it is
	// invisible to the admin through every write, like every read.
	foreign := fixtureWriteClusterID("02")
	env.seedWriteCluster(t, fixtureWriteAssetID("02"), graceID,
		time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), foreign,
		writeFaceSpec{fixtureWriteFaceID("03"), 0.85})

	t.Run("LabelUpsertMovesClusterIntoNamedRegion", func(t *testing.T) {
		rec := env.putPersonLabel(t, env.cookie, c1, `{"name":"  Ada  "}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT label status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		state := decodePersonLabel(t, rec)
		if uuid.UUID(state.ClusterId) != c1 {
			t.Errorf("clusterId = %s, want %s", state.ClusterId, c1)
		}
		if state.Name != "Ada" {
			t.Errorf("name = %q, want the trimmed %q", state.Name, "Ada")
		}

		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("/people holds %d items, want the one named cluster", len(people))
		}
		if people[0].Name != "Ada" || uuid.UUID(people[0].ClusterId) != c1 {
			t.Errorf("/people entry = %+v, want Ada on %s", people[0], c1)
		}
		// Two faces of the caller's cluster sit on ONE asset: the
		// card counts photos, so the honest number is 1.
		if people[0].PhotoCount != 1 {
			t.Errorf("photoCount = %d, want 1 (two faces on one photo)", people[0].PhotoCount)
		}

		page := env.getUnnamedPage(t, env.cookie, "")
		if len(page.Items) != 0 {
			t.Errorf("unnamed region holds %v after the label, want empty", unnamedClusterIDs(page.Items))
		}
	})

	t.Run("ReLabelRenamesInPlace", func(t *testing.T) {
		rec := env.putPersonLabel(t, env.cookie, c1, `{"name":"Ada Lovelace"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT label status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if state := decodePersonLabel(t, rec); state.Name != "Ada Lovelace" {
			t.Errorf("name = %q, want the rename", state.Name)
		}
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || people[0].Name != "Ada Lovelace" {
			t.Fatalf("/people = %+v, want exactly the renamed entry", people)
		}
	})

	t.Run("LabelValidationRejectsOutOfBoundsNames", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"EmptyName", `{"name":""}`},
			{"WhitespaceOnlyName", `{"name":"   "}`},
			{"NameOver64Characters", `{"name":"` + strings.Repeat("é", 65) + `"}`},
			{"NameFieldMissing", `{}`},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := env.putPersonLabel(t, env.cookie, c1, testCase.body)
				_, _ = requireValidationError(t, rec, testCase.name)
			})
		}

		// The 64-character name sits exactly on the bound: accepted,
		// and the stored value is the trimmed input.
		bound := `{"name":"` + strings.Repeat("é", 64) + `"}`
		rec := env.putPersonLabel(t, env.cookie, c1, bound)
		if rec.Code != http.StatusOK {
			t.Fatalf("64-character name status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if state := decodePersonLabel(t, rec); len([]rune(state.Name)) != 64 {
			t.Errorf("stored name holds %d runes, want 64", len([]rune(state.Name)))
		}
	})

	t.Run("UnlabelReturnsClusterToUnnamedRegion", func(t *testing.T) {
		rec := env.deletePersonLabel(t, env.cookie, c1)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE label status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
		}
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Fatalf("/people holds %d items after the delete, want 0", len(people))
		}
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, c1)

		// Idempotent: deleting without a label is still a 204 once
		// the cluster is visible.
		rec = env.deletePersonLabel(t, env.cookie, c1)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("second DELETE label status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("LabelUnknownOrInvisibleClusterNotFound", func(t *testing.T) {
		rec := env.putPersonLabel(t, env.cookie, fixtureWriteClusterID("ff"), `{"name":"Ghost"}`)
		requirePersonNotFound(t, rec, "PUT label unknown cluster")

		rec = env.putPersonLabel(t, env.cookie, foreign, `{"name":"Foreign"}`)
		requirePersonNotFound(t, rec, "PUT label foreign-only cluster")

		// The anti-enumeration shape: unknown and invisible answer
		// byte-identical envelopes.
		unknown := env.putPersonLabel(t, env.cookie, fixtureWriteClusterID("ff"), `{"name":"Ghost"}`)
		if unknown.Body.String() != rec.Body.String() {
			t.Errorf("404 bodies differ between unknown and foreign-only clusters: %q vs %q",
				unknown.Body.String(), rec.Body.String())
		}

		rec = env.deletePersonLabel(t, env.cookie, foreign)
		requirePersonNotFound(t, rec, "DELETE label foreign-only cluster")
	})

	t.Run("DuplicateNamesAllowedServerSide", func(t *testing.T) {
		c2 := fixtureWriteClusterID("03")
		env.seedWriteCluster(t, fixtureWriteAssetID("03"), env.sessionUserID(t),
			time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), c2,
			writeFaceSpec{fixtureWriteFaceID("04"), 0.70})

		for _, cluster := range []uuid.UUID{c1, c2} {
			rec := env.putPersonLabel(t, env.cookie, cluster, `{"name":"Ada"}`)
			if rec.Code != http.StatusOK {
				t.Fatalf("duplicate label status = %d, want %d (no NAME_TAKEN exists; body %q)",
					rec.Code, http.StatusOK, rec.Body.String())
			}
		}
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 2 {
			t.Fatalf("/people holds %d items, want both same-named clusters", len(people))
		}
		for _, person := range people {
			if person.Name != "Ada" {
				t.Errorf("person %s name = %q, want Ada", person.ClusterId, person.Name)
			}
		}

		// Leave the fixture lean for the next subtest: one label off.
		if rec := env.deletePersonLabel(t, env.cookie, c2); rec.Code != http.StatusNoContent {
			t.Fatalf("cleanup DELETE status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		for _, probe := range []struct {
			method string
			target string
			body   string
		}{
			{http.MethodPut, "/api/v1/people/" + c1.String() + "/label", `{"name":"Ada"}`},
			{http.MethodDelete, "/api/v1/people/" + c1.String() + "/label", ""},
			{http.MethodPut, "/api/v1/people/" + c1.String() + "/birthdate", `{"birthDate":"1992-03-15"}`},
			{http.MethodPut, "/api/v1/people/" + c1.String() + "/avatar", `{"faceId":"` + fixtureWriteFaceID("01").String() + `"}`},
			{http.MethodPost, "/api/v1/people/merge", `{"sourceClusterIds":["` + c1.String() + `","` + fixtureWriteClusterID("ff").String() + `"],"targetClusterId":"` + fixtureWriteClusterID("fe").String() + `"}`},
			{http.MethodPost, "/api/v1/faces/" + fixtureWriteFaceID("01").String() + "/move", `{"targetClusterId":"` + fixtureWriteClusterID("fe").String() + `"}`},
		} {
			rec := env.writePeopleRaw(t, nil, probe.method, probe.target, probe.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s unauthenticated status = %d, want %d (body %q)",
					probe.method, probe.target, rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
		}
	})
}

// TestPeopleLabelB4Isolation is the mandated isolation case:
// labels are per-caller rows. A's naming never moves the cluster for
// B; both accounts label the same cluster independently; and the
// person ENTITY's attributes are the opposite shape -- one value per
// cluster, so B's write is what A reads.
func TestPeopleLabelB4Isolation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "b4-write-grace@example.com")
	env.insertPeopleModel(t)

	// One global cluster both accounts see through their own asset.
	shared := fixtureWriteClusterID("11")
	assetA := fixtureWriteAssetID("11")
	assetB := fixtureWriteAssetID("12")
	env.insertOwnedAsset(t, assetA, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, assetB, graceID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	fA := fixtureWriteFaceID("11")
	fB := fixtureWriteFaceID("12")
	env.insertFace(t, fA, assetA, 0.90)
	env.insertFace(t, fB, assetB, 0.95)
	env.seedPersonCluster(t, shared, fA, fB)

	t.Run("ALabelKeepsClusterInBUnnamed", func(t *testing.T) {
		rec := env.putPersonLabel(t, env.cookie, shared, `{"name":"Alice"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("A's PUT label status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || people[0].Name != "Alice" {
			t.Fatalf("A's /people = %+v, want exactly Alice", people)
		}
		aPage := env.getUnnamedPage(t, env.cookie, "")
		if len(aPage.Items) != 0 {
			t.Errorf("A's unnamed region holds %v after A's label, want empty", unnamedClusterIDs(aPage.Items))
		}

		// B's view is untouched: B has labeled nothing, so the shared
		// cluster stays in B's unnamed region and out of B's named
		// list -- another account's label never moves it.
		if people := env.getPeopleList(t, graceCookie); len(people) != 0 {
			t.Errorf("B's /people = %+v, want empty (A's label is private)", people)
		}
		bPage := env.getUnnamedPage(t, graceCookie, "")
		requireUnnamedOrder(t, bPage.Items, shared)
		if bPage.Items[0].FaceCount != 1 {
			t.Errorf("B's faceCount = %d, want 1 (B's own visible faces)", bPage.Items[0].FaceCount)
		}
	})

	t.Run("BothLabelsCoexistIndependently", func(t *testing.T) {
		rec := env.putPersonLabel(t, graceCookie, shared, `{"name":"Bob"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("B's PUT label status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		aPeople := env.getPeopleList(t, env.cookie)
		bPeople := env.getPeopleList(t, graceCookie)
		if len(aPeople) != 1 || aPeople[0].Name != "Alice" {
			t.Errorf("A's /people = %+v, want exactly Alice (B's label must not leak)", aPeople)
		}
		if len(bPeople) != 1 || bPeople[0].Name != "Bob" {
			t.Errorf("B's /people = %+v, want exactly Bob (A's label must not leak)", bPeople)
		}

		// The detail view shows each caller their own row only.
		if detail := env.getClusterDetail(t, env.cookie, shared); detail.Label == nil || detail.Label.Name != "Alice" {
			t.Errorf("A's detail label = %+v, want Alice", detail.Label)
		}
		if detail := env.getClusterDetail(t, graceCookie, shared); detail.Label == nil || detail.Label.Name != "Bob" {
			t.Errorf("B's detail label = %+v, want Bob", detail.Label)
		}
	})

	t.Run("EntityAttributesAreGlobal", func(t *testing.T) {
		// B writes the person's birth date: A reads the SAME value --
		// the entity's attribute, not a per-user row (the model
		// asymmetry: names private, entity attributes global).
		rec := env.putPersonBirthdate(t, graceCookie, shared, `{"birthDate":"1992-03-15"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("B's PUT birthdate status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		// The same holds for the avatar pick: one face per person for
		// the whole family, whatever account chose it.
		rec = env.putPersonAvatar(t, graceCookie, shared, `{"faceId":"`+fB.String()+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("B's PUT avatar status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		aDetail := env.getClusterDetail(t, env.cookie, shared)
		if aDetail.BirthDate == nil || aDetail.BirthDate.Format("2006-01-02") != "1992-03-15" {
			t.Errorf("A's birthDate = %v, want B's 1992-03-15 (the entity's own value)", aDetail.BirthDate)
		}
		if aDetail.AvatarFaceId == nil || uuid.UUID(*aDetail.AvatarFaceId) != fB {
			t.Errorf("A's avatarFaceId = %v, want B's pick %s", aDetail.AvatarFaceId, fB)
		}
		// A's name is untouched by B's entity writes, and B's name is
		// B's own: the two state classes never mix.
		if aDetail.Label == nil || aDetail.Label.Name != "Alice" {
			t.Errorf("A's label = %+v after B's entity writes, want Alice", aDetail.Label)
		}
		bDetail := env.getClusterDetail(t, graceCookie, shared)
		if bDetail.Label == nil || bDetail.Label.Name != "Bob" {
			t.Errorf("B's label = %+v, want Bob", bDetail.Label)
		}
	})

	t.Run("BUnlabelLeavesAIntact", func(t *testing.T) {
		rec := env.deletePersonLabel(t, graceCookie, shared)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("B's DELETE label status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if people := env.getPeopleList(t, env.cookie); len(people) != 1 || people[0].Name != "Alice" {
			t.Errorf("A's /people = %+v after B unlabeled, want Alice intact", people)
		}
		bPage := env.getUnnamedPage(t, graceCookie, "")
		requireUnnamedOrder(t, bPage.Items, shared)
	})
}

// TestPeopleMergeThreeStates is the mandated merge case:
// the merge reunifies faces under the target, recomputes the target's
// representative to the strongest member of the merged set, dissolves
// the source clusters WITH their labels (the honest documented cost)
// and publishes the pinned events with the merged ids.
func TestPeopleMergeThreeStates(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPeopleModel(t)

	// Two source clusters and one target, all on the caller's assets.
	// Scores are unique so the best member of the merged set is
	// unambiguous: f3 (0.95) must win the target's representative
	// even though the target's own seeded representative is f4
	// (0.85).
	s1 := fixtureWriteClusterID("21")
	s2 := fixtureWriteClusterID("22")
	target := fixtureWriteClusterID("23")
	f1 := writeFaceSpec{fixtureWriteFaceID("21"), 0.90}
	f2 := writeFaceSpec{fixtureWriteFaceID("22"), 0.80}
	f3 := writeFaceSpec{fixtureWriteFaceID("23"), 0.95}
	f4 := writeFaceSpec{fixtureWriteFaceID("24"), 0.85}
	env.seedWriteCluster(t, fixtureWriteAssetID("21"), adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), s1, f1, f2)
	env.seedWriteCluster(t, fixtureWriteAssetID("22"), adminID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), s2, f3)
	env.seedWriteCluster(t, fixtureWriteAssetID("23"), adminID, time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), target, f4)

	// The caller named both sources and the target.
	labelings := []struct {
		cluster uuid.UUID
		name    string
	}{
		{s1, "Dad"},
		{s2, "Uncle"},
		{target, "Family"},
	}
	for _, labeling := range labelings {
		rec := env.putPersonLabel(t, env.cookie, labeling.cluster, `{"name":"`+labeling.name+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("label %s = %d, want %d (body %q)", labeling.cluster, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	body := fmt.Sprintf(`{"sourceClusterIds":["%s","%s"],"targetClusterId":"%s"}`, s1, s2, target)
	rec := env.postPeopleMerge(t, env.cookie, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST merge status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var merged apigen.MergePeopleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &merged); err != nil {
		t.Fatalf("decode merge response %q: %v", rec.Body.String(), err)
	}
	if uuid.UUID(merged.SurvivingClusterId) != target {
		t.Errorf("survivingClusterId = %s, want %s", merged.SurvivingClusterId, target)
	}
	if len(merged.MergedClusterIds) != 2 || uuid.UUID(merged.MergedClusterIds[0]) != s1 || uuid.UUID(merged.MergedClusterIds[1]) != s2 {
		t.Errorf("mergedClusterIds = %v, want [%s %s] in request order", merged.MergedClusterIds, s1, s2)
	}

	t.Run("FacesReunifyUnderTarget", func(t *testing.T) {
		detail := env.getClusterDetail(t, env.cookie, target)
		wantFaces := []uuid.UUID{f3.id, f1.id, f4.id, f2.id} // score DESC, face id tiebreak
		if len(detail.Faces) != len(wantFaces) {
			t.Fatalf("target detail holds %d faces, want %d", len(detail.Faces), len(wantFaces))
		}
		for i, want := range wantFaces {
			if uuid.UUID(detail.Faces[i].Id) != want {
				t.Errorf("target faces[%d] = %s, want %s (all %v)", i, detail.Faces[i].Id, want, detail.Faces)
			}
		}
	})

	t.Run("TargetRepresentativeRecomputedToBestMember", func(t *testing.T) {
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("/people = %+v, want exactly the surviving Family entry", people)
		}
		if people[0].Name != "Family" {
			t.Errorf("survivor name = %q, want Family (the target's label persists)", people[0].Name)
		}
		if uuid.UUID(people[0].RepresentativeFaceId) != f3.id {
			t.Errorf("representative = %s, want the merged set's best face %s (0.95)", people[0].RepresentativeFaceId, f3.id)
		}
		// Four faces over three assets (one photo carries two of
		// them): the card counts photos, so the merged person reads 3.
		if people[0].PhotoCount != 3 {
			t.Errorf("photoCount = %d, want the merged set's 3 photos", people[0].PhotoCount)
		}
	})

	t.Run("SourceClustersGoneWithTheirLabels", func(t *testing.T) {
		for _, source := range []uuid.UUID{s1, s2} {
			rec := env.getPeopleRaw(t, env.cookie, "/api/v1/people/"+source.String())
			requirePersonNotFound(t, rec, "GET dissolved source")
		}

		// The labels cascaded: no user_face_labels row points at a
		// dissolved cluster, whatever account gave it.
		dangling := env.queryInt(t,
			"SELECT count(*) FROM user_face_labels WHERE cluster_id = ANY($1)",
			pgUUIDs([]uuid.UUID{s1, s2}))
		if dangling != 0 {
			t.Errorf("labels on dissolved clusters = %d, want 0 (labels die with the cluster)", dangling)
		}
		if clusters := env.queryInt(t,
			"SELECT count(*) FROM face_clusters WHERE id = ANY($1)",
			pgUUIDs([]uuid.UUID{s1, s2})); clusters != 0 {
			t.Errorf("dissolved cluster rows = %d, want 0", clusters)
		}

		// The unnamed region is empty: both sources died and the
		// survivor carries the caller's label.
		page := env.getUnnamedPage(t, env.cookie, "")
		if len(page.Items) != 0 {
			t.Errorf("unnamed region = %v after the merge, want empty", unnamedClusterIDs(page.Items))
		}
	})

	t.Run("EventsCarryMergedIds", func(t *testing.T) {
		mergedEvents := busEventsOf(t, env.bus, "person.merged")
		if len(mergedEvents) != 1 {
			t.Fatalf("person.merged events = %d, want exactly 1", len(mergedEvents))
		}
		payload := mergedEvents[0]
		if payload["survivingClusterId"] != target.String() {
			t.Errorf("survivingClusterId = %q, want %s", payload["survivingClusterId"], target)
		}
		if want := s1.String() + "," + s2.String(); payload["mergedClusterIds"] != want {
			t.Errorf("mergedClusterIds = %q, want the comma-joined %q", payload["mergedClusterIds"], want)
		}
		if payload["userId"] != adminID.String() {
			t.Errorf("userId = %q, want the caller %s", payload["userId"], adminID)
		}

		// The three label writes published their pinned event —
		// ids only: the label text is the caller's private state
		// and never rides the shared stream.
		labeledEvents := busEventsOf(t, env.bus, "person.labeled")
		if len(labeledEvents) != 3 {
			t.Fatalf("person.labeled events = %d, want 3 (one per label write)", len(labeledEvents))
		}
		byCluster := map[string]bool{}
		for _, labeled := range labeledEvents {
			if labeled["name"] != "" {
				t.Errorf("person.labeled payload carries the private label name %q — it must not leave the account", labeled["name"])
			}
			if labeled["userId"] == "" {
				t.Error("person.labeled payload missing the acting user id")
			}
			byCluster[labeled["clusterId"]] = true
		}
		for _, labeling := range labelings {
			if !byCluster[labeling.cluster.String()] {
				t.Errorf("person.labeled for cluster %s missing from the bus", labeling.cluster)
			}
		}
	})
}

// TestPeopleMergeValidation pins the 400/404 mapping of the merge
// rules: the source count bounds are structural 400s, while an
// unknown, invisible or self-referencing cluster answers the same
// byte-identical 404 as every people read.
func TestPeopleMergeValidation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "merge-validation-grace@example.com")
	env.insertPeopleModel(t)

	c1 := fixtureWriteClusterID("31")
	c2 := fixtureWriteClusterID("32")
	env.seedWriteCluster(t, fixtureWriteAssetID("31"), adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), c1,
		writeFaceSpec{fixtureWriteFaceID("31"), 0.90})
	env.seedWriteCluster(t, fixtureWriteAssetID("32"), adminID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), c2,
		writeFaceSpec{fixtureWriteFaceID("32"), 0.80})
	foreign := fixtureWriteClusterID("33")
	env.seedWriteCluster(t, fixtureWriteAssetID("33"), graceID, time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), foreign,
		writeFaceSpec{fixtureWriteFaceID("33"), 0.70})

	mergeBody := func(target uuid.UUID, sources ...uuid.UUID) string {
		ids := make([]string, len(sources))
		for i, source := range sources {
			ids[i] = `"` + source.String() + `"`
		}
		return `{"sourceClusterIds":[` + strings.Join(ids, ",") + `],"targetClusterId":"` + target.String() + `"}`
	}

	t.Run("SingleSourceMerges", func(t *testing.T) {
		// One source is the pairwise fold the cluster detail's
		// "merge into" action posts: that person merges into the
		// picked one, the source cluster dissolves.
		c4 := fixtureWriteClusterID("34")
		c5 := fixtureWriteClusterID("35")
		env.seedWriteCluster(t, fixtureWriteAssetID("34"), adminID, time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC), c4,
			writeFaceSpec{fixtureWriteFaceID("34"), 0.60})
		env.seedWriteCluster(t, fixtureWriteAssetID("35"), adminID, time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC), c5,
			writeFaceSpec{fixtureWriteFaceID("35"), 0.50})
		rec := env.postPeopleMerge(t, env.cookie, mergeBody(c5, c4))
		if rec.Code != http.StatusOK {
			t.Fatalf("single-source merge status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var merged apigen.MergePeopleResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &merged); err != nil {
			t.Fatalf("decode single-source merge response: %v", err)
		}
		if uuid.UUID(merged.SurvivingClusterId) != c5 {
			t.Errorf("single-source merge surviving = %s, want %s", merged.SurvivingClusterId, c5)
		}
		if len(merged.MergedClusterIds) != 1 || uuid.UUID(merged.MergedClusterIds[0]) != c4 {
			t.Errorf("single-source merge merged ids = %v, want [%s]", merged.MergedClusterIds, c4)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(c4)); got != 0 {
			t.Errorf("dissolved source cluster rows = %d, want 0", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(c5)); got != 2 {
			t.Errorf("target cluster faces = %d, want 2", got)
		}
	})

	t.Run("EmptySourceListRejected", func(t *testing.T) {
		rec := env.postPeopleMerge(t, env.cookie, mergeBody(c2))
		_, _ = requireValidationError(t, rec, "empty merge")
	})

	t.Run("FiftyOneSourcesRejected", func(t *testing.T) {
		// The size bound is structural: it fires before any cluster
		// resolution, so unknown ids are enough to reach it.
		sources := make([]uuid.UUID, 51)
		for i := range sources {
			sources[i] = fixtureWriteClusterID(fmt.Sprintf("%02x", i+40))
		}
		rec := env.postPeopleMerge(t, env.cookie, mergeBody(c2, sources...))
		_, _ = requireValidationError(t, rec, "51-source merge")
	})

	t.Run("SourceEqualsTargetNotFound", func(t *testing.T) {
		rec := env.postPeopleMerge(t, env.cookie, mergeBody(c1, c1, c2))
		requirePersonNotFound(t, rec, "self-referencing merge")
	})

	t.Run("UnknownOrInvisibleClusterNotFound", func(t *testing.T) {
		unknown := fixtureWriteClusterID("ff")
		for _, probe := range []struct {
			what string
			body string
		}{
			{"unknown source", mergeBody(c2, c1, unknown)},
			{"unknown target", mergeBody(unknown, c1, c2)},
			{"foreign-only source", mergeBody(c2, c1, foreign)},
			{"foreign-only target", mergeBody(foreign, c1, c2)},
		} {
			rec := env.postPeopleMerge(t, env.cookie, probe.body)
			requirePersonNotFound(t, rec, probe.what)
		}

		// The anti-enumeration shape: every 404 above is
		// byte-identical.
		rec := env.postPeopleMerge(t, env.cookie, mergeBody(c2, c1, unknown))
		foreignRec := env.postPeopleMerge(t, env.cookie, mergeBody(foreign, c1, c2))
		if rec.Body.String() != foreignRec.Body.String() {
			t.Errorf("404 bodies differ between unknown and foreign-only sources: %q vs %q",
				rec.Body.String(), foreignRec.Body.String())
		}

		// Nothing was merged by any rejected request: the two known
		// clusters still hold their own faces.
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(c1)); got != 1 {
			t.Errorf("cluster %s faces = %d after rejected merges, want 1", c1, got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(c2)); got != 1 {
			t.Errorf("cluster %s faces = %d after rejected merges, want 1", c2, got)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.postPeopleMerge(t, nil, mergeBody(c2, c1, fixtureWriteClusterID("fe")))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated merge status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestPeopleFaceMove walks the single-face move: the face lands on
// the target, the target's representative recomputes, the vacated
// source re-homes its representative (or dissolves with its label
// when the move empties it), a clusterless face joins a target with
// no source maintenance, and every invisible constellation answers
// the anti-enumeration 404.
func TestPeopleFaceMove(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "face-move-grace@example.com")
	env.insertPeopleModel(t)

	// The move fixtures: a two-face source (its label rides along)
	// and a one-face target.
	source := fixtureWriteClusterID("41")
	target := fixtureWriteClusterID("42")
	g1 := writeFaceSpec{fixtureWriteFaceID("41"), 0.90} // source representative
	g2 := writeFaceSpec{fixtureWriteFaceID("42"), 0.70}
	g3 := writeFaceSpec{fixtureWriteFaceID("43"), 0.80} // target's seeded representative
	env.seedWriteCluster(t, fixtureWriteAssetID("41"), adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), source, g1, g2)
	env.seedWriteCluster(t, fixtureWriteAssetID("42"), adminID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), target, g3)

	rec := env.putPersonLabel(t, env.cookie, source, `{"name":"Mum"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("label the source: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	moveBody := `{"targetClusterId":"` + target.String() + `"}`
	t.Run("MoveLandsFaceAndMaintainsTargetRepresentative", func(t *testing.T) {
		rec := env.postFaceMove(t, env.cookie, g1.id, moveBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var moved apigen.MoveFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &moved); err != nil {
			t.Fatalf("decode move response %q: %v", rec.Body.String(), err)
		}
		if uuid.UUID(moved.FaceId) != g1.id || uuid.UUID(moved.ClusterId) != target {
			t.Errorf("move response = (%s, %s), want (%s, %s)", moved.FaceId, moved.ClusterId, g1.id, target)
		}

		// The target now holds both faces and its representative
		// recomputed to the strongest member of its new membership.
		detail := env.getClusterDetail(t, env.cookie, target)
		if len(detail.Faces) != 2 || uuid.UUID(detail.Faces[0].Id) != g1.id || uuid.UUID(detail.Faces[1].Id) != g3.id {
			t.Fatalf("target faces = %v, want [%s %s] strongest first", detail.Faces, g1.id, g3.id)
		}
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, target)
		if uuid.UUID(page.Items[0].RepresentativeFaceId) != g1.id {
			t.Errorf("target representative = %s, want the moved best face %s", page.Items[0].RepresentativeFaceId, g1.id)
		}
	})

	t.Run("VacatedSourceRepointsAndSurvives", func(t *testing.T) {
		// The source lost its representative face but kept a member:
		// it survives, re-points to its best survivor and keeps the
		// caller's label.
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || people[0].Name != "Mum" {
			t.Fatalf("/people = %+v, want exactly Mum", people)
		}
		if uuid.UUID(people[0].RepresentativeFaceId) != g2.id {
			t.Errorf("source representative = %s, want the surviving best member %s", people[0].RepresentativeFaceId, g2.id)
		}
		if people[0].PhotoCount != 1 {
			t.Errorf("source photoCount = %d, want 1", people[0].PhotoCount)
		}
	})

	t.Run("SameClusterMoveIsAnEventlessNoOp", func(t *testing.T) {
		rec := env.postFaceMove(t, env.cookie, g1.id, moveBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("no-op move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(target)); got != 2 {
			t.Errorf("target faces = %d after the no-op move, want 2", got)
		}

		// The event discipline of the batch surface applies: a move
		// onto the cluster the face already occupies reports no
		// change, so the no-op stays silent.
		movedEvents := busEventsOf(t, env.bus, "person.face_moved")
		if len(movedEvents) != 1 {
			t.Fatalf("person.face_moved events = %d, want 1 (the real move only; the no-op stays silent)", len(movedEvents))
		}
		if movedEvents[0]["faceId"] != g1.id.String() || movedEvents[0]["clusterId"] != target.String() || movedEvents[0]["userId"] != adminID.String() {
			t.Errorf("face_moved payload = %v, want face %s on %s by %s", movedEvents[0], g1.id, target, adminID)
		}
	})

	t.Run("EmptyingSourceDissolvesWithItsLabel", func(t *testing.T) {
		rec := env.postFaceMove(t, env.cookie, g2.id, moveBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		// The vacated cluster dissolved; the label cascaded with it.
		rec = env.getPeopleRaw(t, env.cookie, "/api/v1/people/"+source.String())
		requirePersonNotFound(t, rec, "GET emptied source")
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(source)); got != 0 {
			t.Errorf("emptied source rows = %d, want 0 (dissolved)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(source)); got != 0 {
			t.Errorf("labels on the emptied source = %d, want 0 (cascaded)", got)
		}
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Errorf("/people = %+v after the dissolve, want empty (Mum died with the cluster)", people)
		}

		// The target absorbed everything; the move's event carries
		// the face's new cluster and the acting user.
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, target)
		movedEvents := busEventsOf(t, env.bus, "person.face_moved")
		if len(movedEvents) != 2 {
			t.Fatalf("person.face_moved events = %d, want 2 (the two real moves)", len(movedEvents))
		}
		last := movedEvents[len(movedEvents)-1]
		if last["faceId"] != g2.id.String() || last["clusterId"] != target.String() || last["userId"] != adminID.String() {
			t.Errorf("last face_moved payload = %v, want face %s on %s by %s", last, g2.id, target, adminID)
		}
	})

	t.Run("InvisibleConstellationsNotFound", func(t *testing.T) {
		// Grace's two clusters: targets and movables inside her own
		// visible world, all invisible to the admin.
		graceFrom := fixtureWriteClusterID("43")
		graceTo := fixtureWriteClusterID("44")
		env.seedWriteCluster(t, fixtureWriteAssetID("43"), graceID,
			time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), graceFrom,
			writeFaceSpec{fixtureWriteFaceID("44"), 0.60})
		env.seedWriteCluster(t, fixtureWriteAssetID("44"), graceID,
			time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC), graceTo,
			writeFaceSpec{fixtureWriteFaceID("47"), 0.55})

		for _, probe := range []struct {
			what string
			face uuid.UUID
			body string
		}{
			{"unknown face", fixtureWriteFaceID("ff"), moveBody},
			{"foreign-asset face", fixtureWriteFaceID("44"), moveBody},
			{"invisible target", g3.id, `{"targetClusterId":"` + graceFrom.String() + `"}`},
			{"unknown target", g3.id, `{"targetClusterId":"` + fixtureWriteClusterID("fe").String() + `"}`},
		} {
			rec := env.postFaceMove(t, env.cookie, probe.face, probe.body)
			requirePersonNotFound(t, rec, probe.what)
		}

		// Grace moves her own face onto her own second cluster: the
		// move works inside her visible world (and empties her first
		// cluster, which dissolves).
		rec := env.postFaceMove(t, graceCookie, fixtureWriteFaceID("44"),
			`{"targetClusterId":"`+graceTo.String()+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("grace's own move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(graceFrom)); got != 0 {
			t.Errorf("grace's emptied cluster rows = %d, want 0 (dissolved)", got)
		}
	})

	t.Run("ClusterlessSourceMovesIntoATarget", func(t *testing.T) {
		// The clusterless-source pin: a face the clustering has not
		// grouped yet may JOIN an existing person. The move's object
		// is the face's membership, and with no source cluster there
		// is nothing to maintain -- no representative re-home, no
		// dissolution -- so only the target's own side runs.
		clusterless := fixtureWriteFaceID("48")
		env.insertFace(t, clusterless, fixtureWriteAssetID("41"), 0.40)

		clustersBefore := env.queryInt(t, "SELECT count(*) FROM face_clusters")
		eventsBefore := len(busEventsOf(t, env.bus, "person.face_moved"))

		rec := env.postFaceMove(t, env.cookie, clusterless, moveBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("clusterless POST move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var moved apigen.MoveFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &moved); err != nil {
			t.Fatalf("decode clusterless move response %q: %v", rec.Body.String(), err)
		}
		if uuid.UUID(moved.FaceId) != clusterless || uuid.UUID(moved.ClusterId) != target {
			t.Errorf("clusterless move response = (%s, %s), want (%s, %s)", moved.FaceId, moved.ClusterId, clusterless, target)
		}

		// The target absorbed the face and its representative stayed
		// its strongest member: the newcomer is weaker than every
		// face already there.
		detail := env.getClusterDetail(t, env.cookie, target)
		if len(detail.Faces) != 4 || uuid.UUID(detail.Faces[0].Id) != g1.id {
			t.Fatalf("target faces = %v, want 4 strongest-first from %s", detail.Faces, g1.id)
		}

		// No source maintenance ran: nothing dissolved, nothing was
		// minted, and no cluster took the moved face as its
		// representative.
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters"); got != clustersBefore {
			t.Errorf("face_clusters rows = %d after the clusterless move, want %d (no dissolution, no mint)", got, clustersBefore)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE representative_face_id = $1", pgUUID(clusterless)); got != 0 {
			t.Errorf("clusters represented by the moved face = %d, want 0 (the target kept its strongest member)", got)
		}

		// The write reports the change like any other move.
		movedEvents := busEventsOf(t, env.bus, "person.face_moved")
		if len(movedEvents) != eventsBefore+1 {
			t.Fatalf("person.face_moved events = %d, want %d", len(movedEvents), eventsBefore+1)
		}
		last := movedEvents[len(movedEvents)-1]
		if last["faceId"] != clusterless.String() || last["clusterId"] != target.String() || last["userId"] != adminID.String() {
			t.Errorf("face_moved payload = %v, want face %s on %s by %s", last, clusterless, target, adminID)
		}
	})

	t.Run("MalformedBodyRejected", func(t *testing.T) {
		rec := env.postFaceMove(t, env.cookie, g3.id, `not json`)
		_, _ = requireValidationError(t, rec, "malformed move body")
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.postFaceMove(t, nil, g3.id, moveBody)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated move status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestPeopleBirthdateWrite pins the person entity's birth date: a
// normal date lands and reads back for every account, null clears it,
// a body without the field is a client bug (never a silent clear), a
// malformed date answers the field's 400, the anti-enumeration 404
// covers unknown and invisible clusters alike, and concurrent writes
// converge on one value without an error.
func TestPeopleBirthdateWrite(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "birthdate-grace@example.com")
	env.insertPeopleModel(t)

	// The caller's cluster: one face on the caller's asset.
	c1 := fixtureWriteClusterID("71")
	env.seedWriteCluster(t, fixtureWriteAssetID("71"), adminID,
		time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), c1,
		writeFaceSpec{fixtureWriteFaceID("71"), 0.90})

	// A cluster whose only face sits on Grace's asset: invisible to
	// the admin, so its 404 is the anti-enumeration pair.
	foreign := fixtureWriteClusterID("72")
	env.seedWriteCluster(t, fixtureWriteAssetID("72"), graceID,
		time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), foreign,
		writeFaceSpec{fixtureWriteFaceID("72"), 0.85})

	// A cluster the two accounts share (a face on each account's own
	// asset): the global-read arm needs one both can see.
	sharedCluster := fixtureWriteClusterID("73")
	adminFace := fixtureWriteFaceID("73")
	graceFace := fixtureWriteFaceID("74")
	env.insertOwnedAsset(t, fixtureWriteAssetID("73"), adminID,
		time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, fixtureWriteAssetID("74"), graceID,
		time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertFace(t, adminFace, fixtureWriteAssetID("73"), 0.90)
	env.insertFace(t, graceFace, fixtureWriteAssetID("74"), 0.88)
	env.seedPersonCluster(t, sharedCluster, adminFace, graceFace)

	t.Run("SetOnUnnamedCluster", func(t *testing.T) {
		// The birth date does not depend on a name: the cluster is
		// unnamed for the caller and the write still lands.
		rec := env.putPersonBirthdate(t, env.cookie, c1, `{"birthDate":"1992-03-15"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT birthdate status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response apigen.PersonBirthdateResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode birthdate response %q: %v", rec.Body.String(), err)
		}
		if uuid.UUID(response.ClusterId) != c1 {
			t.Errorf("clusterId = %s, want %s", response.ClusterId, c1)
		}
		if response.BirthDate == nil || response.BirthDate.Format("2006-01-02") != "1992-03-15" {
			t.Errorf("birthDate = %v, want 1992-03-15", response.BirthDate)
		}

		detail := env.getClusterDetail(t, env.cookie, c1)
		if detail.BirthDate == nil || detail.BirthDate.Format("2006-01-02") != "1992-03-15" {
			t.Errorf("detail birthDate = %v, want 1992-03-15", detail.BirthDate)
		}
		if detail.Label != nil {
			t.Errorf("detail label = %+v, want null (the date writes onto an unnamed cluster)", detail.Label)
		}

		// The event carries ids only: the date itself never rides the
		// shared stream.
		events := busEventsOf(t, env.bus, "person.birthdate_updated")
		if len(events) != 1 {
			t.Fatalf("person.birthdate_updated events = %d, want 1", len(events))
		}
		requirePayloadIDs(t, events[0], "person.birthdate_updated", map[string]string{
			"clusterId": c1.String(),
			"userId":    adminID.String(),
		})
	})

	t.Run("ClearWithNull", func(t *testing.T) {
		rec := env.putPersonBirthdate(t, env.cookie, c1, `{"birthDate":null}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("clear status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response apigen.PersonBirthdateResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode clear response %q: %v", rec.Body.String(), err)
		}
		if response.BirthDate != nil {
			t.Errorf("birthDate = %v after the clear, want null", response.BirthDate)
		}
		if detail := env.getClusterDetail(t, env.cookie, c1); detail.BirthDate != nil {
			t.Errorf("detail birthDate = %v after the clear, want null", detail.BirthDate)
		}
	})

	t.Run("BodyWithoutTheFieldRejected", func(t *testing.T) {
		// A missing field is a bug, not a clear: the clear is an act.
		requireValidationFields(t, env.putPersonBirthdate(t, env.cookie, c1, `{}`), "birthDate")
		requireValidationError(t, env.putPersonBirthdate(t, env.cookie, c1, `not json`), "malformed birthdate body")
	})

	t.Run("MalformedDateRejected", func(t *testing.T) {
		for _, body := range []string{`{"birthDate":"15/03/1992"}`, `{"birthDate":""}`, `{"birthDate":123}`} {
			requireValidationFields(t, env.putPersonBirthdate(t, env.cookie, c1, body), "birthDate")
		}
		// The rejected writes changed nothing.
		if detail := env.getClusterDetail(t, env.cookie, c1); detail.BirthDate != nil {
			t.Errorf("birthDate = %v after rejected writes, want the untouched null", detail.BirthDate)
		}
	})

	t.Run("UnknownOrInvisibleClusterNotFound", func(t *testing.T) {
		unknown := env.putPersonBirthdate(t, env.cookie, fixtureWriteClusterID("ff"), `{"birthDate":"1992-03-15"}`)
		requirePersonNotFound(t, unknown, "birthdate unknown cluster")
		invisible := env.putPersonBirthdate(t, env.cookie, foreign, `{"birthDate":"1992-03-15"}`)
		requirePersonNotFound(t, invisible, "birthdate foreign-only cluster")
		if unknown.Body.String() != invisible.Body.String() {
			t.Errorf("404 bodies differ between unknown and foreign-only clusters: %q vs %q",
				unknown.Body.String(), invisible.Body.String())
		}
	})

	t.Run("AnyAccountReadsTheSameValue", func(t *testing.T) {
		// Grace sees the cluster through her own asset; the admin's
		// write is what she reads (one birthday per person).
		if rec := env.putPersonBirthdate(t, env.cookie, sharedCluster, `{"birthDate":"1990-12-01"}`); rec.Code != http.StatusOK {
			t.Fatalf("admin write status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		graceDetail := env.getClusterDetail(t, graceCookie, sharedCluster)
		if graceDetail.BirthDate == nil || graceDetail.BirthDate.Format("2006-01-02") != "1990-12-01" {
			t.Errorf("grace birthDate = %v, want the admin's 1990-12-01", graceDetail.BirthDate)
		}
	})

	t.Run("ConcurrentWritesConverge", func(t *testing.T) {
		// Concurrent writers race honestly: every request answers 200
		// (the write is one column, no uniqueness to violate) and the
		// last one wins -- the reads answer ONE of the written dates,
		// never a half state.
		dates := []string{"1985-01-02", "1986-02-03", "1987-03-04", "1988-04-05"}
		results := make(chan *httptest.ResponseRecorder, len(dates))
		var wg sync.WaitGroup
		for _, date := range dates {
			wg.Add(1)
			go func(day string) {
				defer wg.Done()
				results <- env.putPersonBirthdate(t, env.cookie, c1, `{"birthDate":"`+day+`"}`)
			}(date)
		}
		wg.Wait()
		close(results)
		for rec := range results {
			if rec.Code != http.StatusOK {
				t.Fatalf("concurrent write status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
			}
		}
		detail := env.getClusterDetail(t, env.cookie, c1)
		if detail.BirthDate == nil {
			t.Fatal("birthDate = null after concurrent writes, want one of the written dates")
		}
		got := detail.BirthDate.Format("2006-01-02")
		matched := false
		for _, date := range dates {
			if got == date {
				matched = true
			}
		}
		if !matched {
			t.Errorf("birthDate = %q after concurrent writes, want one of %v", got, dates)
		}
	})
}

// TestPeopleAvatarWrite pins the person entity's avatar: the pick lands
// on the cluster's own face and the card renders it even when it is not
// the best-scoring member, every account reads the same pick, a face
// outside the cluster or on an invisible asset answers the 404, and a
// pick whose face later leaves the cluster degrades to the best visible
// member instead of breaking.
func TestPeopleAvatarWrite(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "avatar-grace@example.com")
	env.insertPeopleModel(t)

	// The caller's cluster: two faces on two assets -- the strongest
	// is the second one, so the avatar pick has something to override.
	c1 := fixtureWriteClusterID("81")
	weak := fixtureWriteFaceID("81")   // 0.70, on the pick's own asset
	strong := fixtureWriteFaceID("82") // 0.95, the fallback card face
	avA := fixtureWriteAssetID("81")
	env.insertOwnedAsset(t, avA, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertFace(t, weak, avA, 0.70)
	avB := fixtureWriteAssetID("82")
	env.insertOwnedAsset(t, avB, adminID, time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertFace(t, strong, avB, 0.95)
	env.seedPersonCluster(t, c1, weak, strong)

	// A second cluster of the caller and a cluster of Grace's: both
	// supply faces the pick must refuse.
	other := fixtureWriteClusterID("82")
	otherFace := fixtureWriteFaceID("83")
	env.seedWriteCluster(t, fixtureWriteAssetID("83"), adminID,
		time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), other,
		writeFaceSpec{otherFace, 0.90})
	foreign := fixtureWriteClusterID("83")
	foreignFace := fixtureWriteFaceID("84")
	env.seedWriteCluster(t, fixtureWriteAssetID("84"), graceID,
		time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), foreign,
		writeFaceSpec{foreignFace, 0.80})

	// Between the writes the caller's cluster carries a label, so the
	// card checks run against the named listing; the unnamed tile
	// reads the same pick.
	if rec := env.putPersonLabel(t, env.cookie, c1, `{"name":"Ada"}`); rec.Code != http.StatusOK {
		t.Fatalf("label: status = %d, want %d", rec.Code, http.StatusOK)
	}

	t.Run("FallbackIsTheBestVisibleFace", func(t *testing.T) {
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("/people = %+v, want exactly the labeled cluster", people)
		}
		if uuid.UUID(people[0].RepresentativeFaceId) != strong {
			t.Errorf("card face = %s, want the best-scoring member %s before any pick",
				people[0].RepresentativeFaceId, strong)
		}
		if detail := env.getClusterDetail(t, env.cookie, c1); detail.AvatarFaceId != nil {
			t.Errorf("avatarFaceId = %v before any pick, want null", detail.AvatarFaceId)
		}
	})

	t.Run("PickOverridesTheBestMember", func(t *testing.T) {
		rec := env.putPersonAvatar(t, env.cookie, c1, `{"faceId":"`+weak.String()+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT avatar status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response apigen.PersonAvatarResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode avatar response %q: %v", rec.Body.String(), err)
		}
		if uuid.UUID(response.ClusterId) != c1 || uuid.UUID(response.AvatarFaceId) != weak {
			t.Errorf("avatar response = %+v, want %s on %s", response, weak, c1)
		}

		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || uuid.UUID(people[0].RepresentativeFaceId) != weak {
			t.Fatalf("/people card face = %+v, want the picked avatar %s", people, weak)
		}
		wantThumbnail := "/api/v1/faces/" + weak.String() + "/thumbnail"
		if people[0].Thumbnail != wantThumbnail {
			t.Errorf("card thumbnail = %q, want the picked face's crop %q", people[0].Thumbnail, wantThumbnail)
		}
		if detail := env.getClusterDetail(t, env.cookie, c1); detail.AvatarFaceId == nil || uuid.UUID(*detail.AvatarFaceId) != weak {
			t.Errorf("detail avatarFaceId = %v, want %s", detail.AvatarFaceId, weak)
		}

		events := busEventsOf(t, env.bus, "person.avatar_updated")
		if len(events) != 1 {
			t.Fatalf("person.avatar_updated events = %d, want 1", len(events))
		}
		requirePayloadIDs(t, events[0], "person.avatar_updated", map[string]string{
			"clusterId": c1.String(),
			"userId":    adminID.String(),
		})
	})

	t.Run("EveryAccountReadsTheSamePick", func(t *testing.T) {
		// Grace reaches the cluster through the admin's label? No --
		// through her own face? She has none here, so the pick is
		// proven global on a cluster she CAN see: the unnamed region's
		// tile renders the picked avatar too.
		unnamed := env.getUnnamedPage(t, env.cookie, "")
		found := false
		for _, item := range unnamed.Items {
			if uuid.UUID(item.ClusterId) == other {
				found = true
			}
		}
		if !found {
			t.Fatalf("unnamed page = %v, want the unlabeled second cluster", unnamedClusterIDs(unnamed.Items))
		}
		rec := env.putPersonAvatar(t, env.cookie, other, `{"faceId":"`+otherFace.String()+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("unnamed-cluster avatar pick status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		page := env.getUnnamedPage(t, env.cookie, "")
		for _, item := range page.Items {
			if uuid.UUID(item.ClusterId) == other && uuid.UUID(item.RepresentativeFaceId) != otherFace {
				t.Errorf("unnamed tile face = %s, want the picked avatar %s", item.RepresentativeFaceId, otherFace)
			}
		}
		// Grace reads the admin-picked avatar of the shared cluster
		// through the same global column: the detail is admin-only
		// here, so the proof is the row itself.
		var avatar uuid.UUID
		row := env.pool.QueryRow(t.Context(),
			"SELECT avatar_face_id FROM face_clusters WHERE id = $1", pgUUID(other))
		if err := row.Scan(&avatar); err != nil {
			t.Fatalf("read the stored avatar: %v", err)
		}
		if avatar != otherFace {
			t.Errorf("stored avatar_face_id = %s, want %s", avatar, otherFace)
		}
	})

	t.Run("FaceOutsideTheClusterRefused", func(t *testing.T) {
		rec := env.putPersonAvatar(t, env.cookie, c1, `{"faceId":"`+otherFace.String()+`"}`)
		requirePersonNotFound(t, rec, "avatar from another cluster")

		// An invisible face: Grace's cluster is invisible to the
		// admin, and so is its face -- both answer the same 404.
		invisible := env.putPersonAvatar(t, env.cookie, foreign, `{"faceId":"`+foreignFace.String()+`"}`)
		requirePersonNotFound(t, invisible, "avatar on an invisible cluster")
		if rec.Body.String() != invisible.Body.String() {
			t.Errorf("404 bodies differ between another cluster's face and an invisible cluster: %q vs %q",
				rec.Body.String(), invisible.Body.String())
		}

		// Nothing moved.
		if detail := env.getClusterDetail(t, env.cookie, c1); detail.AvatarFaceId == nil || uuid.UUID(*detail.AvatarFaceId) != weak {
			t.Errorf("detail avatarFaceId = %v after the refusals, want the earlier pick %s", detail.AvatarFaceId, weak)
		}
	})

	t.Run("MalformedBodyRejected", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"faceId":"not-a-uuid"}`, `not json`} {
			rec := env.putPersonAvatar(t, env.cookie, c1, body)
			_, _ = requireValidationError(t, rec, "malformed avatar "+body)
		}
	})

	t.Run("AvatarThatLeavesTheClusterFallsBack", func(t *testing.T) {
		// The picked face moves to the other cluster: the pick stays
		// stored but stops counting, and the card falls back to the
		// best remaining visible member.
		rec := env.postFaceMove(t, env.cookie, weak, `{"targetClusterId":"`+other.String()+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("face move status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		people := env.getPeopleList(t, env.cookie)
		var card *apigen.PersonSummary
		for i := range people {
			if uuid.UUID(people[i].ClusterId) == c1 {
				card = &people[i]
			}
		}
		if card == nil {
			t.Fatalf("/people = %+v, want the labeled cluster still listed", people)
		}
		if uuid.UUID(card.RepresentativeFaceId) != strong {
			t.Errorf("card face = %s after the avatar left, want the fallback %s",
				card.RepresentativeFaceId, strong)
		}
	})
}

// TestPeopleWriteDeleteAssetCascades is the delete-cascade linkage at
// the API level: a permanent asset delete dissolves the clusters its
// faces held, and the caller's labels cascade with the cluster -- the
// people surface is left with zero empty clusters and zero dangling
// names.
func TestPeopleWriteDeleteAssetCascades(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPeopleModel(t)

	asset := fixtureWriteAssetID("61")
	cluster := fixtureWriteClusterID("61")
	env.seedWriteCluster(t, asset, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), cluster,
		writeFaceSpec{fixtureWriteFaceID("61"), 0.90},
		writeFaceSpec{fixtureWriteFaceID("62"), 0.80})
	rec := env.putPersonLabel(t, env.cookie, cluster, `{"name":"Gone"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("label: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	// The permanent delete (the owner's own purge surface) removes
	// the row; the sweep re-homes or dissolves the clusters its faces
	// held, here dissolving the only cluster.
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assets/"+asset.String(), nil)
	req.AddCookie(env.cookie)
	del := httptest.NewRecorder()
	env.router.ServeHTTP(del, req)
	if del.Code != http.StatusNoContent {
		t.Fatalf("DELETE asset status = %d, want %d (body %q)", del.Code, http.StatusNoContent, del.Body.String())
	}

	rec = env.getPeopleRaw(t, env.cookie, "/api/v1/people/"+cluster.String())
	requirePersonNotFound(t, rec, "GET cluster after its asset died")
	if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
		t.Errorf("/people = %+v after the delete, want empty (the label cascaded)", people)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(cluster)); got != 0 {
		t.Errorf("labels on the dissolved cluster = %d, want 0", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(cluster)); got != 0 {
		t.Errorf("dissolved cluster rows = %d, want 0", got)
	}
}

// TestPeopleFaceClusterCreate walks the from-face person creation:
// the seeding face re-homes onto a brand-new cluster as its sole
// member and representative, the vacated source follows the
// maintenance discipline (representative re-home, or dissolution
// with the label cascade), a repeat call CREATES again (never a
// no-op), a clusterless face seeds a cluster with no source to
// maintain, and every invisible constellation answers the
// anti-enumeration 404.
func TestPeopleFaceClusterCreate(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "face-create-grace@example.com")
	env.insertPeopleModel(t)

	// The fixtures: a labeled two-face source (representative
	// first) and one clusterless face on the owner's asset.
	source := fixtureWriteClusterID("71")
	k1 := writeFaceSpec{fixtureWriteFaceID("71"), 0.90} // source representative
	k2 := writeFaceSpec{fixtureWriteFaceID("72"), 0.70}
	asset := fixtureWriteAssetID("71")
	env.seedWriteCluster(t, asset, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), source, k1, k2)
	clusterless := fixtureWriteFaceID("73")
	env.insertFace(t, clusterless, asset, 0.50)

	if rec := env.putPersonLabel(t, env.cookie, source, `{"name":"Kim"}`); rec.Code != http.StatusOK {
		t.Fatalf("label the source: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var firstCluster uuid.UUID
	t.Run("ClusteredFaceBecomesSoleMemberAndRepresentative", func(t *testing.T) {
		rec := env.postFaceCluster(t, env.cookie, k2.id)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST cluster status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var created apigen.ClusterFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode cluster response %q: %v", rec.Body.String(), err)
		}
		firstCluster = uuid.UUID(created.ClusterId)
		if uuid.UUID(created.FaceId) != k2.id {
			t.Errorf("cluster response faceId = %s, want the seeded %s", created.FaceId, k2.id)
		}
		if firstCluster == source {
			t.Fatalf("the new cluster is the source itself -- the creation must build a new row")
		}
		env.requireSoleMemberAndRepresentative(t, firstCluster, k2.id)

		// The new cluster is unnamed, so it surfaces in the
		// caller's unnamed region holding one face.
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, firstCluster)

		// The source survives with its remaining face: the
		// representative stays the strongest member (k2 was
		// never it), and the label rides along untouched.
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || people[0].Name != "Kim" || uuid.UUID(people[0].ClusterId) != source {
			t.Fatalf("/people = %+v, want exactly Kim on the surviving source", people)
		}
		if uuid.UUID(people[0].RepresentativeFaceId) != k1.id || people[0].PhotoCount != 1 {
			t.Errorf("source state = %+v, want representative %s with faceCount 1", people[0], k1.id)
		}
	})

	t.Run("RepeatCallCreatesAnotherCluster", func(t *testing.T) {
		// The pin: a call on a face that already holds a cluster
		// is a CREATE, never a no-op -- the face re-homes again
		// and the cluster it left dies with zero members.
		rec := env.postFaceCluster(t, env.cookie, k2.id)
		if rec.Code != http.StatusOK {
			t.Fatalf("repeat POST cluster status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var created apigen.ClusterFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode repeat cluster response %q: %v", rec.Body.String(), err)
		}
		secondCluster := uuid.UUID(created.ClusterId)
		if secondCluster == firstCluster {
			t.Fatalf("repeat creation returned the same cluster %s -- it must build another", secondCluster)
		}
		env.requireSoleMemberAndRepresentative(t, secondCluster, k2.id)
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(firstCluster)); got != 0 {
			t.Errorf("the vacated first cluster holds %d rows, want 0 (dissolved with no member)", got)
		}
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, secondCluster)
	})

	t.Run("ClusterlessFaceSeedsANewCluster", func(t *testing.T) {
		// The clusterless arm: no source to maintain, the face
		// simply gains its own person.
		rec := env.postFaceCluster(t, env.cookie, clusterless)
		if rec.Code != http.StatusOK {
			t.Fatalf("clusterless POST cluster status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var created apigen.ClusterFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode clusterless cluster response %q: %v", rec.Body.String(), err)
		}
		env.requireSoleMemberAndRepresentative(t, uuid.UUID(created.ClusterId), clusterless)
	})

	t.Run("EmptyingTheSourceDissolvesItWithTheLabelCascade", func(t *testing.T) {
		// k1 is the source's last member now (k2 left twice).
		rec := env.postFaceCluster(t, env.cookie, k1.id)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST cluster of the last member status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(source)); got != 0 {
			t.Errorf("emptied source rows = %d, want 0 (dissolved)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(source)); got != 0 {
			t.Errorf("labels on the emptied source = %d, want 0 (cascaded)", got)
		}
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Errorf("/people = %+v after the dissolve, want empty (Kim died with the cluster)", people)
		}
	})

	t.Run("GraceCreatesFromHerOwnFace", func(t *testing.T) {
		// The gate is owner-scoped curation, not admin power:
		// Grace creates from a face on her own asset.
		graceCluster := fixtureWriteClusterID("72")
		graceFace := fixtureWriteFaceID("74")
		env.seedWriteCluster(t, fixtureWriteAssetID("72"), graceID,
			time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), graceCluster,
			writeFaceSpec{graceFace, 0.60})
		rec := env.postFaceCluster(t, graceCookie, graceFace)
		if rec.Code != http.StatusOK {
			t.Fatalf("grace's own POST cluster status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var created apigen.ClusterFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode grace's cluster response %q: %v", rec.Body.String(), err)
		}
		env.requireSoleMemberAndRepresentative(t, uuid.UUID(created.ClusterId), graceFace)
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(graceCluster)); got != 0 {
			t.Errorf("grace's emptied source rows = %d, want 0 (dissolved)", got)
		}
	})

	t.Run("InvisibleConstellationsNotFound", func(t *testing.T) {
		// The revoked arms: the caller's own assets, but not
		// live -- trashed and hidden both fail the same gate.
		trashed := fixtureWriteFaceID("75")
		env.insertOwnedAsset(t, fixtureWriteAssetID("73"), adminID,
			time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
		env.insertFace(t, trashed, fixtureWriteAssetID("73"), 0.40)
		hidden := fixtureWriteFaceID("76")
		env.insertOwnedAsset(t, fixtureWriteAssetID("74"), adminID,
			time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})
		env.insertFace(t, hidden, fixtureWriteAssetID("74"), 0.45)

		probes := []struct {
			what string
			face uuid.UUID
		}{
			{"unknown face", fixtureWriteFaceID("ff")},
			{"foreign-asset face", fixtureWriteFaceID("74")},
			{"trashed-asset face", trashed},
			{"hidden-asset face", hidden},
		}
		var bodies []string
		for _, probe := range probes {
			rec := env.postFaceCluster(t, env.cookie, probe.face)
			requirePersonNotFound(t, rec, probe.what)
			bodies = append(bodies, rec.Body.String())
		}
		for _, body := range bodies[1:] {
			if body != bodies[0] {
				t.Errorf("404 bodies differ across the refusal matrix: %q vs %q", body, bodies[0])
			}
		}
	})

	t.Run("EventsCarryFaceAndNewCluster", func(t *testing.T) {
		clustered := busEventsOf(t, env.bus, "person.face_clustered")
		// One per committed creation: k2, k2 again, the
		// clusterless face, k1 emptying the source, grace's own.
		if len(clustered) != 5 {
			t.Fatalf("person.face_clustered events = %d, want 5 (one per creation)", len(clustered))
		}
		for _, payload := range clustered {
			if payload["faceId"] == "" || payload["clusterId"] == "" {
				t.Errorf("person.face_clustered payload = %v, want the face and cluster ids", payload)
			}
			if payload["userId"] != adminID.String() && payload["userId"] != graceID.String() {
				t.Errorf("person.face_clustered userId = %q, want the acting user", payload["userId"])
			}
		}
		requirePayloadIDs(t, clustered[0], "first creation", map[string]string{
			"faceId":    k2.id.String(),
			"clusterId": firstCluster.String(),
			"userId":    adminID.String(),
		})

		// The detach event family stays silent: nothing was
		// released, and a release is the only thing it reports.
		if detached := busEventsOf(t, env.bus, "person.face_detached"); len(detached) != 0 {
			t.Errorf("person.face_detached events = %d, want 0 (no detach happened)", len(detached))
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.postFaceCluster(t, nil, fixtureWriteFaceID("ff"))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated POST cluster status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestPeopleFaceDetach walks the single-face detach: the face's
// cluster assignment goes NULL (the not-yet-grouped pool its asset's
// face list reads), the source survives with a re-homed
// representative unless the detach emptied it (then the dissolution
// and label cascade), a released face can seed a new person again,
// and every invisible constellation -- a clusterless face included,
// part of no person -- answers the anti-enumeration 404.
func TestPeopleFaceDetach(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "face-detach-grace@example.com")
	env.insertPeopleModel(t)

	// The fixtures: a labeled two-face source (representative
	// first) on the owner's asset.
	source := fixtureWriteClusterID("81")
	h1 := writeFaceSpec{fixtureWriteFaceID("81"), 0.90} // source representative
	h2 := writeFaceSpec{fixtureWriteFaceID("82"), 0.70}
	asset := fixtureWriteAssetID("81")
	env.seedWriteCluster(t, asset, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), source, h1, h2)

	if rec := env.putPersonLabel(t, env.cookie, source, `{"name":"Dad"}`); rec.Code != http.StatusOK {
		t.Fatalf("label the source: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	t.Run("DetachReleasesTheFaceAndRehomesTheSource", func(t *testing.T) {
		rec := env.deleteFace(t, env.cookie, h2.id)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE face status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE id = $1 AND cluster_id IS NULL", pgUUID(h2.id)); got != 1 {
			t.Errorf("the detached face holds a cluster still: rows with NULL = %d, want 1", got)
		}

		// The source survives: its representative re-homes to
		// the best survivor (h1 was it already -- the detach of
		// h2 changes nothing), and the label rides along.
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || people[0].Name != "Dad" || uuid.UUID(people[0].ClusterId) != source {
			t.Fatalf("/people = %+v, want exactly Dad on the surviving source", people)
		}
		if uuid.UUID(people[0].RepresentativeFaceId) != h1.id || people[0].PhotoCount != 1 {
			t.Errorf("source state = %+v, want representative %s with faceCount 1", people[0], h1.id)
		}
	})

	t.Run("DetachedFaceStillListsOnItsAsset", func(t *testing.T) {
		faces := env.getAssetFaces(t, env.cookie, asset)
		if len(faces) != 2 {
			t.Fatalf("asset faces = %d, want both (the row survives the detach)", len(faces))
		}
		// Strongest first: h1 (0.90) leads, the detached h2
		// follows carrying no cluster and no person to label.
		if uuid.UUID(faces[0].Id) != h1.id || faces[0].ClusterId == nil || *faces[0].ClusterId != source {
			t.Errorf("faces[0] = %+v, want %s on cluster %s", faces[0], h1.id, source)
		}
		if uuid.UUID(faces[1].Id) != h2.id || faces[1].ClusterId != nil || faces[1].Label != nil {
			t.Errorf("faces[1] = %+v, want the detached %s with no cluster and no label", faces[1], h2.id)
		}
	})

	t.Run("DetachedFaceCanSeedANewPerson", func(t *testing.T) {
		// The roundtrip: a released face is exactly the
		// clusterless state the creation accepts.
		rec := env.postFaceCluster(t, env.cookie, h2.id)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST cluster of a detached face status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var created apigen.ClusterFaceResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode cluster response %q: %v", rec.Body.String(), err)
		}
		env.requireSoleMemberAndRepresentative(t, uuid.UUID(created.ClusterId), h2.id)
	})

	t.Run("EmptyingDetachDissolvesWithTheLabelCascade", func(t *testing.T) {
		// h1 is the source's last member now (h2 left).
		rec := env.deleteFace(t, env.cookie, h1.id)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE last member status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE id = $1 AND cluster_id IS NULL", pgUUID(h1.id)); got != 1 {
			t.Errorf("the last detached face holds a cluster still: rows with NULL = %d, want 1", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(source)); got != 0 {
			t.Errorf("emptied source rows = %d, want 0 (dissolved)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE cluster_id = $1", pgUUID(source)); got != 0 {
			t.Errorf("labels on the emptied source = %d, want 0 (cascaded)", got)
		}
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Errorf("/people = %+v after the dissolve, want empty (Dad died with the cluster)", people)
		}
	})

	t.Run("InvisibleConstellationsNotFound", func(t *testing.T) {
		// Grace's face on her own asset, a face on the caller's
		// trashed asset, and a clusterless face: a face part of
		// no person has no membership to release, the same 404
		// the move answers.
		graceFace := fixtureWriteFaceID("84")
		env.seedWriteCluster(t, fixtureWriteAssetID("82"), graceID,
			time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), fixtureWriteClusterID("82"),
			writeFaceSpec{graceFace, 0.55})
		trashed := fixtureWriteFaceID("83")
		env.insertOwnedAsset(t, fixtureWriteAssetID("83"), adminID,
			time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
		env.insertFace(t, trashed, fixtureWriteAssetID("83"), 0.40)
		clusterless := fixtureWriteFaceID("85")
		env.insertFace(t, clusterless, asset, 0.35)

		probes := []struct {
			what string
			face uuid.UUID
		}{
			{"unknown face", fixtureWriteFaceID("ff")},
			{"foreign-asset face", graceFace},
			{"trashed-asset face", trashed},
			{"clusterless face", clusterless},
		}
		var bodies []string
		for _, probe := range probes {
			rec := env.deleteFace(t, env.cookie, probe.face)
			requirePersonNotFound(t, rec, probe.what)
			bodies = append(bodies, rec.Body.String())
		}
		for _, body := range bodies[1:] {
			if body != bodies[0] {
				t.Errorf("404 bodies differ across the refusal matrix: %q vs %q", body, bodies[0])
			}
		}

		// Nothing was written by any rejected request: the
		// clusterless face still holds no cluster, grace's face
		// still sits on grace's cluster.
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE id = $1 AND cluster_id IS NULL", pgUUID(clusterless)); got != 1 {
			t.Errorf("clusterless face rows with NULL = %d, want 1 (no side effect)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM faces WHERE id = $1 AND cluster_id = $2", pgUUID(graceFace), pgUUID(fixtureWriteClusterID("82"))); got != 1 {
			t.Errorf("grace's face moved = %d rows, want 1 (no side effect)", got)
		}
	})

	t.Run("EventsCarryTheLeftCluster", func(t *testing.T) {
		detached := busEventsOf(t, env.bus, "person.face_detached")
		// One per committed release: h2, then h1 -- both left
		// the same source, whose person surface shrank each
		// time (until it died).
		if len(detached) != 2 {
			t.Fatalf("person.face_detached events = %d, want 2 (one per release)", len(detached))
		}
		requirePayloadIDs(t, detached[0], "first detach", map[string]string{
			"faceId":    h2.id.String(),
			"clusterId": source.String(),
			"userId":    adminID.String(),
		})
		requirePayloadIDs(t, detached[1], "emptying detach", map[string]string{
			"faceId":    h1.id.String(),
			"clusterId": source.String(),
			"userId":    adminID.String(),
		})
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.deleteFace(t, nil, fixtureWriteFaceID("ff"))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated DELETE face status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
