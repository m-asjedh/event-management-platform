package events

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
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

const testSecret = "dev-only-not-a-real-secret-000000000000"

type fixture struct {
	db      *sqlx.DB
	handler http.Handler
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
	store := NewStore(db)
	mux := http.NewServeMux()
	mux.Handle("GET /events", users.Require(testSecret)(List(store, grants)))
	mux.Handle("POST /events", users.Require(testSecret)(Create(store)))

	return &fixture{db: db, handler: mux}
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

func (f *fixture) get(t *testing.T, userID, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(f.signIn(t, userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) post(t *testing.T, userID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(raw))
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

func TestListExcludesEventsCallerCannotSee(t *testing.T) {
	f := setup(t)

	var userID string
	err := f.db.Get(&userID, `
		SELECT user_id
		FROM   event_members
		WHERE  role = 'attendee'
		  AND  user_id NOT IN (
		        SELECT user_id FROM event_members WHERE role <> 'attendee'
		  )
		LIMIT  1
	`)
	if err != nil {
		t.Fatalf("attendee-only user: %v", err)
	}

	var allowed []string
	if err := f.db.Select(&allowed, `SELECT event_id FROM event_members WHERE user_id = $1 ORDER BY event_id`, userID); err != nil {
		t.Fatal(err)
	}
	if len(allowed) == 0 {
		t.Fatal("expected at least one membership")
	}

	var hidden string
	err = f.db.Get(&hidden, `
		SELECT id FROM events
		WHERE  id NOT IN (SELECT event_id FROM event_members WHERE user_id = $1)
		LIMIT  1
	`, userID)
	if err != nil {
		t.Fatalf("hidden event: %v", err)
	}

	rec := f.get(t, userID, "/events?limit=100")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}

	var page struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != len(allowed) {
		t.Fatalf("got %d events, want %d", len(page.Items), len(allowed))
	}

	seen := map[string]bool{}
	for _, item := range page.Items {
		id, _ := item["id"].(string)
		seen[id] = true
		if _, ok := item["members"]; ok {
			t.Fatalf("attendee list item leaked members: %v", item)
		}
		if _, ok := item["email"]; ok {
			t.Fatalf("attendee list item leaked email: %v", item)
		}
		raw, _ := json.Marshal(item)
		if bytes.Contains(bytes.ToLower(raw), []byte(`"email"`)) {
			t.Fatalf("attendee list item contains email: %s", raw)
		}
	}
	for _, id := range allowed {
		if !seen[id] {
			t.Fatalf("missing permitted event %s", id)
		}
	}
	if seen[hidden] {
		t.Fatalf("listed event the caller is not a member of: %s", hidden)
	}
}

func TestCreateUnknownZone(t *testing.T) {
	f := setup(t)
	rec := f.post(t, "usr_0000", map[string]any{
		"name":     "Bad Zone",
		"timeZone": "Not/ARealZone",
		"startsAt": "2026-08-17T09:00:00",
		"endsAt":   "2026-08-21T18:00:00",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeErr(t, rec)
	if out["code"] != "VALIDATION_ERROR" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
}

func TestCreateMakesCreatorAdmin(t *testing.T) {
	f := setup(t)
	rec := f.post(t, "usr_0000", map[string]any{
		"name":        "CRUD Create Test",
		"description": "created by test",
		"timeZone":    "Pacific/Auckland",
		"startsAt":    "2026-08-17T09:00:00",
		"endsAt":      "2026-08-21T18:00:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatalf("missing id: %s", rec.Body.String())
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM events WHERE id = $1`, created.ID)
	})

	var role string
	err := f.db.Get(&role, `
		SELECT role FROM event_members WHERE event_id = $1 AND user_id = $2
	`, created.ID, "usr_0000")
	if err != nil {
		t.Fatalf("membership: %v", err)
	}
	if role != "admin" {
		t.Fatalf("role=%s", role)
	}
}
