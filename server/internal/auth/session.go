// Session credentials: the raw token the client presents and the hash
// the database stores. Only the hash is persisted, so a leaked database
// cannot mint usable sessions; the raw token exists exactly twice --
// once in the Set-Cookie / bearer response, once in the client.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// sessionTokenBytes is the entropy of one session credential: 32 bytes
// (256 bits) of crypto/rand, enough that guessing is hopeless while the
// hex form stays URL- and cookie-safe.
const sessionTokenBytes = 32

// NewSessionToken mints a session credential: it returns the raw token
// (32 random bytes, hex-encoded) together with the SHA-256 hex hash of
// those bytes that the sessions table stores.
//
// The two-value signature is pinned by the plan. crypto/rand only fails
// when the system CSPRNG itself is broken; like uuid.New for the same
// reason, this panics loudly rather than silently weakening a
// credential -- a process without working entropy must not serve
// authenticated traffic.
func NewSessionToken() (raw string, hash string) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("lumemor/auth: read session token entropy: %v", err))
	}
	raw = hex.EncodeToString(buf)
	return raw, HashSessionToken(raw)
}

// HashSessionToken derives the stored form of a raw session token:
// the lowercase hex SHA-256 of its bytes. The middleware uses it to
// look up the session behind a presented cookie or bearer credential,
// exactly the way the token was stored at mint time.
func HashSessionToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
