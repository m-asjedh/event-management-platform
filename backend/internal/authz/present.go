package authz

import "time"

type Event struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	TimeZone    string    `db:"time_zone"`
	StartsAt    time.Time `db:"starts_at"`
	EndsAt      time.Time `db:"ends_at"`
}

type Member struct {
	UserID string `db:"user_id" json:"userId"`
	Name   string `db:"name"    json:"name"`
	Email  string `db:"email"   json:"email,omitempty"`
	Role   string `db:"role"    json:"role"`
}

type EventView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TimeZone    string    `json:"timeZone"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	Members     []Member  `json:"members,omitempty"`
}

// PresentEvent is the body filter. Same Can() the handler used for the URL.
//
// Passing members in does not leak them: if the grant lacks member.read the
// roster is dropped. If it lacks user.email.read, every email is cleared.
func PresentEvent(event Event, members []Member, g Grant) EventView {
	view := EventView{
		ID:          event.ID,
		Name:        event.Name,
		Description: event.Description,
		TimeZone:    event.TimeZone,
		StartsAt:    event.StartsAt,
		EndsAt:      event.EndsAt,
	}
	if !g.Can(MemberRead) {
		return view
	}

	out := make([]Member, 0, len(members))
	for _, m := range members {
		if !g.Can(UserEmailRead) {
			m.Email = ""
		}
		out = append(out, m)
	}
	view.Members = out
	return view
}
