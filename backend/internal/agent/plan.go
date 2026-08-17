package agent

import (
	"fmt"
	"regexp"
	"strings"
)

const maxSteps = 8

type intent int

const (
	intentUnknown intent = iota
	intentEventsInZone
	intentSessionCount
	intentInvitations
)

type action struct {
	tool    string
	eventID string
	finish  bool
	answer  string
}

// next decides the next GET from the question and observations so far.
// Event ids always come from an earlier list, never from the user.
func next(question string, hist []Observation) action {
	q := strings.TrimSpace(question)
	kind := classify(q)

	list := lastByPrefix(hist, "/events?")
	if list.Call.Path == "" {
		list = lastExactPrefix(hist, "/events")
	}

	switch kind {
	case intentEventsInZone:
		if !called(hist, "/events") {
			return action{tool: "list_events"}
		}
		if list.Status != 200 {
			return action{finish: true, answer: apiFailure("list events", list)}
		}
		zone := zoneFrom(q)
		if zone == "" {
			return action{finish: true, answer: "I could not tell which time zone you meant from the question."}
		}
		var names []string
		for _, ev := range jsonItems(list.Body) {
			if jsonString(ev, "timeZone") == zone {
				names = append(names, jsonString(ev, "name"))
			}
		}
		if len(names) == 0 {
			return action{finish: true, answer: fmt.Sprintf("GET /events returned no events in %s.", zone)}
		}
		return action{finish: true, answer: fmt.Sprintf("These events are in %s: %s.", zone, strings.Join(names, ", "))}

	case intentSessionCount:
		if !called(hist, "/events") {
			return action{tool: "list_events"}
		}
		if list.Status != 200 {
			return action{finish: true, answer: apiFailure("list events", list)}
		}
		name := eventNameFrom(q)
		id := findEventID(list.Body, name)
		if id == "" {
			return action{finish: true, answer: fmt.Sprintf("GET /events did not include %q, so I cannot count its sessions.", name)}
		}
		sess := lastBySuffix(hist, "/sessions")
		if sess.Call.Path == "" {
			return action{tool: "list_sessions", eventID: id}
		}
		if sess.Status != 200 {
			return action{finish: true, answer: apiFailure("list sessions", sess)}
		}
		n := len(jsonItems(sess.Body))
		return action{finish: true, answer: fmt.Sprintf("%s has %d sessions.", name, n)}

	case intentInvitations:
		if !called(hist, "/events") {
			return action{tool: "list_events"}
		}
		if list.Status != 200 {
			return action{finish: true, answer: apiFailure("list events", list)}
		}
		name := eventNameFrom(q)
		id := findEventID(list.Body, name)
		if id == "" {
			return action{finish: true, answer: fmt.Sprintf("GET /events did not include %q, so I cannot ask for its invitations.", name)}
		}
		inv := lastBySuffix(hist, "/invitations")
		if inv.Call.Path == "" {
			return action{tool: "list_invitations", eventID: id}
		}
		if inv.Status == 403 {
			code := jsonString(jsonObject(inv.Body), "code")
			if code == "" {
				code = "FORBIDDEN"
			}
			return action{finish: true, answer: fmt.Sprintf(
				"The API returned 403 %s. I am not allowed to see invitations for %s, so I have no list to report.",
				code, name,
			)}
		}
		if inv.Status != 200 {
			return action{finish: true, answer: apiFailure("list invitations", inv)}
		}
		n := len(jsonItems(inv.Body))
		return action{finish: true, answer: fmt.Sprintf("%s has a page of %d invitations.", name, n)}
	}

	if !called(hist, "/events") {
		return action{tool: "list_events"}
	}
	return action{finish: true, answer: "I only answer from GET responses, and I do not recognise that question."}
}

func classify(q string) intent {
	l := strings.ToLower(q)
	switch {
	case strings.Contains(l, "invitation"):
		return intentInvitations
	case strings.Contains(l, "session"):
		return intentSessionCount
	case strings.Contains(l, "which events") || strings.Contains(l, "events are in"):
		return intentEventsInZone
	default:
		return intentUnknown
	}
}

func apiFailure(what string, obs Observation) string {
	code := jsonString(jsonObject(obs.Body), "code")
	if code == "" {
		code = httpStatusName(obs.Status)
	}
	return fmt.Sprintf("The API refused to %s (%d %s). I am not inventing a result.", what, obs.Status, code)
}

func httpStatusName(status int) string {
	switch status {
	case 401:
		return "UNAUTHENTICATED"
	case 403:
		return "FORBIDDEN"
	case 404:
		return "NOT_FOUND"
	default:
		return "ERROR"
	}
}

func called(hist []Observation, prefix string) bool {
	for _, obs := range hist {
		p := strings.SplitN(obs.Call.Path, "?", 2)[0]
		if p == prefix || strings.HasPrefix(obs.Call.Path, prefix+"?") {
			return true
		}
	}
	return false
}

func lastByPrefix(hist []Observation, prefix string) Observation {
	var last Observation
	for _, obs := range hist {
		if strings.HasPrefix(obs.Call.Path, prefix) {
			last = obs
		}
	}
	return last
}

func lastExactPrefix(hist []Observation, prefix string) Observation {
	var last Observation
	for _, obs := range hist {
		p := strings.SplitN(obs.Call.Path, "?", 2)[0]
		if p == prefix {
			last = obs
		}
	}
	return last
}

func lastBySuffix(hist []Observation, suffix string) Observation {
	var last Observation
	for _, obs := range hist {
		p := strings.SplitN(obs.Call.Path, "?", 2)[0]
		if strings.HasSuffix(p, suffix) {
			last = obs
		}
	}
	return last
}

func findEventID(body, name string) string {
	if name == "" {
		return ""
	}
	want := strings.ToLower(name)
	for _, ev := range jsonItems(body) {
		if strings.ToLower(jsonString(ev, "name")) == want {
			return jsonString(ev, "id")
		}
	}
	for _, ev := range jsonItems(body) {
		if strings.Contains(strings.ToLower(jsonString(ev, "name")), want) {
			return jsonString(ev, "id")
		}
	}
	return ""
}

var (
	ianaZone = regexp.MustCompile(`[A-Za-z]+/[A-Za-z_]+`)
	confNum  = regexp.MustCompile(`(?i)Conference\s+\d{2}`)
)

func zoneFrom(q string) string {
	if m := ianaZone.FindString(q); m != "" {
		return m
	}
	l := strings.ToLower(q)
	switch {
	case strings.Contains(l, "new york"):
		return "America/New_York"
	case strings.Contains(l, "london"):
		return "Europe/London"
	case strings.Contains(l, "colombo"):
		return "Asia/Colombo"
	case strings.Contains(l, "sydney"):
		return "Australia/Sydney"
	case strings.Contains(l, "berlin"):
		return "Europe/Berlin"
	case strings.Contains(l, "los angeles"):
		return "America/Los_Angeles"
	case strings.Contains(l, "tokyo"):
		return "Asia/Tokyo"
	case strings.Contains(l, "auckland"):
		return "Pacific/Auckland"
	default:
		return ""
	}
}

func eventNameFrom(q string) string {
	known := []string{
		"Prompt Injection Conference",
		"DST Spring Forward",
		"DST Fall Back",
	}
	for _, name := range known {
		if strings.Contains(strings.ToLower(q), strings.ToLower(name)) {
			return name
		}
	}
	if m := confNum.FindString(q); m != "" {
		return strings.Join(strings.Fields(m), " ")
	}
	return strings.TrimSpace(q)
}
