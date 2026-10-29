// Pairing token credentials: the secret behind the add-a-device QR
// code. A pairing token is deliberately shorter-lived and shorter than
// a session credential -- a QR code must stay scannable -- but it is
// stored under the same SHA-256 hash, so a leaked database can redeem
// neither.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// pairingTokenBytes is the entropy of one pairing token: 16 bytes
// (128 bits) of crypto/rand. That stays far beyond any guessing budget
// against a five-minute, one-time window while the hex form stays
// compact enough for a QR code.
const pairingTokenBytes = 16

// NewPairingToken mints a pairing token: it returns the raw token (16
// random bytes, hex-encoded) together with the SHA-256 hex hash of
// those bytes that the pairing_tokens table stores.
//
// Like NewSessionToken, a failing CSPRNG panics loudly instead of
// silently weakening a credential: a process without working entropy
// must not mint pairing secrets.
func NewPairingToken() (raw string, hash string) {
	buf := make([]byte, pairingTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("lumemor/auth: read pairing token entropy: %v", err))
	}
	raw = hex.EncodeToString(buf)
	return raw, HashPairingToken(raw)
}

// HashPairingToken derives the stored form of a raw pairing token. It
// is deliberately the same derivation as HashSessionToken: both
// credentials are opaque random hex strings, and one helper keeps the
// hashing policy a single point of change.
func HashPairingToken(raw string) string {
	return HashSessionToken(raw)
}
