// The in-memory grid index the GPS resolution runs against. The
// route is pinned: the Go process loads the places
// centroids ONCE per process and answers nearest-city queries in
// memory -- no SQL, no extensions, no country polygons.
//
// The structure is a fixed 1-degree lat/lon grid. Every centroid
// lands in exactly one cell (row = floor(lat) clamped to the row
// range, column = floor(lon) modulo 360, so the antimeridian wrap is
// plain modular arithmetic: columns 179 and 180 are neighbors across
// the +/-180 line). A query scans the cells that intersect the
// query's geodesic disk of the threshold radius -- the window proven
// below -- and tracks the best (smallest distance, smaller geonameid
// on an exact tie). The window is derived from two spherical facts,
// both exact:
//
//   - the central angle between two points is at least their
//     latitude difference (cos of the angle = sin phi1 sin phi2 +
//     cos phi1 cos phi2 cos of the longitude difference, which stays
//     below cos of the latitude difference), so every point of the
//     disk sits within delta of the query latitude -- the lat window;
//   - the disk's extreme longitude deviation is reached at the pure
//     east/west bearings, where the destination formula collapses to
//     atan2(sin delta, cos delta cos phi0) -- the lon window,
//     computed at the QUERY latitude and valid for every row of the
//     lat window. When the disk reaches a pole every longitude is
//     reachable and the far side of the pole mirrors into the
//     opposite latitudes: the window widens to all rows and columns.
//
// Each window edge gains one cell of margin: the exact windows bound
// the disk, and the margin absorbs the floating-point rounding of
// their computation without ever excluding a point of the disk.
// Scanning extra cells cannot break exactness -- a candidate outside
// the disk answers a distance beyond the threshold, and the
// NearestWithin gate filters it out -- so the grid's answer is the
// brute-force answer by construction, which
// TestIndexNearestMatchesBruteForce pins over a random field.
//
// Memory: the centroids ride one flat slice (20 bytes per row), the
// grid holds one int32 index per centroid plus one slice header per
// cell (64,800 cells) -- about 7 MiB for the ~185k-row cities500
// bake, well inside the data model's budget.
package geodata

import "math"

// earthRadiusKm is the mean Earth radius the haversine distance and
// every window computation share: the same sphere for the whole
// index, so distances and windows can never disagree.
const earthRadiusKm = 6371.0088

// gridRows is the number of latitude cells: 1-degree cells over
// [-90, 90), the last row absorbing the +90 pole itself.
const gridRows = 180

// gridCols is the number of longitude cells: 1-degree cells over the
// full circle, [-180, 180) modulo 360.
const gridCols = 360

// Centroid is one places row in the index's working set: the geoname
// id the resolution writes into assets.place_id, plus the centroid
// coordinates. The canonical name and the rest of the row stay in the
// database -- the index answers WHERE, never WHAT.
type Centroid struct {
	ID  int32
	Lat float64
	Lon float64
}

// Index is the immutable in-memory grid. Build once per process with
// NewIndex, query concurrently with NearestWithin: nothing mutates
// after construction, so one index safely serves every worker.
type Index struct {
	// centroids is the flat working set; cells index into it.
	centroids []Centroid
	// cells maps row*gridCols+column to the centroid indices of that
	// cell, in insertion order (the order never leaks into answers:
	// the best-tracking comparison is order-independent).
	cells [][]int32
}

// NewIndex builds the grid over the centroids. Duplicated ids are
// the caller's data problem (the import deduplicates by primary key);
// the index itself keeps every row it is given.
func NewIndex(centroids []Centroid) *Index {
	index := &Index{
		centroids: append([]Centroid(nil), centroids...),
		cells:     make([][]int32, gridRows*gridCols),
	}
	for i, centroid := range index.centroids {
		key := cellKey(centroid.Lat, centroid.Lon)
		index.cells[key] = append(index.cells[key], int32(i))
	}
	return index
}

// Len answers the number of centroids the index carries.
func (ix *Index) Len() int {
	return len(ix.centroids)
}

