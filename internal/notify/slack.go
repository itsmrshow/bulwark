package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type slackTextObject struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

type slackBlock struct {
	Type     string            `json:"type"`
	Text     *slackTextObject  `json:"text,omitempty"`
	Fields   []slackTextObject `json:"fields,omitempty"`
	Elements []slackTextObject `json:"elements,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color,omitempty"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackPayload struct {
	Text        string            `json:"text,omitempty"`
	Blocks      []slackBlock      `json:"blocks,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

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
		return s.doSendPayload(ctx, slackPayload{Text: message})
	})
}

// SendRich sends a structured Slack payload with retry.
func (s *SlackNotifier) SendRich(ctx context.Context, payload slackPayload) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack webhook url missing")
	}

	return sendWithRetry(ctx, "slack", func(ctx context.Context) (int, error) {
		return s.doSendPayload(ctx, payload)
	})
}

func (s *SlackNotifier) doSendPayload(ctx context.Context, payload slackPayload) (int, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

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

func embedToSlackPayload(embed discordEmbed) slackPayload {
	blocks := make([]slackBlock, 0, 2+len(embed.Fields))
	plainText := firstNonEmpty(embed.Title, embed.Description, "Bulwark notification")

	if embed.Title != "" {
		blocks = append(blocks, slackBlock{
			Type: "section",
			Text: &slackTextObject{Type: "mrkdwn", Text: fmt.Sprintf("*%s*", escapeSlack(embed.Title))},
		})
	}
	if embed.Description != "" {
		blocks = append(blocks, slackBlock{
			Type: "section",
			Text: &slackTextObject{Type: "mrkdwn", Text: escapeSlack(embed.Description)},
		})
	}

	fields := make([]slackTextObject, 0, 10)
	flushFields := func() {
		if len(fields) == 0 {
			return
		}
		blocks = append(blocks, slackBlock{Type: "section", Fields: append([]slackTextObject(nil), fields...)})
		fields = fields[:0]
	}
	for _, field := range embed.Fields {
		fields = append(fields, slackTextObject{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s*\n%s", escapeSlack(field.Name), escapeSlack(field.Value)),
		})
		if len(fields) == 10 {
			flushFields()
		}
	}
	flushFields()

	contextItems := make([]slackTextObject, 0, 2)
	if embed.Footer != nil && embed.Footer.Text != "" {
		contextItems = append(contextItems, slackTextObject{Type: "mrkdwn", Text: escapeSlack(embed.Footer.Text)})
	}
	if embed.Timestamp != "" {
		contextItems = append(contextItems, slackTextObject{Type: "mrkdwn", Text: escapeSlack(embed.Timestamp)})
	}
	if len(contextItems) > 0 {
		blocks = append(blocks, slackBlock{Type: "context", Elements: contextItems})
	}

	payload := slackPayload{Text: plainText}
	if embed.Color != 0 {
		payload.Attachments = []slackAttachment{{
			Color:  fmt.Sprintf("#%06x", embed.Color),
			Blocks: blocks,
		}}
		return payload
	}

	payload.Blocks = blocks
	return payload
}

func escapeSlack(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
