package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	corsMiddleware(handleHealth)(rec, httptest.NewRequest("GET", "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || !body["ok"] {
		t.Errorf("body = %v, err = %v", body, err)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestCorsPreflight(t *testing.T) {
	called := false
	h := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { called = true })
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("OPTIONS", "/download-album", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if called {
		t.Error("preflight should not reach the handler")
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Error("POST should be allowed")
	}
}

// These cases are all rejected before any request is sent to Deezer.
func TestDownloadHandlersRejectBadRequests(t *testing.T) {
	handlers := map[string]func(http.ResponseWriter, *http.Request, configuration){
		"album":    handleDownloadAlbum,
		"playlist": handleDownloadPlaylist,
	}
	cases := []struct {
		name, body, wantErr string
	}{
		{"invalid json", "{not json", "Invalid request body"},
		{"missing id", `{"license_token":"x"}`, "ID is required"},
		{"missing token", `{"id":"123"}`, "License token is required"},
	}
	for kind, handler := range handlers {
		for _, c := range cases {
			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest("POST", "/", strings.NewReader(c.body)), configuration{})

			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s/%s: status = %d, want 400", kind, c.name, rec.Code)
			}
			var resp DownloadResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("%s/%s: %v", kind, c.name, err)
			}
			if resp.Success || !strings.Contains(resp.Error, c.wantErr) {
				t.Errorf("%s/%s: response = %+v", kind, c.name, resp)
			}
		}
	}
}
