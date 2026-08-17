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

func (s *Store) ZoneExists(ctx context.Context, name string) (bool, error) {
	var found string
	err := s.db.GetContext(ctx, &found, `SELECT name FROM time_zones WHERE name = $1`, name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) ListForUser(ctx context.Context, userID, after string, limit int) ([]authz.Event, error) {
	var afterArg any
	if after != "" {
		afterArg = after
	}
	var rows []authz.Event
	err := s.db.SelectContext(ctx, &rows, `
		SELECT e.id, e.name, e.description, e.time_zone, e.starts_at, e.ends_at
		FROM   events e
		JOIN   event_members m ON m.event_id = e.id AND m.user_id = $1
		JOIN   role_permissions rp ON rp.role = m.role AND rp.permission = 'event.read'
		WHERE  ($2::uuid IS NULL OR e.id > $2::uuid)
		ORDER  BY e.id
		LIMIT  $3
	`, userID, afterArg, limit)
	return rows, err
}

func (s *Store) Create(ctx context.Context, e authz.Event, creatorID string) (authz.Event, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return authz.Event{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var out authz.Event
	err = tx.GetContext(ctx, &out, `
		INSERT INTO events (name, description, time_zone, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, time_zone, starts_at, ends_at
	`, e.Name, e.Description, e.TimeZone, e.StartsAt, e.EndsAt)
	if err != nil {
		return authz.Event{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO event_members (event_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, out.ID, creatorID)
	if err != nil {
		return authz.Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return authz.Event{}, err
	}
	return out, nil
}
