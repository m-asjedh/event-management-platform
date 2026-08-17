package sessions

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

const testSecret = "dev-only-not-a-real-secret-000000000000"

type fixture struct {
	db      *sqlx.DB
	handler http.Handler
	admin   string
	attend  string
	talkA1  Session
	talkA2  Session
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	users := identity.NewStore(db)
	grants := authz.NewStore(db)
	mux := http.NewServeMux()
	mux.Handle("PATCH /sessions/{id}", users.Require(testSecret)(Patch(NewStore(db), grants)))

	f := &fixture{db: db, handler: mux}

	var eventID string
	err = db.Get(&eventID, `SELECT id FROM events WHERE name = $1`, "DST Spring Forward")
	if err != nil {
		t.Fatalf("spring event: %v", err)
	}
	err = db.Get(&f.admin, `
		SELECT user_id FROM event_members WHERE event_id = $1 AND role = 'admin' LIMIT 1
	`, eventID)
	if err != nil {
		t.Fatalf("admin: %v", err)
	}
	err = db.Get(&f.attend, `
		SELECT user_id FROM event_members WHERE event_id = $1 AND role = 'attendee' LIMIT 1
	`, eventID)
	if err != nil {
		t.Fatalf("attendee: %v", err)
	}

	err = db.Get(&f.talkA1, `
		SELECT s.id, s.event_id, s.room_id, s.title, s.description,
		       s.starts_at, s.ends_at, s.version, e.time_zone
		FROM   sessions s JOIN events e ON e.id = s.event_id
		WHERE  e.name = $1 AND s.title = $2
	`, "DST Spring Forward", "Talk A1")
	if err != nil {
		t.Fatalf("Talk A1: %v", err)
	}
	err = db.Get(&f.talkA2, `
		SELECT s.id, s.event_id, s.room_id, s.title, s.description,
		       s.starts_at, s.ends_at, s.version, e.time_zone
		FROM   sessions s JOIN events e ON e.id = s.event_id
		WHERE  e.name = $1 AND s.title = $2
	`, "DST Spring Forward", "Talk A2")
	if err != nil {
		t.Fatalf("Talk A2: %v", err)
	}

	t.Cleanup(func() { f.restore(t, f.talkA1) })
	return f
}

func (f *fixture) restore(t *testing.T, orig Session) {
	t.Helper()
	_, err := f.db.Exec(`
		UPDATE sessions
		SET title = $1, description = $2, room_id = $3,
		    starts_at = $4, ends_at = $5, version = $6
		WHERE id = $7
	`, orig.Title, orig.Description, orig.RoomID, orig.StartsAt, orig.EndsAt, orig.Version, orig.ID)
	if err != nil {
		t.Errorf("restore: %v", err)
	}
}

