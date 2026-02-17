package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends messages to a Slack webhook.
type SlackNotifier struct {
	WebhookURL string
	Client     *http.Client
}

// Send sends a message to Slack with retry.
func (s *SlackNotifier) Send(ctx context.Context, message string) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack webhook url missing")
	}

	return sendWithRetry(ctx, "slack", func(ctx context.Context) (int, error) {
		return s.doSend(ctx, message)
	})
}

func (s *SlackNotifier) doSend(ctx context.Context, message string) (int, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]string{"text": message}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
