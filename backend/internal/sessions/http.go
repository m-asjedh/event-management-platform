package sessions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/m-asjedh/event-management-platform/backend/internal/apierr"
	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/events"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
	"github.com/m-asjedh/event-management-platform/backend/internal/tz"
)

type patchBody struct {
	Version     *int    `json:"version"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	RoomID      *roomID `json:"roomId"`
	StartsAt    *string `json:"startsAt"`
	EndsAt      *string `json:"endsAt"`
}

// roomID distinguishes omit / JSON null / a uuid.
type roomID struct {
	Clear bool
	ID    string
}

func (r *roomID) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		r.Clear = true
		return nil
	}
	var id string
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	r.ID = id
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Patch(store *Store, grants *authz.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "missing user")
			return
		}

		id := r.PathValue("id")
		eventID, err := store.eventID(r.Context(), id)
		if errors.Is(err, ErrNotFound) {
			apierr.Simple(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "session lookup failed")
			return
		}

		grant, err := grants.For(r.Context(), user.ID, eventID)
		if errors.Is(err, authz.ErrNotMember) || (err == nil && !grant.Can(authz.SessionUpdate)) {
			apierr.Simple(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "grant lookup failed")
			return
		}

		cur, err := store.Get(r.Context(), id)
		if errors.Is(err, ErrNotFound) {
			apierr.Simple(w, http.StatusNotFound, "NOT_FOUND", "session not found")
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "session lookup failed")
			return
		}

		var body patchBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.Validation(w, "invalid json", apierr.FieldError{Field: "body", Reason: "invalid json"})
			return
		}
		if body.Version == nil {
			apierr.Validation(w, "version is required", apierr.FieldError{Field: "version", Reason: "required"})
			return
		}

		next := cur
		if body.Title != nil {
			if *body.Title == "" {
				apierr.Validation(w, "title is required", apierr.FieldError{Field: "title", Reason: "required"})
				return
			}
			next.Title = *body.Title
		}
		if body.Description != nil {
			next.Description = *body.Description
		}
		if body.RoomID != nil {
			if body.RoomID.Clear {
				next.RoomID = nil
			} else {
				ok, err := store.RoomBelongsToEvent(r.Context(), body.RoomID.ID, cur.EventID)
				if err != nil {
					apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "room lookup failed")
					return
				}
				if !ok {
					apierr.Validation(w, "room not on this event", apierr.FieldError{Field: "roomId", Reason: "unknown room"})
					return
				}
				id := body.RoomID.ID
				next.RoomID = &id
			}
		}

		if body.StartsAt != nil {
			t, err := tz.ParseLocal(cur.TimeZone, *body.StartsAt)
			if ferr := localTimeField("startsAt", err); ferr != nil {
				apierr.Validation(w, ferr.Reason, *ferr)
				return
			}
			next.StartsAt = t
		}
		if body.EndsAt != nil {
			t, err := tz.ParseLocal(cur.TimeZone, *body.EndsAt)
			if ferr := localTimeField("endsAt", err); ferr != nil {
				apierr.Validation(w, ferr.Reason, *ferr)
				return
			}
			next.EndsAt = t
		}
		if !next.EndsAt.After(next.StartsAt) {
			apierr.Validation(w, "endsAt must be after startsAt", apierr.FieldError{Field: "endsAt", Reason: "must be after startsAt"})
			return
		}

		out, clash, err := store.Apply(r.Context(), *body.Version, next)
		if errors.Is(err, ErrRoomConflict) {
			apierr.Write(w, http.StatusConflict, apierr.Body{
				Code:   "ROOM_CONFLICT",
				Reason: "room is taken",
				Conflict: &apierr.Conflict{
					ConflictingSessionID: clash.ID,
					ConflictingTitle:     clash.Title,
				},
			})
			return
		}
		if errors.Is(err, ErrStaleVersion) {
			fresh, gerr := store.Get(r.Context(), id)
			if errors.Is(gerr, ErrNotFound) {
				apierr.Simple(w, http.StatusNotFound, "NOT_FOUND", "session not found")
				return
			}
			if gerr != nil {
				apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "session lookup failed")
				return
			}
			apierr.Write(w, http.StatusConflict, apierr.Body{
				Code:   "STALE_VERSION",
				Reason: "version has changed",
				Conflict: &apierr.Conflict{
					CurrentVersion: fresh.Version,
					CurrentState:   fresh.view(),
				},
			})
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "update failed")
			return
		}

		writeJSON(w, http.StatusOK, out.view())
	})
}

func (s Session) view() Session {
	s.TimeZone = ""
	return s
}

func localTimeField(field string, err error) *apierr.FieldError {
	if err == nil {
		return nil
	}
	reason := "invalid local time"
	switch {
	case errors.Is(err, tz.ErrNonexistentLocalTime):
		reason = "nonexistent local time"
	case errors.Is(err, tz.ErrAmbiguousLocalTime):
		reason = "ambiguous local time"
	}
	return &apierr.FieldError{Field: field, Reason: reason}
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
		if !grant.Can(authz.SessionRead) {
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
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "session list failed")
			return
		}
		items := make([]Session, 0, len(rows))
		for _, s := range rows {
			items = append(items, s.view())
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
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
		if !grant.Can(authz.SessionCreate) {
			apierr.Simple(w, http.StatusForbidden, "FORBIDDEN", "not allowed")
			return
		}

		event, err := evstore.Get(r.Context(), eventID)
		if errors.Is(err, events.ErrNotFound) {
			apierr.Simple(w, http.StatusNotFound, "NOT_FOUND", "event not found")
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "event lookup failed")
			return
		}

		var body struct {
			Title       string  `json:"title"`
			Description string  `json:"description"`
			RoomID      *string `json:"roomId"`
			StartsAt    string  `json:"startsAt"`
			EndsAt      string  `json:"endsAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.Validation(w, "invalid json", apierr.FieldError{Field: "body", Reason: "invalid json"})
			return
		}
		if body.Title == "" {
			apierr.Validation(w, "title is required", apierr.FieldError{Field: "title", Reason: "required"})
			return
		}

		start, err := tz.ParseLocal(event.TimeZone, body.StartsAt)
		if ferr := localTimeField("startsAt", err); ferr != nil {
			apierr.Validation(w, ferr.Reason, *ferr)
			return
		}
		end, err := tz.ParseLocal(event.TimeZone, body.EndsAt)
		if ferr := localTimeField("endsAt", err); ferr != nil {
			apierr.Validation(w, ferr.Reason, *ferr)
			return
		}
		if !end.After(start) {
			apierr.Validation(w, "endsAt must be after startsAt", apierr.FieldError{Field: "endsAt", Reason: "must be after startsAt"})
			return
		}

		if body.RoomID != nil && *body.RoomID != "" {
			ok, err := store.RoomBelongsToEvent(r.Context(), *body.RoomID, eventID)
			if err != nil {
				apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "room lookup failed")
				return
			}
			if !ok {
				apierr.Validation(w, "room not on this event", apierr.FieldError{Field: "roomId", Reason: "unknown room"})
				return
			}
		} else {
			body.RoomID = nil
		}

		out, clash, err := store.Insert(r.Context(), Session{
			EventID: eventID, RoomID: body.RoomID, Title: body.Title,
			Description: body.Description, StartsAt: start, EndsAt: end,
			TimeZone: event.TimeZone,
		})
		if errors.Is(err, ErrRoomConflict) {
			apierr.Write(w, http.StatusConflict, apierr.Body{
				Code:   "ROOM_CONFLICT",
				Reason: "room is taken",
				Conflict: &apierr.Conflict{
					ConflictingSessionID: clash.ID,
					ConflictingTitle:     clash.Title,
				},
			})
			return
		}
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "create failed")
			return
		}
		writeJSON(w, http.StatusCreated, out.view())
	})
}
