// Face crop thumbnail tests: the people surface's pixel
// endpoint. The crop geometry is pinned deterministically -- exact
// window rectangles through media.FaceCropWindow, and solid-color
// quadrant derivatives whose averaged output proves which region the
// window covered (the colors survive the lossy AVIF encode nearly
// exactly, so the assertions never flake). The permission gate, the
// byte-identical 404 family, and the ETag/304 discipline follow the
// assets serving tests' style; the derivative fixtures run through
// the real thumbnail pipeline (media.GenerateThumbnails), so the
// endpoint reads exactly the bytes the jobs would leave behind.
package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// The geometry test's derivative source: 800x600 pixels of four solid
// color quadrants, run through the real derivation pipeline. The size
// sits under the view's 1080 long-edge cap, so the view derivative
// keeps the source's dimensions and framing -- the coordinate space
// the face bounding boxes' per-mille values address.
const (
	faceTestSourceW = 800
	faceTestSourceH = 600
)

// faceTestQuadrants are the derivative's four solid regions, indexed
// by (right bit, bottom bit): red top-left, green top-right, blue
// bottom-left, white bottom-right.
var faceTestQuadrants = [4]color.RGBA{
	{R: 255, A: 255},
	{G: 255, A: 255},
	{B: 255, A: 255},
	{R: 255, G: 255, B: 255, A: 255},
}

// faceTestTolerance is how far the averaged output of a solid-quadrant
// crop may sit from the quadrant's color: the lossy AVIF encode moves
// flat regions by about one unit of 255, while distinct quadrants sit
// a full channel apart -- the margin is enormous on both sides.
const faceTestTolerance = 30

// requireFaceCropMedia skips the pixel-level tests when this build
// carries no libvips: the crop pipeline and the derivative fixtures
// need it (make docker-test links it).
func requireFaceCropMedia(t *testing.T) {
	t.Helper()
	if !media.ThumbnailsSupported {
		t.Skip("this build carries no libvips; run the suite inside the dev container (make docker-test)")
	}
}

