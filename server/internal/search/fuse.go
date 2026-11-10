// The merge rule's pure arithmetic: the deterministic
// two-path rank fusion and the ILIKE wildcard escaping. The rule is
// integer by construction -- no float ties, no rounding, page cursors
// carry exact values.
package search

import (
	"bytes"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// compareUUID answers the bytewise order of two uuids -- PostgreSQL's
// uuid comparison order, the order every SQL ORDER BY asset_id uses,
// kept identical in the Go-side tiebreaks.
func compareUUID(a, b uuid.UUID) int {
	return bytes.Compare(a[:], b[:])
}

// escapeLike makes the query a LITERAL substring for ILIKE: the
// pattern wildcards (% _) and the escape character itself lose their
// pattern meaning (backslash first, then the wildcards, in one pass
// so nothing double-escapes). The contract pins this: the search
// offers no query syntax, the query matches itself verbatim.
func escapeLike(query string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
}

// rankedHit is one asset as one arm ranked it, or as the fusion
// scored it. The arm's position in its slice (not the similarity
// value) is what the fusion consumes: the similarity decided the
// arm's ORDER, the order is the rank.
type rankedHit struct {
	assetID uuid.UUID
	// score is the fused score (arm depth minus the summed
	// positions); zero on the arm input rows.
	score int64
}

// fuse merges the arms' ranked hit lists into the one deterministic
// page order: every asset hit by an arm at zero-based position p
// gains (armDepth - p) from that arm; the fused score is the sum over
// the arms that hit the asset; the result orders by (score DESC,
// asset id ASC). An asset hit by both arms ranks by its combined
// standing, dedup happens by construction (one row per asset), and
// the same inputs answer the same slice: integer sums over fixed
// positions, no ordering ambiguity left.
func fuse(arms ...[]rankedHit) []rankedHit {
	scores := make(map[uuid.UUID]*rankedHit)
	for _, arm := range arms {
		for position, hit := range arm {
			contribution := int64(armDepth - position)
			if existing, ok := scores[hit.assetID]; ok {
				existing.score += contribution
				continue
			}
			scores[hit.assetID] = &rankedHit{assetID: hit.assetID, score: contribution}
		}
	}

	fused := make([]rankedHit, 0, len(scores))
	for _, hit := range scores {
		fused = append(fused, *hit)
	}
	sort.Slice(fused, func(i, j int) bool {
		if fused[i].score != fused[j].score {
			return fused[i].score > fused[j].score
		}
		return compareUUID(fused[i].assetID, fused[j].assetID) < 0
	})
	return fused
}
