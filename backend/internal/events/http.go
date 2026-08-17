package events

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/m-asjedh/event-management-platform/backend/internal/apierr"
	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
	"github.com/m-asjedh/event-management-platform/backend/internal/tz"
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

func List(events *Store, grants *authz.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "missing user")
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

		rows, err := events.ListForUser(r.Context(), user.ID, after, limit+1)
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "event list failed")
			return
		}

		var next string
		if len(rows) > limit {
			next = encodeCursor(rows[limit-1].ID)
			rows = rows[:limit]
		}

		items := make([]authz.EventView, 0, len(rows))
		for _, e := range rows {
			grant, err := grants.For(r.Context(), user.ID, e.ID)
			if err != nil {
				apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "grant lookup failed")
				return
			}
			if !grant.Can(authz.EventRead) {
				continue
			}
			items = append(items, authz.PresentEvent(e, nil, grant))
		}

		page := map[string]any{"items": items}
		if next != "" {
			page["nextCursor"] = next
		}
		writeJSON(w, http.StatusOK, page)
	})
}

func Create(events *Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "missing user")
			return
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			TimeZone    string `json:"timeZone"`
			StartsAt    string `json:"startsAt"`
			EndsAt      string `json:"endsAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.Validation(w, "invalid json", apierr.FieldError{Field: "body", Reason: "invalid json"})
			return
		}
		if body.Name == "" {
			apierr.Validation(w, "name is required", apierr.FieldError{Field: "name", Reason: "required"})
			return
		}
		if body.TimeZone == "" {
			apierr.Validation(w, "timeZone is required", apierr.FieldError{Field: "timeZone", Reason: "required"})
			return
		}

		okZone, err := events.ZoneExists(r.Context(), body.TimeZone)
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "zone lookup failed")
			return
		}
		if !okZone {
			apierr.Validation(w, "unknown IANA time zone", apierr.FieldError{Field: "timeZone", Reason: "unknown IANA time zone"})
			return
		}

		start, err := tz.ParseLocal(body.TimeZone, body.StartsAt)
		if ferr := localTimeField("startsAt", err); ferr != nil {
			apierr.Validation(w, ferr.Reason, *ferr)
			return
		}
		end, err := tz.ParseLocal(body.TimeZone, body.EndsAt)
		if ferr := localTimeField("endsAt", err); ferr != nil {
			apierr.Validation(w, ferr.Reason, *ferr)
			return
		}
		if !end.After(start) {
			apierr.Validation(w, "endsAt must be after startsAt", apierr.FieldError{Field: "endsAt", Reason: "must be after startsAt"})
			return
		}

		out, err := events.Create(r.Context(), authz.Event{
			Name: body.Name, Description: body.Description, TimeZone: body.TimeZone,
			StartsAt: start, EndsAt: end,
		}, user.ID)
		if err != nil {
			apierr.Simple(w, http.StatusInternalServerError, "INTERNAL", "create failed")
			return
		}

		writeJSON(w, http.StatusCreated, authz.PresentEvent(out, nil, authz.NewGrant("admin", nil)))
	})
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
