// People surface tests: the named region, the unnamed
// region's keyset pagination and the cluster detail, all with the
// per-user-state discipline under load -- clusters are global, labels are private
// per user, and every face count is honest for the calling user's
// visible assets. Fixtures insert models, faces, clusters, labels
// and assets directly in the database (the write endpoints have
// their own suite; the read side must be honest about rows no API can
// produce yet, exactly like the timeline fixtures before them).
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// peopleModelID is the detect model every people fixture face
// references: faces.model_id is NOT NULL, so the fixtures need one
// models row before any face row.
const peopleModelID = "people-detect-fixture"

// fixturePeopleAssetID builds a deterministic fixture asset id from a
// two-character hex suffix, in a namespace disjoint from the timeline
// (00..) and album (10..) fixture ids.
func fixturePeopleAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("40000000-0000-0000-0000-0000000000" + suffix)
}

// fixturePeopleClusterID builds a deterministic fixture cluster id,
// in a namespace disjoint from every other fixture family.
func fixturePeopleClusterID(suffix string) uuid.UUID {
	return uuid.MustParse("20000000-0000-0000-0000-0000000000" + suffix)
}

// fixturePeopleFaceID builds a deterministic fixture face id, in a
// namespace disjoint from every other fixture family.
func fixturePeopleFaceID(suffix string) uuid.UUID {
	return uuid.MustParse("30000000-0000-0000-0000-0000000000" + suffix)
}

// bulkPeopleAssetID builds the i-th id of the bulk pagination
// fixtures, in a namespace disjoint from every hand-picked fixture id.
func bulkPeopleAssetID(i int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("40000000-0000-0000-0002-%012x", i))
}

// bulkPeopleClusterID builds the i-th bulk cluster id.
func bulkPeopleClusterID(i int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("20000000-0000-0000-0002-%012x", i))
}

// bulkPeopleFaceID builds the i-th bulk face id.
func bulkPeopleFaceID(i int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("30000000-0000-0000-0002-%012x", i))
}

// insertPeopleModel seeds the fixture detect model the face rows
// reference. The model stays inactive: these tests read the people
// surface, which filters no rows by model state.
func (e *assetEnv) insertPeopleModel(t *testing.T) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO models (id, slot, version, source, file_path)
		VALUES ($1, 'face_detect', '1.0.0', 'builtin', '/models/people-detect.onnx')`,
		peopleModelID); err != nil {
		t.Fatalf("insert fixture model: %v", err)
	}
}

// insertFace inserts one fixture face row without a cluster
// assignment (the circular faces/face_clusters foreign keys force the
// cluster wiring to run after both rows exist).
func (e *assetEnv) insertFace(t *testing.T, id, assetID uuid.UUID, score float32) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id)
		VALUES ($1, $2, 100, 150, 200, 250, $3, $4)`,
		pgUUID(id), pgUUID(assetID), score, peopleModelID); err != nil {
		t.Fatalf("insert fixture face %s: %v", id, err)
	}
}

// insertFaceCluster inserts one cluster row whose representative is
// the given (already inserted) face.
func (e *assetEnv) insertFaceCluster(t *testing.T, clusterID, representativeFaceID uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO face_clusters (id, representative_face_id) VALUES ($1, $2)",
		pgUUID(clusterID), pgUUID(representativeFaceID)); err != nil {
		t.Fatalf("insert fixture cluster %s: %v", clusterID, err)
	}
}

// assignFaceCluster points one face at its cluster.
func (e *assetEnv) assignFaceCluster(t *testing.T, faceID, clusterID uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE faces SET cluster_id = $1 WHERE id = $2",
		pgUUID(clusterID), pgUUID(faceID)); err != nil {
		t.Fatalf("assign face %s to cluster %s: %v", faceID, clusterID, err)
	}
}

// seedPersonCluster wires one cluster over already-inserted faces:
// the first face listed becomes the representative, every face joins
// the cluster.
func (e *assetEnv) seedPersonCluster(t *testing.T, clusterID uuid.UUID, faceIDs ...uuid.UUID) {
	t.Helper()
	e.insertFaceCluster(t, clusterID, faceIDs[0])
	for _, faceID := range faceIDs {
		e.assignFaceCluster(t, faceID, clusterID)
	}
}

