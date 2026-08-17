package invitations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/m-asjedh/event-management-platform/backend/internal/apierr"
	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/events"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func List(store *Store, evstore *events.Store, grants *authz.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "missing user")
			return
		}

		eventID := r.PathValue("eventId")
		grant, err := grants.For(r.Context(), user.ID, eventID)
		if errors.Is(err, authz.ErrNotMember) {
			apierr.Simple(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "grant lookup failed")
			return
		}
		if !grant.Can(authz.InvitationRead) {
			apierr.Simple(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}

		if _, err := evstore.Get(r.Context(), eventID); errors.Is(err, events.ErrNotFound) {
			apierr.Simple(w, http.StatusNotFound, "NOT_FOUND", "event not found")
			return
		} else if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "event lookup failed")
			return
		}

		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				apierr.Validation(w, "limit must be 1–100", apierr.FieldError{Field: "limit", Reason: "must be 1–100"})
				return
			}
			limit = n
		}

		var after string
		if raw := r.URL.Query().Get("cursor"); raw != "" {
			id, err := decodeCursor(raw)
			if err != nil {
				apierr.Validation(w, "invalid cursor", apierr.FieldError{Field: "cursor", Reason: "invalid"})
				return
			}
			after = id
		}

		rows, err := store.ListByEvent(r.Context(), eventID, after, limit+1)
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "invitation list failed")
			return
		}

		var next string
		if len(rows) > limit {
			next = encodeCursor(rows[limit-1].ID)
			rows = rows[:limit]
		}

		items := make([]Invitation, 0, len(rows))
		for _, inv := range rows {
			items = append(items, present(inv, grant))
		}

		page := map[string]any{"items": items}
		if next != "" {
			page["nextCursor"] = next
		}
		writeJSON(w, http.StatusOK, page)
	})
}

func present(inv Invitation, g authz.Grant) Invitation {
	if !g.Can(authz.UserEmailRead) {
		inv.Email = ""
	}
	return inv
}
