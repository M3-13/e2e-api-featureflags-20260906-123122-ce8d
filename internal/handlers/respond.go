package handlers

import (
	"encoding/json"
	"net/http"
)

// WriteJSON serializes v as JSON and writes it with the given status code,
// setting Content-Type to application/json; charset=utf-8.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Nothing else can be written at this point; the status is already sent.
		return
	}
}

// WriteError writes the canonical JSON error object {"error": msg} with the
// given status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
