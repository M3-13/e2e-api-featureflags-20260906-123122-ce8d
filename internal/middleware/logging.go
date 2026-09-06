package middleware

import (
	"log"
	"net/http"
)

// responseWriter wraps http.ResponseWriter to capture the status code written
// by the handler.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging wraps next and logs one line per request with the HTTP method, the
// path (without query string) and the resulting status code.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %q %d", r.Method, r.URL.EscapedPath(), rw.status)
	})
}
