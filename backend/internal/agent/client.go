package agent

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Call is one request the agent made. Tests assert these, not just the prose.
type Call struct {
	Method string
	Path   string
}

func (c Call) String() string {
	return c.Method + " " + c.Path
}

// Observation is what came back. The planner may only answer from these.
type Observation struct {
	Call    Call
	Status  int
	Body    string
	Summary string
}

// Client talks to the public API as the signed-in user. GET only.
type Client struct {
	base   string
	cookie *http.Cookie
	http   *http.Client
}

func NewClient(apiURL string, cookie *http.Cookie) *Client {
	return &Client{
		base:   strings.TrimRight(apiURL, "/"),
		cookie: cookie,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Get(path string) (Observation, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return Observation{}, err
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Observation{}, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Observation{}, err
	}
	obs := Observation{
		Call:   Call{Method: http.MethodGet, Path: path},
		Status: resp.StatusCode,
		Body:   string(raw),
	}
	obs.Summary = summarise(obs)
	return obs, nil
}

func summarise(obs Observation) string {
	if obs.Status == http.StatusForbidden {
		return "403 FORBIDDEN"
	}
	if obs.Status == http.StatusUnauthorized {
		return "401 UNAUTHENTICATED"
	}
	if obs.Status == http.StatusNotFound {
		return "404 NOT_FOUND"
	}
	if obs.Status != http.StatusOK {
		return fmt.Sprintf("%d", obs.Status)
	}
	items := jsonItems(obs.Body)
	switch {
	case strings.HasSuffix(strings.SplitN(obs.Call.Path, "?", 2)[0], "/sessions"):
		return fmt.Sprintf("200 — %d sessions", len(items))
	case strings.HasSuffix(strings.SplitN(obs.Call.Path, "?", 2)[0], "/rooms"):
		return fmt.Sprintf("200 — %d rooms", len(items))
	case strings.HasSuffix(strings.SplitN(obs.Call.Path, "?", 2)[0], "/invitations"):
		return fmt.Sprintf("200 — %d invitations", len(items))
	default:
		if strings.HasPrefix(obs.Call.Path, "/events/") && !strings.Contains(obs.Call.Path[1:], "/") {
			return "200 — one event"
		}
		if strings.HasPrefix(obs.Call.Path, "/events") {
			return fmt.Sprintf("200 — %d events", len(items))
		}
		return "200"
	}
}
