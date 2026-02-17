package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackNotifier_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &SlackNotifier{
		WebhookURL: server.URL,
		Client:     server.Client(),
	}

	err := notifier.Send(context.Background(), "test message")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSlackNotifier_ServerError(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := &SlackNotifier{
		WebhookURL: server.URL,
		Client:     server.Client(),
	}

	err := notifier.Send(context.Background(), "test message")
	if err == nil {
		t.Error("expected error for server error")
	}
	if attempt < 2 {
		t.Errorf("expected retries, got %d attempts", attempt)
	}
}

func TestSlackNotifier_MissingURL(t *testing.T) {
	notifier := &SlackNotifier{}
	err := notifier.Send(context.Background(), "test")
	if err == nil {
		t.Error("expected error for missing webhook URL")
	}
}
