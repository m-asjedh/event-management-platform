package seed

import (
	"strings"
	"testing"
	_ "time/tzdata"
)

func TestGenerateCountsAndFixtures(t *testing.T) {
	d := Generate("salt:hash")

	if len(d.Users) != UserCount {
		t.Fatalf("users: got %d", len(d.Users))
	}
	if len(d.Events) != EventCount {
		t.Fatalf("events: got %d", len(d.Events))
	}
	if len(d.Invitations) != EventCount*InvitesPerEvent {
		t.Fatalf("invitations: got %d", len(d.Invitations))
	}

	zones := map[string]int{}
	var spring, fall, injection int
	for _, e := range d.Events {
		zones[e.TimeZone]++
		switch e.Name {
		case SpringEventName:
			spring++
			if e.TimeZone != "America/New_York" {
				t.Fatalf("spring zone %s", e.TimeZone)
			}
		case FallEventName:
			fall++
		case InjectionEvent:
			injection++
			if e.Description != InjectionText {
				t.Fatalf("injection text mismatch: %q", e.Description)
			}
		}
	}
	if spring != 1 || fall != 1 || injection != 1 {
		t.Fatalf("fixtures spring=%d fall=%d injection=%d", spring, fall, injection)
	}
	if len(zones) < 4 {
		t.Fatalf("want several zones, got %v", zones)
	}

	adminOn0 := ""
	attendeeOn1 := map[string]bool{}
	for _, m := range d.Members {
		if m.EventID == d.Events[0].ID && m.Role == "admin" {
			adminOn0 = m.UserID
		}
		if m.EventID == d.Events[1].ID && m.Role == "attendee" {
			attendeeOn1[m.UserID] = true
		}
	}
	if adminOn0 == "" || !attendeeOn1[adminOn0] {
		t.Fatal("expected event-0 admin to be an attendee on event 1")
	}

	for _, e := range d.Events {
		if !strings.Contains(e.Description, InjectionText) && e.Name != InjectionEvent {
			continue
		}
		if e.Name != InjectionEvent {
			t.Fatalf("injection text leaked onto %q", e.Name)
		}
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	a := Generate("x")
	b := Generate("x")
	if a.Events[0].ID != b.Events[0].ID {
		t.Fatal("event ids should be stable")
	}
	if a.Users[7].Email != b.Users[7].Email {
		t.Fatal("user emails should be stable")
	}
	if a.Invitations[123].Email != b.Invitations[123].Email {
		t.Fatal("invitation emails should be stable")
	}
}
