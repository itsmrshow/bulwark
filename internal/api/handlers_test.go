package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer creates a minimal Server suitable for handler unit tests.
// It bypasses NewServer (which requires Docker/SQLite) and wires up only
// what the simple handlers need.
func testServer() *Server {
	return &Server{
		cfg: Config{
			ReadOnly:  true,
			UIEnabled: true,
		},
		runs:      NewRunManager(10, 100, 50, nil),
		planCache: newPlanCache(0),
		sessions:  newSessionStore(),
	}
}

func TestHandleHealth_GET(t *testing.T) {
	s := testServer()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if !resp.ReadOnly {
		t.Error("expected read_only true")
	}
	if !resp.UIEnabled {
		t.Error("expected ui_enabled true")
	}
}

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleRun_NotFound(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/runs/nonexistent", nil)
	w := httptest.NewRecorder()
	s.handleRun(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleRun_Found(t *testing.T) {
	s := testServer()
	run := s.runs.CreateRun("apply")

	req := httptest.NewRequest(http.MethodGet, "/api/runs/"+run.ID, nil)
	w := httptest.NewRecorder()
	s.handleRun(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp Run
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.ID != run.ID {
		t.Errorf("expected ID %s, got %s", run.ID, resp.ID)
	}
}

func TestHandleRun_MethodNotAllowed(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/runs/test", nil)
	w := httptest.NewRecorder()
	s.handleRun(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleRun_MissingID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/runs/", nil)
	w := httptest.NewRecorder()
	s.handleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleHistory_NoStore(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	w := httptest.NewRecorder()
	s.handleHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp historyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestHandleHistory_MethodNotAllowed(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/history", nil)
	w := httptest.NewRecorder()
	s.handleHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		query    string
		key      string
		defValue int
		expected int
	}{
		{"", "page", 1, 1},
		{"page=5", "page", 1, 5},
		{"page=abc", "page", 1, 1},
		{"page=0", "page", 1, 0},
		{"other=5", "page", 1, 1},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
		got := parseIntQuery(req, tt.key, tt.defValue)
		if got != tt.expected {
			t.Errorf("parseIntQuery(%q, %q, %d) = %d, want %d", tt.query, tt.key, tt.defValue, got, tt.expected)
		}
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		header   string
		expected string
	}{
		{"", ""},
		{"Bearer mytoken", "mytoken"},
		{"bearer MYTOKEN", "MYTOKEN"},
		{"Basic auth", ""},
		{"Bearermissing", ""},
		{"Bearer  spaced ", "spaced"},
	}

	for _, tt := range tests {
		got := bearerToken(tt.header)
		if got != tt.expected {
			t.Errorf("bearerToken(%q) = %q, want %q", tt.header, got, tt.expected)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %s", cc)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad request", "details here")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp apiError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.Error != "bad request" {
		t.Errorf("expected error 'bad request', got %s", resp.Error)
	}
	if resp.Details != "details here" {
		t.Errorf("expected details 'details here', got %s", resp.Details)
	}
}
