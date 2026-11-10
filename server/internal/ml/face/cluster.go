// The clusterer of the recognition pipeline: a textbook
// brute-force DBSCAN over the L2-normalized embeddings, with the
// cosine distance (one minus the dot product) as the neighborhood
// metric. The faces of a family-scale library number in the low
// thousands, so the O(n^2) pairwise scan -- every pair computed once
// per job, in application memory -- is the honest cost
// accepted; the incremental clustering of a larger library is a later
// optimization.
package mlface

// The clustering defaults: the recognize model's params carry eps
// and minPts when the manifest pins them, and anything absent or
// malformed falls back to these.
const (
	// DefaultClusterEps is the cosine-distance radius of one
	// neighborhood: embeddings closer than 0.55 (a cosine similarity
	// above 0.45) are neighbors.
	DefaultClusterEps = 0.55
	// DefaultClusterMinPts is the neighborhood size a point needs to
	// be a cluster's seed.
	DefaultClusterMinPts = 2
)

// Cluster runs DBSCAN over the embeddings and returns one label per
// embedding: the cluster's index (numbered in the scan order of each
// cluster's first seed), or -1 for noise. The embeddings are the
// Embed wrapper's output -- L2-normalized, so the dot product is the
// cosine similarity and one minus it is the cosine distance -- and
// the neighborhood test is inclusive (distance <= eps), the textbook
// convention. A minPts below 1 is clamped to 1: every point is then
// its own seed and nothing is noise.
//
// The clustering is deterministic: the pairwise scan walks the
// embeddings in order, a cluster's expansion queue follows discovery
// order, and ties resolve toward the smaller index.
func Cluster(embeddings [][]float32, eps float64, minPts int) []int {
	labels := make([]int, len(embeddings))
	for i := range labels {
		labels[i] = unvisited
	}
	if len(embeddings) == 0 {
		return labels
	}
	if minPts < 1 {
		minPts = 1
	}

	nextCluster := 0
	for i := range embeddings {
		if labels[i] != unvisited {
			continue
		}
		neighbors := regionOf(embeddings, i, eps)
		if len(neighbors) < minPts {
			// Not a seed: a lone point or an edge of thin air. It may
			// still join a cluster later as a border point of one of
			// its neighbors; if no such neighbor exists, the label
			// stands as noise.
			labels[i] = noise
			continue
		}
		labels[i] = nextCluster
		nextCluster++

		// The expansion: every unvisited neighbor joins the cluster,
		// and a neighbor that is itself a seed contributes its own
		// neighborhood to the frontier -- density connectivity, not
		// pairwise proximity, is what a cluster is.
		frontier := make([]int, 0, len(neighbors))
		frontier = append(frontier, neighbors...)
		for len(frontier) > 0 {
			j := frontier[0]
			frontier = frontier[1:]
			if labels[j] == noise {
				// A point earlier marked noise lies inside this
				// cluster's reach: it joins as a border point.
				labels[j] = labels[i]
				continue
			}
			if labels[j] != unvisited {
				continue
			}
			labels[j] = labels[i]
			jNeighbors := regionOf(embeddings, j, eps)
			if len(jNeighbors) >= minPts {
				frontier = append(frontier, jNeighbors...)
			}
		}
	}

	// Any label still unvisited is unreachable state (the loop visits
	// every index); noise is the only negative label that escapes.
	for i, label := range labels {
		if label == unvisited {
			labels[i] = noise
		}
	}
	return labels
}

// The label states of the run: noise is the public -1, unvisited is
// the run's internal bookkeeping.
const (
	noise     = -1
	unvisited = -2
)

// regionOf lists the indexes whose embedding lies within eps of the
// i-th (the point itself included: a normalized embedding is at
// distance zero from itself).
func regionOf(embeddings [][]float32, i int, eps float64) []int {
	var neighbors []int
	for j := range embeddings {
		if cosineDistance(embeddings[i], embeddings[j]) <= eps {
			neighbors = append(neighbors, j)
		}
	}
	return neighbors
}

// cosineDistance is one minus the dot product of two embeddings:
// with L2-normalized inputs this is the cosine distance, in [0, 2].
// The accumulation runs in float64 so a thousand-term dot product
// keeps its precision.
func cosineDistance(a, b []float32) float64 {
	dot := 0.0
	n := min(len(a), len(b))
	for k := 0; k < n; k++ {
		dot += float64(a[k]) * float64(b[k])
	}
	return 1 - dot
}

// ClusterOptionsOf reads the clustering steering values out of the
// recognize model's params: eps (default 0.55) and minPts (default
// 2). Malformed values fall back to the defaults -- the params
// arrived with a registered model, so a drifted entry degrades the
// radius, it does not fail every asset's ML work.
func ClusterOptionsOf(params map[string]any) (eps float64, minPts int) {
	eps, minPts = DefaultClusterEps, DefaultClusterMinPts
	if v, ok := paramFloat(params, "eps"); ok {
		eps = v
	}
	if v, ok := paramFloat(params, "minPts"); ok && v >= 1 {
		minPts = int(v)
	}
	return eps, minPts
}
