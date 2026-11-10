// The DBSCAN clusterer tests: handcrafted unit-circle
// embeddings whose pairwise cosine distances are exact by
// construction, pinning the textbook semantics -- two density groups
// plus noise separate, the eps boundary is inclusive, a border point
// joins through the density chain, and the degenerate inputs (empty,
// singletons) answer honestly. The options reader's defaults and
// malformed-value fallbacks are pinned alongside: the values arrive
// with a registered model's params, so a drifted entry degrades to
// the defaults instead of failing every asset's ML work.
package mlface

import (
	"math"
	"testing"
)

// unitAt builds the unit vector at angle degrees on the circle: the
// two-dimensional stand-in for an L2-normalized embedding.
func unitAt(degrees float64) []float32 {
	radians := degrees * math.Pi / 180
	return []float32{float32(math.Cos(radians)), float32(math.Sin(radians))}
}

// TestClusterTwoGroupsPlusNoise pins the separation: two density
// groups (0/15/-15 degrees and 110/125/95 degrees -- every
// within-group pair below eps, every cross pair far above it) and one
// outlier at 250 degrees nobody reaches. The labels are the cluster
// indices in scan order of the first core point, noise is -1.
func TestClusterTwoGroupsPlusNoise(t *testing.T) {
	embeddings := [][]float32{
		unitAt(0), unitAt(15), unitAt(-15),
		unitAt(110), unitAt(125), unitAt(95),
		unitAt(250),
	}

	labels := Cluster(embeddings, 0.55, 2)

	want := []int{0, 0, 0, 1, 1, 1, -1}
	if len(labels) != len(want) {
		t.Fatalf("labels = %v, want one label per embedding (%d)", labels, len(want))
	}
	for i, label := range want {
		if labels[i] != label {
			t.Errorf("labels[%d] = %d, want %d (labels = %v)", i, labels[i], label, labels)
		}
	}
}

// TestClusterEpsBoundaryIsInclusive pins the boundary: two vectors at
// cosine distance exactly eps are neighbors (distance <= eps, the
// textbook convention), so minPts 2 forms their cluster; eps a hair
// below leaves both as noise. The distance is exact in floating
// point: only the first component contributes to the dot product.
func TestClusterEpsBoundaryIsInclusive(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0.5, 0.8660254}

	if got := labelsOf(Cluster([][]float32{a, b}, 0.5, 2)); got[0] != 0 || got[1] != 0 {
		t.Errorf("labels at eps 0.5 = %v, want one cluster for both (boundary inclusive)", got)
	}
	if got := labelsOf(Cluster([][]float32{a, b}, 0.499, 2)); got[0] != -1 || got[1] != -1 {
		t.Errorf("labels at eps 0.499 = %v, want both noise (distance 0.5 is above)", got)
	}
}

// TestClusterChainsThroughBorderPoints pins the density chain: three
// points at 0, 40 and 80 degrees with eps at the 45-degree cosine
// distance -- the outer pair is NOT directly within eps, but each is
// within eps of the middle one, so the expansion carries the whole
// chain into one cluster.
func TestClusterChainsThroughBorderPoints(t *testing.T) {
	embeddings := [][]float32{unitAt(0), unitAt(40), unitAt(80)}
	eps := 1 - math.Cos(math.Pi/4) // the 45-degree cosine distance

	labels := Cluster(embeddings, eps, 2)

	for i, label := range labels {
		if label != 0 {
			t.Errorf("labels[%d] = %d, want 0 (density-connected through the middle point)", i, label)
		}
	}
}

// TestClusterEmptyAndSingletonInputs pins the degenerate inputs: an
// empty embedding set answers an empty label set (no panic), and a
// lone embedding cannot meet minPts 2, so it is noise.
func TestClusterEmptyAndSingletonInputs(t *testing.T) {
	if got := Cluster(nil, 0.55, 2); len(got) != 0 {
		t.Errorf("Cluster(nil) = %v, want no labels", got)
	}
	if got := Cluster([][]float32{}, 0.55, 2); len(got) != 0 {
		t.Errorf("Cluster(empty) = %v, want no labels", got)
	}
	if got := labelsOf(Cluster([][]float32{unitAt(0)}, 0.55, 2)); got[0] != -1 {
		t.Errorf("labels of a lone embedding = %v, want noise (minPts 2 unmet)", got)
	}
}

// labelsOf guards the label slice's shape before the assertions read
// it (a panicked clusterer fails its caller, not the test harness).
func labelsOf(labels []int) []int {
	if len(labels) == 0 {
		return []int{-2}
	}
	return labels
}

// TestClusterOptionsOf pins the eps and minPts reader: the params of
// the recognize model steer the clustering (the manifest's default
// group carries eps 0.55 and minPts 2), absent or malformed values
// fall back to the same defaults -- the drifted-params discipline
// the decode options established.
func TestClusterOptionsOf(t *testing.T) {
	if eps, minPts := ClusterOptionsOf(nil); eps != DefaultClusterEps || minPts != DefaultClusterMinPts {
		t.Errorf("ClusterOptionsOf(nil) = (%v, %d), want the defaults (%v, %d)",
			eps, minPts, DefaultClusterEps, DefaultClusterMinPts)
	}
	if eps, minPts := ClusterOptionsOf(map[string]any{"eps": 0.4, "minPts": 5}); eps != 0.4 || minPts != 5 {
		t.Errorf("ClusterOptionsOf(params) = (%v, %d), want (0.4, 5)", eps, minPts)
	}
	if eps, minPts := ClusterOptionsOf(map[string]any{"eps": "wide", "minPts": 0}); eps != DefaultClusterEps || minPts != DefaultClusterMinPts {
		t.Errorf("ClusterOptionsOf(malformed) = (%v, %d), want the defaults", eps, minPts)
	}
	if eps, minPts := ClusterOptionsOf(map[string]any{"eps": float32(0.3), "minPts": int64(3)}); eps < 0.299 || eps > 0.301 || minPts != 3 {
		t.Errorf("ClusterOptionsOf(json numbers) = (%v, %d), want (0.3, 3)", eps, minPts)
	}
}
