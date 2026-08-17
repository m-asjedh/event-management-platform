package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"
)

func signLikeBetterAuth(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(token))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return url.PathEscape(token + "." + sig)
}

func TestTokenFromCookie(t *testing.T) {
	const secret = "dev-only-not-a-real-secret-000000000000"
	const token = "Eg7bMW53FupzHJ8vCgba0sWSwvoX29Iq"

	t.Run("valid signed cookie", func(t *testing.T) {
		got, ok := TokenFromCookie(signLikeBetterAuth(token, secret), secret)
		if !ok || got != token {
			t.Fatalf("got %q ok=%v", got, ok)
		}
	})

	t.Run("already unescaped value", func(t *testing.T) {
		raw, _ := url.PathUnescape(signLikeBetterAuth(token, secret))
		got, ok := TokenFromCookie(raw, secret)
		if !ok || got != token {
			t.Fatalf("got %q ok=%v", got, ok)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		if _, ok := TokenFromCookie(signLikeBetterAuth(token, secret), "other-secret-other-secret-000000"); ok {
			t.Fatal("expected reject")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, ok := TokenFromCookie("", secret); ok {
			t.Fatal("expected reject")
		}
	})

	t.Run("raw token with no signature", func(t *testing.T) {
		if _, ok := TokenFromCookie(token, secret); ok {
			t.Fatal("expected reject")
		}
	})
}
