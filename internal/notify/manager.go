package notify

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/planner"
	"github.com/itsmrshow/bulwark/internal/scheduler"
	"github.com/itsmrshow/bulwark/internal/state"
)

type PlanFunc func(ctx context.Context) (*planner.Plan, error)

// Manager orchestrates notification settings and scheduled jobs.
type Manager struct {
	logger   *logging.Logger
	store    Store
	planFn   PlanFunc
	mu       sync.RWMutex
	config   Settings
	lastHash string
	sched    *scheduler.Scheduler
	envLock  Settings
}

// NewManager creates a notification manager.
func NewManager(store Store, planFn PlanFunc, logger *logging.Logger) *Manager {
	if logger == nil {
		logger = logging.Default()
	}
	return &Manager{
		logger: logger.WithComponent("notify"),
		store:  store,
		planFn: planFn,
		config: Defaults(),
	}
}

// Load loads settings from the store if available.
func (m *Manager) Load(ctx context.Context) error {
	if m.store != nil {
		if settings, err := m.store.Load(ctx); err == nil {
			m.mu.Lock()
			m.config = settings.Normalize()
			m.mu.Unlock()
		}
	}
	m.applyEnvOverrides()
	return nil
}

// Settings returns current settings.
func (m *Manager) Settings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Normalize()
}

// Reload applies env overrides to current config.
func (m *Manager) Reload(ctx context.Context) {
	m.applyEnvOverrides()
	m.restartScheduler()
}

// EnvLocked returns environment-provided webhook settings.
func (m *Manager) EnvLocked() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.envLock.Normalize()
}

// Update saves settings and restarts scheduled jobs.
func (m *Manager) Update(ctx context.Context, settings Settings) error {
	settings = settings.Normalize()
	if err := settings.Validate(); err != nil {
		return err
	}

	if m.store != nil {
		if err := m.store.Save(ctx, settings); err != nil {
			return err
		}
	}

	m.mu.Lock()
	merged := settings
	if m.envLock.DiscordWebhook != "" {
		merged.DiscordWebhook = m.envLock.DiscordWebhook
		merged.DiscordEnabled = true
	}
	if m.envLock.SlackWebhook != "" {
		merged.SlackWebhook = m.envLock.SlackWebhook
		merged.SlackEnabled = true
	}
	m.config = merged
	m.mu.Unlock()

	m.applyEnvOverrides()
	m.restartScheduler()
	return nil
}

// Start loads settings and starts scheduled jobs.
func (m *Manager) Start(ctx context.Context) {
	_ = m.Load(ctx)
	m.restartScheduler()
}

// Stop stops scheduled jobs.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sched != nil {
		m.sched.Stop()
		m.sched = nil
	}
}

// Test sends a test notification.
func (m *Manager) Test(ctx context.Context) error {
	settings := m.Settings()
	message := "Bulwark test notification ✅"
	return m.send(ctx, settings, message)
}

func (m *Manager) restartScheduler() {
	m.Stop()

	settings := m.Settings()
	if !settings.NotifyOnFind && !settings.DigestEnabled {
		return
	}

	sched := scheduler.NewScheduler(m.logger)

	if settings.NotifyOnFind {
		if err := sched.AddJob(settings.CheckCron, &notifyJob{manager: m, mode: "immediate"}); err != nil {
			m.logger.Error().Err(err).Msg("failed to schedule immediate notifications")
		}
	}
	if settings.DigestEnabled {
		if err := sched.AddJob(settings.DigestCron, &notifyJob{manager: m, mode: "digest"}); err != nil {
			m.logger.Error().Err(err).Msg("failed to schedule digest notifications")
		}
	}

	sched.Start()

	m.mu.Lock()
	m.sched = sched
	m.mu.Unlock()
}

