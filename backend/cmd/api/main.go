package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/m-asjedh/event-management-platform/backend/internal/authz"
	"github.com/m-asjedh/event-management-platform/backend/internal/events"
	"github.com/m-asjedh/event-management-platform/backend/internal/identity"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	secret := os.Getenv("BETTER_AUTH_SECRET")
	if secret == "" {
		return errors.New("BETTER_AUTH_SECRET is not set")
	}

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	users := identity.NewStore(db)
	grants := authz.NewStore(db)
	eventStore := events.NewStore(db)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /me", users.Require(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := identity.UserFrom(r)
		if !ok {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "missing user")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		})
	})))
	mux.Handle("GET /events/{id}", users.Require(secret)(events.Show(eventStore, grants)))

	addr := ":" + getenv("API_PORT", "8080")
	log.Printf("api listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func writeError(w http.ResponseWriter, status int, code, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":   code,
		"reason": reason,
	})
}
