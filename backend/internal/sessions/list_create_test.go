package sessions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/events"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

func setupListCreate(t *testing.T) *fixture {
	t.Helper()
	f := setup(t)

	users := identity.NewStore(f.db)
	grants := authz.NewStore(f.db)
	mux := http.NewServeMux()
	mux.Handle("PATCH /sessions/{id}", users.Require(testSecret)(Patch(NewStore(f.db), grants)))
	mux.Handle("GET /events/{eventId}/sessions", users.Require(testSecret)(List(NewStore(f.db), events.NewStore(f.db), grants)))
	mux.Handle("POST /events/{eventId}/sessions", users.Require(testSecret)(Create(NewStore(f.db), events.NewStore(f.db), grants)))
	f.handler = mux
	return f
}

func (f *fixture) getSessions(t *testing.T, userID, eventID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID+"/sessions", nil)
	req.AddCookie(f.signIn(t, userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) postSession(t *testing.T, userID, eventID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/events/"+eventID+"/sessions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.signIn(t, userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func TestListSessionsStartOrderedWithVersion(t *testing.T) {
	f := setupListCreate(t)
	rec := f.getSessions(t, f.admin, f.talkA1.EventID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var page struct {
		Items []Session `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) < 2 {
		t.Fatalf("want at least 2 sessions, got %d", len(page.Items))
	}
	for i, s := range page.Items {
		if s.Version < 1 {
			t.Fatalf("item %d missing version: %+v", i, s)
		}
		if i > 0 {
			prev := page.Items[i-1]
			if s.StartsAt.Before(prev.StartsAt) {
				t.Fatalf("not start-ordered: %s then %s", prev.StartsAt, s.StartsAt)
			}
			if s.StartsAt.Equal(prev.StartsAt) && s.ID < prev.ID {
				t.Fatalf("id tie-break not ordered: %s then %s", prev.ID, s.ID)
			}
		}
	}
}

func TestCreateSessionSpringForward(t *testing.T) {
	f := setupListCreate(t)
	rec := f.postSession(t, f.admin, f.talkA1.EventID, map[string]any{
		"title":    "Gap",
		"roomId":   *f.talkA1.RoomID,
		"startsAt": "2026-03-08T02:30:00",
		"endsAt":   "2026-03-08T03:30:00",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeErr(t, rec)
	if out["code"] != "VALIDATION_ERROR" {
		t.Fatalf("code=%v", out["code"])
	}
}

func TestCreateSessionRoomConflictFromConstraint(t *testing.T) {
	f := setupListCreate(t)
	rec := f.postSession(t, f.admin, f.talkA1.EventID, map[string]any{
		"title":    "Clash",
		"roomId":   *f.talkA1.RoomID,
		"startsAt": "2026-03-08T09:15:00",
		"endsAt":   "2026-03-08T09:45:00",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeErr(t, rec)
	if out["code"] != "ROOM_CONFLICT" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
	conflict, _ := out["conflict"].(map[string]any)
	if conflict["conflictingSessionId"] != f.talkA1.ID {
		t.Fatalf("clash id=%v want A1 %s", conflict["conflictingSessionId"], f.talkA1.ID)
	}

	var n int
	if err := f.db.Get(&n, `SELECT count(*) FROM sessions WHERE title = $1 AND event_id = $2`, "Clash", f.talkA1.EventID); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("insert stuck despite conflict: %d", n)
	}
}

func TestUnknownEventIsForbidden(t *testing.T) {
	f := setupListCreate(t)
	missing := "01999999-9999-7000-8000-000000000000"
	cookie := f.signIn(t, f.admin)

	req := httptest.NewRequest(http.MethodGet, "/events/"+missing+"/sessions", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if decodeErr(t, rec)["code"] != "FORBIDDEN" {
		t.Fatalf("body=%s", rec.Body.String())
	}

	raw, err := json.Marshal(map[string]any{
		"title":    "Nope",
		"startsAt": "2026-03-08T16:00:00",
		"endsAt":   "2026-03-08T17:00:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/events/"+missing+"/sessions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("create status %d body=%s", rec.Code, rec.Body.String())
	}
	if decodeErr(t, rec)["code"] != "FORBIDDEN" {
		t.Fatalf("create body=%s", rec.Body.String())
	}
}

func TestCreateSessionForbiddenAttendee(t *testing.T) {
	f := setupListCreate(t)
	rec := f.postSession(t, f.attend, f.talkA1.EventID, map[string]any{
		"title":    "Nope",
		"roomId":   *f.talkA1.RoomID,
		"startsAt": "2026-03-08T16:00:00",
		"endsAt":   "2026-03-08T17:00:00",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if decodeErr(t, rec)["code"] != "FORBIDDEN" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestCreateSessionReturnsVersion(t *testing.T) {
	f := setupListCreate(t)
	rec := f.postSession(t, f.admin, f.talkA1.EventID, map[string]any{
		"title":    "Late slot",
		"roomId":   *f.talkA1.RoomID,
		"startsAt": "2026-03-08T16:00:00",
		"endsAt":   "2026-03-08T17:00:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var got Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM sessions WHERE id = $1`, got.ID)
	})
	if got.Version != 1 {
		t.Fatalf("version=%d", got.Version)
	}
	if got.Title != "Late slot" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.StartsAt.IsZero() || !got.EndsAt.After(got.StartsAt) {
		t.Fatalf("times %s %s", got.StartsAt, got.EndsAt)
	}
}
