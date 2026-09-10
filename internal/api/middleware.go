package api

import (
	"net/http"

	"github.com/relentlessworks/timestampkit/internal/auth"
)

// requireAuth wraps a handler with bearer token authentication.
func requireAuth(a *auth.Auth, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.ExtractBearer(r)
		if token == "" {
			writeError(w, r, http.StatusUnauthorized, "missing auth token", "call POST /auth/request with email to get an OTP, then POST /auth/verify to get a bearer token")
			return
		}
		email, ok := a.ValidateToken(token)
		if !ok {
			writeError(w, r, http.StatusUnauthorized, "invalid or expired token", "call POST /auth/request with email to get a new OTP, then POST /auth/verify to get a bearer token")
			return
		}
		r.Header.Set("X-User-Email", email)
		next(w, r)
	}
}
