package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordNotifier_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	notifier := &DiscordNotifier{
		WebhookURL: server.URL,
		Client:     server.Client(),
	}

	err := notifier.Send(context.Background(), "test message")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDiscordNotifier_ServerError(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := &DiscordNotifier{
		WebhookURL: server.URL,
		Client:     server.Client(),
	}

	err := notifier.Send(context.Background(), "test message")
	if err == nil {
		t.Error("expected error for server error")
	}
	// Should have retried
	if attempt < 2 {
		t.Errorf("expected retries, got %d attempts", attempt)
	}
}

func TestDiscordNotifier_MissingURL(t *testing.T) {
	notifier := &DiscordNotifier{}
	err := notifier.Send(context.Background(), "test")
	if err == nil {
		t.Error("expected error for missing webhook URL")
	}
}
