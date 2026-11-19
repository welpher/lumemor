// The instance settings' storage leaves: the map's tile sources, and
// their compatibility facts -- an unwritten row and a row written
// before a field existed both read that field's zero value rather
// than fail or invent a source.
package db

import (
	"testing"

	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// TestTileSettingsRoundTrip pins the pair's write/read round-trip and
// the compatibility answers: no row and an old-shaped row both read
// as unconfigured, and the second save replaces both fields.
func TestTileSettingsRoundTrip(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}
	queries := dbgen.New(pool)

	// The shipped state: no row at all.
	unset, err := TileSettingsOf(ctx, queries)
	if err != nil {
		t.Fatalf("TileSettingsOf(unwritten) error = %v, want nil", err)
	}
	if unset != (TileSettings{}) {
		t.Fatalf("TileSettingsOf(unwritten) = %+v, want the zero pair", unset)
	}

	// A row written before the dark field existed: the dark style is
	// the struct's zero value, so the instance keeps one source for
	// both themes. The coordinate flags read false (the standard
	// WGS-84) for the same reason.
	if err := queries.SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   TilesSettingsKey,
		Value: []byte(`{"tilejsonUrl":"https://tiles.example.net/osm/tilejson.json"}`),
	}); err != nil {
		t.Fatalf("write the old-shaped row: %v", err)
	}
	legacy, err := TileSettingsOf(ctx, queries)
	if err != nil {
		t.Fatalf("TileSettingsOf(old row) error = %v, want nil", err)
	}
	want := TileSettings{TileJSONURL: "https://tiles.example.net/osm/tilejson.json"}
	if legacy != want {
		t.Fatalf("TileSettingsOf(old row) = %+v, want %+v", legacy, want)
	}
	if legacy.GCJ02 || legacy.DarkGCJ02 {
		t.Fatalf("TileSettingsOf(old row) flags = %v/%v, want both false", legacy.GCJ02, legacy.DarkGCJ02)
	}

	// The pair survives the round-trip verbatim, the coordinate
	// flags included; the two sources may carry different datums.
	pair := TileSettings{
		TileJSONURL:     "https://tiles.example.net/osm/tilejson.json",
		DarkTileJSONURL: "https://tiles.example.net/osm-dark/tilejson.json",
		GCJ02:           true,
		DarkGCJ02:       false,
	}
	if err := SaveTileSettings(ctx, queries, pair); err != nil {
		t.Fatalf("SaveTileSettings() error = %v, want nil", err)
	}
	saved, err := TileSettingsOf(ctx, queries)
	if err != nil {
		t.Fatalf("TileSettingsOf(saved) error = %v, want nil", err)
	}
	if saved != pair {
		t.Fatalf("TileSettingsOf(saved) = %+v, want %+v", saved, pair)
	}

	// The save replaces the document: clearing the dark style clears
	// it rather than leaving the old one behind, and a flag that is
	// no longer saved reads false.
	if err := SaveTileSettings(ctx, queries, TileSettings{TileJSONURL: pair.TileJSONURL}); err != nil {
		t.Fatalf("SaveTileSettings(light only) error = %v, want nil", err)
	}
	cleared, err := TileSettingsOf(ctx, queries)
	if err != nil {
		t.Fatalf("TileSettingsOf(cleared) error = %v, want nil", err)
	}
	if cleared != (TileSettings{TileJSONURL: pair.TileJSONURL}) {
		t.Fatalf("TileSettingsOf(cleared) = %+v, want the light source alone", cleared)
	}
}

// TestTileSettingsCorruptRowIsUnconfigured pins the corruption
// answer: a row that is not a JSON object reads as unconfigured with
// an error, never as a source the browser would load.
func TestTileSettingsCorruptRowIsUnconfigured(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}
	queries := dbgen.New(pool)
	if err := queries.SetInstanceSettingRaw(ctx, dbgen.SetInstanceSettingRawParams{
		Key:   TilesSettingsKey,
		Value: []byte(`[1,2,3]`),
	}); err != nil {
		t.Fatalf("write the corrupt row: %v", err)
	}
	settings, err := TileSettingsOf(ctx, queries)
	if err == nil {
		t.Fatal("TileSettingsOf(corrupt row) error = nil, want the corruption reported")
	}
	if settings != (TileSettings{}) {
		t.Fatalf("TileSettingsOf(corrupt row) = %+v, want the zero pair", settings)
	}
}
