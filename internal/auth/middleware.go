package auth

import (
	"net/http"
	"strings"
)

// SessionValidator loads a principal from a session ID.
type SessionValidator func(sessionID string) (*Principal, error)

// SessionMiddleware attaches an authenticated Principal to the request context when a valid session cookie is present.
func SessionMiddleware(validate SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if validate != nil {
				if id := sessionIDFromRequest(r); id != "" {
					if p, err := validate(id); err == nil && p != nil {
						ctx = WithPrincipal(ctx, p)
					}
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth rejects requests without a valid principal.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFrom(r.Context()); !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sessionIDFromRequest(r *http.Request) string {
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}
