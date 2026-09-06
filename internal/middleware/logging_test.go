package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingLogsMethodPathStatusWithoutQuery(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags/foo/evaluate?user=alice&secret=xyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "GET") {
		t.Fatalf("expected log to contain method, got %q", output)
	}
	if !strings.Contains(output, "/flags/foo/evaluate") {
		t.Fatalf("expected log to contain path, got %q", output)
	}
	if !strings.Contains(output, "418") {
		t.Fatalf("expected log to contain status code, got %q", output)
	}
	if strings.Contains(output, "user=alice") || strings.Contains(output, "secret=xyz") {
		t.Fatalf("expected log to exclude query string, got %q", output)
	}
}

func TestLoggingPreservesResponse(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/flags", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}
