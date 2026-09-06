package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"featureflags/internal/handlers"
)

// Auth wraps next and requires a valid Bearer API key in the Authorization
// header. The expected key is read from the ADMIN_API_KEY environment
// variable. Requests without a key, with a malformed header, or with a key
// that does not match receive 401 {"error":"unauthorized"}.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("ADMIN_API_KEY")
		if apiKey == "" {
			handlers.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			handlers.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token := header[len(prefix):]
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			handlers.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
