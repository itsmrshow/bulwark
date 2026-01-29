package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireWriteReadOnly(t *testing.T) {
	srv := &Server{cfg: Config{ReadOnly: true}}
	h := srv.requireWrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/apply", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestRequireWriteMissingToken(t *testing.T) {
	srv := &Server{cfg: Config{ReadOnly: false, WebToken: "secret"}}
	h := srv.requireWrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/apply", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestRequireWriteValidToken(t *testing.T) {
	srv := &Server{cfg: Config{ReadOnly: false, WebToken: "secret"}}
	h := srv.requireWrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/apply", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}
