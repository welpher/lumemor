// Package geodata carries the place reference-data plane:
// the GeoNames bake's loader, its idempotent database import,
// and the in-memory grid index the GPS resolution runs against.
//
// The distribution discipline is build-time: the Docker image
// downloads the three GeoNames dumps (cities500, admin1CodesASCII,
// and the zh alternate names of exactly those geoname ids), verifies
// them by sha256, and stages them under /usr/local/share/lumemor/geo
// -- the runtime performs ZERO network access. The startup load is
// idempotent through a data-version marker: the version is a sha256
// digest of the exact loaded bytes, stored in instance_settings; an
// unchanged dataset (the normal reboot) imports nothing, a changed
// one (a rebuilt image) replaces places/place_names atomically --
// and because assets.place_id is ON DELETE SET NULL, the replacement
// releases every stale resolution, the state the admin
// re-resolution run then refills.
//
// The locale discipline follows the data model: place_names carries
// one row per (geoname id, locale) for the UI's language set -- the
// en row of every city AND every admin1 entity comes from the
// canonical name (one uniform rule, no special casing), the zh-Hans
// row comes from the alternateNames dump filtered by language, where
// the isPreferredName entry of a geoname wins and file order breaks
// ties. Display names join at read time; the assets rows never carry
// locale strings, so a locale switch touches no asset row.
package geodata

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The file names of the bake, as staged under the geo dir.
const (
	// CitiesFile is the GeoNames cities500 dump: every populated
	// place with population >= 500, one tab-separated row per city.
	CitiesFile = "cities500.txt"

	// Admin1File is the GeoNames admin1CodesASCII dump: the
	// first-level administrative divisions, "<country>.<code>"
	// keyed, each row carrying the division's own geonameid.
	Admin1File = "admin1CodesASCII.txt"

	// AlternateZhFile is the build-side filtered subset of the
	// GeoNames alternateNames dump: the zh (Simplified Chinese)
	// rows of exactly the geoname ids the two files above carry.
	// The loader re-applies both filters, so an unfiltered dump
	// parses to the same dataset.
	AlternateZhFile = "alternateNames-zh.txt"
)

// The locales place_names carries, mirroring the UI's language set
// (the i18n table's file names): en is the source language and
// fallback, zh-Hans the Chinese translation. The GeoNames zh rows
// (language code "zh") are the Simplified Chinese corpus and map
// onto the zh-Hans UI locale.
const (
	LocaleEn     = "en"
	LocaleZhHans = "zh-Hans"

	// geoNamesZhLanguage is the GeoNames language code whose rows
	// feed the zh-Hans locale.
	geoNamesZhLanguage = "zh"
)

// ResolutionRadiusKm is the reverse-geocoding threshold the data
// model pins: the nearest city within this radius resolves the
// asset's place; a farther one or no GPS at all leaves place_id
// NULL (no country-polygon fallback).
const ResolutionRadiusKm = 25.0

// Place is one places row: a GeoNames city centroid. The place
// tables carry no audit columns (the data model exempts them:
// read-only imports, no application-side updates).
type Place struct {
	// ID is the GeoNames geonameid.
	ID int32
	// Name is the canonical (English or romanized) name -- the
	// display fallback when no place_names row matches.
	Name string
	// CountryCode is the ISO 3166-1 alpha-2 code.
	CountryCode string
	// Admin1Code is the full first-level division code, "CN.30".
	Admin1Code string
	// Timezone is the IANA zone name.
	Timezone string
	// Lat and Lon are the centroid's coordinates, within the
	// schema's CHECK bounds.
	Lat float64
	Lon float64
}

// PlaceName is one place_names row: a geoname id's display name in
// one locale. Geoname ids may address admin1 entities that carry no
// places row, so there is deliberately no foreign key.
type PlaceName struct {
	GeonameID int32
	Locale    string
	Name      string
}