// labelPersonCluster inserts the calling user's private label row.
// The read-side fixtures seed the table directly (the write endpoint
// has its own tests); the name is the ONLY per-user state the people
// domain stores.
func (e *assetEnv) labelPersonCluster(t *testing.T, userID, clusterID uuid.UUID, name string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO user_face_labels (user_id, cluster_id, name) VALUES ($1, $2, $3)",
		pgUUID(userID), pgUUID(clusterID), name); err != nil {
		t.Fatalf("label cluster %s for %s: %v", clusterID, userID, err)
	}
}

// getPeopleRaw fires a GET against the people surface with the given
// cookie (nil sends it unauthenticated) and returns the recorder.
func (e *assetEnv) getPeopleRaw(t *testing.T, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// getPeopleList calls GET /people and decodes the 200 body.
func (e *assetEnv) getPeopleList(t *testing.T, cookie *http.Cookie) []apigen.PersonSummary {
	t.Helper()
	rec := e.getPeopleRaw(t, cookie, "/api/v1/people")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /people status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	var people []apigen.PersonSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &people); err != nil {
		t.Fatalf("decode people body %q: %v", rec.Body.String(), err)
	}
	return people
}

// getUnnamedPage calls GET /people/unnamed with the raw query string
// and decodes the 200 body.
func (e *assetEnv) getUnnamedPage(t *testing.T, cookie *http.Cookie, rawQuery string) apigen.UnnamedClustersResponse {
	t.Helper()
	target := "/api/v1/people/unnamed"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	rec := e.getPeopleRaw(t, cookie, target)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /people/unnamed?%s status = %d, want %d (body %q)",
			rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var page apigen.UnnamedClustersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode unnamed page body %q: %v", rec.Body.String(), err)
	}
	return page
}

// getClusterDetail calls GET /people/{clusterId} and decodes the 200
// body.
func (e *assetEnv) getClusterDetail(t *testing.T, cookie *http.Cookie, clusterID uuid.UUID) apigen.ClusterDetail {
	t.Helper()
	rec := e.getPeopleRaw(t, cookie, "/api/v1/people/"+clusterID.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /people/%s status = %d, want %d (body %q)",
			clusterID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.ClusterDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode cluster detail body %q: %v", rec.Body.String(), err)
	}
	return detail
}

// unnamedClusterIDs collects the ids of a page's items in order.
func unnamedClusterIDs(items []apigen.UnnamedClusterSummary) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = uuid.UUID(item.ClusterId)
	}
	return ids
}

// requireUnnamedOrder asserts the page carries exactly the given
// cluster ids, in order.
func requireUnnamedOrder(t *testing.T, items []apigen.UnnamedClusterSummary, want ...uuid.UUID) {
	t.Helper()
	got := unnamedClusterIDs(items)
	if len(got) != len(want) {
		t.Fatalf("unnamed page holds %d items, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unnamed item %d = %s, want %s (page %v)", i, got[i], want[i], got)
		}
	}
}

// TestPeopleEmptyDB walks the three endpoints against an empty
// instance: empty shapes everywhere, the detail 404s, and nothing
// answers without a session.
func TestPeopleEmptyDB(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)

	t.Run("NamedListIsEmpty", func(t *testing.T) {
		people := env.getPeopleList(t, env.cookie)
		if people == nil {
			t.Fatalf("GET /people body = null, want an (empty) array")
		}
		if len(people) != 0 {
			t.Fatalf("GET /people holds %d items, want 0", len(people))
		}
	})

	t.Run("UnnamedPageIsEmpty", func(t *testing.T) {
		page := env.getUnnamedPage(t, env.cookie, "")
		if page.Items == nil {
			t.Fatalf("unnamed items = null, want an (empty) array")
		}
		if len(page.Items) != 0 {
			t.Fatalf("unnamed page holds %d items, want 0", len(page.Items))
		}
		if page.NextCursor != nil {
			t.Error("nextCursor set on the empty unnamed page, want absent")
		}
	})

	t.Run("UnknownClusterNotFound", func(t *testing.T) {
		rec := env.getPeopleRaw(t, env.cookie,
			"/api/v1/people/"+fixturePeopleClusterID("ff").String())
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body %q)",
				rec.Code, http.StatusNotFound, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.PERSONNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.PERSONNOTFOUND)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		for _, target := range []string{
			"/api/v1/people",
			"/api/v1/people/unnamed",
			"/api/v1/people/" + fixturePeopleClusterID("01").String(),
		} {
			rec := env.getPeopleRaw(t, nil, target)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET %s unauthenticated status = %d, want %d",
					target, rec.Code, http.StatusUnauthorized)
			}
		}
	})
}

