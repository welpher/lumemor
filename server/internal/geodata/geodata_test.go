// Dataset loading tests: the bake's three files parse into the
// places/place_names rows exactly, with the locale discipline the
// data model pins -- the en row of every city AND every admin1
// entity comes from the canonical name (one uniform rule, no
// special casing), the zh-Hans row comes from the alternateNames
// dump filtered by language, and within one geoname's zh rows the
// isPreferredName entry wins (file order breaks ties). The version
// marker is a digest of the exact loaded bytes: identical files
// answer identical versions, any byte change answers a different
// one, and the order of the files inside the digest is fixed.
package geodata

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// loadFixtureDataset loads the committed testdata directory.
func loadFixtureDataset(t *testing.T) *Dataset {
	t.Helper()
	ds, err := Load(filepath.Join("testdata"))
	if err != nil {
		t.Fatalf("Load(testdata): %v", err)
	}
	return ds
}

// wantPlace is one expected places row.
type wantPlace struct {
	id          int32
	name        string
	countryCode string
	admin1Code  string
	timezone    string
	lat, lon    float64
}

// TestLoadParsesPlaces pins the cities500 mapping: geonameid, the
// canonical name, the ISO country code, the FULL admin1 code
// ("<country>.<admin1>"), the IANA time zone and the centroid.
func TestLoadParsesPlaces(t *testing.T) {
	ds := loadFixtureDataset(t)
	want := []wantPlace{
		{id: 3038832, name: "Vila", countryCode: "AD", admin1Code: "AD.06", timezone: "Europe/Andorra", lat: 42.53176, lon: 1.56654},
		{id: 2988507, name: "Paris", countryCode: "FR", admin1Code: "FR.11", timezone: "Europe/Paris", lat: 48.85341, lon: 2.3488},
		{id: 1816670, name: "Beijing", countryCode: "CN", admin1Code: "CN.22", timezone: "Asia/Shanghai", lat: 39.9075, lon: 116.39723},
		{id: 2208623, name: "Nowhere", countryCode: "NZ", admin1Code: "NZ.N", timezone: "Pacific/Auckland", lat: -45.03333, lon: 166.48333},
	}
	if len(ds.Places) != len(want) {
		t.Fatalf("Places = %d rows, want %d", len(ds.Places), len(want))
	}
	for i, exp := range want {
		got := ds.Places[i]
		if got.ID != exp.id || got.Name != exp.name || got.CountryCode != exp.countryCode ||
			got.Admin1Code != exp.admin1Code || got.Timezone != exp.timezone ||
			got.Lat != exp.lat || got.Lon != exp.lon {
			t.Errorf("Places[%d] = %+v, want %+v", i, got, exp)
		}
	}
}

// TestLoadBuildsPlaceNames pins the locale discipline: the en row of
// every city and every admin1 entity is the canonical name; the
// zh-Hans row is the alternateNames pick -- the preferred entry of
// the geoname when GeoNames marks one, else the first entry in file
// order; empty names, other languages and unknown geoname ids never
// produce rows.
func TestLoadBuildsPlaceNames(t *testing.T) {
	ds := loadFixtureDataset(t)

	got := map[[2]string]string{} // (geoname id, locale) -> name
	for _, name := range ds.Names {
		key := [2]string{strconv.Itoa(int(name.GeonameID)), name.Locale}
		if _, dup := got[key]; dup {
			t.Fatalf("duplicate place_names row for geoname %d locale %s", name.GeonameID, name.Locale)
		}
		got[key] = name.Name
	}

	// The en rows: canonical names of the four cities plus the three
	// admin1 entities with real geonameids (the gid-0 row is skipped).
	for key, want := range map[[2]string]string{
		{"3038832", LocaleEn}: "Vila",
		{"2988507", LocaleEn}: "Paris",
		{"1816670", LocaleEn}: "Beijing",
		{"2208623", LocaleEn}: "Nowhere",
		{"3039162", LocaleEn}: "Sant Julià de Loria",
		{"3012874", LocaleEn}: "Île-de-France",
		{"2038349", LocaleEn}: "Beijing Shi",
	} {
		if got[key] != want {
			t.Errorf("place_names[%v] = %q, want %q", key, got[key], want)
		}
	}

	// The zh-Hans rows: preferred beats file order (Paris), file
	// order breaks a no-preferred tie (Beijing), the admin1 entity
	// joins through the same rule, and the empty-name, en-language
	// and unknown-id rows never appear.
	for key, want := range map[[2]string]string{
		{"2988507", LocaleZhHans}: "巴黎",
		{"1816670", LocaleZhHans}: "北京",
		{"3012874", LocaleZhHans}: "法兰西岛",
	} {
		if got[key] != want {
			t.Errorf("place_names[%v] = %q, want %q", key, got[key], want)
		}
	}

	// Rows that must not exist: Vila's empty zh name, the unknown
	// geoname id, and the admin1 gid 0. (The en alternate of Paris
	// is already pinned above: the canonical en row wins.)
	for _, key := range [][2]string{
		{"3038832", LocaleZhHans},
		{"9999999", LocaleZhHans},
		{"0", LocaleEn},
	} {
		if name, exists := got[key]; exists {
			t.Errorf("place_names carries an unexpected row for %v: %q", key, name)
		}
	}

	if len(ds.Names) != 10 { // 4 city en + 3 admin1 en + 3 zh
		t.Errorf("Names = %d rows, want 10: %+v", len(ds.Names), ds.Names)
	}
}

// TestLoadVersionIsByteDigest pins the data-version marker: the
// digest of the three loaded files, stable for identical bytes and
// different for any change -- and the file order inside the digest
// is fixed (cities, admin1, alternates).
func TestLoadVersionIsByteDigest(t *testing.T) {
	ds := loadFixtureDataset(t)
	if len(ds.Version) != 64 {
		t.Fatalf("Version = %q, want a 64-character sha256 hex digest", ds.Version)
	}

	// A byte-for-byte copy of the directory answers the same version.
	copyDir := t.TempDir()
	for _, name := range []string{CitiesFile, Admin1File, AlternateZhFile} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(copyDir, name), data, 0o644); err != nil {
			t.Fatalf("copy fixture %s: %v", name, err)
		}
	}
	again, err := Load(copyDir)
	if err != nil {
		t.Fatalf("Load(copy): %v", err)
	}
	if again.Version != ds.Version {
		t.Errorf("Version of the identical copy = %q, want %q", again.Version, ds.Version)
	}

	// One changed byte in the alternates file answers a different
	// version (the marker detects a data update).
	data, err := os.ReadFile(filepath.Join("testdata", AlternateZhFile))
	if err != nil {
		t.Fatalf("read alternates fixture: %v", err)
	}
	data[0] ^= 0x01
	if err := os.WriteFile(filepath.Join(copyDir, AlternateZhFile), data, 0o644); err != nil {
		t.Fatalf("mutate alternates fixture: %v", err)
	}
	mutated, err := Load(copyDir)
	if err != nil {
		t.Fatalf("Load(mutated): %v", err)
	}
	if mutated.Version == ds.Version {
		t.Error("Version ignored a changed byte in the alternates file")
	}
}

// TestLoadRejectsMissingFiles pins the loud failure: a directory
// without the three files is a deployment error, not an empty
// dataset.
func TestLoadRejectsMissingFiles(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("Load of an empty directory succeeded, want an error")
	}
	if _, err := Load(filepath.Join("testdata", "does-not-exist")); err == nil {
		t.Fatal("Load of a missing directory succeeded, want an error")
	}
}
