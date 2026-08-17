package contract

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

const cookieName = identity.CookieName

type harness struct {
	t       *testing.T
	db      *sqlx.DB
	api     string
	secret  string
	router  routers.Router
	client  *http.Client
	checked int

	admin        string
	attend       string
	adminEvent   string
	attendEvent  string
	springEvent  string
	springAttend string
	talkA1       string
	talkA1Room   string
	talkA1Ver    int
	talkA2       string
}

func setup(t *testing.T) *harness {
	t.Helper()
	api := os.Getenv("API_URL")
	if api == "" {
		t.Skip("API_URL is not set (contract runs against the live api service)")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is not set")
	}
	secret := os.Getenv("BETTER_AUTH_SECRET")
	if secret == "" {
		secret = "dev-only-not-a-real-secret-000000000000"
	}
	specPath := os.Getenv("OPENAPI_PATH")
	if specPath == "" {
		t.Fatal("OPENAPI_PATH is not set")
	}

	ctx := context.Background()
	loader := &openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("spec invalid: %v", err)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("router: %v", err)
	}

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	h := &harness{
		t: t, db: db, api: api, secret: secret, router: router,
		client: &http.Client{Timeout: 15 * time.Second},
		admin:  "usr_0000",
		attend: "usr_0001",
	}
	if err := db.Get(&h.adminEvent, `
		SELECT event_id FROM event_members WHERE user_id = $1 AND role = 'admin' LIMIT 1
	`, h.admin); err != nil {
		t.Fatalf("admin event: %v", err)
	}
	if err := db.Get(&h.attendEvent, `
		SELECT event_id FROM event_members WHERE user_id = $1 AND role = 'attendee' LIMIT 1
	`, h.admin); err != nil {
		t.Fatalf("attendee event: %v", err)
	}
	if err := db.Get(&h.springEvent, `SELECT id FROM events WHERE name = $1`, "DST Spring Forward"); err != nil {
		t.Fatalf("spring event: %v", err)
	}
	if err := db.Get(&h.springAttend, `
		SELECT user_id FROM event_members WHERE event_id = $1 AND role = 'attendee' LIMIT 1
	`, h.springEvent); err != nil {
		t.Fatalf("spring attendee: %v", err)
	}
	_, _ = db.Exec(`DELETE FROM events WHERE name = $1`, "Contract Create")
	_, _ = db.Exec(`DELETE FROM sessions WHERE title = $1`, "Contract Slot")
	if err := db.Get(&h.talkA1, `
		SELECT s.id FROM sessions s JOIN events e ON e.id = s.event_id
		WHERE e.name = $1 AND s.title = $2
	`, "DST Spring Forward", "Talk A1"); err != nil {
		t.Fatalf("Talk A1: %v", err)
	}
	if err := db.Get(&h.talkA1Room, `SELECT room_id FROM sessions WHERE id = $1`, h.talkA1); err != nil {
		t.Fatalf("Talk A1 room: %v", err)
	}
	if err := db.Get(&h.talkA1Ver, `SELECT version FROM sessions WHERE id = $1`, h.talkA1); err != nil {
		t.Fatalf("Talk A1 version: %v", err)
	}
	if err := db.Get(&h.talkA2, `
		SELECT s.id FROM sessions s JOIN events e ON e.id = s.event_id
		WHERE e.name = $1 AND s.title = $2
	`, "DST Spring Forward", "Talk A2"); err != nil {
		t.Fatalf("Talk A2: %v", err)
	}
	return h
}

func (h *harness) signIn(userID string) *http.Cookie {
	h.t.Helper()
	n := time.Now().UnixNano()
	token := fmt.Sprintf("tok-contract-%s-%d", userID, n)
	sid := fmt.Sprintf("sid-contract-%s-%d", userID, n)
	_, err := h.db.Exec(`
		INSERT INTO auth.sessions (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, now() + interval '1 hour')
	`, sid, userID, token)
	if err != nil {
		h.t.Fatalf("session: %v", err)
	}
	h.t.Cleanup(func() {
		_, _ = h.db.Exec(`DELETE FROM auth.sessions WHERE id = $1`, sid)
	})
	mac := hmac.New(sha256.New, []byte(h.secret))
	_, _ = mac.Write([]byte(token))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return &http.Cookie{Name: cookieName, Value: url.PathEscape(token + "." + sig)}
}

func (h *harness) do(method, path string, body any, cookie *http.Cookie) *http.Response {
	h.t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, h.api+path, rdr)
	if err != nil {
		h.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func (h *harness) expect(name, method, path string, body any, cookie *http.Cookie, wantStatus int) map[string]any {
	h.t.Helper()
	resp := h.do(method, path, body, cookie)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		h.t.Fatalf("%s: status %d want %d body=%s", name, resp.StatusCode, wantStatus, raw)
	}

	valReq, err := http.NewRequest(method, "http://localhost:8080"+path, nil)
	if err != nil {
		h.t.Fatal(err)
	}
	route, pathParams, err := h.router.FindRoute(valReq)
	if err != nil {
		h.t.Fatalf("%s: spec has no operation for %s %s: %v", name, method, path, err)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    valReq,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
	}
	input.SetBodyBytes(raw)
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		h.t.Fatalf("%s: response does not honour the spec: %v\nbody=%s", name, err, raw)
	}
	h.checked++

	var parsed map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			h.t.Fatalf("%s: json: %v body=%s", name, err, raw)
		}
	}
	return parsed
}
