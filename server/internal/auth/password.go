// Package auth provides credential hashing for local lumemor accounts.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// The Argon2id profile is pinned for the first release. Verify accepts
// only hashes with exactly these parameters, so a stored credential can
// never drag an unexpected cost profile into a verification call.
// Parameter agility (rehash-on-login for stronger profiles) is deferred
// to the credential upgrade flow of a later milestone.
const (
	argon2Version = argon2.Version // 19

	argon2MemoryKiB = 64 * 1024 // 64 MiB
	argon2Time      = 3
	argon2Threads   = 1

	saltBytes = 16
	keyBytes  = 32
)

// Hash derives the Argon2id hash of password with a fresh random salt
// and returns it as a PHC string:
// $argon2id$v=19$m=65536,t=3,p=1$<b64salt>$<b64key>, with unpadded
// standard base64 segments.
func Hash(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt entropy: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2MemoryKiB, argon2Threads, keyBytes)
	return phcPrefix() +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key), nil
}

// Verify reports whether password matches the Argon2id PHC string hash.
// It returns false — without panicking — for any malformed hash and for
// hashes whose profile differs from the pinned parameters.
func Verify(hash, password string) bool {
	salt, key, ok := parsePHC(hash)
	if !ok {
		return false
	}
	derived := argon2.IDKey([]byte(password), salt, argon2Time, argon2MemoryKiB, argon2Threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(derived, key) == 1
}

// parsePHC splits an Argon2id PHC string into its salt and key segments,
// accepting only the pinned parameter profile. Base64 segments decode
// strictly and must carry exactly the pinned salt and key lengths.
func parsePHC(hash string) (salt, key []byte, ok bool) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" {
		return nil, nil, false
	}
	if parts[1] != "argon2id" {
		return nil, nil, false
	}
	if parts[2] != "v="+strconv.Itoa(argon2Version) {
		return nil, nil, false
	}
	if parts[3] != parameterBlock() {
		return nil, nil, false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != saltBytes {
		return nil, nil, false
	}
	key, err = base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) != keyBytes {
		return nil, nil, false
	}
	return salt, key, true
}

// phcPrefix renders the fixed head of every hash this package produces,
// derived from the pinned constants so the string and the key derivation
// cannot drift apart.
func phcPrefix() string {
	return fmt.Sprintf("$argon2id$v=%d$%s$", argon2Version, parameterBlock())
}

// parameterBlock renders the pinned Argon2id parameters in PHC order.
func parameterBlock() string {
	return fmt.Sprintf("m=%d,t=%d,p=%d", argon2MemoryKiB, argon2Time, argon2Threads)
}
