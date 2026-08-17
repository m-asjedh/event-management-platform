package sessions

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound     = errors.New("session not found")
	ErrStaleVersion = errors.New("stale version")
	ErrRoomConflict = errors.New("room conflict")
)

type Session struct {
	ID          string    `db:"id"          json:"id"`
	EventID     string    `db:"event_id"    json:"eventId"`
	RoomID      *string   `db:"room_id"     json:"roomId"`
	Title       string    `db:"title"       json:"title"`
	Description string    `db:"description" json:"description"`
	StartsAt    time.Time `db:"starts_at"   json:"startsAt"`
	EndsAt      time.Time `db:"ends_at"     json:"endsAt"`
	Version     int       `db:"version"     json:"version"`
	TimeZone    string    `db:"time_zone"   json:"-"`
}

type Clash struct {
	ID    string `db:"id"`
	Title string `db:"title"`
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context, id string) (Session, error) {
	var row Session
	err := s.db.GetContext(ctx, &row, `
		SELECT s.id, s.event_id, s.room_id, s.title, s.description,
		       s.starts_at, s.ends_at, s.version, e.time_zone
		FROM   sessions s
		JOIN   events   e ON e.id = s.event_id
		WHERE  s.id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return row, err
}

func (s *Store) eventID(ctx context.Context, sessionID string) (string, error) {
	var eventID string
	err := s.db.GetContext(ctx, &eventID, `SELECT event_id FROM sessions WHERE id = $1`, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return eventID, err
}

func (s *Store) RoomBelongsToEvent(ctx context.Context, roomID, eventID string) (bool, error) {
	var found string
	err := s.db.GetContext(ctx, &found, `
		SELECT id FROM rooms WHERE id = $1 AND event_id = $2
	`, roomID, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Apply writes next where the stored version still matches, then increments.
// Room overlap is the exclusion constraint, not a SELECT beforehand.
func (s *Store) Apply(ctx context.Context, expectedVersion int, next Session) (Session, Clash, error) {
	var out Session
	err := s.db.GetContext(ctx, &out, `
		UPDATE sessions
		SET    title       = $1,
		       description = $2,
		       room_id     = $3,
		       starts_at   = $4,
		       ends_at     = $5,
		       version     = version + 1
		WHERE  id = $6
		  AND  version = $7
		RETURNING id, event_id, room_id, title, description, starts_at, ends_at, version
	`, next.Title, next.Description, next.RoomID, next.StartsAt, next.EndsAt, next.ID, expectedVersion)

	if err == nil {
		out.TimeZone = next.TimeZone
		return out, Clash{}, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" { // exclusion_violation
		clash, _ := s.clash(ctx, next)
		return Session{}, clash, ErrRoomConflict
	}

	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, Clash{}, ErrStaleVersion
	}
	return Session{}, Clash{}, err
}

func (s *Store) clash(ctx context.Context, next Session) (Clash, error) {
	if next.RoomID == nil {
		return Clash{}, nil
	}
	var c Clash
	err := s.db.GetContext(ctx, &c, `
		SELECT id, title
		FROM   sessions
		WHERE  room_id = $1
		  AND  id <> $2
		  AND  tstzrange(starts_at, ends_at, '[)') && tstzrange($3, $4, '[)')
		LIMIT  1
	`, *next.RoomID, next.ID, next.StartsAt, next.EndsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Clash{}, nil
	}
	return c, err
}

func (s *Store) ListByEvent(ctx context.Context, eventID string) ([]Session, error) {
	var rows []Session
	err := s.db.SelectContext(ctx, &rows, `
		SELECT s.id, s.event_id, s.room_id, s.title, s.description,
		       s.starts_at, s.ends_at, s.version, e.time_zone
		FROM   sessions s
		JOIN   events   e ON e.id = s.event_id
		WHERE  s.event_id = $1
		ORDER  BY s.starts_at, s.id
	`, eventID)
	return rows, err
}

func (s *Store) Insert(ctx context.Context, in Session) (Session, Clash, error) {
	var out Session
	err := s.db.GetContext(ctx, &out, `
		INSERT INTO sessions (event_id, room_id, title, description, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, event_id, room_id, title, description, starts_at, ends_at, version
	`, in.EventID, in.RoomID, in.Title, in.Description, in.StartsAt, in.EndsAt)
	if err == nil {
		out.TimeZone = in.TimeZone
		return out, Clash{}, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" { // exclusion_violation
		if in.ID == "" {
			in.ID = "00000000-0000-0000-0000-000000000000"
		}
		clash, _ := s.clash(ctx, in)
		return Session{}, clash, ErrRoomConflict
	}
	return Session{}, Clash{}, err
}
