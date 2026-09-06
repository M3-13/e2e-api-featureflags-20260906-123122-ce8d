package handlers

import "net/http"

// Health answers GET /healthz with 200 and {"status":"ok"}.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
