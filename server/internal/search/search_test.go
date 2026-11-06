// The search engine's pure-logic tests (M3d Task 3): the ILIKE
// wildcard escaping and the deterministic two-arm rank fusion --
// dedup by asset id, the integer rank-sum score, the (score DESC, id
// ASC) ordering, and the arm-depth cut. The SQL behavior (scoping,
// arms, facets) is exercised through the api package's endpoint
// tests over a real database; this file pins the arithmetic the
// merge rule is.
package search

import (
	"testing"

	"github.com/google/uuid"
)

// TestEscapeLikeWildcards pins the literal-substring contract: the
// ILIKE wildcards (% _) and the escape character itself lose their
// pattern meaning inside the query, so the query matches itself
// verbatim. Non-wildcard characters -- including CJK, accents and
// digits -- pass through untouched.
func TestEscapeLikeWildcards(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"海边", "海边"},
		{"beach", "beach"},
		{"café", "café"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`back\slash`, `back\\slash`},
		{"%_", `\%\_`},
		{"50% off", `50\% off`},
		{"東京", "東京"},
	}
	for _, tc := range cases {
		if got := escapeLike(tc.in); got != tc.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// fuseHits builds one arm's hit list from ranked ids.
func fuseHits(ids ...uuid.UUID) []rankedHit {
	hits := make([]rankedHit, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, rankedHit{assetID: id})
	}
	return hits
}

// TestFuseRanksDeterministically pins the merge rule: the score is
// the sum over the arms of (arm depth minus the hit's arm position),
// an asset hit by both arms outranks single-arm hits near the top,
// the ordering is (score DESC, asset id ASC), and the same inputs
// answer the same list.
func TestFuseRanksDeterministically(t *testing.T) {
	low := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mid := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	high := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	// The FTS arm ranks [mid, low, high]; the vector arm ranks
	// [high, mid]. Scores: mid = (N-0)+(N-1); high = (N-2)+(N-0);
	// low = N-1. With N=armDepth: mid > high > low.
	fts := fuseHits(mid, low, high)
	vec := fuseHits(high, mid)

	fused := fuse(fts, vec)
	wantOrder := []uuid.UUID{mid, high, low}
	if len(fused) != len(wantOrder) {
		t.Fatalf("fused = %v, want %v", fused, wantOrder)
	}
	for i, want := range wantOrder {
		if fused[i].assetID != want {
			t.Errorf("fused[%d] = %s, want %s", i, fused[i].assetID, want)
		}
	}

	// The same inputs answer the same list.
	again := fuse(fts, vec)
	for i := range fused {
		if fused[i] != again[i] {
			t.Errorf("fused[%d] drifted across identical runs", i)
		}
	}
}

// TestFuseDedupsByAssetID pins the dedup: one asset hit by both arms
// contributes one fused row.
func TestFuseDedupsByAssetID(t *testing.T) {
	a := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	b := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")

	fused := fuse(fuseHits(a, b), fuseHits(b, a))
	if len(fused) != 2 {
		t.Fatalf("fused = %v, want the two deduped rows", fused)
	}
	// Both carry the both-arms score; the tie resolves by id.
	if fused[0].assetID != a || fused[1].assetID != b {
		t.Errorf("fused order = %v, want the id tiebreak", fused)
	}
}

// TestFuseTiesBreakByIDAsc pins the tiebreak: equal scores order by
// the asset id ascending, a total order that makes paging stable.
func TestFuseTiesBreakByIDAsc(t *testing.T) {
	b := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	a := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	c := uuid.MustParse("00000000-0000-0000-0000-0000000000cc")

	// Three one-hit arms: every asset carries the same score (the
	// top of its own arm), so the tie is genuine.
	fused := fuse(fuseHits(b), fuseHits(a), fuseHits(c))
	wantOrder := []uuid.UUID{a, b, c}
	for i, want := range wantOrder {
		if fused[i].assetID != want {
			t.Errorf("fused[%d] = %s, want %s (id ascending)", i, fused[i].assetID, want)
		}
	}
}

// TestFuseSingleArmStillRanks pins the one-arm case: the vector arm
// alone (the text arm empty) still produces a fully-ranked page.
func TestFuseSingleArmStillRanks(t *testing.T) {
	a := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	b := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")

	fused := fuse(nil, fuseHits(a, b))
	if len(fused) != 2 || fused[0].assetID != a || fused[1].assetID != b {
		t.Errorf("fused = %v, want [a b] (a single arm still ranks)", fused)
	}
}
