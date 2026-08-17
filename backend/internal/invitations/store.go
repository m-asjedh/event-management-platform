package invitations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Invitation struct {
	ID        string  `db:"id"         json:"id"`
	EventID   string  `db:"event_id"   json:"eventId"`
	Email     string  `db:"email"      json:"email,omitempty"`
	Role      string  `db:"role"       json:"role"`
	Status    string  `db:"status"     json:"status"`
	InvitedBy *string `db:"invited_by" json:"invitedBy"`
	UserID    *string `db:"user_id"    json:"userId"`
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListByEvent(ctx context.Context, eventID, after string, limit int) ([]Invitation, error) {
	var afterArg any
	if after != "" {
		afterArg = after
	}
	var rows []Invitation
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, event_id, email, role, status, invited_by, user_id
		FROM   invitations
		WHERE  event_id = $1
		  AND  ($2::uuid IS NULL OR id > $2::uuid)
		ORDER  BY event_id, id
		LIMIT  $3
	`, eventID, afterArg, limit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return rows, nil
}
