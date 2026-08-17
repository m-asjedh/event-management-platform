package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
)

// CookieName is what Better Auth sets on sign-in.
const CookieName = "better-auth.session_token"

// TokenFromCookie pulls the session token out of Better Auth's signed cookie.
//
// The cookie is not a JWT. It is the raw token plus an HMAC-SHA256, which we
// check with the same secret Better Auth uses. A bad signature is treated as
// "not signed in", not as a different error.
func TokenFromCookie(raw, secret string) (string, bool) {
	if raw == "" || secret == "" {
		return "", false
	}

	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", false
	}

	dot := strings.LastIndex(decoded, ".")
	if dot < 1 {
		return "", false
	}

	token := decoded[:dot]
	sigB64 := decoded[dot+1:]
	if len(sigB64) != 44 || !strings.HasSuffix(sigB64, "=") {
		return "", false
	}

	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return "", false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(token))
	if !hmac.Equal(mac.Sum(nil), sig) {
		return "", false
	}

	return token, true
}
