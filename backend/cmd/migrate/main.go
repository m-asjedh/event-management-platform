// Command migrate applies the embedded SQL migrations.
//
// The SQL is compiled into this binary, so the container that runs it needs no goose
// CLI, no source tree and no Go toolchain. `make up` can call it on a clean machine
// where only Docker and make exist.
//
// Usage: migrate [up|down|status]
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/m-asjedh/event-management-platform/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Compose already waits for the healthcheck, so this should succeed first time. It
	// is here for the case where the container is started without that guarantee.
	if err := waitForDB(ctx, db); err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("creating goose provider: %w", err)
	}

	switch command {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("applying migrations: %w", err)
		}
		if len(results) == 0 {
			fmt.Println("already up to date")
			return nil
		}
		for _, r := range results {
			fmt.Printf("applied %d %s in %s\n", r.Source.Version, r.Source.Path, r.Duration)
		}
		return nil

	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return fmt.Errorf("rolling back: %w", err)
		}
		fmt.Printf("rolled back %d %s\n", result.Source.Version, result.Source.Path)
		return nil

	case "status":
		status, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("reading status: %w", err)
		}
		for _, s := range status {
			applied := "pending"
			if !s.AppliedAt.IsZero() {
				applied = s.AppliedAt.Format(time.RFC3339)
			}
			fmt.Printf("%-8d %-32s %s\n", s.Source.Version, s.Source.Path, applied)
		}
		return nil

	default:
		return fmt.Errorf("unknown command %q, want up, down or status", command)
	}
}

func waitForDB(ctx context.Context, db *sql.DB) error {
	var lastErr error
	for attempt := range 30 {
		if err := db.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
	return fmt.Errorf("database unreachable: %w", lastErr)
}
