package invitations

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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
	mux.Handle("GET /events/{eventId}/invitations", users.Require(testSecret)(List(NewStore(db), events.NewStore(db), grants)))

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
	n := time.Now().UnixNano()
	token := fmt.Sprintf("tok-%s-%d", t.Name(), n)
	sid := fmt.Sprintf("sid-%s-%d", t.Name(), n)
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

func (f *fixture) list(t *testing.T, userID, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/events/"+f.eventID+"/invitations"+query, nil)
	req.AddCookie(f.signIn(t, userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

type page struct {
	Items      []map[string]any `json:"items"`
	NextCursor string           `json:"nextCursor"`
}

func decodePage(t *testing.T, rec *httptest.ResponseRecorder) page {
	t.Helper()
	var p page
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return p
}

func idsOf(items []map[string]any) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		id, _ := it["id"].(string)
		out = append(out, id)
	}
	return out
}

func TestCursorMatchesEventsScheme(t *testing.T) {
	id := "01999999-9999-7000-8000-000000000000"
	got := encodeCursor(id)
	want := base64.RawURLEncoding.EncodeToString([]byte(id))
	if got != want {
		t.Fatalf("encodeCursor=%q want %q (GET /events scheme)", got, want)
	}
	back, err := decodeCursor(got)
	if err != nil || back != id {
		t.Fatalf("roundtrip %q err=%v", back, err)
	}
}

func TestFirstPageHasLimitAndNextCursor(t *testing.T) {
	f := setup(t)
	rec := f.list(t, f.admin, "?limit=20")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	p := decodePage(t, rec)
	if len(p.Items) != 20 {
		t.Fatalf("got %d items, want 20", len(p.Items))
	}
	if p.NextCursor == "" {
		t.Fatal("expected nextCursor")
	}
}

func TestNextPageIsDistinct(t *testing.T) {
	f := setup(t)
	first := decodePage(t, f.list(t, f.admin, "?limit=20"))
	second := decodePage(t, f.list(t, f.admin, "?limit=20&cursor="+url.QueryEscape(first.NextCursor)))
	if len(second.Items) != 20 {
		t.Fatalf("page 2 got %d items", len(second.Items))
	}
	seen := map[string]bool{}
	for _, id := range idsOf(first.Items) {
		seen[id] = true
	}
	for _, id := range idsOf(second.Items) {
		if seen[id] {
			t.Fatalf("page 2 overlapped page 1: %s", id)
		}
	}
	if second.Items[0]["id"] == first.Items[len(first.Items)-1]["id"] {
		t.Fatal("page 2 started on page 1's last row")
	}
}

func TestWalkToEndHasNoNextCursor(t *testing.T) {
	f := setup(t)
	var cursor string
	var n int
	seen := map[string]bool{}
	for range 20 {
		q := "?limit=100"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		rec := f.list(t, f.admin, q)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
		}
		p := decodePage(t, rec)
		for _, id := range idsOf(p.Items) {
			if seen[id] {
				t.Fatalf("duplicate across pages: %s", id)
			}
			seen[id] = true
		}
		n += len(p.Items)
		if p.NextCursor == "" {
			if n != 1000 {
				t.Fatalf("walked %d rows, seed has 1000 invitations on this event", n)
			}
			return
		}
		cursor = p.NextCursor
	}
	t.Fatal("did not reach the last page")
}