func (f *fixture) signIn(t *testing.T, userID string) *http.Cookie {
	t.Helper()
	token := "tok-" + t.Name()
	sid := "sid-" + t.Name()
	_, err := f.db.Exec(`
		INSERT INTO auth.sessions (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, now() + interval '1 hour')
	`, sid, userID, token)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM auth.sessions WHERE id = $1`, sid)
	})
	return &http.Cookie{Name: identity.CookieName, Value: signCookie(token)}
}

func signCookie(token string) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	_, _ = mac.Write([]byte(token))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return url.PathEscape(token + "." + sig)
}

func (f *fixture) patch(t *testing.T, userID, sessionID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/sessions/"+sessionID, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.signIn(t, userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func decodeErr(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return out
}

func TestSuccessfulUpdate(t *testing.T) {
	f := setup(t)
	rec := f.patch(t, f.admin, f.talkA1.ID, map[string]any{
		"version": f.talkA1.Version,
		"title":   "Keynote",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var got Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Keynote" || got.Version != f.talkA1.Version+1 {
		t.Fatalf("title=%q version=%d", got.Title, got.Version)
	}
}

func TestStaleVersionDoesNotOverwrite(t *testing.T) {
	f := setup(t)
	rec := f.patch(t, f.admin, f.talkA1.ID, map[string]any{
		"version": f.talkA1.Version - 1,
		"title":   "Should not stick",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeErr(t, rec)
	if out["code"] != "STALE_VERSION" {
		t.Fatalf("code=%v", out["code"])
	}
	conflict, _ := out["conflict"].(map[string]any)
	if conflict["currentVersion"] != float64(f.talkA1.Version) {
		t.Fatalf("currentVersion=%v", conflict["currentVersion"])
	}
	state, _ := conflict["currentState"].(map[string]any)
	if state["title"] != "Talk A1" {
		t.Fatalf("currentState title=%v", state["title"])
	}

	var title string
	var version int
	if err := f.db.QueryRow(`SELECT title, version FROM sessions WHERE id = $1`, f.talkA1.ID).Scan(&title, &version); err != nil {
		t.Fatal(err)
	}
	if title != "Talk A1" || version != f.talkA1.Version {
		t.Fatalf("overwrote: title=%s version=%d", title, version)
	}
}

func TestRoomConflictFromConstraint(t *testing.T) {
	f := setup(t)
	if f.talkA1.RoomID == nil || f.talkA2.RoomID == nil || *f.talkA1.RoomID != *f.talkA2.RoomID {
		t.Fatal("Talk A1 and A2 must share a room in the seed")
	}

	rec := f.patch(t, f.admin, f.talkA1.ID, map[string]any{
		"version":  f.talkA1.Version,
		"startsAt": "2026-03-08T10:00:00",
		"endsAt":   "2026-03-08T11:00:00",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeErr(t, rec)
	if out["code"] != "ROOM_CONFLICT" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
	conflict, _ := out["conflict"].(map[string]any)
	if conflict["conflictingSessionId"] != f.talkA2.ID {
		t.Fatalf("clash id=%v want A2 %s", conflict["conflictingSessionId"], f.talkA2.ID)
	}
	if conflict["conflictingTitle"] != "Talk A2" {
		t.Fatalf("clash title=%v", conflict["conflictingTitle"])
	}

	var starts time.Time
	if err := f.db.Get(&starts, `SELECT starts_at FROM sessions WHERE id = $1`, f.talkA1.ID); err != nil {
		t.Fatal(err)
	}
	if !starts.Equal(f.talkA1.StartsAt) {
		t.Fatalf("A1 time changed after conflict: %s", starts)
	}
}

func TestForbiddenAttendee(t *testing.T) {
	f := setup(t)
	rec := f.patch(t, f.attend, f.talkA1.ID, map[string]any{
		"version": f.talkA1.Version,
		"title":   "Nope",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if decodeErr(t, rec)["code"] != "FORBIDDEN" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestSpringForwardLocalTime(t *testing.T) {
	f := setup(t)
	rec := f.patch(t, f.admin, f.talkA1.ID, map[string]any{
		"version":  f.talkA1.Version,
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
	fields, _ := out["fieldErrors"].([]any)
	if len(fields) == 0 {
		t.Fatalf("want fieldErrors, body=%s", rec.Body.String())
	}
	first, _ := fields[0].(map[string]any)
	if first["field"] != "startsAt" || first["reason"] != "nonexistent local time" {
		t.Fatalf("fieldErrors=%v", fields)
	}
}

func TestPartialTitleLeavesTimeAndRoom(t *testing.T) {
	f := setup(t)
	rec := f.patch(t, f.admin, f.talkA1.ID, map[string]any{
		"version": f.talkA1.Version,
		"title":   "Only the title",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var got Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Only the title" {
		t.Fatalf("title=%q", got.Title)
	}
	if !got.StartsAt.Equal(f.talkA1.StartsAt) || !got.EndsAt.Equal(f.talkA1.EndsAt) {
		t.Fatalf("times changed: %s %s", got.StartsAt, got.EndsAt)
	}
	if got.RoomID == nil || f.talkA1.RoomID == nil || *got.RoomID != *f.talkA1.RoomID {
		t.Fatalf("room changed: %v -> %v", f.talkA1.RoomID, got.RoomID)
	}
}
