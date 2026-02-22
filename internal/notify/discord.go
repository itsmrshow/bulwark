package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// discordEmbedField is a single field in a Discord embed.
type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// discordEmbedFooter is the footer of a Discord embed.
type discordEmbedFooter struct {
	Text string `json:"text"`
}

// discordEmbed is a Discord rich embed object.
type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Footer      *discordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// discordPayload is the top-level Discord webhook payload.
type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

// DiscordNotifier sends messages to a Discord webhook.
type DiscordNotifier struct {
	WebhookURL string
	Client     *http.Client
}

// Send sends a plain-text message to Discord with retry.
func (d *DiscordNotifier) Send(ctx context.Context, message string) error {
	if d.WebhookURL == "" {
		return fmt.Errorf("discord webhook url missing")
	}

	return sendWithRetry(ctx, "discord", func(ctx context.Context) (int, error) {
		return d.doSend(ctx, message)
	})
}

// sendEmbed sends a rich embed to Discord with retry.
func (d *DiscordNotifier) sendEmbed(ctx context.Context, embed discordEmbed) error {
	if d.WebhookURL == "" {
		return fmt.Errorf("discord webhook url missing")
	}

	return sendWithRetry(ctx, "discord", func(ctx context.Context) (int, error) {
		return d.doSendEmbed(ctx, embed)
	})
}

func (d *DiscordNotifier) doSend(ctx context.Context, message string) (int, error) {
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]string{"content": message}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.WebhookURL, bytes.NewReader(body))
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
		return resp.StatusCode, fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (d *DiscordNotifier) doSendEmbed(ctx context.Context, embed discordEmbed) (int, error) {
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	payload := discordPayload{Embeds: []discordEmbed{embed}}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.WebhookURL, bytes.NewReader(body))
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
		return resp.StatusCode, fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
