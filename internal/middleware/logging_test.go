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

func TestLoggingEscapesPathControlCharacters(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags/foo%0Abar", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	output := buf.String()
	line := strings.TrimSuffix(output, "\n")
	if strings.Count(line, "\n") != 0 {
		t.Fatalf("expected single log line with no newline in the path, got %q", output)
	}
	if !strings.Contains(line, "%0A") {
		t.Fatalf("expected escaped path to preserve %q, got %q", "%0A", line)
	}
	if strings.Contains(line, "/flags/foo\nbar") {
		t.Fatalf("expected path not to contain a decoded newline, got %q", line)
	}
}
