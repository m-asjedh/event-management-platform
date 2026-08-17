package events

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
)

var ErrNotFound = errors.New("event not found")

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context, id string) (authz.Event, error) {
	var e authz.Event
	err := s.db.GetContext(ctx, &e, `
		SELECT id, name, description, time_zone, starts_at, ends_at
		FROM   events
		WHERE  id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return authz.Event{}, ErrNotFound
	}
	return e, err
}

func (s *Store) Members(ctx context.Context, eventID string) ([]authz.Member, error) {
	var members []authz.Member
	err := s.db.SelectContext(ctx, &members, `
		SELECT m.user_id, u.name, u.email, m.role
		FROM   event_members m
		JOIN   auth.users    u ON u.id = m.user_id
		WHERE  m.event_id = $1
		ORDER  BY m.role, u.name
	`, eventID)
	return members, err
}
