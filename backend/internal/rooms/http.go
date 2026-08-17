package rooms

import (
	"encoding/json"
	"errors"
	"net/http"

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
		if !grant.Can(authz.RoomRead) {
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

		rows, err := store.ListByEvent(r.Context(), eventID)
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "room list failed")
			return
		}
		if rows == nil {
			rows = []Room{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": rows})
	})
}

func Create(store *Store, evstore *events.Store, grants *authz.Store) http.Handler {
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
		if !grant.Can(authz.RoomManage) {
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

		var body struct {
			Name     string `json:"name"`
			Capacity int    `json:"capacity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.Validation(w, "invalid json", apierr.FieldError{Field: "body", Reason: "invalid json"})
			return
		}
		if body.Name == "" {
			apierr.Validation(w, "name is required", apierr.FieldError{Field: "name", Reason: "required"})
			return
		}
		if body.Capacity < 1 {
			apierr.Validation(w, "capacity must be at least 1", apierr.FieldError{Field: "capacity", Reason: "must be at least 1"})
			return
		}

		out, err := store.Insert(r.Context(), Room{EventID: eventID, Name: body.Name, Capacity: body.Capacity})
		if errors.Is(err, ErrDuplicateName) {
			apierr.Validation(w, "room name already used", apierr.FieldError{Field: "name", Reason: "already used"})
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "create failed")
			return
		}
		writeJSON(w, http.StatusCreated, out)
	})
}
