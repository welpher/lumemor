// Handlers of the device pairing surface: POST /auth/pairing/token
// (authed) issues the one-time secret behind the add-a-device QR code
// to the signed-in user, and the public POST /auth/pairing/consume
// burns it for the bearer credential of a new mobile session. The raw
// pairing token appears exactly once, in the creation response; the
// store keeps only its hash.
package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// pairingTokenTTL bounds a pairing token's life: five minutes is
	// long enough to scan a QR code, short enough that a leaked one
	// dies quickly.
	pairingTokenTTL = 5 * time.Minute

	// pairingDailyLimit caps how many tokens one user may mint per UTC
	// day: enough QR codes for a family's devices, few enough that
	// farming codes stays pointless.
	pairingDailyLimit = 10

	// maxDeviceIDRunes bounds the client-generated device id: it is
	// free-form input persisted with the session, so it gets a sane
	// ceiling like the device name.
	maxDeviceIDRunes = 200
)

// CreatePairingToken implements POST /auth/pairing/token: mint the
// one-time QR secret of the calling user. Generation is bounded to
// pairingDailyLimit tokens per user per UTC day, so a compromised
// account cannot farm a pile of live codes; the 429 carries the time
// until the budget resets.
func (s *server) CreatePairingToken(w http.ResponseWriter, r *http.Request) {
	user, ok := authFrom(r.Context())
	if !ok {
		// Unreachable through the authed group; a routing regression
		// would land here.
		s.internalError(w, r, "create pairing token without session context", errors.New("authed route ran without auth context"))
		return
	}

	// The budget gate runs before any minting work. Allow counts the
	// event when it says yes, so count and minted tokens agree; a
	// mint that fails downstream burns the allowance, the
	// conservative direction.
	key := user.userID.String()
	if !s.pairingBudget.Allow(key) {
		seconds := int64(math.Ceil(s.pairingBudget.RetryAfter(key).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
		s.writeError(w, http.StatusTooManyRequests, apigen.AUTHRATELIMITED,
			"pairing token budget exhausted, retry later", nil)
		return
	}

	// A fresh 16-byte secret; the store keeps only its hash.
	raw, hash := auth.NewPairingToken()
	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate pairing token id", err)
		return
	}
	expires := time.Now().Add(pairingTokenTTL)
	if _, err := s.queries.CreatePairingToken(r.Context(), dbgen.CreatePairingTokenParams{
		ID:        pgUUID(id),
		TokenHash: hash,
		CreatedBy: pgUUID(user.userID),
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	}); err != nil {
		s.internalError(w, r, "create pairing token", err)
		return
	}

	// The mint reached the audit stream before the secret leaves in
	// the response: the payload names the minting user only, never the
	// token or its hash.
	s.publishPairingTokenCreated(user.userID)

	// The raw token exists exactly twice: this response and the QR
	// code the client renders from it.
	s.writeJSON(w, http.StatusOK, apigen.PairingTokenResponse{
		Token:     raw,
		ExpiresAt: expires.UTC(),
	})
}

// publishPairingTokenCreated reports a pairing token mint onto the
// activity stream: the security spec's audit minimum includes pairing
// generation. The payload names the minting user only -- the secret
// and its hash never travel on the stream, and the event deliberately
// fires only for tokens that exist in the store, so a failed mint
// audits nothing.
func (s *server) publishPairingTokenCreated(userID uuid.UUID) {
	if s.events == nil {
		return
	}
	s.events.Publish("pairing.token_created", map[string]string{"userId": userID.String()})
}

// ConsumePairingToken implements POST /auth/pairing/consume: the
// mobile app presents the QR token with its device metadata, the
// server burns the one-time token and returns the bearer credential
// of a new mobile session of the user who minted the token.
//
// The burn and the session insert share one transaction: the
// conditional UPDATE grants the one-time right to exactly one
// consumer, and committing it together with the session means a crash
// between the two can never strand a burned token without its
// session -- nor, on rollback, leave the token unburned under a
// session that was never answered.
func (s *server) ConsumePairingToken(w http.ResponseWriter, r *http.Request) {
	var req apigen.PairingConsumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// The capped reader reports its own limit violation; a body
		// that is not JSON at all has no field to name.
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}

	in, invalid := normalizePairingConsume(req)
	if len(invalid) > 0 {
		fields := make([]interface{}, len(invalid))
		for i, field := range invalid {
			fields[i] = field
		}
		details := map[string]interface{}{"fields": fields}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"invalid pairing payload", &details)
		return
	}

	// One transaction for the burn and the session: see the doc
	// comment. The rollback is a no-op after a successful commit; the
	// context is captured while the request is still live.
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, "begin pairing transaction", err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	queries := dbgen.New(tx)

	// The burn. Unknown, expired and already-used tokens all match no
	// row, and the answer is the same 401 for each: the endpoint never
	// says which.
	burned, err := queries.ConsumePairingToken(r.Context(), auth.HashPairingToken(in.token))
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusUnauthorized, apigen.PAIRINGTOKENINVALID,
			"pairing token is invalid or expired", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "consume pairing token", err)
		return
	}

	// The session belongs to the token's creator -- never to whoever
	// walked up with the QR code.
	raw, hash := auth.NewSessionToken()
	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate session id", err)
		return
	}
	now := time.Now()
	session, err := queries.CreateSession(r.Context(), dbgen.CreateSessionParams{
		ID:        pgUUID(id),
		UserID:    burned.CreatedBy,
		TokenHash: hash,
		Kind:      dbgen.SessionKindMobile,
		// The device binding is the pairing payload's whole point: the
		// session list later shows name and id together.
		DeviceName: in.deviceName,
		DeviceID:   pgtype.Text{String: in.deviceId, Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: now, Valid: true},
		// Bearer sessions are revoked, not expired: the NOT NULL
		// column parks at creation and every lookup gates on kind,
		// never on this sentinel.
		ExpiresAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		s.internalError(w, r, "create pairing session", err)
		return
	}

	// The identity half of the response: the burn answers the token
	// creator's id, the role lookup completes the account identity the
	// minted session carries (the paired device knows who it paired
	// as from this response alone).
	role, err := queries.GetUserRole(r.Context(), burned.CreatedBy)
	if err != nil {
		s.internalError(w, r, "lookup pairing user role", err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		s.internalError(w, r, "commit pairing transaction", err)
		return
	}

	// The event reflects committed reality; the payload names the
	// session and its kind, never any credential.
	s.publishSessionEvent("session.started", session.ID, session.Kind)
	s.writeJSON(w, http.StatusOK, apigen.BearerSession{
		Token:   raw,
		Session: sessionInfoFrom(session, uuid.UUID(burned.CreatedBy.Bytes), role),
	})
}

// pairingConsumeInput is the normalized form of a validated consume
// payload.
type pairingConsumeInput struct {
	token      string
	deviceName string
	deviceId   string
}

// normalizePairingConsume validates req against the pairing rules and
// returns the normalized input together with the offending field names
// (empty when the payload is valid): every field must be present after
// trimming, and the device name and id are bounded. The token is only
// checked for presence -- its shape is the server's secret, and a
// wrong token fails the burn exactly like an unknown one.
func normalizePairingConsume(req apigen.PairingConsumeRequest) (pairingConsumeInput, []string) {
	var in pairingConsumeInput
	var invalid []string

	in.token = strings.TrimSpace(req.Token)
	if in.token == "" {
		invalid = append(invalid, "token")
	}

	in.deviceName = strings.TrimSpace(req.DeviceName)
	if in.deviceName == "" || utf8.RuneCountInString(in.deviceName) > maxDeviceNameRunes {
		invalid = append(invalid, "deviceName")
	}

	in.deviceId = strings.TrimSpace(req.DeviceId)
	if in.deviceId == "" || utf8.RuneCountInString(in.deviceId) > maxDeviceIDRunes {
		invalid = append(invalid, "deviceId")
	}

	return in, invalid
}
