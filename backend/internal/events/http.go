package events

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, reason string) {
	writeJSON(w, status, map[string]string{"code": code, "reason": reason})
}

// Show is GET /events/{id}. Cookie check happens before this handler.
// This function only asks the grant: may they read, and how much of the body.
func Show(events *Store, grants *authz.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "missing user")
			return
		}

		eventID := r.PathValue("id")
		grant, err := grants.For(r.Context(), user.ID, eventID)
		if errors.Is(err, authz.ErrNotMember) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "grant lookup failed")
			return
		}
		if !grant.Can(authz.EventRead) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}

		event, err := events.Get(r.Context(), eventID)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "event not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "event lookup failed")
			return
		}

		var members []authz.Member
		if grant.Can(authz.MemberRead) {
			members, err = events.Members(r.Context(), eventID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL", "member lookup failed")
				return
			}
		}

		writeJSON(w, http.StatusOK, authz.PresentEvent(event, members, grant))
	})
}
