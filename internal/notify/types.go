package notify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/itsmrshow/bulwark/internal/planner"
)

const (
	settingsKey   = "notifications.settings"
	lastHashKey   = "notifications.last_hash"
	defaultCheck  = "*/15 * * * *"
	defaultDigest = "0 9 * * *"
)

// Settings controls notification behavior.
type Settings struct {
	DiscordWebhook string `json:"discord_webhook"`
	SlackWebhook   string `json:"slack_webhook"`
	DiscordEnabled bool   `json:"discord_enabled"`
	SlackEnabled   bool   `json:"slack_enabled"`
	NotifyOnFind   bool   `json:"notify_on_find"`
	DigestEnabled  bool   `json:"digest_enabled"`
	CheckCron      string `json:"check_cron"`
	DigestCron     string `json:"digest_cron"`
}

// Defaults returns default notification settings.
func Defaults() Settings {
	return Settings{
		DiscordEnabled: false,
		SlackEnabled:   false,
		NotifyOnFind:   false,
		DigestEnabled:  false,
		CheckCron:      defaultCheck,
		DigestCron:     defaultDigest,
	}
}

// Normalize applies defaults to empty fields.
func (s Settings) Normalize() Settings {
	if s.CheckCron == "" {
		s.CheckCron = defaultCheck
	}
	if s.DigestCron == "" {
		s.DigestCron = defaultDigest
	}
	return s
}

// Validate ensures configuration is reasonable.
func (s Settings) Validate() error {
	if s.DiscordEnabled && s.DiscordWebhook == "" {
		return fmt.Errorf("discord webhook required when enabled")
	}
	if s.SlackEnabled && s.SlackWebhook == "" {
		return fmt.Errorf("slack webhook required when enabled")
	}
	if s.NotifyOnFind {
		if _, err := cron.ParseStandard(s.CheckCron); err != nil {
			return fmt.Errorf("invalid check cron: %w", err)
		}
	}
	if s.DigestEnabled {
		if _, err := cron.ParseStandard(s.DigestCron); err != nil {
			return fmt.Errorf("invalid digest cron: %w", err)
		}
	}
	return nil
}

// Encode converts settings to JSON.
func Encode(settings Settings) (string, error) {
	normalized := settings.Normalize()
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Decode parses settings from JSON.
func Decode(value string) (Settings, error) {
	if value == "" {
		return Defaults(), nil
	}
	var settings Settings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return Settings{}, err
	}
	return settings.Normalize(), nil
}

// HashUpdates computes a stable hash for update items.
func HashUpdates(items []planner.PlanItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if !item.UpdateAvailable {
			continue
		}
		parts = append(parts, item.TargetID+":"+item.ServiceID+":"+item.RemoteDigest)
	}
	joined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}
