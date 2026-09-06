package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func newMux(s *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /flags", CreateFlag(s))
	mux.HandleFunc("GET /flags", ListFlags(s))
	mux.HandleFunc("GET /flags/{key}", GetFlag(s))
	mux.HandleFunc("PUT /flags/{key}", UpdateFlag(s))
	mux.HandleFunc("DELETE /flags/{key}", DeleteFlag(s))
	return mux
}

func doRequest(t *testing.T, mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeFlag(t *testing.T, rec *httptest.ResponseRecorder) store.Flag {
	t.Helper()
	var f store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&f); err != nil {
		t.Fatalf("expected valid flag JSON, got error %v", err)
	}
	return f
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("expected valid error JSON, got error %v", err)
	}
	if e["error"] == "" {
		t.Fatalf("expected non-empty error field, got %v", e)
	}
	return e["error"]
}

func TestCreateFlag(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodPost, "/flags", `{"key":"my.flag_1","enabled":true,"description":"desc","rollout_percent":50}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.Key != "my.flag_1" || !f.Enabled || f.Description != "desc" || f.RolloutPercent != 50 {
		t.Fatalf("unexpected flag returned: %+v", f)
	}

	rec = doRequest(t, mux, http.MethodPost, "/flags", `{"key":"plain","enabled":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	f = decodeFlag(t, rec)
	if f.Key != "plain" || f.Enabled || f.Description != "" || f.RolloutPercent != 0 {
		t.Fatalf("unexpected defaults: %+v", f)
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	doRequest(t, mux, http.MethodPost, "/flags", `{"key":"dup","enabled":true}`)
	rec := doRequest(t, mux, http.MethodPost, "/flags", `{"key":"dup","enabled":false}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestCreateFlagValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no JSON", "not json"},
		{"missing key", `{"enabled":true}`},
		{"empty key", `{"key":"","enabled":true}`},
		{"whitespace key", `{"key":"   ","enabled":true}`},
		{"invalid key chars", `{"key":"bad key!","enabled":true}`},
		{"key too long", `{"key":"` + strings.Repeat("a", 129) + `","enabled":true}`},
		{"enabled missing", `{"key":"k"}`},
		{"enabled not bool", `{"key":"k","enabled":"yes"}`},
		{"rollout below", `{"key":"k","enabled":true,"rollout_percent":-1}`},
		{"rollout above", `{"key":"k","enabled":true,"rollout_percent":101}`},
		{"rollout wrong type", `{"key":"k","enabled":true,"rollout_percent":"50"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := store.NewStore()
			mux := newMux(s)
			rec := doRequest(t, mux, http.MethodPost, "/flags", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
			decodeError(t, rec)
		})
	}
}

func TestCreateFlagBodyTooLarge(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)
	big := `{"key":"k","enabled":true,"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rec := doRequest(t, mux, http.MethodPost, "/flags", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestListFlags(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "a", Enabled: true}); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if err := s.Create(store.Flag{Key: "b", Enabled: false}); err != nil {
		t.Fatalf("create b: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodGet, "/flags", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var flags []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&flags); err != nil {
		t.Fatalf("expected valid array JSON, got error %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
}

func TestListFlagsEmpty(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodGet, "/flags", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var flags []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&flags); err != nil {
		t.Fatalf("expected valid array JSON, got error %v", err)
	}
	if flags == nil {
		t.Fatalf("expected empty array, got null")
	}
	if len(flags) != 0 {
		t.Fatalf("expected 0 flags, got %d", len(flags))
	}
}

func TestGetFlag(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "known", Enabled: true, RolloutPercent: 30}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodGet, "/flags/known", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.Key != "known" || f.RolloutPercent != 30 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodGet, "/flags/unknown", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestUpdateFlag(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true, Description: "old", RolloutPercent: 10}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodPut, "/flags/k", `{"enabled":false,"rollout_percent":80}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.Key != "k" || f.Enabled || f.RolloutPercent != 80 || f.Description != "old" {
		t.Fatalf("unexpected merged flag: %+v", f)
	}
}

func TestUpdateFlagNotFound(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodPut, "/flags/missing", `{"enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestUpdateFlagNoFields(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodPut, "/flags/k", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestUpdateFlagValidation(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodPut, "/flags/k", `{"rollout_percent":150}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestUpdateFlagBodyTooLarge(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	big := `{"enabled":true,"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rec := doRequest(t, mux, http.MethodPut, "/flags/k", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	decodeError(t, rec)
}

func TestDeleteFlag(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodDelete, "/flags/k", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	if _, ok := s.Get("k"); ok {
		t.Fatalf("expected flag to be deleted")
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	s := store.NewStore()
	mux := newMux(s)

	rec := doRequest(t, mux, http.MethodDelete, "/flags/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	decodeError(t, rec)
}
