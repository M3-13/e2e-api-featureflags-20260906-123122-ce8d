package middleware

import (
	"log"
	"net/http"

	"featureflags/internal/handlers"
)

// Recover wraps next and catches any panic it raises, logs it, and answers the
// request with 500 {"error":"internal server error"} instead of terminating the
// process.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				handlers.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
