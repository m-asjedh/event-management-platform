package authz

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func attendeeGrant() Grant {
	return NewGrant("attendee", []string{EventRead, RoomRead, SessionRead})
}

func adminGrant() Grant {
	return NewGrant("admin", []string{
		EventRead, "event.update", "event.delete",
		RoomRead, RoomManage,
		SessionRead, SessionCreate, SessionUpdate, "session.delete",
		MemberRead, "member.invite", "member.remove", "member.role.update",
		"invitation.read", "invitation.create", "invitation.revoke",
		UserEmailRead,
	})
}

func sampleEvent() Event {
	return Event{
		ID:          "01924000-0000-7000-8000-000000000001",
		Name:        "Berlin Conf",
		Description: "A conference. No emails in this field.",
		TimeZone:    "Europe/Berlin",
		StartsAt:    time.Date(2026, 3, 28, 19, 0, 0, 0, time.UTC),
		EndsAt:      time.Date(2026, 3, 30, 16, 0, 0, 0, time.UTC),
	}
}

func sampleRoster() []Member {
	return []Member{
		{UserID: "user-admin", Name: "Ada", Email: "secret.roster@example.com", Role: "admin"},
		{UserID: "user-attendee", Name: "Bob", Email: "bob.hidden@example.com", Role: "attendee"},
	}
}

func TestAttendeeEventOmitsRosterAndEmails(t *testing.T) {
	view := PresentEvent(sampleEvent(), sampleRoster(), attendeeGrant())
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)

	if view.Members != nil {
		t.Fatalf("attendee must not receive a roster, got %#v", view.Members)
	}
	if strings.Contains(raw, `"members"`) {
		t.Fatalf("attendee JSON must not contain a members key:\n%s", raw)
	}
	for _, leak := range []string{
		"secret.roster@example.com",
		"bob.hidden@example.com",
		"@example.com",
	} {
		if strings.Contains(raw, leak) {
			t.Fatalf("attendee JSON leaked %q:\n%s", leak, raw)
		}
	}
}

func TestAdminEventIncludesRosterEmails(t *testing.T) {
	view := PresentEvent(sampleEvent(), sampleRoster(), adminGrant())
	if len(view.Members) != 2 {
		t.Fatalf("admin should see the roster, got %d members", len(view.Members))
	}
	if view.Members[0].Email != "secret.roster@example.com" {
		t.Fatalf("admin should see emails, got %q", view.Members[0].Email)
	}
}

func TestGrantDoesNotSwitchOnRoleName(t *testing.T) {
	// A new role with the same rows as attendee must behave the same.
	// If handlers switched on Role == "attendee", this would need a code change.
	guest := NewGrant("guest", []string{EventRead, RoomRead, SessionRead})
	view := PresentEvent(sampleEvent(), sampleRoster(), guest)
	if view.Members != nil {
		t.Fatalf("role with no member.read must not see the roster")
	}
	if !guest.Can(EventRead) || guest.Can(MemberRead) {
		t.Fatal("Can must follow the permission rows, not the role name")
	}
}
