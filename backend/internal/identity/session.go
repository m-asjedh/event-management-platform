package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNoSession = errors.New("no session")

type User struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// UserForToken loads the user for a Better Auth session token.
// Expired rows count as no session.
func (s *Store) UserForToken(ctx context.Context, token string) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, `
		SELECT u.id, u.name, u.email
		FROM   auth.sessions s
		JOIN   auth.users    u ON u.id = s.user_id
		WHERE  s.token = $1
		  AND  s.expires_at > now()
	`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNoSession
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}
