package rooms

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

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/events"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

const testSecret = "dev-only-not-a-real-secret-000000000000"

type fixture struct {
	db      *sqlx.DB
	handler http.Handler
	eventID string
	admin   string
	attend  string
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
	mux.Handle("GET /events/{eventId}/rooms", users.Require(testSecret)(List(NewStore(db), events.NewStore(db), grants)))
	mux.Handle("POST /events/{eventId}/rooms", users.Require(testSecret)(Create(NewStore(db), events.NewStore(db), grants)))

	f := &fixture{db: db, handler: mux}
	err = db.Get(&f.eventID, `SELECT id FROM events WHERE name = $1`, "DST Spring Forward")
	if err != nil {
		t.Fatalf("spring event: %v", err)
	}
	err = db.Get(&f.admin, `
		SELECT user_id FROM event_members WHERE event_id = $1 AND role = 'admin' LIMIT 1
	`, f.eventID)
	if err != nil {
		t.Fatalf("admin: %v", err)
	}
	err = db.Get(&f.attend, `
		SELECT user_id FROM event_members WHERE event_id = $1 AND role = 'attendee' LIMIT 1
	`, f.eventID)
	if err != nil {
		t.Fatalf("attendee: %v", err)
	}
	return f
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

func TestListRooms(t *testing.T) {
	f := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/events/"+f.eventID+"/rooms", nil)
	req.AddCookie(f.signIn(t, f.admin))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var page struct {
		Items []Room `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected rooms")
	}
	if page.Items[0].Name == "" || page.Items[0].Capacity < 1 {
		t.Fatalf("room=%+v", page.Items[0])
	}
}

func TestUnknownEventIsForbidden(t *testing.T) {
	f := setup(t)
	missing := "01999999-9999-7000-8000-000000000000"
	req := httptest.NewRequest(http.MethodGet, "/events/"+missing+"/rooms", nil)
	req.AddCookie(f.signIn(t, f.admin))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["code"] != "FORBIDDEN" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestCreateRoomForbiddenAttendee(t *testing.T) {
	f := setup(t)
	raw, _ := json.Marshal(map[string]any{"name": "Should not exist", "capacity": 10})
	req := httptest.NewRequest(http.MethodPost, "/events/"+f.eventID+"/rooms", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.signIn(t, f.attend))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateRoom(t *testing.T) {
	f := setup(t)
	raw, _ := json.Marshal(map[string]any{"name": "Overflow", "capacity": 12})
	req := httptest.NewRequest(http.MethodPost, "/events/"+f.eventID+"/rooms", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.signIn(t, f.admin))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var got Room
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM rooms WHERE id = $1`, got.ID)
	})
	if got.Name != "Overflow" || got.Capacity != 12 || got.EventID != f.eventID {
		t.Fatalf("room=%+v", got)
	}
}