func (m *Manager) applyEnvOverrides() {
	discord := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
	slack := strings.TrimSpace(os.Getenv("SLACK_WEBHOOK_URL"))
	notifyOnFind, notifyOnFindSet := readEnvBool("BULWARK_NOTIFY_ON_FIND")
	digestEnabled, digestEnabledSet := readEnvBool("BULWARK_NOTIFY_DIGEST")
	checkCron := strings.TrimSpace(os.Getenv("BULWARK_NOTIFY_CHECK_CRON"))
	digestCron := strings.TrimSpace(os.Getenv("BULWARK_NOTIFY_DIGEST_CRON"))
	if discord == "" && slack == "" && !notifyOnFindSet && !digestEnabledSet && checkCron == "" && digestCron == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if notifyOnFindSet {
		m.config.NotifyOnFind = notifyOnFind
		m.envLock.NotifyOnFind = notifyOnFind
	}
	if digestEnabledSet {
		m.config.DigestEnabled = digestEnabled
		m.envLock.DigestEnabled = digestEnabled
	}
	if checkCron != "" {
		m.config.CheckCron = checkCron
		m.envLock.CheckCron = checkCron
	}
	if digestCron != "" {
		m.config.DigestCron = digestCron
		m.envLock.DigestCron = digestCron
	}

	if discord != "" {
		m.config.DiscordWebhook = discord
		m.config.DiscordEnabled = true
		m.envLock.DiscordWebhook = discord
		m.envLock.DiscordEnabled = true
	}
	if slack != "" {
		m.config.SlackWebhook = slack
		m.config.SlackEnabled = true
		m.envLock.SlackWebhook = slack
		m.envLock.SlackEnabled = true
	}

	m.config = m.config.Normalize()
}

