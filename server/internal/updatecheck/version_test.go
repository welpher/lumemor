// The release version comparison's tests: the tag shapes a release
// feed actually serves, and the honest refusal of everything else.
package updatecheck

import "testing"

// TestParseVersion pins the accepted tag shapes: the optional v
// prefix, the numeric core, a suffix that must not change the order,
// and the shapes that are not comparable versions.
func TestParseVersion(t *testing.T) {
	accepted := []struct {
		tag  string
		want Version
	}{
		{"v0.4.1", Version{0, 4, 1}},
		{"0.4.1", Version{0, 4, 1}},
		{"V1.0", Version{1, 0, 0}},
		{"v1.5.0-rc1", Version{1, 5, 0}},
		{"v1.5.0+build.7", Version{1, 5, 0}},
		{"  v2.0.0  ", Version{2, 0, 0}},
		{"v10.2.31", Version{10, 2, 31}},
	}
	for _, tc := range accepted {
		got, ok := ParseVersion(tc.tag)
		if !ok {
			t.Errorf("ParseVersion(%q) = not comparable, want %v", tc.tag, tc.want)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}

	refused := []string{"", "dev", "latest", "v", "v.", "nightly-2026-11-14", "vNext"}
	for _, tag := range refused {
		if _, ok := ParseVersion(tag); ok {
			t.Errorf("ParseVersion(%q) = comparable, want not comparable", tag)
		}
	}
}

// TestVersionCompare pins the order: the numeric fields, missing
// fields as zero, no claim about a suffix (it was cut at parse time).
func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b Version
		want int
	}{
		{Version{0, 4, 1}, Version{0, 5, 0}, -1},
		{Version{0, 5, 0}, Version{0, 4, 1}, 1},
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{0, 4}, Version{0, 4, 0}, 0},
		{Version{0, 10, 0}, Version{0, 9, 9}, 1},
	}
	for _, tc := range cases {
		if got := tc.a.Compare(tc.b); got != tc.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
