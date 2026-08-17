package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func copyAll(ctx context.Context, conn *pgx.Conn, d Data) error {
	type job struct {
		table   pgx.Identifier
		columns []string
		n       int
		row     func(int) ([]any, error)
	}

	jobs := []job{
		{
			table:   pgx.Identifier{"auth", "users"},
			columns: []string{"id", "name", "email", "email_verified"},
			n:       len(d.Users),
			row: func(i int) ([]any, error) {
				u := d.Users[i]
				return []any{u.ID, u.Name, u.Email, true}, nil
			},
		},
		{
			table:   pgx.Identifier{"auth", "accounts"},
			columns: []string{"id", "user_id", "account_id", "provider_id", "password"},
			n:       len(d.Accounts),
			row: func(i int) ([]any, error) {
				a := d.Accounts[i]
				return []any{a.ID, a.UserID, a.AccountID, a.ProviderID, a.Password}, nil
			},
		},
		{
			table:   pgx.Identifier{"events"},
			columns: []string{"id", "name", "description", "time_zone", "starts_at", "ends_at"},
			n:       len(d.Events),
			row: func(i int) ([]any, error) {
				e := d.Events[i]
				return []any{e.ID, e.Name, e.Description, e.TimeZone, e.StartsAt, e.EndsAt}, nil
			},
		},
		{
			table:   pgx.Identifier{"rooms"},
			columns: []string{"id", "event_id", "name", "capacity"},
			n:       len(d.Rooms),
			row: func(i int) ([]any, error) {
				r := d.Rooms[i]
				return []any{r.ID, r.EventID, r.Name, r.Capacity}, nil
			},
		},
		{
			table:   pgx.Identifier{"event_members"},
			columns: []string{"event_id", "user_id", "role"},
			n:       len(d.Members),
			row: func(i int) ([]any, error) {
				m := d.Members[i]
				return []any{m.EventID, m.UserID, m.Role}, nil
			},
		},
		{
			table:   pgx.Identifier{"sessions"},
			columns: []string{"id", "event_id", "room_id", "title", "description", "starts_at", "ends_at"},
			n:       len(d.Sessions),
			row: func(i int) ([]any, error) {
				s := d.Sessions[i]
				return []any{s.ID, s.EventID, s.RoomID, s.Title, s.Description, s.StartsAt, s.EndsAt}, nil
			},
		},
		{
			table:   pgx.Identifier{"session_speakers"},
			columns: []string{"session_id", "user_id"},
			n:       len(d.Speakers),
			row: func(i int) ([]any, error) {
				s := d.Speakers[i]
				return []any{s.SessionID, s.UserID}, nil
			},
		},
		{
			table:   pgx.Identifier{"invitations"},
			columns: []string{"id", "event_id", "email", "role", "status", "invited_by", "user_id"},
			n:       len(d.Invitations),
			row: func(i int) ([]any, error) {
				in := d.Invitations[i]
				return []any{in.ID, in.EventID, in.Email, in.Role, in.Status, in.InvitedBy, in.UserID}, nil
			},
		},
	}

	for _, j := range jobs {
		n, err := conn.CopyFrom(ctx, j.table, j.columns, pgx.CopyFromSlice(j.n, j.row))
		if err != nil {
			return fmt.Errorf("copy %s: %w", j.table.Sanitize(), err)
		}
		fmt.Printf("copied %d rows into %s\n", n, j.table.Sanitize())
	}
	return nil
}
