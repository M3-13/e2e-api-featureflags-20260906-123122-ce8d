package handlers

import (
	"net/http"

	"featureflags/internal/store"
)

// EvaluateFlag is a skeleton stub. The full implementation is delivered by the
// ticket "Deterministischen Evaluate-Endpunkt implementieren".
func EvaluateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}
