package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func TestHTTPProbe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewHTTPProbe(server.URL, 200, config, logger)
	result := probe.Execute(context.Background())

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Type != state.ProbeTypeHTTP {
		t.Errorf("expected type=http, got %s", result.Type)
	}
}

func TestHTTPProbe_WrongStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewHTTPProbe(server.URL, 200, config, logger)
	result := probe.Execute(context.Background())

	if result.Success {
		t.Error("expected failure for wrong status code")
	}
}

func TestHTTPProbe_CustomStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewHTTPProbe(server.URL, 202, config, logger)
	result := probe.Execute(context.Background())

	if !result.Success {
		t.Errorf("expected success for status 202, got: %s", result.Message)
	}
}

func TestHTTPProbe_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logging.Default()
	config := Config{
		Timeout:      100 * time.Millisecond,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewHTTPProbe(server.URL, 200, config, logger)
	result := probe.Execute(context.Background())

	if result.Success {
		t.Error("expected failure for timeout")
	}
}

func TestHTTPProbe_DefaultStatus(t *testing.T) {
	logger := logging.Default()
	config := Config{Timeout: 5 * time.Second, Retries: 1, RetryBackoff: 10 * time.Millisecond}

	probe := NewHTTPProbe("http://example.com", 0, config, logger)
	if probe.expectStatus != 200 {
		t.Errorf("expected default status 200, got %d", probe.expectStatus)
	}
}