// Dataset is one loaded bake: the rows the import writes, plus the
// data-version marker (a sha256 digest of the exact loaded bytes in
// fixed file order: cities, admin1, alternates).
type Dataset struct {
	Version string
	Places  []Place
	Names   []PlaceName
}

// Load reads and parses the bake under dir. A missing file or a
// malformed row is an error, not an empty dataset: the directory is
// sha256-pinned at build time, so anything unreadable here is a
// deployment or pin problem the boot must surface.
func Load(dir string) (*Dataset, error) {
	citiesBytes, err := os.ReadFile(filepath.Join(dir, CitiesFile))
	if err != nil {
		return nil, fmt.Errorf("geodata: read %s: %w", CitiesFile, err)
	}
	admin1Bytes, err := os.ReadFile(filepath.Join(dir, Admin1File))
	if err != nil {
		return nil, fmt.Errorf("geodata: read %s: %w", Admin1File, err)
	}
	alternateBytes, err := os.ReadFile(filepath.Join(dir, AlternateZhFile))
	if err != nil {
		return nil, fmt.Errorf("geodata: read %s: %w", AlternateZhFile, err)
	}

	digest := sha256.New()
	for _, blob := range [][]byte{citiesBytes, admin1Bytes, alternateBytes} {
		digest.Write(blob)
	}

	places, err := parseCities(citiesBytes)
	if err != nil {
		return nil, err
	}
	admin1, err := parseAdmin1(admin1Bytes)
	if err != nil {
		return nil, err
	}
	names := buildPlaceNames(places, admin1, alternateBytes)

	return &Dataset{
		Version: hex.EncodeToString(digest.Sum(nil)),
		Places:  places,
		Names:   names,
	}, nil
}

// admin1Division is one admin1CodesASCII row.
type admin1Division struct {
	// Code is the full division code, "CN.30".
	Code string
	// Name is the canonical (UTF-8) division name.
	Name string
	// GeonameID is the division's own GeoNames id (the place_names
	// key for its locale rows); rows without a usable id are
	// skipped.
	GeonameID int32
}

