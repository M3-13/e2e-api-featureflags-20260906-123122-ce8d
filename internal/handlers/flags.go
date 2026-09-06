package handlers

import (
	"net/http"

	"featureflags/internal/store"
)

// CreateFlag is a skeleton stub. The full implementation is delivered by the
// ticket "Flag-Verwaltung mit POST, GET, PUT und DELETE implementieren".
func CreateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}

// ListFlags is a skeleton stub. The full implementation is delivered by the
// ticket "Flag-Verwaltung mit POST, GET, PUT und DELETE implementieren".
func ListFlags(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}

// GetFlag is a skeleton stub. The full implementation is delivered by the
// ticket "Flag-Verwaltung mit POST, GET, PUT und DELETE implementieren".
func GetFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}

// UpdateFlag is a skeleton stub. The full implementation is delivered by the
// ticket "Flag-Verwaltung mit POST, GET, PUT und DELETE implementieren".
func UpdateFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}

// DeleteFlag is a skeleton stub. The full implementation is delivered by the
// ticket "Flag-Verwaltung mit POST, GET, PUT und DELETE implementieren".
func DeleteFlag(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not implemented")
	}
}
