// The places surface (M3d Task 1): the caller's city grouping behind
// GET /places and the admin re-resolution trigger behind
// POST /admin/places/reresolve.
//
// The read's domain is the caller-visible asset union (the same
// caller_asset_visible predicate every count-bearing surface of the
// server applies): a place lists when it holds at least one
// caller-visible live asset, the count is honest per caller, and the
// covers are the place's own visible members -- never the global
// newest, which could sit on an asset this caller cannot see (the
// people surface's card-crop discipline, applied to cities).
//
// Display names are the locale join: place_names carries one row per
// (geoname id, locale) and the request's locale picks it, the
// canonical name answering the fallback. The locale is a REQUEST
// parameter (the web app owns the active locale and passes it
// along); the server never guesses it from headers, and the assets
// rows never carry locale strings, so a locale switch touches no
// asset row.
//
// The re-resolution trigger mirrors the model rerun endpoint's
// idempotence: a run of the kind still in flight answers 200 with
// its identifier, anything else starts a fresh run through the queue
// seam. The heavy work -- walking every GPS-bearing asset -- lives
// on the queue (the places_rerun worker of the jobs package), not
// here.
package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// ListPlaces implements GET /api/v1/places: the caller's city
// grouping -- every place holding at least one caller-visible live
// asset, ordered by that count descending. Names resolve through the
// requested locale's place_names row with the canonical name as the
// fallback; the covers are each place's four most recent
// caller-visible live members, the wall's derived-cover discipline.
func (s *server) ListPlaces(w http.ResponseWriter, r *http.Request, params apigen.ListPlacesParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The locale parameter: the contract's enum is the instance's
	// language set, and the generated binder binds the raw string, so
	// the membership check lives here -- an out-of-set locale is a
	// client bug the 400 names.
	locale := apigen.En
	if params.Locale != nil {
		if !params.Locale.Valid() {
			s.writeValidationError(w, "locale must be one of: en, zh-Hans", []string{"locale"})
			return
		}
		locale = *params.Locale
	}

	rows, err := s.queries.ListCallerPlaces(r.Context(), dbgen.ListCallerPlacesParams{
		UserID: pgUUID(requester.userID),
		Locale: string(locale),
	})
	if err != nil {
		s.internalError(w, r, "load places", err)
		return
	}

	// The derived covers: one bounded second read over the listed
	// places' visible members (at most four rows per place, the wall's
	// cover cost class). A listed place always carries at least one
	// member, so the map always fills for every place id.
	placeIDs := make([]int32, 0, len(rows))
	for _, row := range rows {
		placeIDs = append(placeIDs, row.ID)
	}
	covers, err := s.queries.ListPlaceCoverCandidates(r.Context(), dbgen.ListPlaceCoverCandidatesParams{
		UserID:   pgUUID(requester.userID),
		PlaceIds: placeIDs,
	})
	if err != nil {
		s.internalError(w, r, "load place covers", err)
		return
	}
	coverIDs := make(map[int32][]uuid.UUID, len(rows))
	for _, candidate := range covers {
		id := uuid.UUID(candidate.AssetID.Bytes)
		coverIDs[candidate.PlaceID.Int32] = append(coverIDs[candidate.PlaceID.Int32], id)
	}

	places := make([]apigen.PlaceSummary, 0, len(rows))
	for _, row := range rows {
		candidates := coverIDs[row.ID]
		if candidates == nil {
			// The count row proved a visible member; a missing cover
			// here can only mean the member left between the two
			// reads. The empty array keeps the response honest rather
			// than fabricating a cover from an asset the caller may
			// no longer see; the web renders a name tile then.
			candidates = []uuid.UUID{}
		}
		places = append(places, apigen.PlaceSummary{
			PlaceId:                     row.ID,
			Name:                        row.Name,
			Country:                     row.CountryCode,
			AssetCount:                  int(row.AssetCount),
			RepresentativeCoverAssetIds: candidates,
		})
	}

	s.writeJSON(w, http.StatusOK, places)
}

// ReresolvePlaces implements POST /api/v1/admin/places/reresolve:
// one run that re-resolves the place of every GPS-bearing asset
// against the reference data the server currently runs on -- the
// places-data-update path. Idempotent the way the model rerun is: a
// run of the kind still in flight answers 200 with its identifier,
// anything else starts a fresh run. This milestone wires no admin
// page button to it; the job panel lists the run like any other.
func (s *server) ReresolvePlaces(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	// A run of the kind still in flight answers 200 with its
	// identifier: the request joins the running re-resolution instead
	// of starting a second one.
	if run, err := s.queries.FindInFlightJobRunByKind(ctx, dbgen.PlacesRerunKind); err == nil {
		s.writeJSON(w, http.StatusOK, apigen.PlacesRerunResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight places rerun run", err)
		return
	}

	if s.placesReruns == nil {
		s.logger.Error("places re-resolution requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the re-resolution could not be started", nil)
		return
	}
	jobRunID, err := s.placesReruns.EnqueuePlacesRerun(ctx)
	if err != nil {
		// The queue refuses the enqueue when this server runs without
		// the place reference data (no GeoNames bake, so no grid to
		// resolve against) exactly as it does for a broken
		// transaction; the log line carries which.
		s.logger.Error("enqueue the places re-resolution", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the re-resolution could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.PlacesRerunResponse{JobRunId: jobRunID})
}
