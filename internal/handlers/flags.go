package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"featureflags/internal/store"
)

// maxBodyBytes is the maximum accepted size of a POST/PUT request body (1 MiB).
const maxBodyBytes = 1 << 20

// keyPattern restricts flag keys to a safe character set.
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// validKey reports whether key is an acceptable flag key: non-empty, not only
// whitespace, at most 128 characters, and matching [A-Za-z0-9._-].
func validKey(key string) bool {
	if key == "" || strings.TrimSpace(key) == "" {
		return false
	}
	if len(key) > 128 {
		return false
	}
	return keyPattern.MatchString(key)
}

// createFlagRequest is the JSON body accepted by POST /flags. Pointer fields
// distinguish a missing field from its zero value.
type createFlagRequest struct {
	Key            *string  `json:"key"`
	Enabled        *bool    `json:"enabled"`
	Description    *string  `json:"description"`
	RolloutPercent *float64 `json:"rollout_percent"`
}

// updateFlagRequest is the JSON body accepted by PUT /flags/{key}. Pointer
// fields distinguish a missing field from its zero value.
type updateFlagRequest struct {
	Enabled        *bool    `json:"enabled"`
	Description    *string  `json:"description"`
	RolloutPercent *float64 `json:"rollout_percent"`
}

// decodeJSONBody reads and decodes the request body into v, enforcing the
// 1 MiB limit. It returns true when decoding succeeded, and otherwise writes
// a 413 (body too large) or 400 (invalid JSON) response.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// CreateFlag handles POST /flags.
func CreateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createFlagRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		if req.Key == nil || !validKey(*req.Key) {
			WriteError(w, http.StatusBadRequest, "invalid key")
			return
		}
		if req.Enabled == nil {
			WriteError(w, http.StatusBadRequest, "enabled is required and must be a boolean")
			return
		}

		rollout := 0.0
		if req.RolloutPercent != nil {
			rollout = *req.RolloutPercent
			if rollout < 0 || rollout > 100 {
				WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
				return
			}
		}

		description := ""
		if req.Description != nil {
			description = *req.Description
		}

		flag := store.Flag{
			Key:            *req.Key,
			Enabled:        *req.Enabled,
			Description:    description,
			RolloutPercent: rollout,
		}
		if err := s.Create(flag); err != nil {
			if errors.Is(err, store.ErrDuplicate) {
				WriteError(w, http.StatusConflict, "flag already exists")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		WriteJSON(w, http.StatusCreated, flag)
	}
}

// ListFlags handles GET /flags.
func ListFlags(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, s.List())
	}
}

// GetFlag handles GET /flags/{key}.
func GetFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		flag, ok := s.Get(key)
		if !ok {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		WriteJSON(w, http.StatusOK, flag)
	}
}

// UpdateFlag handles PUT /flags/{key}.
func UpdateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		var req updateFlagRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		if req.Enabled == nil && req.Description == nil && req.RolloutPercent == nil {
			WriteError(w, http.StatusBadRequest, "at least one of enabled, description, rollout_percent is required")
			return
		}
		if req.RolloutPercent != nil && (*req.RolloutPercent < 0 || *req.RolloutPercent > 100) {
			WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
			return
		}

		current, ok := s.Get(key)
		if !ok {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}

		if req.Enabled != nil {
			current.Enabled = *req.Enabled
		}
		if req.Description != nil {
			current.Description = *req.Description
		}
		if req.RolloutPercent != nil {
			current.RolloutPercent = *req.RolloutPercent
		}

		updated, _ := s.Update(key, current)
		WriteJSON(w, http.StatusOK, updated)
	}
}

// DeleteFlag handles DELETE /flags/{key}.
func DeleteFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if _, ok := s.Get(key); !ok {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		s.Delete(key)
		w.WriteHeader(http.StatusNoContent)
	}
}
