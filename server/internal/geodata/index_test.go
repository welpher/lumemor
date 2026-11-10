// Grid index tests: the resolution contract's determinism and
// correctness. Determinism is pinned three ways -- the same query
// against the same index always answers the same place, a rebuilt
// index over the same centroids answers the same place, and an exact
// distance tie between two cities resolves to the smaller geonameid
// (order-independent, so the scan order can never leak into the
// answer). Correctness is pinned against a brute-force nearest
// scan over a synthetic archipelago: the grid must agree with the
// linear scan on every point, including the antimeridian (the
// cell window wraps in longitude) and the 25 km threshold boundary.
package geodata

import (
	"math"
	"math/rand"
	"testing"
)

// archipelago is a deterministic synthetic dataset spanning the
// corners the grid must survive: the antimeridian, high latitudes
// (the longitude ring widens there), and dense clusters.
func archipelago() []Centroid {
	return []Centroid{
		{ID: 1, Lat: 48.85341, Lon: 2.3488},   // Paris
		{ID: 2, Lat: 51.50722, Lon: -0.1275},  // London
		{ID: 3, Lat: 39.9075, Lon: 116.39723}, // Beijing
		{ID: 4, Lat: 1.35208, Lon: 103.81984}, // Singapore (equator)
		{ID: 5, Lat: 78.2186, Lon: 15.6401},   // Longyearbyen (Svalbard)
		{ID: 6, Lat: -33.8688, Lon: 151.2093}, // Sydney
		{ID: 7, Lat: 64.1466, Lon: -21.9426},  // Reykjavik
		{ID: 8, Lat: 0.0, Lon: 179.9},         // antimeridian west side
		{ID: 9, Lat: 0.0, Lon: -179.9},        // antimeridian east side
		{ID: 10, Lat: 89.0, Lon: 45.0},        // near the pole
		{ID: 11, Lat: 48.0, Lon: 2.0},         // ~90 km south of Paris
		{ID: 12, Lat: 48.0, Lon: 3.0},         // ~100 km south-east of Paris
		{ID: 13, Lat: -0.5, Lon: -78.5},       // Quito region (south, wrap-free)
		{ID: 20, Lat: 40.0, Lon: 179.95},      // antimeridian mid-latitude
		{ID: 21, Lat: 40.0, Lon: -179.95},     // its wrap partner
		{ID: 22, Lat: 0.0, Lon: 179.99},       // a hair east of the antimeridian
	}
}

// bruteForceNearest is the linear scan the grid is checked against:
// every centroid, haversine, smallest distance wins (id as the
// deterministic tiebreak).
func bruteForceNearest(centroids []Centroid, lat, lon, maxKm float64) (int32, bool) {
	best := int32(0)
	bestDist := math.Inf(1)
	for _, c := range centroids {
		d := haversineKm(lat, lon, c.Lat, c.Lon)
		if d < bestDist || (d == bestDist && c.ID < best) {
			bestDist = d
			best = c.ID
		}
	}
	if best == 0 || bestDist > maxKm {
		return 0, false
	}
	return best, true
}

// TestIndexNearestMatchesBruteForce walks a fixed-seed random field
// of query points and asserts the grid and the linear scan agree on
// every one, whatever the nearest city's distance (the not-found arm
// included: both must say "beyond the threshold").
func TestIndexNearestMatchesBruteForce(t *testing.T) {
	centroids := archipelago()
	ix := NewIndex(centroids)
	if got := ix.Len(); got != len(centroids) {
		t.Fatalf("Len() = %d, want %d", got, len(centroids))
	}

	random := rand.New(rand.NewSource(20261106)) //nolint:gosec // fixed fixture seed, no security role
	mismatches, beyond := 0, 0
	for i := 0; i < 20000; i++ {
		lat := random.Float64()*178 - 89 // [-89, 89)
		lon := random.Float64()*358 - 179
		gridID, _, gridOK := ix.NearestWithin(lat, lon, ResolutionRadiusKm)
		wantID, wantOK := bruteForceNearest(centroids, lat, lon, ResolutionRadiusKm)
		if gridOK != wantOK || gridID != wantID {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("point (%.5f, %.5f): grid = (%d, %v), brute force = (%d, %v)",
					lat, lon, gridID, gridOK, wantID, wantOK)
			}
		}
		if !wantOK {
			beyond++
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d of 20000 query points disagree with the brute-force scan", mismatches)
	}
	if beyond == 0 {
		t.Fatal("the random field never exercised the beyond-threshold arm; widen the fixtures")
	}
}

// TestIndexResolvesKnownPoints pins the hand-computed answers:
// queries well inside the threshold resolve to the obvious city, the
// antimeridian pair finds each other across the wrap, and the
// Svalbard query survives the wide high-latitude longitude ring.
func TestIndexResolvesKnownPoints(t *testing.T) {
	ix := NewIndex(archipelago())
	cases := []struct {
		name string
		lat  float64
		lon  float64
		want int32
	}{
		{"Paris", 48.8584, 2.2945, 1},
		{"London", 51.5074, -0.1278, 2},
		{"Beijing", 39.9042, 116.4074, 3},
		{"Singapore equator", 1.3521, 103.8198, 4},
		{"Longyearbyen", 78.2232, 15.6467, 5},
		{"Sydney", -33.865, 151.2099, 6},
		{"Reykjavik", 64.1466, -21.9426, 7},
		// The equatorial pair: the query hugs the -180 side, so the
		// same-side city (9) is nearer than the wrap partner (8) --
		// and the mirrored query on the +180 side answers 8.
		{"antimeridian same side", 0.0, -179.95, 9},
		{"antimeridian same side reverse", 0.0, 179.93, 8},
		// The discriminating wrap case: the query west of -180 finds
		// city 22 EAST of +180 across the line -- without the column
		// wrap its cell sits outside the window and 9 would win.
		{"antimeridian nearest across the line", 0.0, -179.995, 22},
		{"mid-latitude wrap", 40.0, 179.99, 20},
		{"mid-latitude wrap reverse", 40.0, -179.99, 21},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, _, ok := ix.NearestWithin(tc.lat, tc.lon, ResolutionRadiusKm)
			if !ok {
				t.Fatalf("NearestWithin(%.4f, %.4f) found nothing, want place %d", tc.lat, tc.lon, tc.want)
			}
			if id != tc.want {
				t.Errorf("NearestWithin(%.4f, %.4f) = %d, want %d", tc.lat, tc.lon, id, tc.want)
			}
		})
	}
}

