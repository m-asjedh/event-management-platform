package authz

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrNotMember = errors.New("not a member of this event")

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// For loads the caller's grant on one event. No membership means ErrNotMember,
// not an empty grant: an empty grant would look like "attendee with no permissions"
// and hide a missing row.
func (s *Store) For(ctx context.Context, userID, eventID string) (Grant, error) {
	var rows []struct {
		Role       string `db:"role"`
		Permission string `db:"permission"`
	}
	err := s.db.SelectContext(ctx, &rows, `
		SELECT m.role, rp.permission
		FROM   event_members m
		JOIN   role_permissions rp ON rp.role = m.role
		WHERE  m.event_id = $1
		  AND  m.user_id  = $2
	`, eventID, userID)
	if err != nil {
		return Grant{}, err
	}
	if len(rows) == 0 {
		// Distinguish "member with zero permissions" from "no row". The join
		// always returns at least one row if the membership exists, because
		// every role in the seed matrix has permissions.
		var role string
		err := s.db.GetContext(ctx, &role, `
			SELECT role
			FROM   event_members
			WHERE  event_id = $1 AND user_id = $2
		`, eventID, userID)
		if errors.Is(err, sql.ErrNoRows) {
			return Grant{}, ErrNotMember
		}
		if err != nil {
			return Grant{}, err
		}
		return NewGrant(role, nil), nil
	}

	perms := make([]string, 0, len(rows))
	for _, r := range rows {
		perms = append(perms, r.Permission)
	}
	return NewGrant(rows[0].Role, perms), nil
}
