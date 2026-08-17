package rooms

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var ErrDuplicateName = errors.New("duplicate room name")

type Room struct {
	ID       string `db:"id"       json:"id"`
	EventID  string `db:"event_id" json:"eventId"`
	Name     string `db:"name"     json:"name"`
	Capacity int    `db:"capacity" json:"capacity"`
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListByEvent(ctx context.Context, eventID string) ([]Room, error) {
	var rows []Room
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, event_id, name, capacity
		FROM   rooms
		WHERE  event_id = $1
		ORDER  BY name
	`, eventID)
	return rows, err
}

func (s *Store) Insert(ctx context.Context, in Room) (Room, error) {
	var out Room
	err := s.db.GetContext(ctx, &out, `
		INSERT INTO rooms (event_id, name, capacity)
		VALUES ($1, $2, $3)
		RETURNING id, event_id, name, capacity
	`, in.EventID, in.Name, in.Capacity)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Room{}, ErrDuplicateName
	}
	return out, err
}
