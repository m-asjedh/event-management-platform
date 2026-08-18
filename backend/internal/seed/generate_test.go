package seed

import (
	"strings"
	"testing"
	_ "time/tzdata"
)

func TestGenerateCountsAndFixtures(t *testing.T) {
	t.Run("full", func(t *testing.T) { assertFixtures(t, Generate("salt:hash", SizeFull), SizeFull) })
	t.Run("small", func(t *testing.T) { assertFixtures(t, Generate("salt:hash", SizeSmall), SizeSmall) })
}

func assertFixtures(t *testing.T, d Data, size Size) {
	t.Helper()
	if len(d.Users) != size.Users {
		t.Fatalf("users: got %d want %d", len(d.Users), size.Users)
	}
	if len(d.Events) != size.Events {
		t.Fatalf("events: got %d want %d", len(d.Events), size.Events)
	}
	if len(d.Invitations) != size.Events*size.InvitesPerEvent {
		t.Fatalf("invitations: got %d want %d", len(d.Invitations), size.Events*size.InvitesPerEvent)
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

	if d.Users[0].Email != "seed.admin@example.com" || d.Users[1].Email != "seed.attendee@example.com" {
		t.Fatalf("demo logins: %s %s", d.Users[0].Email, d.Users[1].Email)
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

	var rooms, sessions int
	for _, r := range d.Rooms {
		if r.EventID == d.Events[0].ID {
			rooms++
		}
	}
	for _, s := range d.Sessions {
		if s.EventID == d.Events[0].ID {
			sessions++
		}
	}
	if rooms < 2 || sessions < 8 {
		t.Fatalf("spring schedule too thin: rooms=%d sessions=%d", rooms, sessions)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	for _, size := range []Size{SizeSmall, SizeFull} {
		a := Generate("x", size)
		b := Generate("x", size)
		if a.Events[0].ID != b.Events[0].ID {
			t.Fatalf("%s: event ids should be stable", size.Label)
		}
		if a.Users[7].Email != b.Users[7].Email {
			t.Fatalf("%s: user emails should be stable", size.Label)
		}
		if a.Invitations[123].Email != b.Invitations[123].Email {
			t.Fatalf("%s: invitation emails should be stable", size.Label)
		}
	}
}

func TestParseSize(t *testing.T) {
	if ParseSize("").Label != "small" || ParseSize("SMALL").Label != "small" {
		t.Fatal("empty / small should be the default")
	}
	if ParseSize("full").Label != "full" || ParseSize("FULL").Label != "full" {
		t.Fatal("full")
	}
}
