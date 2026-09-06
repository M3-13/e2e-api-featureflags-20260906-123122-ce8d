package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func doRequest(t *testing.T, h http.HandlerFunc, method, target string, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if key != "" {
		req.SetPathValue("key", key)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func decodeFlag(t *testing.T, rec *httptest.ResponseRecorder) store.Flag {
	t.Helper()
	var f store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&f); err != nil {
		t.Fatalf("decode flag: %v", err)
	}
	return f
}

func TestCreateFlagSuccess(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"feature.x","enabled":true,"description":"hi","rollout_percent":42}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if f.Key != "feature.x" || !f.Enabled || f.Description != "hi" || f.RolloutPercent != 42 {
		t.Fatalf("unexpected flag: %+v", f)
	}

	got, ok := s.Get("feature.x")
	if !ok {
		t.Fatalf("flag not stored")
	}
	if got != f {
		t.Fatalf("stored flag mismatch: %+v vs %+v", got, f)
	}
}

func TestCreateFlagDefaults(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"minimal","enabled":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if f.RolloutPercent != 0 {
		t.Fatalf("expected default rollout 0, got %v", f.RolloutPercent)
	}
	if f.Description != "" {
		t.Fatalf("expected empty description, got %q", f.Description)
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "dup", Enabled: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"dup","enabled":true}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	var e map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("decode error object: %v", err)
	}
	if e["error"] == "" {
		t.Fatalf("expected error object, got %+v", e)
	}
}