func readEnvBool(key string) (bool, bool) {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return false, false
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func (m *Manager) run(ctx context.Context, mode string) error {
	settings := m.Settings()
	if !settings.NotifyOnFind && !settings.DigestEnabled {
		return nil
	}

	plan, err := m.planFn(ctx)
	if err != nil {
		return err
	}

	updates := make([]planner.PlanItem, 0)
	for _, item := range plan.Items {
		if item.UpdateAvailable {
			updates = append(updates, item)
		}
	}
	if len(updates) == 0 {
		return nil
	}

	if mode == "immediate" {
		hash := HashUpdates(updates)
		m.mu.RLock()
		lastHash := m.lastHash
		m.mu.RUnlock()
		if hash == lastHash {
			return nil
		}
		m.mu.Lock()
		m.lastHash = hash
		m.mu.Unlock()

		if m.store != nil {
			m.store.SetHash(ctx, hash)
		}
	}

	embed := formatDiscoveryEmbed(mode, updates, plan)
	return m.sendDiscordEmbed(ctx, settings, embed)
}

func (m *Manager) send(ctx context.Context, settings Settings, message string) error {
	var errs []string

	if settings.DiscordEnabled {
		discord := DiscordNotifier{WebhookURL: settings.DiscordWebhook}
		if err := discord.Send(ctx, message); err != nil {
			errs = append(errs, fmt.Sprintf("discord: %v", err))
		}
	}

	if settings.SlackEnabled {
		slack := SlackNotifier{WebhookURL: settings.SlackWebhook}
		if err := slack.Send(ctx, message); err != nil {
			errs = append(errs, fmt.Sprintf("slack: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// sendDiscordEmbed sends a rich embed via Discord and falls back to plain text for Slack.
func (m *Manager) sendDiscordEmbed(ctx context.Context, settings Settings, embed discordEmbed) error {
	var errs []string

	if settings.DiscordEnabled {
		discord := DiscordNotifier{WebhookURL: settings.DiscordWebhook}
		if err := discord.sendEmbed(ctx, embed); err != nil {
			errs = append(errs, fmt.Sprintf("discord: %v", err))
		}
	}

	if settings.SlackEnabled {
		// Slack does not support Discord embed format; format as plain text.
		text := embedToText(embed)
		slack := SlackNotifier{WebhookURL: settings.SlackWebhook}
		if err := slack.Send(ctx, text); err != nil {
			errs = append(errs, fmt.Sprintf("slack: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// NotifyResult sends a per-update result notification after an update completes.
func (m *Manager) NotifyResult(ctx context.Context, result *state.UpdateResult, image string) {
	settings := m.Settings()
	if !settings.DiscordEnabled && !settings.SlackEnabled {
		return
	}

	var title string
	var color int
	switch {
	case result.RollbackPerformed:
		title = "⚠️ Update Rolled Back"
		color = 0xFEE75C
	case result.Success:
		title = "✅ Updated"
		color = 0x57F287
	default:
		title = "❌ Update Failed"
		color = 0xED4245
	}

	oldDigest := shortDigest(result.OldDigest)
	newDigest := shortDigest(result.NewDigest)
	duration := result.CompletedAt.Sub(result.StartedAt).Round(time.Millisecond).String()

	embed := discordEmbed{
		Title: title,
		Color: color,
		Fields: []discordEmbedField{
			{Name: "Container", Value: result.ServiceName, Inline: true},
			{Name: "Image", Value: image, Inline: true},
			{Name: "Old Digest", Value: "`" + oldDigest + "`", Inline: true},
			{Name: "New Digest", Value: "`" + newDigest + "`", Inline: true},
			{Name: "Duration", Value: duration, Inline: true},
		},
		Footer:    &discordEmbedFooter{Text: "Bulwark"},
		Timestamp: result.CompletedAt.UTC().Format(time.RFC3339),
	}

	if err := m.sendDiscordEmbed(ctx, settings, embed); err != nil {
		m.logger.Warn().Err(err).Msg("failed to send result notification")
	}
}

func formatDiscoveryEmbed(mode string, updates []planner.PlanItem, plan *planner.Plan) discordEmbed {
	title := "🔄 Updates Available"
	if mode == "digest" {
		title = "📋 Daily Digest"
	}

	limit := len(updates)
	if limit > 20 {
		limit = 20
	}

	fields := make([]discordEmbedField, 0, limit)
	for i := 0; i < limit; i++ {
		item := updates[i]
		cur := shortDigest(item.CurrentDigest)
		rem := shortDigest(item.RemoteDigest)
		value := fmt.Sprintf("%s\n`%s` → `%s`", item.Image, cur, rem)
		fields = append(fields, discordEmbedField{
			Name:  fmt.Sprintf("%s/%s", item.TargetName, item.ServiceName),
			Value: value,
		})
	}

	if len(updates) > limit {
		fields = append(fields, discordEmbedField{
			Name:  "…",
			Value: fmt.Sprintf("and %d more", len(updates)-limit),
		})
	}

	return discordEmbed{
		Title:       title,
		Description: fmt.Sprintf("%d image update(s) detected", len(updates)),
		Color:       0xF4A22C,
		Fields:      fields,
		Footer:      &discordEmbedFooter{Text: "Bulwark"},
		Timestamp:   plan.GeneratedAt.UTC().Format(time.RFC3339),
	}
}

func shortDigest(digest string) string {
	bare := digest
	if strings.HasPrefix(bare, "sha256:") {
		bare = bare[7:]
	}
	if len(bare) > 12 {
		bare = bare[:12]
	}
	if bare == "" {
		return "none"
	}
	return bare
}

func embedToText(embed discordEmbed) string {
	var b strings.Builder
	if embed.Title != "" {
		b.WriteString(embed.Title)
		b.WriteString("\n")
	}
	if embed.Description != "" {
		b.WriteString(embed.Description)
		b.WriteString("\n")
	}
	for _, f := range embed.Fields {
		b.WriteString(fmt.Sprintf("• %s: %s\n", f.Name, f.Value))
	}
	if embed.Timestamp != "" {
		b.WriteString(fmt.Sprintf("\n%s", embed.Timestamp))
	}
	return b.String()
}

type notifyJob struct {
	manager *Manager
	mode    string
}

func (j *notifyJob) Name() string {
	if j.mode == "digest" {
		return "notify-digest"
	}
	return "notify-immediate"
}

func (j *notifyJob) Execute(ctx context.Context) error {
	return j.manager.run(ctx, j.mode)
}
