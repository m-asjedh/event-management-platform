package contract

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
)

func TestServerHonoursOpenAPI(t *testing.T) {
	h := setup(t)
	admin := h.signIn(h.admin)
	springAttend := h.signIn(h.springAttend)

	t.Run("GET /healthz 200", func(t *testing.T) {
		h.t = t
		body := h.expect("healthz", http.MethodGet, "/healthz", nil, nil, http.StatusOK)
		if body["status"] != "ok" {
			t.Fatalf("status=%v", body["status"])
		}
	})

	t.Run("GET /me 200", func(t *testing.T) {
		h.t = t
		h.expect("me", http.MethodGet, "/me", nil, admin, http.StatusOK)
	})

	t.Run("GET /me 401", func(t *testing.T) {
		h.t = t
		h.expect("me unauthenticated", http.MethodGet, "/me", nil, nil, http.StatusUnauthorized)
	})

	t.Run("GET /events 200", func(t *testing.T) {
		h.t = t
		page := h.expect("list events", http.MethodGet, "/events?limit=20", nil, admin, http.StatusOK)
		items, _ := page["items"].([]any)
		if len(items) == 0 {
			t.Fatal("expected items")
		}
	})

	t.Run("GET /events/{id} admin 200 with members", func(t *testing.T) {
		h.t = t
		got := h.expect("get event admin", http.MethodGet, "/events/"+h.adminEvent, nil, admin, http.StatusOK)
		if _, ok := got["members"]; !ok {
			t.Fatal("admin body missing members")
		}
		members, _ := got["members"].([]any)
		if len(members) == 0 {
			t.Fatal("admin members empty")
		}
		first, _ := members[0].(map[string]any)
		if first["email"] == nil || first["email"] == "" {
			t.Fatal("admin member missing email")
		}
	})

	t.Run("GET /events/{id} attendee 200 without members or email", func(t *testing.T) {
		h.t = t
		got := h.expect("get event attendee", http.MethodGet, "/events/"+h.attendEvent, nil, admin, http.StatusOK)
		if _, ok := got["members"]; ok {
			t.Fatalf("attendee body leaked members: %v", got["members"])
		}
		raw, _ := json.Marshal(got)
		if bytesContainEmailKey(raw) {
			t.Fatalf("attendee body leaked email: %s", raw)
		}
	})

	t.Run("POST /events 201", func(t *testing.T) {
		h.t = t
		got := h.expect("create event", http.MethodPost, "/events", map[string]any{
			"name":        "Contract Create",
			"description": "contract",
			"timeZone":    "Pacific/Auckland",
			"startsAt":    "2026-08-17T09:00:00",
			"endsAt":      "2026-08-21T18:00:00",
		}, admin, http.StatusCreated)
		id, _ := got["id"].(string)
		if id == "" {
			t.Fatal("missing id")
		}
		t.Cleanup(func() {
			_, _ = h.db.Exec(`DELETE FROM events WHERE id = $1`, id)
		})
	})

	t.Run("POST /events 400 unknown zone", func(t *testing.T) {
		h.t = t
		got := h.expect("create event bad zone", http.MethodPost, "/events", map[string]any{
			"name":     "Bad Zone",
			"timeZone": "Not/ARealZone",
			"startsAt": "2026-08-17T09:00:00",
			"endsAt":   "2026-08-21T18:00:00",
		}, admin, http.StatusBadRequest)
		if got["code"] != "VALIDATION_ERROR" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("POST /events/{eventId}/sessions 201", func(t *testing.T) {
		h.t = t
		got := h.expect("create session", http.MethodPost, "/events/"+h.springEvent+"/sessions", map[string]any{
			"title":    "Contract Slot",
			"roomId":   h.talkA1Room,
			"startsAt": "2026-03-08T16:00:00",
			"endsAt":   "2026-03-08T17:00:00",
		}, admin, http.StatusCreated)
		id, _ := got["id"].(string)
		if id == "" {
			t.Fatal("missing id")
		}
		if got["version"] == nil {
			t.Fatal("missing version")
		}
		t.Cleanup(func() {
			_, _ = h.db.Exec(`DELETE FROM sessions WHERE id = $1`, id)
		})
	})

	t.Run("GET /events/{eventId}/sessions 200", func(t *testing.T) {
		h.t = t
		h.expect("list sessions", http.MethodGet, "/events/"+h.springEvent+"/sessions", nil, admin, http.StatusOK)
	})

	t.Run("GET /events/{eventId}/rooms 200", func(t *testing.T) {
		h.t = t
		h.expect("list rooms", http.MethodGet, "/events/"+h.springEvent+"/rooms", nil, admin, http.StatusOK)
	})

	t.Run("GET /events/{eventId}/invitations 200 nextCursor present", func(t *testing.T) {
		h.t = t
		page := h.expect("list invitations", http.MethodGet, "/events/"+h.adminEvent+"/invitations?limit=20", nil, admin, http.StatusOK)
		if page["nextCursor"] == nil || page["nextCursor"] == "" {
			t.Fatal("expected nextCursor")
		}
	})

	t.Run("GET /events/{eventId}/invitations 200 nextCursor absent", func(t *testing.T) {
		h.t = t
		cur := base64.RawURLEncoding.EncodeToString([]byte("ffffffff-ffff-7fff-8000-ffffffffffff"))
		page := h.expect("list invitations last", http.MethodGet, "/events/"+h.adminEvent+"/invitations?cursor="+cur, nil, admin, http.StatusOK)
		if _, ok := page["nextCursor"]; ok {
			t.Fatalf("last page should omit nextCursor, got %v", page["nextCursor"])
		}
		items, _ := page["items"].([]any)
		if len(items) != 0 {
			t.Fatalf("expected empty items, got %d", len(items))
		}
	})

	t.Run("GET invitations 400 malformed cursor", func(t *testing.T) {
		h.t = t
		got := h.expect("bad cursor", http.MethodGet, "/events/"+h.adminEvent+"/invitations?cursor=not-a-cursor", nil, admin, http.StatusBadRequest)
		if got["code"] != "VALIDATION_ERROR" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("GET invitations 403 attendee", func(t *testing.T) {
		h.t = t
		got := h.expect("invitations forbidden", http.MethodGet, "/events/"+h.springEvent+"/invitations", nil, springAttend, http.StatusForbidden)
		if got["code"] != "FORBIDDEN" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("GET missing event 403 not 404", func(t *testing.T) {
		h.t = t
		got := h.expect("missing event", http.MethodGet, "/events/01999999-9999-7000-8000-000000000000", nil, admin, http.StatusForbidden)
		if got["code"] != "FORBIDDEN" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("PATCH session 400 spring-forward", func(t *testing.T) {
		h.t = t
		got := h.expect("spring forward", http.MethodPatch, "/sessions/"+h.talkA1, map[string]any{
			"version":  h.talkA1Ver,
			"startsAt": "2026-03-08T02:30:00",
			"endsAt":   "2026-03-08T03:30:00",
		}, admin, http.StatusBadRequest)
		if got["code"] != "VALIDATION_ERROR" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("PATCH session 403 attendee", func(t *testing.T) {
		h.t = t
		got := h.expect("patch forbidden", http.MethodPatch, "/sessions/"+h.talkA1, map[string]any{
			"version": h.talkA1Ver,
			"title":   "Nope",
		}, springAttend, http.StatusForbidden)
		if got["code"] != "FORBIDDEN" {
			t.Fatalf("code=%v", got["code"])
		}
	})

	t.Run("PATCH session 409 ROOM_CONFLICT", func(t *testing.T) {
		h.t = t
		got := h.expect("room conflict", http.MethodPatch, "/sessions/"+h.talkA1, map[string]any{
			"version":  h.talkA1Ver,
			"startsAt": "2026-03-08T10:00:00",
			"endsAt":   "2026-03-08T11:00:00",
		}, admin, http.StatusConflict)
		if got["code"] != "ROOM_CONFLICT" {
			t.Fatalf("code=%v body=%v", got["code"], got)
		}
	})

	t.Run("PATCH session 409 STALE_VERSION", func(t *testing.T) {
		h.t = t
		got := h.expect("stale version", http.MethodPatch, "/sessions/"+h.talkA1, map[string]any{
			"version": h.talkA1Ver - 1,
			"title":   "Should not stick",
		}, admin, http.StatusConflict)
		if got["code"] != "STALE_VERSION" {
			t.Fatalf("code=%v body=%v", got["code"], got)
		}
	})

	t.Logf("validated %d live responses against openapi/openapi.yaml", h.checked)
	if h.checked < 16 {
		t.Fatalf("too few validated responses: %d", h.checked)
	}
}

func bytesContainEmailKey(raw []byte) bool {
	return json.Valid(raw) && containsEmailKey(raw)
}

func containsEmailKey(raw []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m["email"]
	return ok
}
