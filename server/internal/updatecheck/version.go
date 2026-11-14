// The release version comparison: tags only. A tag is compared as
// the numeric fields of its prefix, so `v1.4.0` and `1.4.0` are the
// same release and a pre-release suffix (`v1.5.0-rc1`) compares as
// the release it leads to. Anything that does not start with a
// numeric field is not a comparable version, and the caller reports
// the honest unknown instead of inventing an order.
package updatecheck

import (
	"strconv"
	"strings"
)

// Version is one comparable version: the up-to-three numeric fields
// of a tag, missing fields counting as zero.
type Version [3]int

// ParseVersion parses a release tag. It accepts an optional `v`/`V`
// prefix, then a dotted numeric prefix; it stops at the first
// character that is neither a digit nor a dot, so a suffix
// (`-rc1`, `+build.5`) does not change the order. The trailing dot
// and empty fields the prefix may leave (`v1.`) are ignored.
func ParseVersion(tag string) (Version, bool) {
	trimmed := strings.TrimSpace(tag)
	trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "v"), "V")
	if trimmed == "" {
		return Version{}, false
	}
	end := 0
	for end < len(trimmed) {
		c := trimmed[end]
		if (c < '0' || c > '9') && c != '.' {
			break
		}
		end++
	}
	core := trimmed[:end]
	core = strings.TrimSuffix(core, ".")
	if core == "" {
		return Version{}, false
	}
	parts := strings.Split(core, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}
	var version Version
	for i, part := range parts {
		if part == "" {
			return Version{}, false
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, false
		}
		version[i] = value
	}
	return version, true
}

// Compare orders two versions: negative when a is older, zero when
// they are the same release, positive when a is newer.
func (a Version) Compare(b Version) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}
