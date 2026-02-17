package notify

import (
	"context"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	store := &memoryStore{settings: Defaults()}

	settings, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.CheckCron != defaultCheck {
		t.Errorf("expected default check cron, got %s", settings.CheckCron)
	}

	settings.DiscordEnabled = true
	settings.DiscordWebhook = "https://discord.com/api/webhooks/test"
	err = store.Save(context.Background(), settings)
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	loaded, _ := store.Load(context.Background())
	if !loaded.DiscordEnabled {
		t.Error("expected discord enabled after save")
	}
}

func TestSettingsDefaults(t *testing.T) {
	d := Defaults()
	if d.DiscordEnabled || d.SlackEnabled || d.NotifyOnFind || d.DigestEnabled {
		t.Error("expected all disabled by default")
	}
	if d.CheckCron != "*/15 * * * *" {
		t.Errorf("unexpected check cron: %s", d.CheckCron)
	}
	if d.DigestCron != "0 9 * * *" {
		t.Errorf("unexpected digest cron: %s", d.DigestCron)
	}
}

func TestSettingsNormalize(t *testing.T) {
	s := Settings{}
	n := s.Normalize()
	if n.CheckCron == "" {
		t.Error("expected non-empty check cron after normalize")
	}
	if n.DigestCron == "" {
		t.Error("expected non-empty digest cron after normalize")
	}
}

func TestSettingsValidate(t *testing.T) {
	// Discord enabled without webhook
	s := Settings{DiscordEnabled: true}
	if err := s.Validate(); err == nil {
		t.Error("expected error for discord without webhook")
	}

	// Slack enabled without webhook
	s = Settings{SlackEnabled: true}
	if err := s.Validate(); err == nil {
		t.Error("expected error for slack without webhook")
	}

	// Valid config
	s = Settings{
		DiscordEnabled: true,
		DiscordWebhook: "https://discord.com/test",
		NotifyOnFind:   true,
		CheckCron:      "*/15 * * * *",
	}
	if err := s.Validate(); err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}

	// Invalid cron
	s = Settings{
		NotifyOnFind: true,
		CheckCron:    "invalid-cron",
	}
	if err := s.Validate(); err == nil {
		t.Error("expected error for invalid cron")
	}
}

func TestHashUpdates_Deterministic(t *testing.T) {
	// Empty items
	hash1 := HashUpdates(nil)
	hash2 := HashUpdates(nil)
	if hash1 != hash2 {
		t.Error("expected same hash for empty items")
	}
}

func TestEncodeDecode(t *testing.T) {
	original := Settings{
		DiscordEnabled: true,
		DiscordWebhook: "https://test",
		CheckCron:      "*/5 * * * *",
		DigestCron:     "0 8 * * *",
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.DiscordEnabled != original.DiscordEnabled {
		t.Error("discord enabled mismatch")
	}
	if decoded.DiscordWebhook != original.DiscordWebhook {
		t.Error("discord webhook mismatch")
	}
}

func TestDecodeEmpty(t *testing.T) {
	settings, err := Decode("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.CheckCron != defaultCheck {
		t.Errorf("expected default check cron, got %s", settings.CheckCron)
	}
}