func TestInsertMidTraversalNoDuplicateNoSkip(t *testing.T) {
	f := setup(t)
	first := decodePage(t, f.list(t, f.admin, "?limit=5"))
	lastID := first.Items[4]["id"].(string)

	var nextID string
	err := f.db.Get(&nextID, `
		SELECT id FROM invitations
		WHERE  event_id = $1 AND id > $2
		ORDER  BY id
		LIMIT  1
	`, f.eventID, lastID)
	if err != nil {
		t.Fatalf("next id: %v", err)
	}

	mid, err := uuid.Parse(lastID)
	if err != nil {
		t.Fatal(err)
	}
	mid[15] = 1
	if mid.String() >= nextID {
		t.Fatalf("no gap between %s and %s", lastID, nextID)
	}

	_, err = f.db.Exec(`
		INSERT INTO invitations (id, event_id, email, role, status, invited_by)
		VALUES ($1, $2, $3, 'attendee', 'pending', $4)
	`, mid.String(), f.eventID, "mid-keyset@test.example", f.admin)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM invitations WHERE id = $1`, mid.String())
	})

	second := decodePage(t, f.list(t, f.admin, "?limit=5&cursor="+url.QueryEscape(first.NextCursor)))
	got := idsOf(second.Items)
	if len(got) == 0 {
		t.Fatal("empty page 2")
	}
	if got[0] != mid.String() {
		t.Fatalf("inserted row skipped: first of page 2=%s want %s", got[0], mid.String())
	}
	if got[1] != nextID {
		t.Fatalf("original next row skipped: %s want %s", got[1], nextID)
	}
	for _, id := range idsOf(first.Items) {
		for _, g := range got {
			if g == id {
				t.Fatalf("page 1 row %s repeated on page 2", id)
			}
		}
	}
}

func TestMalformedCursorIsValidationError(t *testing.T) {
	f := setup(t)
	rec := f.list(t, f.admin, "?cursor=not-a-cursor")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["code"] != "VALIDATION_ERROR" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
}

func TestUnknownCursorYieldsEmptyPage(t *testing.T) {
	f := setup(t)
	cur := encodeCursor("ffffffff-ffff-7fff-8000-ffffffffffff")
	rec := f.list(t, f.admin, "?cursor="+url.QueryEscape(cur))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	p := decodePage(t, rec)
	if len(p.Items) != 0 {
		t.Fatalf("want empty page, got %d", len(p.Items))
	}
	if p.NextCursor != "" {
		t.Fatalf("nextCursor=%q", p.NextCursor)
	}
}

func TestAttendeeForbiddenDoesNotLeakExistence(t *testing.T) {
	f := setup(t)
	rec := f.list(t, f.attend, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["code"] != "FORBIDDEN" {
		t.Fatalf("code=%v", out["code"])
	}

	cookie := f.signIn(t, f.attend)
	req := httptest.NewRequest(http.MethodGet, "/events/01999999-9999-7000-8000-000000000000/invitations", nil)
	req.AddCookie(cookie)
	missing := httptest.NewRecorder()
	f.handler.ServeHTTP(missing, req)
	if missing.Code != http.StatusForbidden {
		t.Fatalf("missing event status %d body=%s", missing.Code, missing.Body.String())
	}
}

func TestNoEmailWithoutEmailPermission(t *testing.T) {
	f := setup(t)
	role := "invite_reader_test"
	_, err := f.db.Exec(`INSERT INTO roles (name, description) VALUES ($1, 'test') ON CONFLICT DO NOTHING`, role)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.db.Exec(`
		INSERT INTO role_permissions (role, permission) VALUES ($1, 'event.read'), ($1, 'invitation.read')
		ON CONFLICT DO NOTHING
	`, role)
	if err != nil {
		t.Fatal(err)
	}
	userID := "usr_0400"
	_, err = f.db.Exec(`
		INSERT INTO event_members (event_id, user_id, role) VALUES ($1, $2, $3)
		ON CONFLICT (event_id, user_id) DO UPDATE SET role = $3
	`, f.eventID, userID, role)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM event_members WHERE event_id = $1 AND user_id = $2`, f.eventID, userID)
		_, _ = f.db.Exec(`DELETE FROM roles WHERE name = $1`, role)
	})

	rec := f.list(t, userID, "?limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	p := decodePage(t, rec)
	if len(p.Items) == 0 {
		t.Fatal("expected items")
	}
	for _, it := range p.Items {
		if _, ok := it["email"]; ok {
			t.Fatalf("email leaked: %v", it)
		}
		raw, _ := json.Marshal(it)
		if bytes.Contains(bytes.ToLower(raw), []byte(`"email"`)) {
			t.Fatalf("email key present: %s", raw)
		}
	}
}

func TestKeysetUsesInvitationsIndex(t *testing.T) {
	f := setup(t)
	var ids []string
	err := f.db.Select(&ids, `
		SELECT id FROM invitations WHERE event_id = $1 ORDER BY id LIMIT 201
	`, f.eventID)
	if err != nil {
		t.Fatalf("deep cursor: %v", err)
	}
	if len(ids) < 201 {
		t.Fatalf("want 201 ids, got %d", len(ids))
	}
	after := ids[200]

	var lines []string
	err = f.db.Select(&lines, `
		EXPLAIN
		SELECT id, event_id, email, role, status, invited_by, user_id
		FROM   invitations
		WHERE  event_id = $1
		  AND  id > $2
		ORDER  BY event_id, id
		LIMIT  50
	`, f.eventID, after)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	plan := strings.Join(lines, "\n")
	if !strings.Contains(plan, "invitations_event_id_id_idx") {
		t.Fatalf("expected invitations_event_id_id_idx, plan:\n%s", plan)
	}
	if strings.Contains(plan, "Seq Scan") {
		t.Fatalf("seq scan, plan:\n%s", plan)
	}
}
