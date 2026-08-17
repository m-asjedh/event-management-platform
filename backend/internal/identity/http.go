package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type ctxKey struct{}

func contextWithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

func UserFrom(r *http.Request) (User, bool) {
	u, ok := r.Context().Value(ctxKey{}).(User)
	return u, ok
}

// Require is the door. No valid session → 401. Roles are not checked here.
func (s *Store) Require(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				writeUnauthenticated(w)
				return
			}

			token, ok := TokenFromCookie(cookie.Value, secret)
			if !ok {
				writeUnauthenticated(w)
				return
			}

			user, err := s.UserForToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrNoSession) {
					writeUnauthenticated(w)
					return
				}
				writeError(w, http.StatusInternalServerError, "INTERNAL", "session lookup failed")
				return
			}

			ctx := r.Context()
			ctx = contextWithUser(ctx, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthenticated(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "not signed in")
}

func writeError(w http.ResponseWriter, status int, code, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":   code,
		"reason": reason,
	})
}