// cellKey maps a coordinate onto its cell: the row clamps into
// [-90, 89] (a +90 coordinate belongs to the polar row), the column
// wraps modulo 360 (the antimeridian: -180 and +180 share one
// column's neighborhood).
func cellKey(lat, lon float64) int {
	row := math.Floor(lat)
	if row > 89 {
		row = 89
	}
	if row < -90 {
		row = -90
	}
	col := math.Mod(math.Floor(lon), gridCols)
	if col < 0 {
		col += gridCols
	}
	return int(row+90)*gridCols + int(col)
}

// NearestWithin answers the nearest centroid to (lat, lon) when one
// lies within maxKm, together with its distance in kilometers. No
// candidate within the threshold -- a missing grid, an out-of-range
// or non-finite coordinate, a non-positive radius -- answers
// (0, 0, false). An exact distance tie between two centroids
// resolves to the smaller geonameid, so the answer never depends on
// scan order.
func (ix *Index) NearestWithin(lat, lon, maxKm float64) (int32, float64, bool) {
	if ix == nil || len(ix.centroids) == 0 {
		return 0, 0, false
	}
	if !validCoordinate(lat, 90) || !validCoordinate(lon, 180) {
		return 0, 0, false
	}
	delta := maxKm / earthRadiusKm // the threshold as a central angle
	if !(delta > 0) || delta >= math.Pi {
		return 0, 0, false
	}

	// The scan window: rows and columns of cells intersecting the
	// query's geodesic disk, one cell of margin on every edge. The
	// column range rides RAW integer space around the query's own
	// column and each column wraps modulo 360 as it is read -- that
	// is the whole antimeridian story. A disk reaching a pole covers
	// every longitude and every row (the far side of the pole mirrors
	// into the opposite latitudes).
	latDegrees := delta * 180 / math.Pi
	rowLo, rowHi := -90, 89
	colLo, colHi := 0, gridCols-1
	if math.Abs(lat)+latDegrees < 90 {
		rowLo = int(math.Floor(lat-latDegrees)) - 1
		rowHi = int(math.Floor(lat+latDegrees)) + 1
		if rowLo < -90 {
			rowLo = -90
		}
		if rowHi > 89 {
			rowHi = 89
		}
		span := lonSpan(lat*math.Pi/180, delta)
		center := int(math.Floor(lon))
		colLo, colHi = center-span, center+span
	}

	bestID := int32(0)
	bestDist := math.Inf(1)
	for row := rowLo; row <= rowHi; row++ {
		for col := colLo; col <= colHi; col++ {
			cell := ix.cells[(row+90)*gridCols+wrapCol(col)]
			for _, candidate := range cell {
				centroid := ix.centroids[candidate]
				dist := haversineKm(lat, lon, centroid.Lat, centroid.Lon)
				if dist < bestDist || (dist == bestDist && centroid.ID < bestID) {
					bestDist = dist
					bestID = centroid.ID
				}
			}
		}
	}

	if bestID == 0 || bestDist > maxKm {
		return 0, 0, false
	}
	return bestID, bestDist, true
}

// validCoordinate reports whether a coordinate is finite and within
// its axis's range.
func validCoordinate(value, bound float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= bound
}

// lonSpan answers the half-width of the disk's longitude window in
// whole cells (including the margin cell): the pure east/west
// destination formula gives the disk's extreme longitude deviation as
// atan2(sin delta, cos delta cos phi0), and the floor plus one margin
// cell absorbs the formula's floating-point rounding. The span is at
// least one cell so a tiny window still scans the query's own cell.
func lonSpan(latRadians, delta float64) int {
	width := math.Atan2(math.Sin(delta), math.Cos(delta)*math.Cos(latRadians)) * 180 / math.Pi
	span := int(math.Floor(width)) + 1
	if span < 1 {
		span = 1
	}
	return span
}

// wrapCol normalizes a raw column index into [0, 360).
func wrapCol(col int) int {
	col %= gridCols
	if col < 0 {
		col += gridCols
	}
	return col
}

// haversineKm answers the great-circle distance between two points
// on the index's sphere, in kilometers. The haversine form is the
// numerically stable choice for the small distances nearest-city
// resolution works with.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	phi1 := lat1 * rad
	phi2 := lat2 * rad
	dPhi := (lat2 - lat1) * rad
	dLambda := (lon2 - lon1) * rad
	cosPhi1 := math.Cos(phi1)
	cosPhi2 := math.Cos(phi2)
	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) + cosPhi1*cosPhi2*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}
