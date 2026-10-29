package auth

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
)

// testPassword is a representative wizard password: 25 runes, inside the
// 8-128 rune window the API enforces.
const testPassword = "correct horse battery staple"

func TestHashVerifyRoundTrip(t *testing.T) {
	hash, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}
	if !Verify(hash, testPassword) {
		t.Errorf("Verify(hash, %q) = false, want true", testPassword)
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hash, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}

	wrong := []string{
		"",
		"Correct horse battery staple",
		"correct horse battery staple!",
	}
	for _, password := range wrong {
		if Verify(hash, password) {
			t.Errorf("Verify(hash, %q) = true, want false", password)
		}
	}
}

// TestVerifyMalformedHash pins the contract that corrupted stored hashes
// never crash the login path: every malformed input answers false, and a
// panic fails the test through the recover guard with a readable message.
func TestVerifyMalformedHash(t *testing.T) {
	hash, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}
	parts := strings.Split(hash, "$") // ["", "argon2id", "v=19", "m=...", salt, key]

	// jumble rebuilds the PHC string with individual segments replaced.
	jumble := func(replacements map[int]string) string {
		segments := slices.Clone(parts)
		for i, replacement := range replacements {
			segments[i] = replacement
		}
		return strings.Join(segments, "$")
	}

	malformed := map[string]string{
		// The mandatory probe: a corrupted PHC string must answer false
		// without panicking.
		"corrupted base64":  "$argon2id$v=19$m=65536,t=3,p=1$!!!corrupted!!!$xxx",
		"empty":             "",
		"plain text":        "not-a-hash",
		"no leading dollar": strings.TrimPrefix(hash, "$"),
		"missing key":       jumble(map[int]string{5: ""}),
		"missing segments":  "$argon2id$v=19$m=65536,t=3,p=1$",
		"wrong algorithm":   jumble(map[int]string{1: "argon2i"}),
		"wrong version":     jumble(map[int]string{2: "v=16"}),
		"wrong memory":      jumble(map[int]string{3: "m=32768,t=3,p=1"}),
		"wrong iterations":  jumble(map[int]string{3: "m=65536,t=2,p=1"}),
		"wrong parallelism": jumble(map[int]string{3: "m=65536,t=3,p=4"}),
		"unknown parameter": jumble(map[int]string{3: "m=65536,t=3,x=1"}),
		"missing parameter": jumble(map[int]string{3: "m=65536,t=3"}),
		"short salt":        jumble(map[int]string{4: "AAAAAAAAAAAAAA"}),
		"short key":         jumble(map[int]string{5: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}),
	}
	for name, malformedHash := range malformed {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Verify panicked on %q: %v", malformedHash, r)
				}
			}()
			if Verify(malformedHash, testPassword) {
				t.Errorf("Verify(%q, ...) = true, want false", malformedHash)
			}
		})
	}
}

// TestHashPHCFormat asserts the stored representation byte-for-byte where
// it is load-bearing: the algorithm tag, version and pinned parameter
// block, and the base64 lengths decoding to a 16-byte salt and 32-byte
// key.
func TestHashPHCFormat(t *testing.T) {
	hash, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}

	const wantPrefix = "$argon2id$v=19$m=65536,t=3,p=1$"
	if !strings.HasPrefix(hash, wantPrefix) {
		t.Fatalf("Hash() = %q, want prefix %q", hash, wantPrefix)
	}

	segments := strings.Split(strings.TrimPrefix(hash, wantPrefix), "$")
	if len(segments) != 2 {
		t.Fatalf("Hash() = %q, want exactly salt and key segments after the parameter block", hash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(segments[0])
	if err != nil {
		t.Fatalf("decode salt segment %q: %v", segments[0], err)
	}
	if len(salt) != 16 {
		t.Errorf("salt length = %d bytes, want 16", len(salt))
	}
	key, err := base64.RawStdEncoding.DecodeString(segments[1])
	if err != nil {
		t.Fatalf("decode key segment %q: %v", segments[1], err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d bytes, want 32", len(key))
	}
}

// TestHashUsesFreshSalt ensures two hashes of the same password differ,
// proving the salt is random per hash rather than derived.
func TestHashUsesFreshSalt(t *testing.T) {
	first, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}
	second, err := Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v, want nil", testPassword, err)
	}
	if first == second {
		t.Errorf("two Hash() calls produced identical output, want per-hash random salts")
	}
}