// TestPeopleCallerScopedCounts seeds one global cluster whose faces
// sit on assets of two accounts (plus a trashed and a hidden asset of
// the caller) and pins the honest per-caller view: face counts, the
// detail's visible faces with their ordering and asset context, and
// the double-meaning 404.
func TestPeopleCallerScopedCounts(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "people-grace@example.com")
	env.insertPeopleModel(t)

	// The admin's assets: two live ones, one trashed, one archived.
	aA1 := fixturePeopleAssetID("01")
	aA2 := fixturePeopleAssetID("02")
	aTrash := fixturePeopleAssetID("03")
	aHidden := fixturePeopleAssetID("04")
	aB := fixturePeopleAssetID("05")
	env.insertOwnedAsset(t, aA1, adminID, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aA2, adminID, time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aTrash, adminID, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, aHidden, adminID, time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})
	env.insertOwnedAsset(t, aB, graceID, time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// One global cluster: the representative fA, a stronger face fB,
	// a score tie fC (higher id), Grace's own face fD, and faces on
	// the caller's trashed and archived assets.
	fA := fixturePeopleFaceID("01") // representative, on aA1, 0.90
	fB := fixturePeopleFaceID("02") // on aA2, 0.95
	fC := fixturePeopleFaceID("03") // on aA1, 0.90 (tie with fA, higher id)
	fD := fixturePeopleFaceID("04") // on aB, 0.99
	fE := fixturePeopleFaceID("05") // on aTrash, 0.97
	fF := fixturePeopleFaceID("06") // on aHidden, 0.96
	env.insertFace(t, fA, aA1, 0.90)
	env.insertFace(t, fB, aA2, 0.95)
	env.insertFace(t, fC, aA1, 0.90)
	env.insertFace(t, fD, aB, 0.99)
	env.insertFace(t, fE, aTrash, 0.97)
	env.insertFace(t, fF, aHidden, 0.96)
	cluster := fixturePeopleClusterID("01")
	env.seedPersonCluster(t, cluster, fA, fB, fC, fD, fE, fF)

	// A second cluster whose faces sit ONLY on Grace's assets: it is
	// invisible to the admin through every read.
	foreign := fixturePeopleClusterID("02")
	foreignFace := fixturePeopleFaceID("07")
	env.insertFace(t, foreignFace, aB, 0.9)
	env.seedPersonCluster(t, foreign, foreignFace)

	t.Run("UnnamedCountsAreCallerScoped", func(t *testing.T) {
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, cluster)
		item := page.Items[0]
		if item.FaceCount != 3 {
			t.Errorf("admin faceCount = %d, want 3 (the live faces on own assets only)", item.FaceCount)
		}
		// The region's count rides the page: the admin sees exactly
		// the one cluster (the foreign one is invisible to them).
		if page.TotalCount != 1 {
			t.Errorf("admin unnamed totalCount = %d, want 1", page.TotalCount)
		}
		// The card face is the best caller-visible live face
		// (the card-face rule): the strongest of the admin's three is fB
		// (0.95), not the cluster's stored representative.
		if uuid.UUID(item.RepresentativeFaceId) != fB {
			t.Errorf("representativeFaceId = %s, want the best visible %s", item.RepresentativeFaceId, fB)
		}
		wantThumbnail := "/api/v1/faces/" + fB.String() + "/thumbnail"
		if item.Thumbnail != wantThumbnail {
			t.Errorf("thumbnail = %q, want %q", item.Thumbnail, wantThumbnail)
		}

		gracePage := env.getUnnamedPage(t, graceCookie, "")
		// Grace sees both clusters through her own asset -- the shared
		// one via fD, the foreign one via its only face -- both
		// unlabeled for her, the 1/1 tie in cluster-id descending
		// order.
		requireUnnamedOrder(t, gracePage.Items, foreign, cluster)
		if gracePage.Items[1].FaceCount != 1 {
			t.Errorf("grace faceCount = %d, want 1", gracePage.Items[1].FaceCount)
		}
		if gracePage.TotalCount != 2 {
			t.Errorf("grace unnamed totalCount = %d, want 2 (both visible clusters)", gracePage.TotalCount)
		}
	})

	t.Run("NamedListStaysEmptyWithoutOwnLabel", func(t *testing.T) {
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Fatalf("admin /people holds %d items before any label, want 0", len(people))
		}
		if people := env.getPeopleList(t, graceCookie); len(people) != 0 {
			t.Fatalf("grace /people holds %d items before any label, want 0", len(people))
		}
	})

	t.Run("DetailShowsVisibleFacesOnly", func(t *testing.T) {
		detail := env.getClusterDetail(t, env.cookie, cluster)
		if uuid.UUID(detail.ClusterId) != cluster {
			t.Errorf("clusterId = %s, want %s", detail.ClusterId, cluster)
		}
		if detail.Label != nil {
			t.Errorf("label = %+v, want null before the caller labels the cluster", detail.Label)
		}
		// The photo count dedupes by asset: the admin's three visible
		// faces sit on two assets (fA and fC share aA1).
		if detail.PhotoCount != 2 {
			t.Errorf("detail photoCount = %d, want 2 (three faces on two assets)", detail.PhotoCount)
		}
		if detail.BirthDate != nil {
			t.Errorf("birthDate = %v, want null on a fresh cluster", detail.BirthDate)
		}
		if detail.AvatarFaceId != nil {
			t.Errorf("avatarFaceId = %v, want null on a fresh cluster", detail.AvatarFaceId)
		}
		// The admin sees fB (0.95) then the 0.90 tie pair in face id
		// order; the foreign face and the trashed/archived-asset faces
		// never appear.
		wantFaces := []uuid.UUID{fB, fA, fC}
		if len(detail.Faces) != len(wantFaces) {
			t.Fatalf("detail faces = %d, want %d", len(detail.Faces), len(wantFaces))
		}
		for i, want := range wantFaces {
			if uuid.UUID(detail.Faces[i].Id) != want {
				t.Errorf("faces[%d] = %s, want %s (all %v)", i, detail.Faces[i].Id, want, detail.Faces)
			}
		}
		first := detail.Faces[0]
		if uuid.UUID(first.AssetId) != aA2 {
			t.Errorf("faces[0].assetId = %s, want %s", first.AssetId, aA2)
		}
		if first.Score != 0.95 {
			t.Errorf("faces[0].score = %v, want 0.95", first.Score)
		}
		wantThumbnail := "/api/v1/faces/" + fB.String() + "/thumbnail"
		if first.Thumbnail != wantThumbnail {
			t.Errorf("faces[0].thumbnail = %q, want %q", first.Thumbnail, wantThumbnail)
		}
		// The asset context rides along with the face.
		if got := first.Asset.OriginalFilename; got != "fixture-"+aA2.String()+".jpg" {
			t.Errorf("faces[0].asset.originalFilename = %q, want the fixture name", got)
		}
		wantTaken := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
		if !first.Asset.TakenAt.Truncate(time.Microsecond).Equal(wantTaken) {
			t.Errorf("faces[0].asset.takenAt = %v, want %v", first.Asset.TakenAt, wantTaken)
		}

		// Grace's view of the same global cluster: only her own face.
		graceDetail := env.getClusterDetail(t, graceCookie, cluster)
		if len(graceDetail.Faces) != 1 || uuid.UUID(graceDetail.Faces[0].Id) != fD {
			t.Fatalf("grace detail faces = %v, want exactly [%s]", graceDetail.Faces, fD)
		}
	})

	t.Run("DetailNotFoundDoubleMeaning", func(t *testing.T) {
		// An unknown id and a cluster whose faces are all on another
		// account's assets answer the same byte-identical 404.
		for _, clusterID := range []uuid.UUID{
			fixturePeopleClusterID("ff"),
			foreign,
		} {
			rec := env.getPeopleRaw(t, env.cookie, "/api/v1/people/"+clusterID.String())
			if rec.Code != http.StatusNotFound {
				t.Fatalf("GET %s status = %d, want %d (body %q)",
					clusterID, rec.Code, http.StatusNotFound, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.PERSONNOTFOUND {
				t.Errorf("GET %s code = %q, want %q", clusterID, apiErr.Code, apigen.PERSONNOTFOUND)
			}
		}
	})
}

// TestPeopleB4LabelIsolation loads the per-user-state discipline: the caller's
// label moves a cluster into their named region and out of their
// unnamed region -- and does nothing to the other account's view.
func TestPeopleB4LabelIsolation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "b4-grace@example.com")
	env.insertPeopleModel(t)

	// The shared cluster carries a face on each account's asset: one
	// global person both callers see through their own photos.
	assetA := fixturePeopleAssetID("01")
	env.insertOwnedAsset(t, assetA, adminID, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	faceA := fixturePeopleFaceID("01")
	env.insertFace(t, faceA, assetA, 0.9)
	graceAsset := fixturePeopleAssetID("03")
	env.insertOwnedAsset(t, graceAsset, graceID, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	faceB := fixturePeopleFaceID("03")
	env.insertFace(t, faceB, graceAsset, 0.8)
	cluster := fixturePeopleClusterID("01")
	env.seedPersonCluster(t, cluster, faceA, faceB)

	// A second cluster with its only face on a Grace asset: the admin
	// labels it to pin that labeling an invisible cluster never
	// enters the named list.
	foreignFace := fixturePeopleFaceID("02")
	env.insertFace(t, foreignFace, graceAsset, 0.9)
	foreign := fixturePeopleClusterID("02")
	env.seedPersonCluster(t, foreign, foreignFace)

	env.labelPersonCluster(t, adminID, cluster, "Ada")
	env.labelPersonCluster(t, adminID, foreign, "Invisible")

	t.Run("LabelMovesClusterIntoNamedRegion", func(t *testing.T) {
		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("admin /people holds %d items, want 1 (the visible labeled cluster only; got %v)",
				len(people), people)
		}
		person := people[0]
		if uuid.UUID(person.ClusterId) != cluster {
			t.Errorf("clusterId = %s, want %s", person.ClusterId, cluster)
		}
		if person.Name != "Ada" {
			t.Errorf("name = %q, want the caller's own label", person.Name)
		}
		if person.PhotoCount != 1 {
			t.Errorf("photoCount = %d, want 1 (one visible photo)", person.PhotoCount)
		}
	})

	t.Run("LabelMovesClusterOutOfOwnUnnamedRegionOnly", func(t *testing.T) {
		// The admin no longer sees either cluster as unnamed...
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items)
		// ...while Grace still sees both in HER unnamed region: the
		// admin's labels are private and never move them.
		gracePage := env.getUnnamedPage(t, graceCookie, "")
		requireUnnamedOrder(t, gracePage.Items, foreign, cluster)
		// And Grace's named region stays empty.
		if people := env.getPeopleList(t, graceCookie); len(people) != 0 {
			t.Fatalf("grace /people holds %d items, want 0", len(people))
		}
	})

	t.Run("EntityAttributesRideTheUnnamedState", func(t *testing.T) {
		// The person entity's attributes do not depend on a name: the
		// cluster the admin labeled keeps them, and Grace's unnamed
		// view of the SAME cluster reads them too (one value per
		// person, not per caller).
		if _, err := env.pool.Exec(context.Background(),
			"UPDATE face_clusters SET birth_date = '1992-03-15' WHERE id = $1",
			pgUUID(cluster)); err != nil {
			t.Fatalf("seed the fixture birth date: %v", err)
		}
		aDetail := env.getClusterDetail(t, env.cookie, cluster)
		if aDetail.BirthDate == nil || aDetail.BirthDate.Format("2006-01-02") != "1992-03-15" {
			t.Errorf("admin birthDate = %v, want 1992-03-15", aDetail.BirthDate)
		}
		graceDetail := env.getClusterDetail(t, graceCookie, cluster)
		if graceDetail.BirthDate == nil || graceDetail.BirthDate.Format("2006-01-02") != "1992-03-15" {
			t.Errorf("grace birthDate = %v, want the same 1992-03-15 (the entity's own attribute)", graceDetail.BirthDate)
		}
		if graceDetail.Label != nil {
			t.Errorf("grace label = %+v, want null (the admin's name stays private)", graceDetail.Label)
		}
	})
}

