package handlers

import (
	"hash/fnv"
	"net/http"

	"featureflags/internal/store"
)

// EvaluateFlag answers GET /flags/{key}/evaluate?user={id} with a
// deterministic boolean result for a fixed key+user pair. It never mutates the
// store and never persists the user id or the evaluation result.
func EvaluateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		user := r.URL.Query().Get("user")
		if user == "" {
			WriteError(w, http.StatusBadRequest, "user is required")
			return
		}

		flag, ok := s.Get(key)
		if !ok {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}

		if !flag.Enabled {
			WriteJSON(w, http.StatusOK, map[string]bool{"result": false})
			return
		}

		h := fnv.New32a()
		h.Write([]byte(key + ":" + user))
		enabled := int(h.Sum32()%100) < int(flag.RolloutPercent)

		WriteJSON(w, http.StatusOK, map[string]bool{"result": enabled})
	}
}
