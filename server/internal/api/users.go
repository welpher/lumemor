// The instance's minimal member list (M3c Task 3): GET /users, the
// share dialog's pick list. Any authenticated account may enumerate
// the family's accounts -- the family-instance posture the contract
// documents (the accounts of a shared photo home are not secret from
// one another) -- and the entries carry exactly the identity fields
// a pick list renders. This is deliberately NOT the admin management
// face: no display names, no timestamps, no writes; the admin
// surface keeps its own richer, role-guarded listing.
package api

import (
	"net/http"

	"github.com/google/uuid"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// ListInstanceUsers implements GET /api/v1/users: every account of
// the instance, email ascending, through the listing that selects
// only the identity columns (the password hash never leaves the
// table here, exactly like the admin listing).
func (s *server) ListInstanceUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}

	rows, err := s.queries.ListInstanceUsersMinimal(r.Context())
	if err != nil {
		s.internalError(w, r, "list instance users", err)
		return
	}

	response := apigen.UserListResponse{Users: make([]apigen.UserSummary, 0, len(rows))}
	for _, row := range rows {
		response.Users = append(response.Users, apigen.UserSummary{
			Id:         uuid.UUID(row.ID.Bytes),
			Email:      oapitypes.Email(row.Email),
			Role:       apigen.UserRole(row.Role),
			IsDisabled: row.IsDisabled,
		})
	}
	s.writeJSON(w, http.StatusOK, response)
}
