package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const cookieName = "better-auth.session_token"

// SignIn uses Better Auth's public HTTP API. The agent never writes a session
// row itself; it only forwards the cookie the auth service sets.
func SignIn(authURL, email, password string) (*http.Cookie, error) {
	url := strings.TrimRight(authURL, "/") + "/api/auth/sign-in/email"
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Better Auth CSRF accepts JSON from a trusted origin.
	req.Header.Set("Origin", "http://localhost:5173")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sign-in: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sign-in %d: %s", resp.StatusCode, raw)
	}
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			return c, nil
		}
	}
	return nil, fmt.Errorf("sign-in: no %s cookie", cookieName)
}
