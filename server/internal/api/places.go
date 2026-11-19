// The places surface: the caller's city grouping behind
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/geodata"
)

const (
	// placesRerunJobKind is the job_runs kind the re-resolution runs
	// record -- the jobs package's pinned vocabulary, mirrored here
	// the way the model rerun kinds mirror theirs (the endpoint's
	// in-flight join and the rerun test's seeded row drive the two
	// against each other, so they cannot drift apart).
	placesRerunJobKind = "places_rerun"

	// The re-resolution modes, the contract's enum -- the jobs
	// package's pinned vocabulary, mirrored here for the same reason
	// as the kind: the endpoint validates the request against its
	// own copy and passes the value through the enqueue seam.
	placesRerunModeMissing = "missing"
	placesRerunModeAll     = "all"
)

// resolveManualPlace resolves the manual location edit's place: the
// grid's nearest city within the threshold, or NULL (the honest
// absence -- the point sits beyond every city). No grid (a
// deployment without the baked place data) answers the invalid
// value, which leaves the resolved place whatever the asset already
// carried -- the metadata step's own feature-off discipline: a
// data-less boot never destroys a resolution a data-carrying boot
// produced.
func (s *server) resolveManualPlace(ctx context.Context, lat, lon float64) pgtype.Int4 {
	if s.places == nil {
		return pgtype.Int4{}
	}
	if id, _, ok := s.places.NearestWithin(lat, lon, geodata.ResolutionRadiusKm); ok {
		return pgtype.Int4{Int32: id, Valid: true}
	}
	return pgtype.Int4{}
}

// ListPlaces implements GET /api/v1/places: the caller's city
// grouping -- every place holding at least one caller-visible live
// asset, ordered by the latest capture time descending (the most
// recently photographed city first; the place id ascending as the
// tiebreak). Names resolve through the requested locale's
// place_names row with the canonical name as the fallback; the covers
// are each place's four most recent caller-visible live members, the
// wall's derived-cover discipline, and the span carries the members'
// earliest and latest capture times.
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
	locale := apigen.ListPlacesParamsLocaleEn
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
			// reads. The place no longer holds a visible member, so
			// it leaves the listing exactly as the next request's
			// grouping would leave it -- an honest stale-read drop,
			// never a place card without a cover.
			continue
		}
		places = append(places, apigen.PlaceSummary{
			PlaceId: row.ID,
			Name:    row.Name,
			Country: row.CountryCode,
			// The place record's own coordinate: the map face pins it,
			// and it is GeoNames' city point -- never a photo's.
			Latitude:                    row.Lat,
			Longitude:                   row.Lon,
			AssetCount:                  int(row.AssetCount),
			TakenAtFrom:                 row.TakenAtFrom.Time.UTC(),
			TakenAtTo:                   row.TakenAtTo.Time.UTC(),
			RepresentativeCoverAssetIds: candidates,
		})
	}

	s.writeJSON(w, http.StatusOK, places)
}

// GetPlacesDatabase implements GET /api/v1/admin/places: the map &
// places card's read -- the loaded place database's data-version
// marker and locale set, plus the two re-resolution scopes' sizes.
//
// The marker is whatever the startup import recorded (an empty string
// when the instance never imported a bake; the card renders its first
// eight characters when it is a content digest). The locales are the
// baked set's constants -- the display-name join the places read
// performs, not a per-instance setting. The counts are the walk's
// scope sizes, library-wide (place resolution is metadata, not
// visibility): every GPS-bearing asset row the pipeline owns, and the
// subset whose place is absent. A hand-located row is outside both --
// the manual decision owns it and no walk rewrites it -- so a count
// never promises work the walk will not do.
func (s *server) GetPlacesDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	version, err := s.queries.GetGeodataVersion(ctx, geodata.VersionSettingKey)
	if errors.Is(err, pgx.ErrNoRows) {
		// No bake ever imported: the honest empty marker.
		version = ""
	} else if err != nil {
		s.internalError(w, r, "read the place database marker", err)
		return
	}
	gpsAssets, err := s.queries.CountGpsAssets(ctx)
	if err != nil {
		s.internalError(w, r, "count the GPS-bearing assets", err)
		return
	}
	missingPlaces, err := s.queries.CountMissingPlaceAssets(ctx)
	if err != nil {
		s.internalError(w, r, "count the assets without a place", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.PlacesDatabaseResponse{
		Version:       version,
		Languages:     []string{geodata.LocaleEn, geodata.LocaleZhHans},
		GpsAssets:     int(gpsAssets),
		MissingPlaces: int(missingPlaces),
	})
}

// ReresolvePlaces implements POST /api/v1/admin/places/reresolve:
// one run that re-resolves the place of the assets its mode covers
// against the baked place data the server currently runs on -- the
// places-data-update path. The mode is the contract's enum: missing
// walks only the GPS-bearing rows whose place is absent, all walks
// every GPS-bearing row; an absent body means all, any other value is
// 400. Idempotent the way the model rerun is: a run of the kind still
// in flight answers 200 with its identifier, anything else starts a
// fresh run. The in-flight join is per kind, not per mode -- v1 does
// not queue modes, so a missing run in flight answers an all request
// with 200 as well (the card points at the jobs page).
func (s *server) ReresolvePlaces(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	// The mode: an absent body means the endpoint's original full
	// walk; a present body must name one of the enum's values.
	mode := placesRerunModeAll
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	if len(bytes.TrimSpace(body)) > 0 {
		var req apigen.PlacesRerunRequest
		if err := json.Unmarshal(body, &req); err != nil {
			s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
			return
		}
		if req.Mode != nil {
			if !req.Mode.Valid() {
				s.writeValidationError(w, "mode must be one of missing, all", []string{"mode"})
				return
			}
			mode = string(*req.Mode)
		}
	}

	// A run of the kind still in flight answers 200 with its
	// identifier: the request joins the running re-resolution instead
	// of starting a second one.
	if run, err := s.queries.FindInFlightJobRunByKind(ctx, placesRerunJobKind); err == nil {
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
	jobRunID, err := s.placesReruns.EnqueuePlacesRerun(ctx, mode)
	if err != nil {
		// The queue refuses the enqueue when this server runs without
		// the place reference data (no GeoNames bake, so no grid to
		// resolve against) exactly as it does for a broken
		// transaction; the log line carries which.
		s.logger.Error("enqueue the places re-resolution", "error", err, "mode", mode)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the re-resolution could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.PlacesRerunResponse{JobRunId: jobRunID})
}
