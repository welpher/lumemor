// Shared-visibility tests (M3c Task 4): the caller-visible-asset
// predicate's switchover, exercised through every read face it now
// gates -- the people surface (sharee counts and the best-visible
// card crop), the asset detail and media serving, and the favorites
// pair (the batch favorite's resolution and the view's union
// filter) -- plus the owner-only write faces' unchanged shapes
// against a sharee. The fixtures model the production invariant the
// union builds on: every owned asset sits in its owner's sync album.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// visibilityAssetOpts is the fixture alias of the timeline asset
// options, named for this file's intent.
type visibilityAssetOpts = timelineAssetOpts

// requireAssetStatus asserts the asset face answered the given
// status and code.
func requireAssetStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, code apigen.ErrorCode) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, status, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != code {
		t.Errorf("code = %q, want %q", apiErr.Code, code)
	}
}

// TestSharedVisibilityPeople pins the people surface under sharing:
// a sharee's people view counts (and details, and labels) the faces
// on shared-album assets, labels stay per-user (B4), and a revoked
// share retracts the faces. The structural faces stay owner-only.
func TestSharedVisibilityPeople(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "people-shared-grace@example.com")
	env.insertPeopleModel(t)

	// The admin's album shared to Grace, carrying a1 (live) and
	// aTrash (trashed -- live filters still apply through the
	// union). a3 stays unshared.
	sharedAlbum := fixtureAlbumID("d1")
	env.insertTimelineAlbum(t, sharedAlbum, adminID, "virtual")
	env.shareTimelineAlbum(t, sharedAlbum, graceID, adminID)
	a1 := fixturePeopleAssetID("d1")
	aTrash := fixturePeopleAssetID("d2")
	a3 := fixturePeopleAssetID("d3")
	env.insertOwnedAsset(t, a1, adminID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})
	env.insertOwnedAsset(t, aTrash, adminID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{deleted: true})
	env.insertOwnedAsset(t, a3, adminID, time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})
	env.addTimelineItem(t, sharedAlbum, a1, adminID)
	env.addTimelineItem(t, sharedAlbum, aTrash, adminID)

	// Grace's own asset with her face in the same global cluster
	// (B4: one person across accounts), plus a cluster whose faces
	// sit ONLY on the admin's shared album (the retraction arm).
	g1 := fixturePeopleAssetID("d4")
	env.insertOwnedAsset(t, g1, graceID, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})

	fa1 := fixturePeopleFaceID("d1")     // on a1 (shared, live), 0.91
	faTrash := fixturePeopleFaceID("d2") // on aTrash, 0.99
	fa3 := fixturePeopleFaceID("d3")     // on a3 (unshared), 0.95
	fg1 := fixturePeopleFaceID("d4")     // on g1 (grace's own), 0.80
	env.insertFace(t, fa1, a1, 0.91)
	env.insertFace(t, faTrash, aTrash, 0.99)
	env.insertFace(t, fa3, a3, 0.95)
	env.insertFace(t, fg1, g1, 0.80)
	cluster := fixturePeopleClusterID("d1")
	env.seedPersonCluster(t, cluster, faTrash, fa1, fa3, fg1)

	sharedOnlyFace := fixturePeopleFaceID("d5")
	env.insertFace(t, sharedOnlyFace, a1, 0.90)
	sharedOnly := fixturePeopleClusterID("d2")
	env.seedPersonCluster(t, sharedOnly, sharedOnlyFace)

	t.Run("ShareeCountsIncludeSharedFaces", func(t *testing.T) {
		// The admin's unnamed region: both clusters, counts over
		// their own live assets only (the trashed face never
		// counts, whatever the album).
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, cluster, sharedOnly)
		if page.Items[0].FaceCount != 2 { // fa1 + fa3
			t.Errorf("admin cluster faceCount = %d, want 2", page.Items[0].FaceCount)
		}

		// Grace's unnamed region before labeling: the same global
		// clusters through the shared album and her own asset --
		// the union arms, counted honestly for her.
		gracePage := env.getUnnamedPage(t, graceCookie, "")
		requireUnnamedOrder(t, gracePage.Items, cluster, sharedOnly)
		if gracePage.Items[0].FaceCount != 2 { // fa1 + fg1
			t.Errorf("grace cluster faceCount = %d, want 2 (shared + own)", gracePage.Items[0].FaceCount)
		}

		// The detail answers the same union: Grace sees the
		// shared-album face and her own, never the unshared or
		// trashed ones.
		detail := env.getClusterDetail(t, graceCookie, cluster)
		wantFaces := []uuid.UUID{fa1, fg1} // 0.91 then 0.80
		if len(detail.Faces) != len(wantFaces) {
			t.Fatalf("grace detail faces = %v, want %v", detail.Faces, wantFaces)
		}
		for i, want := range wantFaces {
			if uuid.UUID(detail.Faces[i].Id) != want {
				t.Errorf("grace detail faces[%d] = %s, want %s", i, detail.Faces[i].Id, want)
			}
		}
	})

	t.Run("LabelsStayPerUser", func(t *testing.T) {
		// Grace labels the shared cluster: her row, her view.
		rec := env.writeAlbumRaw(t, graceCookie, http.MethodPut,
			"/api/v1/people/"+cluster.String()+"/label", `{"name":"Grandma"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT label as sharee status = %d, want %d (body %q)",
				rec.Code, http.StatusOK, rec.Body.String())
		}
		people := env.getPeopleList(t, graceCookie)
		if len(people) != 1 || uuid.UUID(people[0].ClusterId) != cluster || people[0].Name != "Grandma" {
			t.Fatalf("grace named people = %v, want exactly the labeled shared cluster", people)
		}
		if people[0].FaceCount != 2 {
			t.Errorf("grace labeled faceCount = %d, want 2", people[0].FaceCount)
		}
		// The admin's view is untouched: the cluster stays unnamed
		// for them, no label leaked across accounts (B4).
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, cluster, sharedOnly)
		if people := env.getPeopleList(t, env.cookie); len(people) != 0 {
			t.Errorf("admin named people = %v, want none (the sharee's label is private)", people)
		}
	})

	t.Run("StructuralWritesStayOwnerOnly", func(t *testing.T) {
		// A sharee cannot merge what they see through a share:
		// cluster curation is the owner's alone, and the refusal is
		// the people surface's single 404 shape -- fired while the
		// share is still live, so the clusters are genuinely
		// visible to the caller the gate refuses.
		body := `{"targetClusterId":"` + cluster.String() + `","sourceClusterIds":["` + sharedOnly.String() + `"]}`
		rec := env.writeAlbumRaw(t, graceCookie, http.MethodPost, "/api/v1/people/merge", body)
		requireAssetStatus(t, rec, http.StatusNotFound, apigen.PERSONNOTFOUND)
		// Nothing moved: the source cluster still holds its face.
		if count := env.queryInt(t,
			"SELECT count(*) FROM faces WHERE cluster_id = $1", pgUUID(sharedOnly)); count != 1 {
			t.Errorf("source cluster faces after the sharee's merge = %d, want 1 (untouched)", count)
		}
	})

	t.Run("RetractionRemovesSharedFaces", func(t *testing.T) {
		// The revoke removes Grace's only window onto the
		// shared-only cluster: it leaves her unnamed region and
		// its detail answers the double-meaning 404, while the
		// mixed cluster falls back to her own asset alone.
		env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
			pgUUID(sharedAlbum), pgUUID(graceID))

		people := env.getPeopleList(t, graceCookie)
		if len(people) != 1 || uuid.UUID(people[0].ClusterId) != cluster {
			t.Fatalf("grace named people after retraction = %v, want the labeled cluster (fallen back to her own face)", people)
		}
		if people[0].FaceCount != 1 { // fg1 alone
			t.Errorf("grace labeled faceCount after retraction = %d, want 1", people[0].FaceCount)
		}
		rec := env.getPeopleRaw(t, graceCookie, "/api/v1/people/"+sharedOnly.String())
		requireAssetStatus(t, rec, http.StatusNotFound, apigen.PERSONNOTFOUND)

		// The owner's view never moved.
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, cluster, sharedOnly)
	})
}

// TestPersonCardBestVisibleFace pins the M3b handoff: the person
// card's crop derives from the best CALLER-VISIBLE live face (score
// descending, the face id ascending as the tiebreak), never from the
// global representative -- a representative on a trashed, hidden or
// foreign asset renders the caller's best visible face instead of a
// dead crop URL.
func TestPersonCardBestVisibleFace(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "card-grace@example.com")
	env.insertPeopleModel(t)

	// The admin's album shared to Grace carrying the admin's
	// assets; a2 stays unshared and carries the strongest face of
	// the cluster (the global representative's home too).
	sharedAlbum := fixtureAlbumID("e1")
	env.insertTimelineAlbum(t, sharedAlbum, adminID, "virtual")
	env.shareTimelineAlbum(t, sharedAlbum, graceID, adminID)
	a1 := fixturePeopleAssetID("e1")
	a2 := fixturePeopleAssetID("e2")
	env.insertOwnedAsset(t, a1, adminID, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})
	env.insertOwnedAsset(t, a2, adminID, time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})
	env.addTimelineItem(t, sharedAlbum, a1, adminID)

	// Grace's own asset with the cluster's weakest face.
	g1 := fixturePeopleAssetID("e3")
	env.insertOwnedAsset(t, g1, graceID, time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC), visibilityAssetOpts{})

	// The cluster: the representative sits on the admin's UNSHARED
	// asset with the strongest score; the shared arm's faces are
	// weaker; Grace's own is the weakest.
	fRep := fixturePeopleFaceID("e1")       // on a2 (unshared), 0.99, the representative
	fShared := fixturePeopleFaceID("e2")    // on a1 (shared), 0.90
	fSharedTie := fixturePeopleFaceID("e3") // on a1 (shared), 0.90 (tie, higher id)
	fGrace := fixturePeopleFaceID("e4")     // on g1, 0.80
	env.insertFace(t, fRep, a2, 0.99)
	env.insertFace(t, fShared, a1, 0.90)
	env.insertFace(t, fSharedTie, a1, 0.90)
	env.insertFace(t, fGrace, g1, 0.80)
	cluster := fixturePeopleClusterID("e1")
	env.seedPersonCluster(t, cluster, fRep, fShared, fSharedTie, fGrace)

	t.Run("UnnamedCardUsesBestVisibleFace", func(t *testing.T) {
		// The admin sees every face: their best is the cluster's
		// strongest (0.99), which happens to also be the stored
		// representative -- the pin is that the card agrees with the
		// caller-visible best, and the admin's arms cover it.
		page := env.getUnnamedPage(t, env.cookie, "")
		requireUnnamedOrder(t, page.Items, cluster)
		if got := uuid.UUID(page.Items[0].RepresentativeFaceId); got != fRep {
			t.Errorf("admin card representativeFaceId = %s, want the best visible %s", got, fRep)
		}
		if want := "/api/v1/faces/" + fRep.String() + "/thumbnail"; page.Items[0].Thumbnail != want {
			t.Errorf("admin card thumbnail = %q, want %q", page.Items[0].Thumbnail, want)
		}

		// Grace sees the shared arm and her own face: her best is
		// the shared arm's 0.90 tie winner (score DESC, the face id
		// ascending -- the lower id first), NEVER the global
		// representative, whose unshared asset she cannot view.
		gracePage := env.getUnnamedPage(t, graceCookie, "")
		requireUnnamedOrder(t, gracePage.Items, cluster)
		if got := uuid.UUID(gracePage.Items[0].RepresentativeFaceId); got != fShared {
			t.Errorf("grace card representativeFaceId = %s, want the best visible %s", got, fShared)
		}
	})

	t.Run("NamedCardUsesBestVisibleFace", func(t *testing.T) {
		env.labelPersonCluster(t, adminID, cluster, "Ada", false)
		env.labelPersonCluster(t, graceID, cluster, "Grace", false)

		adminPeople := env.getPeopleList(t, env.cookie)
		if len(adminPeople) != 1 {
			t.Fatalf("admin named people = %v, want the labeled cluster", adminPeople)
		}
		if got := uuid.UUID(adminPeople[0].RepresentativeFaceId); got != fRep {
			t.Errorf("admin named card representativeFaceId = %s, want %s", got, fRep)
		}

		gracePeople := env.getPeopleList(t, graceCookie)
		if len(gracePeople) != 1 {
			t.Fatalf("grace named people = %v, want the labeled cluster", gracePeople)
		}
		if got := uuid.UUID(gracePeople[0].RepresentativeFaceId); got != fShared {
			t.Errorf("grace named card representativeFaceId = %s, want %s", got, fShared)
		}
	})

	t.Run("TrashedRepresentativeNeverCrops", func(t *testing.T) {
		// The shared arm's asset enters the trash: the card stays
		// honest over the caller's best LIVE face (the crop gate
		// 404s trashed assets, so a card pointing at a face on the
		// trashed asset would render a skeleton). The cluster is
		// labeled by now, so the named cards carry the assertion.
		env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(a1))

		people := env.getPeopleList(t, env.cookie)
		if len(people) != 1 {
			t.Fatalf("admin named people = %v, want the labeled cluster", people)
		}
		if got := uuid.UUID(people[0].RepresentativeFaceId); got != fRep {
			t.Errorf("card representativeFaceId after the trash = %s, want the best live visible %s", got, fRep)
		}
		// Grace's card falls back to her own face.
		gracePeople := env.getPeopleList(t, graceCookie)
		if len(gracePeople) != 1 {
			t.Fatalf("grace named people = %v, want the labeled cluster", gracePeople)
		}
		if got := uuid.UUID(gracePeople[0].RepresentativeFaceId); got != fGrace {
			t.Errorf("grace card representativeFaceId after the trash = %s, want her best remaining %s", got, fGrace)
		}
	})
}

// TestSharedVisibilityAssets pins the asset faces under the union
// gate: a sharee reads the detail, the original and the thumbnail of
// a shared album's member (the not-ready code proving the row
// resolved), the share's revocation retracts all of it with the
// single-shape 404, the owner's state filters never change (a
// trashed asset stays readable by its known id, whatever the
// caller), and the owner-only write faces keep their shapes for a
// sharee.
func TestSharedVisibilityAssets(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "assets-shared-grace@example.com")
	root := env.provisionMountRoot(t)

	// A real upload: the original's bytes exist, so the download
	// answers them, and the pipeline never ran (the thumbnail's
	// not-ready state is the gate's discriminating answer).
	uploadRec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "shared-visibility-1",
		"fileName":        "IMG_shared.jpg",
		"deviceCreatedAt": "2026-10-01T12:00:00Z",
	}, []assetUploadFile{{field: uploadPartPhoto, name: "IMG_shared.jpg", content: jpegFixture(t)}})
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d (body %q)", uploadRec.Code, http.StatusCreated, uploadRec.Body.String())
	}
	created := decodeAsset(t, uploadRec)
	shared := uuid.UUID(created.Id)

	album := env.createAlbum(t, env.cookie, "Shared visibility", sourceDir(t, root, "visibility"), nil)
	env.addTimelineItem(t, uuid.UUID(album.Id), shared, adminID)
	env.putAlbumAudience(t, env.cookie, uuid.UUID(album.Id), graceID)

	detailPath := "/api/v1/assets/" + shared.String()
	originalPath := detailPath + "/original"
	thumbPath := detailPath + "/thumbnail?size=tile"

	t.Run("ShareeReadsSharedAsset", func(t *testing.T) {
		rec := env.serveGet(t, detailPath, graceCookie, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("sharee detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var detail apigen.AssetDetail
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatalf("decode sharee detail %q: %v", rec.Body.String(), err)
		}
		if uuid.UUID(detail.OwnerId) != adminID {
			t.Errorf("detail ownerId = %s, want the album owner's %s", detail.OwnerId, adminID)
		}
		if detail.IsTrashed || detail.IsHidden {
			t.Errorf("detail isTrashed/isHidden = %v/%v, want live values", detail.IsTrashed, detail.IsHidden)
		}

		rec = env.serveGet(t, originalPath, graceCookie, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("sharee original status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if rec.Body.Len() == 0 {
			t.Error("sharee original body empty, want the uploaded bytes")
		}

		// The row resolved for the sharee: the derived file's
		// absence is ASSET_NOT_READY, not ASSET_NOT_FOUND -- the
		// two codes are the gate's honest read-out.
		rec = env.serveGet(t, thumbPath, graceCookie, nil)
		requireAssetStatus(t, rec, http.StatusNotFound, apigen.ASSETNOTREADY)
	})

	t.Run("TrashedSharedAssetStaysReadableAtKnownId", func(t *testing.T) {
		// The owner trashes the member: every listing drops it,
		// but the union gate carries no state filter -- the
		// membership row outlives the mark, and both the owner's
		// and the sharee's direct reads keep answering (the
		// contract's documented behavior for the owner, unchanged
		// by the switchover).
		env.batchAssets(t, env.cookie, "trash", shared)

		for name, cookie := range map[string]*http.Cookie{
			"owner": env.cookie, "sharee": graceCookie,
		} {
			rec := env.serveGet(t, detailPath, cookie, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s detail of the trashed member status = %d, want %d (body %q)",
					name, rec.Code, http.StatusOK, rec.Body.String())
			}
			var detail apigen.AssetDetail
			if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
				t.Fatalf("decode %s detail %q: %v", name, rec.Body.String(), err)
			}
			if !detail.IsTrashed {
				t.Errorf("%s detail isTrashed = false, want true", name)
			}
		}

		env.batchAssets(t, env.cookie, "restore", shared)
	})

	t.Run("RevocationRetractsTheReads", func(t *testing.T) {
		env.putAlbumAudience(t, env.cookie, uuid.UUID(album.Id))

		for _, path := range []string{detailPath, originalPath, thumbPath} {
			rec := env.serveGet(t, path, graceCookie, nil)
			requireAssetStatus(t, rec, http.StatusNotFound, apigen.ASSETNOTFOUND)
		}

		// The owner's reads keep answering after the revoke.
		if rec := env.serveGet(t, detailPath, env.cookie, nil); rec.Code != http.StatusOK {
			t.Fatalf("owner detail after revoke status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("OwnerOnlyWriteFacesKeepTheirShapes", func(t *testing.T) {
		env.putAlbumAudience(t, env.cookie, uuid.UUID(album.Id), graceID)

		// PATCH: the 404 is the asset's own single shape.
		rec := env.patchAssetRaw(t, graceCookie, shared, []byte(`{"description":"sharee edit"}`))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("sharee patch status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ASSETNOTFOUND {
			t.Errorf("sharee patch code = %q, want %q", apiErr.Code, apigen.ASSETNOTFOUND)
		}

		// The batch's curation arms resolve owner-scoped: the
		// shared id reports as missing for the sharee and the
		// owner's row is untouched.
		report := env.batchAssets(t, graceCookie, "trash", shared)
		requireBatchReport(t, report, 0, shared)
		if env.assetRowOf(t, shared).DeletedAt.Valid {
			t.Error("the sharee's batch trashed the owner's asset, want untouched")
		}

		// The trash restore answers its owner-scoped 404 for the
		// sharee, and the permanent delete answers the same shape.
		restore := env.restoreAsset(t, graceCookie, shared)
		if restore.Code != http.StatusNotFound {
			t.Errorf("sharee restore status = %d, want %d (body %q)",
				restore.Code, http.StatusNotFound, restore.Body.String())
		}
		del := httptest.NewRequest(http.MethodDelete, "/api/v1/assets/"+shared.String(), nil)
		del.AddCookie(graceCookie)
		delRec := httptest.NewRecorder()
		env.router.ServeHTTP(delRec, del)
		if delRec.Code != http.StatusNotFound {
			t.Errorf("sharee permanent delete status = %d, want %d (body %q)",
				delRec.Code, http.StatusNotFound, delRec.Body.String())
		}
		if count := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(shared)); count != 1 {
			t.Errorf("asset rows after the sharee's delete attempt = %d, want 1 (untouched)", count)
		}
	})
}

// TestSharedVisibilityFavorites pins the favorites pair under the
// union: the batch favorite's resolution accepts a shared-album
// member for a sharee, the view crosses the caller's rows with
// caller-visible assets, and a revoked share removes the assets
// while the favorite row survives.
func TestSharedVisibilityFavorites(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "favorites-shared-grace@example.com")

	sharedAlbum := fixtureAlbumID("f1")
	env.insertTimelineAlbum(t, sharedAlbum, adminID, "virtual")
	env.shareTimelineAlbum(t, sharedAlbum, graceID, adminID)
	taken := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	shared := fixtureAssetID("f1")
	sharedHidden := fixtureAssetID("f2")
	env.insertOwnedAsset(t, shared, adminID, taken, visibilityAssetOpts{})
	env.insertOwnedAsset(t, sharedHidden, adminID, taken.Add(time.Minute), visibilityAssetOpts{})
	env.addTimelineItem(t, sharedAlbum, shared, adminID)
	env.addTimelineItem(t, sharedAlbum, sharedHidden, adminID)

	// Grace's own favorite target and an asset the admin never
	// shared (the resolution's invisible arm).
	graceOwn := fixtureAssetID("f3")
	env.insertOwnedAsset(t, graceOwn, graceID, taken.Add(2*time.Minute), visibilityAssetOpts{})
	unshared := fixtureAssetID("f4")
	env.insertOwnedAsset(t, unshared, adminID, taken.Add(3*time.Minute), visibilityAssetOpts{})

	t.Run("ShareeFavoritesWhatTheySee", func(t *testing.T) {
		report := env.batchAssets(t, graceCookie, "favorite", shared, sharedHidden, unshared, graceOwn)
		requireBatchReport(t, report, 3, unshared)
		for _, id := range []uuid.UUID{shared, sharedHidden, graceOwn} {
			if count := env.queryInt(t,
				"SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
				pgUUID(graceID), pgUUID(id)); count != 1 {
				t.Errorf("favorites row for %s = %d, want 1", id, count)
			}
		}

		// The view crosses her rows with the union: the live shared
		// members and her own sync-stream member appear -- the own
		// arm resolves through the sync membership exactly like the
		// shared arm through the album (the later subtest hides one
		// of the shared pair and watches it filter out).
		page := env.getFavorites(t, graceCookie, "")
		requireTimelineIDs(t, page.Items, graceOwn, sharedHidden, shared)
		requireFavoriteFlags(t, page.Items, true)

		// The admin's view is untouched by the sharee's rows, and
		// the admin CAN favorite the shared member too (their own
		// arm) -- per-user state over the same asset.
		report = env.batchAssets(t, env.cookie, "favorite", shared)
		requireBatchReport(t, report, 1)
		page = env.getFavorites(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, shared)
	})

	t.Run("HiddenSharedFavoriteFiltersOutOfView", func(t *testing.T) {
		// The owner hides the member: the listing surfaces (the
		// timeline, the album view, the favorites view) all drop
		// it, the favorite row survives.
		env.batchAssets(t, env.cookie, "hide", sharedHidden)
		if rows := env.queryInt(t,
			"SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
			pgUUID(graceID), pgUUID(sharedHidden)); rows != 1 {
			t.Fatalf("favorites row for the hidden member = %d, want 1 (the mark survives)", rows)
		}
		env.batchAssets(t, env.cookie, "unhide", sharedHidden)
	})

	t.Run("RetractionRemovesFavoritesFromView", func(t *testing.T) {
		env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
			pgUUID(sharedAlbum), pgUUID(graceID))
		page := env.getFavorites(t, graceCookie, "")
		requireTimelineIDs(t, page.Items, graceOwn)
		if count := env.queryInt(t,
			"SELECT count(*) FROM user_favorites WHERE user_id = $1",
			pgUUID(graceID)); count != 3 {
			t.Errorf("favorites rows after retraction = %d, want 3 (the rows survive)", count)
		}
		// The admin's favorite of their own asset is unaffected.
		page = env.getFavorites(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, shared)
	})
}