// parseCities maps the cities500 dump onto places rows. The column
// layout (19 tab-separated fields, 0-based indices):
//
//	0 geonameid            1 name (canonical, UTF-8)
//	4 latitude             5 longitude
//	8 country code        10 admin1 code (division part)
//	17 timezone
func parseCities(data []byte) ([]Place, error) {
	var places []Place
	seen := make(map[int32]bool)
	err := forEachRow(data, func(line int, fields []string) error {
		if len(fields) < 18 {
			return fmt.Errorf("geodata: %s row %d carries %d fields, want at least 18", CitiesFile, line, len(fields))
		}
		id, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 32)
		if err != nil || id <= 0 {
			return fmt.Errorf("geodata: %s row %d carries an invalid geonameid %q", CitiesFile, line, fields[0])
		}
		name := fields[1]
		if name == "" {
			return fmt.Errorf("geodata: %s row %d carries an empty canonical name", CitiesFile, line)
		}
		lat, latErr := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		lon, lonErr := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
		if latErr != nil || lonErr != nil || math.Abs(lat) > 90 || math.Abs(lon) > 180 {
			return fmt.Errorf("geodata: %s row %d carries invalid coordinates (%q, %q)", CitiesFile, line, fields[4], fields[5])
		}
		if seen[int32(id)] {
			return fmt.Errorf("geodata: %s row %d repeats geonameid %d", CitiesFile, line, id)
		}
		seen[int32(id)] = true
		admin1Part := fields[10]
		admin1Code := ""
		if admin1Part != "" {
			admin1Code = fields[8] + "." + admin1Part
		}
		places = append(places, Place{
			ID:          int32(id),
			Name:        name,
			CountryCode: fields[8],
			Admin1Code:  admin1Code,
			Timezone:    fields[17],
			Lat:         lat,
			Lon:         lon,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(places) == 0 {
		return nil, fmt.Errorf("geodata: %s carries no cities", CitiesFile)
	}
	return places, nil
}

// parseAdmin1 maps the admin1CodesASCII dump onto the division
// table. The column layout (4 tab-separated fields): the full code,
// the canonical name, the ASCII name, and the division's geonameid.
func parseAdmin1(data []byte) ([]admin1Division, error) {
	var divisions []admin1Division
	err := forEachRow(data, func(line int, fields []string) error {
		if len(fields) < 4 {
			return fmt.Errorf("geodata: %s row %d carries %d fields, want 4", Admin1File, line, len(fields))
		}
		id, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 32)
		if err != nil || id <= 0 {
			// Rows without a usable geonameid cannot join
			// place_names; skip them (the dump's real rows all
			// carry ids, this only guards pin mistakes).
			return nil
		}
		if fields[1] == "" {
			return fmt.Errorf("geodata: %s row %d carries an empty canonical name", Admin1File, line)
		}
		divisions = append(divisions, admin1Division{
			Code:      fields[0],
			Name:      fields[1],
			GeonameID: int32(id),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return divisions, nil
}

// buildPlaceNames derives the place_names rows: the en row of every
// city and every admin1 entity from the canonical name (the uniform
// rule -- no special casing, an English alternate never overrides
// it), then the zh-Hans rows from the alternateNames dump: the zh
// rows of exactly the known geoname ids, one row per id, the
// isPreferredName entry winning and file order breaking ties.
func buildPlaceNames(places []Place, admin1 []admin1Division, alternateBytes []byte) []PlaceName {
	known := make(map[int32]bool, len(places)+len(admin1))
	names := make([]PlaceName, 0, len(places)+len(admin1))
	for _, place := range places {
		known[place.ID] = true
		names = append(names, PlaceName{GeonameID: place.ID, Locale: LocaleEn, Name: place.Name})
	}
	for _, division := range admin1 {
		known[division.GeonameID] = true
		names = append(names, PlaceName{GeonameID: division.GeonameID, Locale: LocaleEn, Name: division.Name})
	}

	// The zh picks, in first-occurrence file order: one candidate
	// per geoname id, a preferred entry replacing a non-preferred
	// incumbent (never the reverse, and never a same-class
	// replacement -- the earlier row keeps the slot).
	type candidate struct {
		name      string
		preferred bool
	}
	picked := make(map[int32]candidate)
	var order []int32
	_ = forEachRow(alternateBytes, func(line int, fields []string) error {
		if len(fields) < 4 || fields[2] != geoNamesZhLanguage {
			return nil // the language filter: the en locale is canonical, other languages are not baked
		}
		id, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 32)
		if err != nil || id <= 0 || !known[int32(id)] {
			return nil // the id filter: rows for unknown entities never join
		}
		name := fields[3]
		if name == "" {
			return nil // an empty name row is not a name
		}
		gid := int32(id)
		preferred := len(fields) >= 5 && fields[4] == "1"
		current, exists := picked[gid]
		if !exists {
			order = append(order, gid)
			picked[gid] = candidate{name: name, preferred: preferred}
		} else if preferred && !current.preferred {
			picked[gid] = candidate{name: name, preferred: preferred}
		}
		return nil
	})
	for _, gid := range order {
		names = append(names, PlaceName{GeonameID: gid, Locale: LocaleZhHans, Name: picked[gid].name})
	}
	return names
}

// forEachRow splits tab-separated rows and calls fn per non-empty
// row with its 1-based row number and split fields, stopping at the
// first error.
func forEachRow(data []byte, fn func(line int, fields []string) error) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// GeoNames rows are short (a few hundred bytes); the generous
	// buffer only guards a pathological pin.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		row := scanner.Bytes()
		if len(row) == 0 {
			continue
		}
		if err := fn(line, strings.Split(string(row), "\t")); err != nil {
			return err
		}
	}
	return scanner.Err()
}
