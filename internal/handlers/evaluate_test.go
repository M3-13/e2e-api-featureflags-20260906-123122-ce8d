package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func TestEvaluateFlagDeterministic(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 50}); err != nil {
		t.Fatalf("create: %v", err)
	}

	handler := EvaluateFlag(s)

	var first bool
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user=test", nil)
		req.SetPathValue("key", "f")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var body map[string]bool
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if i == 0 {
			first = body["result"]
			continue
		}
		if body["result"] != first {
			t.Fatalf("non-deterministic result: got %v then %v", first, body["result"])
		}
	}
}

func TestEvaluateFlagRolloutFull(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 100}); err != nil {
		t.Fatalf("create: %v", err)
	}

	handler := EvaluateFlag(s)

	for _, user := range []string{"a", "b", "c"} {
		req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user="+user, nil)
		req.SetPathValue("key", "f")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var body map[string]bool
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !body["result"] {
			t.Fatalf("expected result true for user %q, got false", user)
		}
	}
}

func TestEvaluateFlagRolloutZero(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 0}); err != nil {
		t.Fatalf("create: %v", err)
	}

	handler := EvaluateFlag(s)

	for _, user := range []string{"a", "b", "c"} {
		req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user="+user, nil)
		req.SetPathValue("key", "f")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var body map[string]bool
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["result"] {
			t.Fatalf("expected result false for user %q, got true", user)
		}
	}
}

func TestEvaluateFlagInvalidKey(t *testing.T) {
	s := store.NewStore()
	handler := EvaluateFlag(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/bad!key/evaluate?user=test", nil)
	req.SetPathValue("key", "bad!key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected JSON error object, got %v", body)
	}
}

func TestEvaluateFlagUnknownKey(t *testing.T) {
	s := store.NewStore()
	handler := EvaluateFlag(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/nope/evaluate?user=test", nil)
	req.SetPathValue("key", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestEvaluateFlagMissingUser(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 50}); err != nil {
		t.Fatalf("create: %v", err)
	}
	handler := EvaluateFlag(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate", nil)
	req.SetPathValue("key", "f")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEvaluateFlagEmptyUser(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 50}); err != nil {
		t.Fatalf("create: %v", err)
	}
	handler := EvaluateFlag(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user=", nil)
	req.SetPathValue("key", "f")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEvaluateFlagDoesNotMutateStore(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: true, RolloutPercent: 50}); err != nil {
		t.Fatalf("create: %v", err)
	}
	handler := EvaluateFlag(s)

	before, _ := s.Get("f")

	for _, user := range []string{"a", "b", "c"} {
		req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user="+user, nil)
		req.SetPathValue("key", "f")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}

	after, ok := s.Get("f")
	if !ok {
		t.Fatalf("flag disappeared after evaluate")
	}
	if after != before {
		t.Fatalf("store mutated by evaluate: before=%+v after=%+v", before, after)
	}
}

func TestEvaluateFlagDisabledReturnsFalse(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "f", Enabled: false, RolloutPercent: 100}); err != nil {
		t.Fatalf("create: %v", err)
	}
	handler := EvaluateFlag(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/f/evaluate?user=test", nil)
	req.SetPathValue("key", "f")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["result"] {
		t.Fatalf("expected result false for disabled flag, got true")
	}
}
