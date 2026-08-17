package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	springID = "11111111-1111-7111-8111-111111111111"
	fallID   = "22222222-2222-7222-8222-222222222222"
	injectID = "33333333-3333-7333-8333-333333333333"
	londonID = "44444444-4444-7444-8444-444444444444"
)

func TestEventsInNewYork(t *testing.T) {
	srv := fakeAPI(t, fakeData{events: []map[string]any{
		event(springID, "DST Spring Forward", "America/New_York"),
		event(fallID, "DST Fall Back", "America/New_York"),
		event(londonID, "Prompt Injection Conference", "Europe/London"),
	}})
	out, trace := ask(t, srv, "Which events are in America/New_York?")

	assertCalls(t, out.Calls, "GET /events?limit=100")
	if !strings.Contains(out.Answer, "DST Spring Forward") || !strings.Contains(out.Answer, "DST Fall Back") {
		t.Fatalf("answer missing NY events: %s", out.Answer)
	}
	if strings.Contains(out.Answer, "Prompt Injection Conference") {
		t.Fatalf("answer leaked a London event: %s", out.Answer)
	}
	if !strings.Contains(out.Answer, "America/New_York") {
		t.Fatalf("answer missing zone: %s", out.Answer)
	}
	assertTraceShows(t, trace, "GET /events?limit=100")
}

func TestSessionCountSpringForward(t *testing.T) {
	sessions := make([]map[string]any, 26)
	for i := range sessions {
		sessions[i] = map[string]any{"id": fmt.Sprintf("s%02d", i), "title": fmt.Sprintf("Talk %d", i)}
	}
	srv := fakeAPI(t, fakeData{
		events: []map[string]any{
			event(springID, "DST Spring Forward", "America/New_York"),
			event(fallID, "DST Fall Back", "America/New_York"),
		},
		sessions: map[string][]map[string]any{springID: sessions},
	})
	out, trace := ask(t, srv, "How many sessions does DST Spring Forward have?")

	assertCalls(t, out.Calls,
		"GET /events?limit=100",
		"GET /events/"+springID+"/sessions",
	)
	if !strings.Contains(out.Answer, "26") {
		t.Fatalf("answer missing count 26: %s", out.Answer)
	}
	if !strings.Contains(out.Answer, "DST Spring Forward") {
		t.Fatalf("answer missing event name: %s", out.Answer)
	}
	assertTraceShows(t, trace, "GET /events/"+springID+"/sessions")
}

func TestInvitationsDeniedForAttendee(t *testing.T) {
	srv := fakeAPI(t, fakeData{
		events: []map[string]any{
			event(injectID, "Prompt Injection Conference", "Europe/London"),
			event(fallID, "DST Fall Back", "America/New_York"),
		},
		inviteStatus: map[string]int{injectID: http.StatusForbidden},
	})
	q := "Is seed.attendee allowed to see the invitations for Prompt Injection Conference?"
	out, trace := ask(t, srv, q)

	assertCalls(t, out.Calls,
		"GET /events?limit=100",
		"GET /events/"+injectID+"/invitations",
	)
	if !strings.Contains(out.Answer, "403") {
		t.Fatalf("answer hid the 403: %s", out.Answer)
	}
	if !strings.Contains(strings.ToUpper(out.Answer), "FORBIDDEN") && !strings.Contains(out.Answer, "not allowed") {
		t.Fatalf("answer did not say it was denied: %s", out.Answer)
	}
	if strings.Contains(out.Answer, "invite-") || strings.Contains(out.Answer, "@invite.example") {
		t.Fatalf("answer fabricated invitation rows: %s", out.Answer)
	}
	if strings.Contains(out.Answer, "1000") {
		t.Fatalf("answer fabricated an invitation count: %s", out.Answer)
	}
	if strings.Contains(out.Answer, "page of") {
		t.Fatalf("answer treated a 403 as a list: %s", out.Answer)
	}
	assertTraceShows(t, trace, "403 FORBIDDEN")
}

func TestClientDoesNotSendWrites(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient(srv.URL, nil)
	if _, err := c.Get("/events"); err != nil {
		t.Fatal(err)
	}
	for _, m := range methods {
		if m != http.MethodGet {
			t.Fatalf("non-GET %s", m)
		}
	}
}

func ask(t *testing.T, srv *httptest.Server, question string) (Outcome, string) {
	t.Helper()
	var buf bytes.Buffer
	out, err := Run(&buf, NewClient(srv.URL, nil), question)
	if err != nil {
		t.Fatal(err)
	}
	return out, buf.String()
}

func assertCalls(t *testing.T, got []Call, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls %d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Fatalf("call %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func assertTraceShows(t *testing.T, trace, needle string) {
	t.Helper()
	if !strings.Contains(trace, needle) {
		t.Fatalf("trace missing %q:\n%s", needle, trace)
	}
	if !strings.Contains(trace, "Answer:") {
		t.Fatalf("trace missing the answer:\n%s", trace)
	}
}

type fakeData struct {
	events       []map[string]any
	sessions     map[string][]map[string]any
	inviteStatus map[string]int
}

func event(id, name, zone string) map[string]any {
	return map[string]any{"id": id, "name": name, "timeZone": zone}
}

func fakeAPI(t *testing.T, data fakeData) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": data.events})
	})
	mux.HandleFunc("GET /events/{id}/sessions", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		items := data.sessions[id]
		if items == nil {
			items = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	})
	mux.HandleFunc("GET /events/{id}/invitations", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		status := data.inviteStatus[id]
		if status == http.StatusForbidden {
			writeJSON(w, http.StatusForbidden, map[string]string{"code": "FORBIDDEN", "reason": "not allowed"})
			return
		}
		if status == 0 {
			status = http.StatusOK
		}
		writeJSON(w, status, map[string]any{"items": []map[string]any{
			{"email": "invite-00-0000@invite.example"},
		}})
	})
	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("agent issued a write: POST %s", r.URL.Path)
		http.Error(w, "writes are not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("PATCH /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("agent issued a write: PATCH %s", r.URL.Path)
		http.Error(w, "writes are not allowed", http.StatusMethodNotAllowed)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
