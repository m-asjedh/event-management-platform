package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m-asjedh/event-management-platform/backend/internal/agent"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	api := env("API_URL", "http://localhost:8080")
	auth := env("AUTH_URL", "http://localhost:3001")
	email := env("AGENT_EMAIL", "seed.admin@example.com")
	password := env("AGENT_PASSWORD", "correct-horse-battery")
	question := os.Getenv("QUESTION")

	flag.StringVar(&api, "api", api, "API base URL")
	flag.StringVar(&auth, "auth", auth, "Better Auth base URL")
	flag.StringVar(&email, "email", email, "user email")
	flag.StringVar(&password, "password", password, "user password")
	flag.StringVar(&question, "q", question, "question to ask")
	scenarios := flag.Bool("scenarios", false, "run the three scripted scenarios")
	flag.Parse()

	if *scenarios {
		return runScenarios(os.Stdout, api, auth, password)
	}

	cookie, err := agent.SignIn(auth, email, password)
	if err != nil {
		return err
	}
	client := agent.NewClient(api, cookie)

	if question != "" {
		_, err := agent.Run(os.Stdout, client, question)
		return err
	}

	fmt.Fprintf(os.Stdout, "Signed in as %s. Ask about events (GET only). Ctrl-D to quit.\n", email)
	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "\n> ")
		if !in.Scan() {
			break
		}
		q := strings.TrimSpace(in.Text())
		if q == "" {
			continue
		}
		if _, err := agent.Run(os.Stdout, client, q); err != nil {
			return err
		}
	}
	return in.Err()
}

func runScenarios(w io.Writer, api, auth, password string) error {
	type row struct {
		email, question string
	}
	rows := []row{
		{"seed.admin@example.com", "Which events are in America/New_York?"},
		{"seed.admin@example.com", "How many sessions does DST Spring Forward have?"},
		{"seed.attendee@example.com", "Is seed.attendee allowed to see the invitations for Prompt Injection Conference?"},
	}
	for i, r := range rows {
		fmt.Fprintf(w, "=== scenario %d as %s ===\n%s\n\n", i+1, r.email, r.question)
		cookie, err := agent.SignIn(auth, r.email, password)
		if err != nil {
			return err
		}
		if _, err := agent.Run(w, agent.NewClient(api, cookie), r.question); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
