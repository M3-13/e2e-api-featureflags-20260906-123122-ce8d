package middleware

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRecoverReturns500AndProcessContinues(t *testing.T) {
	// Recover logs the panic via the standard logger; give it a valid writer
	// so the logging itself cannot panic regardless of prior test pollution.
	log.SetOutput(os.Stderr)

	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	// The middleware must not break subsequent requests: serving again proves
	// the process (and this handler) is still alive after the panic.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("expected second request to also return 500, got %d", rec2.Code)
	}
}

func TestRecoverPassesThroughNormalResponse(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/flags", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}