// TestPeopleNamedOrdering pins the named list's order: name
// case-insensitively, cluster id as the tiebreak -- a byte-order sort
// would answer differently.
func TestPeopleNamedOrdering(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPeopleModel(t)

	// "Avocado" sorts before "Zebra" and "apple" case-insensitively
	// (byte order would answer Avocado, Zebra, apple, ivy, Ivy); the
	// two "ivy" labels tie and fall back to the cluster id.
	seeds := []struct {
		suffix string
		name   string
	}{
		{"01", "Zebra"},
		{"02", "apple"},
		{"03", "Avocado"},
		{"04", "ivy"},
		{"05", "Ivy"},
	}
	for _, seed := range seeds {
		clusterID := fixturePeopleClusterID(seed.suffix)
		assetID := fixturePeopleAssetID(seed.suffix)
		faceID := fixturePeopleFaceID(seed.suffix)
		env.insertOwnedAsset(t, assetID, adminID, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
		env.insertFace(t, faceID, assetID, 0.9)
		env.seedPersonCluster(t, clusterID, faceID)
		env.labelPersonCluster(t, adminID, clusterID, seed.name)
	}

	people := env.getPeopleList(t, env.cookie)
	wantOrder := []struct {
		cluster uuid.UUID
		name    string
	}{
		{fixturePeopleClusterID("02"), "apple"},
		{fixturePeopleClusterID("03"), "Avocado"},
		{fixturePeopleClusterID("04"), "ivy"},
		{fixturePeopleClusterID("05"), "Ivy"},
		{fixturePeopleClusterID("01"), "Zebra"},
	}
	if len(people) != len(wantOrder) {
		t.Fatalf("/people holds %d items, want %d (%v)", len(people), len(wantOrder), people)
	}
	for i, want := range wantOrder {
		if uuid.UUID(people[i].ClusterId) != want.cluster {
			t.Errorf("people[%d].clusterId = %s, want %s", i, people[i].ClusterId, want.cluster)
		}
		if people[i].Name != want.name {
			t.Errorf("people[%d].name = %q, want %q", i, people[i].Name, want.name)
		}
	}
}

// TestPeopleUnnamedPagination walks the keyset pagination of the
// unnamed region: the default page size of 60, the disjoint union of
// pages under the faceCount-descending order, and the cluster-id
// tiebreak across a page boundary.
func TestPeopleUnnamedPagination(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPeopleModel(t)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// 61 singleton clusters: a page without an explicit limit must
	// carry exactly the default of 60 plus a follow-up cursor.
	for i := 0; i < 61; i++ {
		assetID := bulkPeopleAssetID(i)
		env.insertOwnedAsset(t, assetID, adminID, base.Add(time.Duration(i)*time.Minute), timelineAssetOpts{})
		faceID := bulkPeopleFaceID(i)
		env.insertFace(t, faceID, assetID, 0.9)
		env.seedPersonCluster(t, bulkPeopleClusterID(i), faceID)
	}

	defaultPage := env.getUnnamedPage(t, env.cookie, "")
	if len(defaultPage.Items) != 60 {
		t.Fatalf("default page holds %d items, want the contract default of 60", len(defaultPage.Items))
	}
	if defaultPage.NextCursor == nil {
		t.Fatal("nextCursor absent on a full default page with more rows after it, want present")
	}

	// Five hand-picked clusters add the tie structure: one with 3
	// faces, a 2/2 tie pair (ids 22 < 23, so the pair walks 23, 22)
	// spanning a page boundary, and a 1/1 tie pair (ids 24 < 25). The
	// fixture-namespace ids sort below the bulk namespace, so the
	// full expected order over all 66 clusters is:
	//
	//	c21 (3), c23 (2), c22 (2),
	//	bulk 60..0 (1 each, id descending),
	//	c25 (1), c24 (1).
	tieAssetBase := 200
	tie := map[string]int{"21": 3, "22": 2, "23": 2, "24": 1, "25": 1}
	faceSeq := 0
	for _, suffix := range []string{"21", "22", "23", "24", "25"} {
		clusterID := fixturePeopleClusterID(suffix)
		count := tie[suffix]
		faces := make([]uuid.UUID, count)
		for j := 0; j < count; j++ {
			assetID := bulkPeopleAssetID(tieAssetBase + faceSeq)
			env.insertOwnedAsset(t, assetID, adminID, base, timelineAssetOpts{})
			faceID := fixturePeopleFaceID(fmt.Sprintf("%02d", faceSeq))
			env.insertFace(t, faceID, assetID, 0.9)
			faces[j] = faceID
			faceSeq++
		}
		env.seedPersonCluster(t, clusterID, faces...)
	}

	wantOrder := []uuid.UUID{fixturePeopleClusterID("21"), fixturePeopleClusterID("23"), fixturePeopleClusterID("22")}
	for i := 60; i >= 0; i-- {
		wantOrder = append(wantOrder, bulkPeopleClusterID(i))
	}
	wantOrder = append(wantOrder, fixturePeopleClusterID("25"), fixturePeopleClusterID("24"))

	seen := make(map[uuid.UUID]bool)
	var walked []uuid.UUID
	cursor := ""
	pages := 0
	for {
		rawQuery := "limit=2"
		if cursor != "" {
			rawQuery += "&cursor=" + cursor
		}
		page := env.getUnnamedPage(t, env.cookie, rawQuery)
		for _, id := range unnamedClusterIDs(page.Items) {
			if seen[id] {
				t.Fatalf("cluster %s appeared on two pages (cursor %q)", id, cursor)
			}
			seen[id] = true
			walked = append(walked, id)
		}
		// Every page carries the whole region's count, whatever the
		// page's own size: the count is not the page walk's answer.
		if page.TotalCount != len(wantOrder) {
			t.Fatalf("page %d totalCount = %d, want the whole region's %d", pages+1, page.TotalCount, len(wantOrder))
		}
		pages++
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	if pages != 33 {
		t.Fatalf("walked %d pages, want 33 (66 clusters at 2 per page)", pages)
	}
	if len(walked) != len(wantOrder) {
		t.Fatalf("walked %d clusters, want %d", len(walked), len(wantOrder))
	}
	for i, want := range wantOrder {
		if walked[i] != want {
			t.Fatalf("walked order[%d] = %s, want %s (walked %v)", i, walked[i], want, walked)
		}
	}

	t.Run("RejectsInvalidPageParams", func(t *testing.T) {
		for _, raw := range []string{"limit=0", "limit=201", "limit=abc", "cursor=not-a-cursor"} {
			rec := env.getPeopleRaw(t, env.cookie, "/api/v1/people/unnamed?"+raw)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("GET ?%s status = %d, want %d (body %q)",
					raw, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("GET ?%s code = %q, want %q", raw, apiErr.Code, apigen.VALIDATIONFAILED)
			}
		}

		// The cap boundary itself answers, it does not reject.
		page := env.getUnnamedPage(t, env.cookie, "limit=200")
		if len(page.Items) != 66 {
			t.Fatalf("limit=200 page holds %d items, want all 66", len(page.Items))
		}
		if page.NextCursor != nil {
			t.Error("nextCursor set on the final page, want absent")
		}
	})
}
