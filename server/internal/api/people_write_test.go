// People surface write tests (M3b Task 2): the label, hidden, merge
// and face-move endpoints, with the pinned semantics under load --
// labels are per-caller upserts (B4), duplicate names are allowed,
// hiding rides on the label row, and every structural cluster change
// re-points representatives and dissolves vacated clusters (source
// labels die with them -- the honest documented cost). Fixtures
// insert models, faces, clusters and assets directly in the database
// and drive every write through the API, exactly like the read-side
// tests before them.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	e.insertTimelineAsset(t, assetID, ownerID, taken, timelineAssetOpts{})
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

// putPersonHidden fires PUT /people/{clusterId}/hidden with the raw
// body.
func (e *assetEnv) putPersonHidden(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writePeopleRaw(t, cookie, http.MethodPut, "/api/v1/people/"+clusterID.String()+"/hidden", body)
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
		if state.IsHidden {
			t.Error("isHidden = true on the first label, want false")
		}

		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("/people holds %d items, want the one named cluster", len(people))
		}
		if people[0].Name != "Ada" || uuid.UUID(people[0].ClusterId) != c1 || people[0].IsHidden {
			t.Errorf("/people entry = %+v, want Ada on %s unhidden", people[0], c1)
		}
		if people[0].FaceCount != 2 {
			t.Errorf("faceCount = %d, want 2", people[0].FaceCount)
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
			{http.MethodPut, "/api/v1/people/" + c1.String() + "/hidden", `{"isHidden":true}`},
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

// TestPeopleLabelB4Isolation is the mandated Review Focus 1 case:
// labels are per-caller rows. A's naming never moves the cluster for
// B; both accounts label the same cluster independently; B's hidden
// switch never touches A's row.
func TestPeopleLabelB4Isolation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "b4-write-grace@example.com")
	env.insertPeopleModel(t)

	// One global cluster both accounts see through their own asset.
	shared := fixtureWriteClusterID("11")
	assetA := fixtureWriteAssetID("11")
	assetB := fixtureWriteAssetID("12")
	env.insertTimelineAsset(t, assetA, adminID, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, assetB, graceID, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
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
		// list -- another account's label never moves it (B4).
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

	t.Run("BHideDoesNotAffectA", func(t *testing.T) {
		rec := env.putPersonHidden(t, graceCookie, shared, `{"isHidden":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("B's PUT hidden status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		// B's badge flips on B's row only; the person stays listed
		// (badge semantics, not a filter).
		bPeople := env.getPeopleList(t, graceCookie)
		if len(bPeople) != 1 || !bPeople[0].IsHidden {
			t.Errorf("B's /people = %+v, want Bob listed carrying isHidden true", bPeople)
		}
		aPeople := env.getPeopleList(t, env.cookie)
		if len(aPeople) != 1 || aPeople[0].IsHidden {
			t.Errorf("A's /people = %+v, want Alice listed unhidden (B's switch is private)", aPeople)
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

// TestPeopleMergeThreeStates is the mandated Review Focus 2 case:
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
		if people[0].FaceCount != 4 {
			t.Errorf("faceCount = %d, want the merged 4", people[0].FaceCount)
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
		// and never rides the shared stream (B4).
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
// when the move empties it), and every invisible constellation
// answers the anti-enumeration 404.
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
		if people[0].FaceCount != 1 {
			t.Errorf("source faceCount = %d, want 1", people[0].FaceCount)
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
		// A face the clustering has not assigned to any cluster is
		// part of no person.
		unclustered := fixtureWriteFaceID("46")
		env.insertFace(t, unclustered, fixtureWriteAssetID("41"), 0.40)

		for _, probe := range []struct {
			what string
			face uuid.UUID
			body string
		}{
			{"unknown face", fixtureWriteFaceID("ff"), moveBody},
			{"foreign-asset face", fixtureWriteFaceID("44"), moveBody},
			{"unclustered face", unclustered, moveBody},
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

// TestPeopleHide walks the hidden switch: hiding rides on the caller's
// label row, so an unlabeled cluster answers the pinned
// person_hide_requires_label key, a labeled cluster toggles, and a
// rename keeps the hidden state.
func TestPeopleHide(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPeopleModel(t)

	c1 := fixtureWriteClusterID("51")
	env.seedWriteCluster(t, fixtureWriteAssetID("51"), adminID,
		time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), c1,
		writeFaceSpec{fixtureWriteFaceID("51"), 0.90},
		writeFaceSpec{fixtureWriteFaceID("52"), 0.80})

	t.Run("HideUnlabeledClusterRejected", func(t *testing.T) {
		rec := env.putPersonHidden(t, env.cookie, c1, `{"isHidden":true}`)
		apiErr, details := requireValidationError(t, rec, "hide unlabeled cluster")
		if got, _ := details["key"].(string); got != "person_hide_requires_label" {
			t.Errorf("details.key = %v, want person_hide_requires_label (details %#v)", got, details)
		}
		if apiErr.Message == nil || !strings.Contains(*apiErr.Message, "label") {
			t.Errorf("message = %v, want it to name the missing label", apiErr.Message)
		}

		// The unhide arm of an unlabeled cluster answers the same
		// key: the label row is the missing piece either way.
		rec = env.putPersonHidden(t, env.cookie, c1, `{"isHidden":false}`)
		_, details = requireValidationError(t, rec, "unhide unlabeled cluster")
		if got, _ := details["key"].(string); got != "person_hide_requires_label" {
			t.Errorf("details.key = %v, want person_hide_requires_label (details %#v)", got, details)
		}
	})

	t.Run("HideLabeledClusterToggles", func(t *testing.T) {
		rec := env.putPersonLabel(t, env.cookie, c1, `{"name":"Ada"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("label: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec = env.putPersonHidden(t, env.cookie, c1, `{"isHidden":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("hide: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if state := decodePersonLabel(t, rec); !state.IsHidden || state.Name != "Ada" {
			t.Errorf("hide state = %+v, want Ada hidden", state)
		}
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 || !people[0].IsHidden {
			t.Errorf("/people = %+v, want Ada listed carrying the badge", people)
		}

		rec = env.putPersonHidden(t, env.cookie, c1, `{"isHidden":false}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("unhide: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if state := decodePersonLabel(t, rec); state.IsHidden {
			t.Errorf("unhide state = %+v, want isHidden false", state)
		}

		// The pinned event carries ids only: the switch value is the
		// caller's private state and never rides the shared stream.
		hiddenEvents := busEventsOf(t, env.bus, "person.hidden")
		if len(hiddenEvents) != 2 {
			t.Fatalf("person.hidden events = %d, want 2 (hide and unhide)", len(hiddenEvents))
		}
		for _, payload := range hiddenEvents {
			if payload["isHidden"] != "" {
				t.Errorf("person.hidden payload carries the private switch %q -- it must not leave the account", payload["isHidden"])
			}
		}
		if hiddenEvents[0]["clusterId"] != c1.String() || hiddenEvents[0]["userId"] != adminID.String() {
			t.Errorf("hide payload ids = %v, want cluster %s by %s", hiddenEvents[0], c1, adminID)
		}
	})

	t.Run("RenameKeepsHiddenState", func(t *testing.T) {
		if rec := env.putPersonHidden(t, env.cookie, c1, `{"isHidden":true}`); rec.Code != http.StatusOK {
			t.Fatalf("hide: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		rec := env.putPersonLabel(t, env.cookie, c1, `{"name":"Ada Lovelace"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("rename: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if state := decodePersonLabel(t, rec); !state.IsHidden || state.Name != "Ada Lovelace" {
			t.Errorf("state after rename = %+v, want the new name with the hidden switch intact", state)
		}
	})

	t.Run("MalformedBodyRejected", func(t *testing.T) {
		rec := env.putPersonHidden(t, env.cookie, c1, `not json`)
		_, _ = requireValidationError(t, rec, "malformed hidden body")
	})

	t.Run("UnknownClusterNotFound", func(t *testing.T) {
		rec := env.putPersonHidden(t, env.cookie, fixtureWriteClusterID("ff"), `{"isHidden":true}`)
		requirePersonNotFound(t, rec, "hide unknown cluster")
	})
}

// TestPeopleWriteDeleteAssetCascades is the Review Focus 3 linkage at
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