func TestCreateFlagMissingKey(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagEmptyKey(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"","enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagWhitespaceKey(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"   ","enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagInvalidKeyChars(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"bad key!","enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagKeyTooLong(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	key := strings.Repeat("a", 129)
	body := `{"key":"` + key + `","enabled":true}`
	rec := doRequest(t, h, http.MethodPost, "/flags", "", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagEnabledNotBool(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"k","enabled":"yes"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagMissingEnabled(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"k"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagRolloutOutOfRange(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	for _, rp := range []string{"-1", "101"} {
		rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"k","enabled":true,"rollout_percent":`+rp+`}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("rollout %s: expected 400, got %d", rp, rec.Code)
		}
	}
}

func TestCreateFlagRolloutNotNumber(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `{"key":"k","enabled":true,"rollout_percent":"50"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagNotJSON(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	rec := doRequest(t, h, http.MethodPost, "/flags", "", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagTooLarge(t *testing.T) {
	s := store.NewStore()
	h := CreateFlag(s)

	body := `{"key":"k","enabled":true,"description":"` + strings.Repeat("x", maxBodyBytes+1) + `"}`
	rec := doRequest(t, h, http.MethodPost, "/flags", "", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestListFlagsEmpty(t *testing.T) {
	s := store.NewStore()
	h := ListFlags(s)

	rec := doRequest(t, h, http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %+v", list)
	}
}

func TestListFlagsMultiple(t *testing.T) {
	s := store.NewStore()
	for _, f := range []store.Flag{{Key: "a", Enabled: true}, {Key: "b", Enabled: false}} {
		if err := s.Create(f); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	h := ListFlags(s)

	rec := doRequest(t, h, http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(list))
	}
}

func TestListFlagsRespectsLimit(t *testing.T) {
	s := store.NewStore()
	for i := 0; i < 5; i++ {
		if err := s.Create(store.Flag{Key: "k" + string(rune('a'+i)), Enabled: true}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	h := ListFlags(s)

	t.Setenv("FLAGS_LIST_LIMIT", "2")

	rec := doRequest(t, h, http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 flags (limited), got %d", len(list))
	}
}

func TestListFlagsDefaultLimit(t *testing.T) {
	s := store.NewStore()
	for i := 0; i < 3; i++ {
		if err := s.Create(store.Flag{Key: "k" + string(rune('a'+i)), Enabled: true}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	h := ListFlags(s)

	t.Setenv("FLAGS_LIST_LIMIT", "")

	rec := doRequest(t, h, http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []store.Flag
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 flags (no limit), got %d", len(list))
	}
}

func TestGetFlagFound(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true, RolloutPercent: 10}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := GetFlag(s)

	rec := doRequest(t, h, http.MethodGet, "/flags/k", "k", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.Key != "k" || !f.Enabled || f.RolloutPercent != 10 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	s := store.NewStore()
	h := GetFlag(s)

	rec := doRequest(t, h, http.MethodGet, "/flags/missing", "missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetFlagInvalidKey(t *testing.T) {
	s := store.NewStore()
	h := GetFlag(s)

	rec := doRequest(t, h, http.MethodGet, "/flags/bad%20key", "bad key", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagSuccess(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: false, Description: "orig", RolloutPercent: 0}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", `{"enabled":true,"rollout_percent":75}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if !f.Enabled || f.RolloutPercent != 75 || f.Description != "orig" {
		t.Fatalf("unexpected merge result: %+v", f)
	}

	got, ok := s.Get("k")
	if !ok || got != f {
		t.Fatalf("store not updated: %+v (ok=%v)", got, ok)
	}
}

func TestUpdateFlagDescriptionOnly(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true, Description: "orig", RolloutPercent: 50}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", `{"description":"new"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if f.Description != "new" || !f.Enabled || f.RolloutPercent != 50 {
		t.Fatalf("unexpected merge result: %+v", f)
	}
}

// fakeFlagStore implements flagStore so the !ok branch of Store.Update can be
// reached deterministically: Get reports the flag exists while Update reports
// it no longer does, which is exactly the concurrent-delete race UpdateFlag
// must handle.
type fakeFlagStore struct {
	existing store.Flag
	getOK    bool
	updateOK bool
}

func (f *fakeFlagStore) Get(key string) (store.Flag, bool) {
	return f.existing, f.getOK
}

func (f *fakeFlagStore) Update(key string, flag store.Flag) (store.Flag, bool) {
	return flag, f.updateOK
}

func TestUpdateFlagDeletedBetweenGetAndUpdate(t *testing.T) {
	fs := &fakeFlagStore{
		existing: store.Flag{Key: "k", Enabled: false},
		getOK:    true,
		updateOK: false,
	}
	h := updateFlag(fs)

	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", `{"enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	var e map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("decode error object: %v", err)
	}
	if e["error"] != "flag not found" {
		t.Fatalf("expected error 'flag not found', got %+v", e)
	}
}

func TestUpdateFlagNotFound(t *testing.T) {
	s := store.NewStore()
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/missing", "missing", `{"enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateFlagNoFields(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagInvalidKey(t *testing.T) {
	s := store.NewStore()
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/bad%20key", "bad key", `{"enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagRolloutOutOfRange(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := UpdateFlag(s)

	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", `{"rollout_percent":200}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagTooLarge(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := UpdateFlag(s)

	body := `{"description":"` + strings.Repeat("x", maxBodyBytes+1) + `"}`
	rec := doRequest(t, h, http.MethodPut, "/flags/k", "k", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestDeleteFlagSuccess(t *testing.T) {
	s := store.NewStore()
	if err := s.Create(store.Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := DeleteFlag(s)

	rec := doRequest(t, h, http.MethodDelete, "/flags/k", "k", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	if _, ok := s.Get("k"); ok {
		t.Fatalf("flag still present after delete")
	}

	getRec := doRequest(t, GetFlag(s), http.MethodGet, "/flags/k", "k", "")
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	s := store.NewStore()
	h := DeleteFlag(s)

	rec := doRequest(t, h, http.MethodDelete, "/flags/missing", "missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteFlagInvalidKey(t *testing.T) {
	s := store.NewStore()
	h := DeleteFlag(s)

	rec := doRequest(t, h, http.MethodDelete, "/flags/bad%20key", "bad key", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestErrorResponsesAreJSONObjects(t *testing.T) {
	s := store.NewStore()

	cases := []struct {
		name   string
		h      http.HandlerFunc
		method string
		target string
		key    string
		body   string
		code   int
	}{
		{"missing key", CreateFlag(s), http.MethodPost, "/flags", "", `{"enabled":true}`, 400},
		{"get not found", GetFlag(s), http.MethodGet, "/flags/nope", "nope", "", 404},
		{"put not found", UpdateFlag(s), http.MethodPut, "/flags/nope", "nope", `{"enabled":true}`, 404},
		{"delete not found", DeleteFlag(s), http.MethodDelete, "/flags/nope", "nope", "", 404},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := doRequest(t, c.h, c.method, c.target, c.key, c.body)
			if rec.Code != c.code {
				t.Fatalf("expected %d, got %d", c.code, rec.Code)
			}
			if rec.Code >= 400 {
				var e map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
					t.Fatalf("error body not a JSON object: %v (body=%q)", err, rec.Body.String())
				}
				if _, ok := e["error"]; !ok {
					t.Fatalf("error object missing \"error\" field: %+v", e)
				}
			}
		})
	}
}
