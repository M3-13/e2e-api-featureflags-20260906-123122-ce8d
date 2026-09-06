package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"featureflags/internal/store"
)

const maxBodyBytes = 1 << 20 // 1 MiB

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

var errBodyTooLarge = errors.New("request body too large")

// validKey reports whether key is a safe flag key: non-empty, at most 128
// characters, and composed only of [A-Za-z0-9._-].
func validKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	return keyPattern.MatchString(key)
}

// decodeJSONBody reads at most maxBodyBytes from r.Body into dst. It returns
// errBodyTooLarge when the body exceeds the limit, or the underlying decoding
// error otherwise.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errBodyTooLarge
		}
		return err
	}
	return nil
}

type createFlagRequest struct {
	Key            *string  `json:"key"`
	Enabled        *bool    `json:"enabled"`
	Description    *string  `json:"description"`
	RolloutPercent *float64 `json:"rollout_percent"`
}

type updateFlagRequest struct {
	Enabled        *bool    `json:"enabled"`
	Description    *string  `json:"description"`
	RolloutPercent *float64 `json:"rollout_percent"`
}

// CreateFlag handles POST /flags.
func CreateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createFlagRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			if errors.Is(err, errBodyTooLarge) {
				WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.Key == nil || *req.Key == "" {
			WriteError(w, http.StatusBadRequest, "key is required")
			return
		}
		if !validKey(*req.Key) {
			WriteError(w, http.StatusBadRequest, "invalid key")
			return
		}
		if req.Enabled == nil {
			WriteError(w, http.StatusBadRequest, "enabled is required")
			return
		}

		flag := store.Flag{
			Key:     *req.Key,
			Enabled: *req.Enabled,
		}
		if req.Description != nil {
			flag.Description = *req.Description
		}
		if req.RolloutPercent != nil {
			if *req.RolloutPercent < 0 || *req.RolloutPercent > 100 {
				WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
				return
			}
			flag.RolloutPercent = *req.RolloutPercent
		}

		if err := s.Create(flag); err != nil {
			if errors.Is(err, store.ErrDuplicate) {
				WriteError(w, http.StatusConflict, "flag already exists")
				return
			}
			WriteError(w, http.StatusInternalServerError, "failed to create flag")
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
		if !validKey(key) {
			WriteError(w, http.StatusBadRequest, "invalid key")
			return
		}

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
		if !validKey(key) {
			WriteError(w, http.StatusBadRequest, "invalid key")
			return
		}

		var req updateFlagRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			if errors.Is(err, errBodyTooLarge) {
				WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		existing, ok := s.Get(key)
		if !ok {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}

		if req.Enabled == nil && req.Description == nil && req.RolloutPercent == nil {
			WriteError(w, http.StatusBadRequest, "no fields to update")
			return
		}

		if req.Enabled != nil {
			existing.Enabled = *req.Enabled
		}
		if req.Description != nil {
			existing.Description = *req.Description
		}
		if req.RolloutPercent != nil {
			if *req.RolloutPercent < 0 || *req.RolloutPercent > 100 {
				WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
				return
			}
			existing.RolloutPercent = *req.RolloutPercent
		}

		updated, _ := s.Update(key, existing)
		WriteJSON(w, http.StatusOK, updated)
	}
}

// DeleteFlag handles DELETE /flags/{key}.
func DeleteFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if !validKey(key) {
			WriteError(w, http.StatusBadRequest, "invalid key")
			return
		}

		if !s.Delete(key) {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