// insertFaceWithBox inserts one fixture face row with an explicit
// bounding box (per-mille) and change signal: the crop endpoint's
// inputs are the box and updated_at, so the tests pin both exactly.
func (e *assetEnv) insertFaceWithBox(t *testing.T, id, assetID uuid.UUID, bbox [4]int, updatedAt time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0.9, $7, $8)`,
		pgUUID(id), pgUUID(assetID), bbox[0], bbox[1], bbox[2], bbox[3],
		peopleModelID, updatedAt); err != nil {
		t.Fatalf("insert fixture face %s: %v", id, err)
	}
}

// deriveQuadrantViewFixture runs the real thumbnail pipeline over a
// solid-quadrant source and leaves the view derivative at the storage
// path the endpoint reads. The fixture is the pipeline's own output,
// so the endpoint decodes exactly the bytes a real asset's view
// derivative carries.
func deriveQuadrantViewFixture(t *testing.T, assetID uuid.UUID) {
	t.Helper()
	source := image.NewRGBA(image.Rect(0, 0, faceTestSourceW, faceTestSourceH))
	for y := 0; y < faceTestSourceH; y++ {
		for x := 0; x < faceTestSourceW; x++ {
			quadrant := 0
			if x >= faceTestSourceW/2 {
				quadrant |= 1
			}
			if y >= faceTestSourceH/2 {
				quadrant |= 2
			}
			source.SetRGBA(x, y, faceTestQuadrants[quadrant])
		}
	}

	original := filepath.Join(t.TempDir(), "quadrants.png")
	file, err := os.Create(original)
	if err != nil {
		t.Fatalf("create quadrant source: %v", err)
	}
	if err := png.Encode(file, source); err != nil {
		t.Fatalf("encode quadrant source: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close quadrant source: %v", err)
	}

	if _, err := media.GenerateThumbnails(original, assetID.String()); err != nil {
		t.Fatalf("derive quadrant view fixture of asset %s: %v", assetID, err)
	}
	width, height, err := media.ImageDimensions(storage.ThumbPath(assetID.String(), media.SizeView, media.FormatAvif))
	if err != nil {
		t.Fatalf("decode quadrant view fixture: %v", err)
	}
	if width != faceTestSourceW || height != faceTestSourceH {
		t.Fatalf("quadrant view fixture = %dx%d, want %dx%d (the source sits under the view cap)",
			width, height, faceTestSourceW, faceTestSourceH)
	}
}

// saveFaceCropBody stores one response body in a scratch file the
// media package's decode helpers can read.
func saveFaceCropBody(t *testing.T, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "face-crop.avif")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("save face crop body: %v", err)
	}
	return path
}

// requireAVIFSignature pins the response's container: an ISO base
// media file whose first box is ftyp with the avif major brand (the
// header the libvips AVIF encoder writes).
func requireAVIFSignature(t *testing.T, body []byte) {
	t.Helper()
	if len(body) < 12 {
		t.Fatalf("face crop body is %d bytes, too short for an AVIF box header", len(body))
	}
	if string(body[4:8]) != "ftyp" {
		t.Errorf("face crop box type = %q, want %q", body[4:8], "ftyp")
	}
	if string(body[8:12]) != "avif" {
		t.Errorf("face crop major brand = %q, want %q", body[8:12], "avif")
	}
}

// requireAverageColor pins the region the crop window covered: the
// averaged output of a solid-quadrant crop sits at the quadrant's
// color (the encode moves flat regions by about one unit of 255,
// while distinct quadrants sit a full channel apart).
func requireAverageColor(t *testing.T, body []byte, want color.RGBA) {
	t.Helper()
	path := saveFaceCropBody(t, body)
	r, g, b, err := media.ImageAverageColor(path)
	if err != nil {
		t.Fatalf("average face crop pixels: %v", err)
	}
	if math.Abs(r-float64(want.R)) > faceTestTolerance ||
		math.Abs(g-float64(want.G)) > faceTestTolerance ||
		math.Abs(b-float64(want.B)) > faceTestTolerance {
		t.Errorf("face crop average = (%.1f, %.1f, %.1f), want the quadrant color (%d, %d, %d)",
			r, g, b, want.R, want.G, want.B)
	}
}

// faceThumbTarget renders the request URL of one face's crop.
func faceThumbTarget(faceID uuid.UUID) string {
	return "/api/v1/faces/" + faceID.String() + "/thumbnail"
}

// faceThumbETag renders the entity tag the endpoint pins: the face id
// with the face row's change signal in unix seconds, quoted.
func faceThumbETag(faceID uuid.UUID, updatedAt time.Time) string {
	return fmt.Sprintf(`"face-%s-%d"`, faceID, updatedAt.Unix())
}

// TestFaceThumbnailServes walks the serving mechanics of the face
// crop endpoint: the 200 body is a real 256x256 AVIF, the static
// asset discipline rides along (the exact face-signal ETag, the
// immutable private cache policy, the conditional 304s), a clusterless
// face on a visible asset serves (the gate is the asset, not cluster
// membership), and an unauthenticated request learns nothing.
func TestFaceThumbnailServes(t *testing.T) {
	requireFaceCropMedia(t)
	env := newAssetEnv(t, testUploadLimit)
	env.insertPeopleModel(t)
	adminID := env.sessionUserID(t)

	assetID := fixturePeopleAssetID("61")
	taken := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	env.insertOwnedAsset(t, assetID, adminID, taken, timelineAssetOpts{width: 800, height: 600})
	deriveQuadrantViewFixture(t, assetID)

	updatedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	faceID := fixturePeopleFaceID("61")
	env.insertFaceWithBox(t, faceID, assetID, [4]int{150, 100, 100, 100}, updatedAt)
	env.seedPersonCluster(t, fixturePeopleClusterID("61"), faceID)
	clusterlessID := fixturePeopleFaceID("62")
	env.insertFaceWithBox(t, clusterlessID, assetID, [4]int{650, 100, 100, 100}, updatedAt)

	wantETag := faceThumbETag(faceID, updatedAt)
	target := faceThumbTarget(faceID)

	t.Run("ServesCrop", func(t *testing.T) {
		rec := env.serveGet(t, target, env.cookie, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != "image/avif" {
			t.Errorf("content-type = %q, want image/avif", got)
		}
		requireAVIFSignature(t, rec.Body.Bytes())
		width, height, err := media.ImageDimensions(saveFaceCropBody(t, rec.Body.Bytes()))
		if err != nil {
			t.Fatalf("decode served face crop: %v", err)
		}
		if width != 256 || height != 256 {
			t.Errorf("face crop = %dx%d, want 256x256", width, height)
		}
	})

	t.Run("ServesClusterlessFace", func(t *testing.T) {
		// The crop's inputs are the face row's box and its asset; a
		// face the clustering has not placed yet still has pixels, so
		// the endpoint serves it (the people listings only ever link
		// clustered faces, but the gate stays the asset).
		rec := env.serveGet(t, faceThumbTarget(clusterlessID), env.cookie, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		requireAVIFSignature(t, rec.Body.Bytes())
	})

	t.Run("HeadersAndConditionals", func(t *testing.T) {
		rec := env.serveGet(t, target, env.cookie, nil)
		if got := rec.Header().Get("ETag"); got != wantETag {
			t.Errorf("etag = %q, want %q", got, wantETag)
		}
		if got := rec.Header().Get("Cache-Control"); got != "private, max-age=31536000, immutable" {
			t.Errorf("cache-control = %q, want the immutable private policy", got)
		}

		cases := []struct {
			name        string
			ifNoneMatch string
			wantStatus  int
		}{
			{"StrongMatch", wantETag, http.StatusNotModified},
			{"WeakMatch", `W/` + wantETag, http.StatusNotModified},
			{"Star", "*", http.StatusNotModified},
			{"ListWithMatch", `"face-stale-0", ` + wantETag, http.StatusNotModified},
			{"Mismatch", `"face-stale-0"`, http.StatusOK},
		}
		for _, testCase := range cases {
			rec := env.serveGet(t, target, env.cookie,
				map[string]string{"If-None-Match": testCase.ifNoneMatch})
			if rec.Code != testCase.wantStatus {
				t.Fatalf("%s status = %d, want %d", testCase.name, rec.Code, testCase.wantStatus)
			}
			if got := rec.Header().Get("ETag"); got != wantETag {
				t.Errorf("%s etag = %q, want %q also on the 304", testCase.name, got, wantETag)
			}
			if testCase.wantStatus == http.StatusNotModified && rec.Body.Len() != 0 {
				t.Errorf("%s body = %q, want no body on the 304", testCase.name, rec.Body.String())
			}
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := env.serveGet(t, target, nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestFaceThumbnailGeometry pins the crop window itself, twice over:
// the exact rectangles FaceCropWindow computes (the clamped corner,
// the oversized face, the degenerate box), and -- through the real
// endpoint -- the averaged pixels of crops whose windows sit inside
// known quadrants of the derivative. The face boxes are per-mille
// against the SAME framing the derivative preserves, so the quadrant
// a crop's average matches is the region the window covered.
func TestFaceThumbnailGeometry(t *testing.T) {
	requireFaceCropMedia(t)

	t.Run("WindowRectangles", func(t *testing.T) {
		cases := []struct {
			name       string
			imgW, imgH int
			bbox       [4]int
			wantLeft   int
			wantTop    int
			wantSide   int
		}{
			// 1.25 x max(80, 60) = 100 centered on (160, 90).
			{"CenteredTopLeft", 800, 600, [4]int{150, 100, 100, 100}, 110, 40, 100},
			{"CenteredBottomRight", 800, 600, [4]int{650, 650, 100, 100}, 510, 370, 100},
			// The box centers on the image's corner: the window clamps
			// from (760, 560) back to the image's bounds.
			{"ClampedCorner", 800, 600, [4]int{960, 960, 80, 80}, 720, 520, 80},
			// A face larger than the image: the side caps at the
			// smaller edge, centered on the box.
			{"OversizedFace", 800, 600, [4]int{0, 0, 1000, 1000}, 100, 0, 600},
			// A degenerate box still crops one pixel at its center.
			{"DegenerateBox", 1000, 1000, [4]int{500, 500, 0, 0}, 500, 500, 1},
		}
		for _, testCase := range cases {
			left, top, side := media.FaceCropWindow(testCase.imgW, testCase.imgH,
				testCase.bbox[0], testCase.bbox[1], testCase.bbox[2], testCase.bbox[3])
			if left != testCase.wantLeft || top != testCase.wantTop || side != testCase.wantSide {
				t.Errorf("%s window = (%d, %d, %d), want (%d, %d, %d)",
					testCase.name, left, top, side,
					testCase.wantLeft, testCase.wantTop, testCase.wantSide)
			}
		}
	})

	env := newAssetEnv(t, testUploadLimit)
	env.insertPeopleModel(t)
	adminID := env.sessionUserID(t)

	assetID := fixturePeopleAssetID("71")
	taken := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	env.insertOwnedAsset(t, assetID, adminID, taken, timelineAssetOpts{width: 800, height: 600})
	deriveQuadrantViewFixture(t, assetID)
	updatedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	// Three faces of the same derivative: one squarely inside the
	// top-left quadrant, one inside the bottom-right, one whose
	// unclamped window would leave the image through the corner (the
	// clamp keeps it inside the bottom-right quadrant).
	topLeft := fixturePeopleFaceID("71")
	env.insertFaceWithBox(t, topLeft, assetID, [4]int{150, 100, 100, 100}, updatedAt)
	env.seedPersonCluster(t, fixturePeopleClusterID("71"), topLeft)
	bottomRight := fixturePeopleFaceID("72")
	env.insertFaceWithBox(t, bottomRight, assetID, [4]int{650, 650, 100, 100}, updatedAt)
	clampedCorner := fixturePeopleFaceID("73")
	env.insertFaceWithBox(t, clampedCorner, assetID, [4]int{960, 960, 80, 80}, updatedAt)

	cases := []struct {
		name   string
		faceID uuid.UUID
		want   color.RGBA
	}{
		{"TopLeftQuadrant", topLeft, faceTestQuadrants[0]},
		{"BottomRightQuadrant", bottomRight, faceTestQuadrants[3]},
		{"ClampedCornerStaysInside", clampedCorner, faceTestQuadrants[3]},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := env.serveGet(t, faceThumbTarget(testCase.faceID), env.cookie, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
			}
			requireAVIFSignature(t, rec.Body.Bytes())
			requireAverageColor(t, rec.Body.Bytes(), testCase.want)
		})
	}
}

// TestFaceThumbnailNotFound pins the single 404 family: an unknown
// face id, a face on another account's asset, a face on an invisible
// (trashed) asset of the caller, and a face whose asset has no view
// derivative yet all answer the SAME bytes -- PERSON_NOT_FOUND, the
// anti-enumeration shape of the whole people surface. An
// unauthenticated request never learns whether the id exists. No
// pixels are needed, so the test runs in every build.
func TestFaceThumbnailNotFound(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	env.insertPeopleModel(t)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "face-thumb-grace@example.com")

	taken := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	// The caller's own asset WITHOUT a derivative (the thumbnail job
	// has not run), a second account's asset, and the caller's
	// trashed asset.
	notReady := fixturePeopleAssetID("81")
	foreign := fixturePeopleAssetID("82")
	trashed := fixturePeopleAssetID("83")
	env.insertOwnedAsset(t, notReady, adminID, taken, timelineAssetOpts{})
	env.insertOwnedAsset(t, foreign, graceID, taken, timelineAssetOpts{})
	env.insertOwnedAsset(t, trashed, adminID, taken, timelineAssetOpts{deleted: true})

	faceNotReady := fixturePeopleFaceID("81")
	faceForeign := fixturePeopleFaceID("82")
	faceTrashed := fixturePeopleFaceID("83")
	env.insertFaceWithBox(t, faceNotReady, notReady, [4]int{150, 100, 100, 100}, updatedAt)
	env.insertFaceWithBox(t, faceForeign, foreign, [4]int{150, 100, 100, 100}, updatedAt)
	env.insertFaceWithBox(t, faceTrashed, trashed, [4]int{150, 100, 100, 100}, updatedAt)
	unknown := fixturePeopleFaceID("8e")

	arms := []struct {
		name   string
		faceID uuid.UUID
	}{
		{"UnknownFace", unknown},
		{"ForeignAssetFace", faceForeign},
		{"TrashedAssetFace", faceTrashed},
		{"NoDerivativeYet", faceNotReady},
	}
	var family [][]byte
	for _, arm := range arms {
		t.Run(arm.name, func(t *testing.T) {
			rec := env.serveGet(t, faceThumbTarget(arm.faceID), env.cookie, nil)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.PERSONNOTFOUND {
				t.Errorf("code = %q, want %q", apiErr.Code, apigen.PERSONNOTFOUND)
			}
			if apiErr.Message == nil || *apiErr.Message != "no such person for this caller" {
				t.Errorf("message = %v, want the contract example's %q", apiErr.Message,
					"no such person for this caller")
			}
			family = append(family, rec.Body.Bytes())
			if len(family) > 1 && !bytes.Equal(family[0], family[len(family)-1]) {
				t.Errorf("404 body = %q, want byte-identical to the first arm's %q",
					rec.Body.String(), family[0])
			}
		})
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := env.serveGet(t, faceThumbTarget(unknown), nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestFaceThumbnailETagChange pins the change signal: the
// first response's ETag, the 304 of a replay, and the NEW
// ETag -- and fresh 200 -- after the face row's updated_at moves. The
// signal bump is applied directly with an explicit later timestamp
// (a cluster move through the API would bump it too, but a move
// inside the same test second would keep the unix value, so the
// deterministic form pins the ETag contract instead of the clock).
func TestFaceThumbnailETagChange(t *testing.T) {
	requireFaceCropMedia(t)
	env := newAssetEnv(t, testUploadLimit)
	env.insertPeopleModel(t)
	adminID := env.sessionUserID(t)

	assetID := fixturePeopleAssetID("91")
	taken := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	env.insertOwnedAsset(t, assetID, adminID, taken, timelineAssetOpts{width: 800, height: 600})
	deriveQuadrantViewFixture(t, assetID)

	first := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	second := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	faceID := fixturePeopleFaceID("91")
	env.insertFaceWithBox(t, faceID, assetID, [4]int{150, 100, 100, 100}, first)
	env.seedPersonCluster(t, fixturePeopleClusterID("91"), faceID)
	target := faceThumbTarget(faceID)

	firstETag := faceThumbETag(faceID, first)
	rec := env.serveGet(t, target, env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("ETag"); got != firstETag {
		t.Fatalf("first etag = %q, want %q", got, firstETag)
	}

	rec = env.serveGet(t, target, env.cookie, map[string]string{"If-None-Match": firstETag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("replay status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	// The face row's change signal moves: the same URL re-renders
	// under a new tag, and the old tag no longer matches.
	env.exec(t, "UPDATE faces SET updated_at = $1 WHERE id = $2", second, pgUUID(faceID))

	rec = env.serveGet(t, target, env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("after the signal moved, status = %d, want %d", rec.Code, http.StatusOK)
	}
	secondETag := faceThumbETag(faceID, second)
	if got := rec.Header().Get("ETag"); got != secondETag {
		t.Fatalf("second etag = %q, want %q", got, secondETag)
	}
	if secondETag == firstETag {
		t.Fatalf("etag stayed %q after the change signal moved", firstETag)
	}

	rec = env.serveGet(t, target, env.cookie, map[string]string{"If-None-Match": secondETag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("new-tag replay status = %d, want %d", rec.Code, http.StatusNotModified)
	}
	rec = env.serveGet(t, target, env.cookie, map[string]string{"If-None-Match": firstETag})
	if rec.Code != http.StatusOK {
		t.Fatalf("stale-tag replay status = %d, want %d (the old tag must miss)", rec.Code, http.StatusOK)
	}
}