// TestIndexBeyondThresholdIsNull pins the hard edge: the
// nearest city more than 25 km away answers nothing (no
// country-polygon fallback, no widening).
func TestIndexBeyondThresholdIsNull(t *testing.T) {
	ix := NewIndex(archipelago())
	// (47.0, 2.0) sits ~111 km south of id 11 (the Paris-region
	// fixture city at (48, 2)), ~134 km from id 12 and ~207 km from
	// Paris itself: far beyond 25 km with nothing else near.
	if id, _, ok := ix.NearestWithin(47.0, 2.0, ResolutionRadiusKm); ok {
		t.Errorf("NearestWithin(47.0, 2.0) = %d, want nothing within 25 km", id)
	}
	// The same point DOES resolve once the radius is widened past the
	// nearest distance -- id 11 at ~111 km -- the threshold is the
	// only gate.
	if id, dist, ok := ix.NearestWithin(47.0, 2.0, 150); !ok || id != 11 {
		t.Errorf("NearestWithin(47.0, 2.0, 150) = (%d, %v, %v), want 11 (the ~111 km-nearest fixture city)", id, dist, ok)
	}
}

// TestIndexDeterministicAcrossCallsAndRebuilds is the resolution
// determinism evidence the plan's review focus #3 demands: the same
// GPS answers the same place on every call, and a freshly built
// index over the same centroids answers the same place (no
// map-iteration or allocation order may leak into the answer).
func TestIndexDeterministicAcrossCallsAndRebuilds(t *testing.T) {
	centroids := archipelago()
	first := NewIndex(centroids)
	queryLat, queryLon := 48.8584, 2.2945
	wantID, _, ok := first.NearestWithin(queryLat, queryLon, ResolutionRadiusKm)
	if !ok {
		t.Fatal("the Paris query found nothing")
	}
	for i := 0; i < 1000; i++ {
		if id, _, ok := first.NearestWithin(queryLat, queryLon, ResolutionRadiusKm); !ok || id != wantID {
			t.Fatalf("call %d answered (%d, %v), want %d", i, id, ok, wantID)
		}
	}
	for i := 0; i < 100; i++ {
		rebuilt := NewIndex(centroids)
		if id, _, ok := rebuilt.NearestWithin(queryLat, queryLon, ResolutionRadiusKm); !ok || id != wantID {
			t.Fatalf("rebuild %d answered (%d, %v), want %d", i, id, ok, wantID)
		}
	}
}

// TestIndexDistanceTieBreaksOnID pins the exact-tie rule: two cities
// at the same distance from the query resolve to the smaller
// geonameid, and the answer is the same from both scan orders
// (the tiebreak makes the grid order-independent).
func TestIndexDistanceTieBreaksOnID(t *testing.T) {
	// Two cities symmetric about the query point, 10 km away each.
	centroids := []Centroid{
		{ID: 700, Lat: 40.09, Lon: 0.0},
		{ID: 300, Lat: 39.91, Lon: 0.0},
	}
	ix := NewIndex(centroids)
	for i := 0; i < 100; i++ {
		id, _, ok := ix.NearestWithin(40.0, 0.0, ResolutionRadiusKm)
		if !ok || id != 300 {
			t.Fatalf("tie query answered (%d, %v), want the smaller id 300", id, ok)
		}
	}
}

// TestIndexRejectsOutOfRangeCoordinates guards the CHECK bounds the
// loader and the database both enforce: a query outside the
// coordinate ranges answers nothing rather than panicking.
func TestIndexRejectsOutOfRangeCoordinates(t *testing.T) {
	ix := NewIndex(archipelago())
	for _, point := range [][2]float64{{91, 0}, {-91, 0}, {0, 181}, {0, -181}, {math.NaN(), 0}, {0, math.Inf(1)}} {
		if id, _, ok := ix.NearestWithin(point[0], point[1], ResolutionRadiusKm); ok {
			t.Errorf("NearestWithin(%v, %v) = %d, want nothing for an out-of-range coordinate", point[0], point[1], id)
		}
	}
}

// TestIndexEmptyAnswersNothing pins the degenerate deployment: no
// places loaded, every query answers nothing.
func TestIndexEmptyAnswersNothing(t *testing.T) {
	ix := NewIndex(nil)
	if id, _, ok := ix.NearestWithin(48.85, 2.35, ResolutionRadiusKm); ok {
		t.Errorf("empty index answered (%d, true), want nothing", id)
	}
	if ix.Len() != 0 {
		t.Errorf("Len() = %d, want 0", ix.Len())
	}
}
