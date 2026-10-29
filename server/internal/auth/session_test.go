package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// TestNewSessionTokenShape pins the credential format: 32 bytes of
// crypto/rand entropy hex-encoded as the raw token, plus the SHA-256
// hex hash of exactly those bytes. The database stores only the hash,
// so a leaked database cannot mint usable credentials.
func TestNewSessionTokenShape(t *testing.T) {
	raw, hash := NewSessionToken()

	if len(raw) != 2*sessionTokenBytes {
		t.Fatalf("len(raw) = %d, want %d hex characters", len(raw), 2*sessionTokenBytes)
	}
	if strings.ToLower(raw) != raw {
		t.Errorf("raw token = %q, want lowercase hex", raw)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		t.Fatalf("raw token is not valid hex: %v", err)
	}
	if len(decoded) != sessionTokenBytes {
		t.Errorf("decoded token length = %d, want %d bytes", len(decoded), sessionTokenBytes)
	}

	if len(hash) != 2*sha256.Size {
		t.Fatalf("len(hash) = %d, want %d hex characters", len(hash), 2*sha256.Size)
	}
	// The hash covers the presented credential verbatim: the raw hex
	// string, not its decoded bytes.
	sum := sha256.Sum256([]byte(raw))
	if hash != hex.EncodeToString(sum[:]) {
		t.Errorf("hash = %q, want the SHA-256 hex of the raw token %q", hash, hex.EncodeToString(sum[:]))
	}

	// The stored hash must never equal the credential itself.
	if hash == raw {
		t.Error("hash equals the raw token: the credential would be stored in the clear")
	}
}

// TestNewSessionTokenRandom verifies consecutive tokens are distinct:
// same-user logins (and any two sessions) must never share a credential.
func TestNewSessionTokenRandom(t *testing.T) {
	const calls = 64
	seen := make(map[string]struct{}, calls)
	for i := 0; i < calls; i++ {
		raw, _ := NewSessionToken()
		if _, dup := seen[raw]; dup {
			t.Fatalf("token %q repeated after %d calls", raw, i+1)
		}
		seen[raw] = struct{}{}
	}
}

// TestHashSessionToken verifies the middleware-side derivation: hashing
// a presented raw token must reproduce the stored hash of that session.
func TestHashSessionToken(t *testing.T) {
	raw, hash := NewSessionToken()

	if got := HashSessionToken(raw); got != hash {
		t.Errorf("HashSessionToken(raw) = %q, want the minted hash %q", got, hash)
	}
	if got := HashSessionToken("a different credential"); got == hash {
		t.Error("HashSessionToken of another token equals this session's hash")
	}
}
