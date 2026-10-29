package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// TestNewPairingTokenShape pins the pairing credential format: 16
// bytes of crypto/rand entropy hex-encoded as the raw token (short
// enough for a QR code), plus the SHA-256 hex hash of exactly those
// bytes. The pairing_tokens table stores only the hash, so a leaked
// database cannot redeem outstanding QR codes.
func TestNewPairingTokenShape(t *testing.T) {
	raw, hash := NewPairingToken()

	if len(raw) != 2*pairingTokenBytes {
		t.Fatalf("len(raw) = %d, want %d hex characters", len(raw), 2*pairingTokenBytes)
	}
	if strings.ToLower(raw) != raw {
		t.Errorf("raw token = %q, want lowercase hex", raw)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		t.Fatalf("raw token is not valid hex: %v", err)
	}
	if len(decoded) != pairingTokenBytes {
		t.Errorf("decoded token length = %d, want %d bytes (128 bits)", len(decoded), pairingTokenBytes)
	}

	if len(hash) != 2*sha256.Size {
		t.Fatalf("len(hash) = %d, want %d hex characters", len(hash), 2*sha256.Size)
	}
	// The hash covers the presented credential verbatim: the raw hex
	// string, not its decoded bytes -- the same derivation as session
	// credentials, so HashPairingToken reproduces the stored form.
	sum := sha256.Sum256([]byte(raw))
	if hash != hex.EncodeToString(sum[:]) {
		t.Errorf("hash = %q, want the SHA-256 hex of the raw token %q", hash, hex.EncodeToString(sum[:]))
	}
	if hash != HashPairingToken(raw) {
		t.Errorf("HashPairingToken(raw) = %q, want the minted hash %q", HashPairingToken(raw), hash)
	}

	// The stored hash must never equal the credential itself.
	if hash == raw {
		t.Error("hash equals the raw token: the credential would be stored in the clear")
	}
}

// TestNewPairingTokenRandom verifies consecutive tokens are distinct:
// a second pairing must never reuse a live QR secret.
func TestNewPairingTokenRandom(t *testing.T) {
	const calls = 64
	seen := make(map[string]struct{}, calls)
	for i := 0; i < calls; i++ {
		raw, _ := NewPairingToken()
		if _, dup := seen[raw]; dup {
			t.Fatalf("token %q repeated after %d calls", raw, i+1)
		}
		seen[raw] = struct{}{}
	}
}

// TestDailyLimiter walks the pairing generation budget: a fixed count
// per key per UTC calendar day, independent keys, the reset at UTC
// midnight, and the Retry-After remainder.
func TestDailyLimiter(t *testing.T) {
	t.Run("AllowsUpToLimit", func(t *testing.T) {
		lim := NewDailyLimiter(10, nil)
		for i := 0; i < 10; i++ {
			if !lim.Allow("ada") {
				t.Fatalf("Allow(ada) #%d = false, want true (limit is 10)", i+1)
			}
		}
		if lim.Allow("ada") {
			t.Fatal("Allow(ada) #11 = true, want false (the day's budget is spent)")
		}
		// Rejected attempts must not grow or shrink the budget: it
		// stays spent for the rest of the day.
		if lim.Allow("ada") {
			t.Fatal("Allow(ada) after a rejection = true, want false")
		}
	})

	t.Run("KeysAreIndependent", func(t *testing.T) {
		lim := NewDailyLimiter(2, nil)
		lim.Allow("ada")
		lim.Allow("ada")
		if lim.Allow("ada") {
			t.Fatal("ada's budget is spent, want no more allowances")
		}
		if !lim.Allow("grace") {
			t.Error("grace's first Allow = false, want true (independent budget)")
		}
	})

	t.Run("ResetsAtNextUTCMidnight", func(t *testing.T) {
		clock := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
		lim := NewDailyLimiter(2, func() time.Time { return clock })
		lim.Allow("ada")
		lim.Allow("ada")
		if lim.Allow("ada") {
			t.Fatal("third Allow of the day = true, want false")
		}

		// One minute before midnight the budget is still spent.
		clock = clock.Add(13*time.Hour + 59*time.Minute)
		if lim.Allow("ada") {
			t.Fatal("Allow at 23:59 UTC = true, want false (the window has not rolled)")
		}

		// At UTC midnight the window rolls and the budget returns.
		clock = time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
		if !lim.Allow("ada") {
			t.Fatal("first Allow after UTC midnight = false, want true (fresh window)")
		}
		if !lim.Allow("ada") {
			t.Fatal("second Allow after UTC midnight = false, want true (fresh budget of 2)")
		}
		if lim.Allow("ada") {
			t.Error("third Allow of the fresh day = true, want false (limit is 2)")
		}
	})

	t.Run("RetryAfterReportsTimeToReset", func(t *testing.T) {
		clock := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
		lim := NewDailyLimiter(2, func() time.Time { return clock })
		if got := lim.RetryAfter("ada"); got != 0 {
			t.Errorf("RetryAfter(unknown key) = %v, want 0", got)
		}
		lim.Allow("ada")
		if got := lim.RetryAfter("ada"); got != 0 {
			t.Errorf("RetryAfter(key with budget left) = %v, want 0", got)
		}
		lim.Allow("ada")
		if lim.Allow("ada") {
			t.Fatal("third Allow = true, want false (limit is 2)")
		}
		// 10:00 UTC has 14 hours left of its calendar day.
		if got, want := lim.RetryAfter("ada"), 14*time.Hour; got != want {
			t.Errorf("RetryAfter(exhausted at 10:00 UTC) = %v, want %v (time to UTC midnight)", got, want)
		}
	})
}
