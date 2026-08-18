package seed

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/scrypt"
)

func Run(ctx context.Context, databaseURL string, size Size) error {
	start := time.Now()

	if err := requireComposePostgres(databaseURL); err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	hash, err := passwordHash(DemoPassword)
	if err != nil {
		return fmt.Errorf("password hash: %w", err)
	}

	data := Generate(hash, size)

	// Domain rows and auth login rows only. roles, permissions, time_zones, and
	// goose_db_version stay. make seed only launches this binary inside Compose,
	// with DATABASE_URL hardcoded to the `postgres` service; requireComposePostgres
	// refuses any other host so a stray go-run cannot truncate a different database.
	if _, err := conn.Exec(ctx, `
		TRUNCATE TABLE
			invitations,
			session_speakers,
			sessions,
			event_members,
			rooms,
			events,
			auth.verifications,
			auth.sessions,
			auth.accounts,
			auth.users
		RESTART IDENTITY CASCADE
	`); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}

	if err := copyAll(ctx, conn, data); err != nil {
		return err
	}

	fmt.Printf("seed %s finished in %s\n", size.Label, time.Since(start).Round(time.Millisecond))
	fmt.Printf("%d events, %d users, %d invitations\n",
		len(data.Events), len(data.Users), len(data.Invitations))
	fmt.Printf("demo logins, password %q:\n  %s\n  %s\n",
		DemoPassword, data.Users[0].Email, data.Users[1].Email)
	return nil
}

// passwordHash is a real Better Auth scrypt hash (N=16384, r=16, p=1, dkLen=64).
// All seed accounts share this one hash so seeding stays fast; not a production pattern.
func passwordHash(password string) (string, error) {
	const salt = "0123456789abcdef0123456789abcdef"
	key, err := scrypt.Key([]byte(password), []byte(salt), 16384, 16, 1, 64)
	if err != nil {
		return "", err
	}
	return salt + ":" + hex.EncodeToString(key), nil
}

func requireComposePostgres(databaseURL string) error {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	if u.Hostname() != "postgres" {
		return fmt.Errorf("seed refuses to run: DATABASE_URL host must be the compose service %q, got %q", "postgres", u.Hostname())
	}
	return nil
}

func Main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "seed: DATABASE_URL is not set")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	size := ParseSize(os.Getenv("SEED_SIZE"))
	if err := Run(ctx, dsn, size); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}
