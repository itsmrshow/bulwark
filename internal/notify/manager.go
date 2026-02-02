package notify

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/planner"
	"github.com/yourusername/bulwark/internal/scheduler"
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
	if discord == "" && slack == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

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

	message := formatMessage(mode, updates, plan)
	return m.send(ctx, settings, message)
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

func formatMessage(mode string, updates []planner.PlanItem, plan *planner.Plan) string {
	var b strings.Builder
	if mode == "digest" {
		b.WriteString("Bulwark daily update digest")
	} else {
		b.WriteString("Bulwark detected new image updates")
	}
	b.WriteString(fmt.Sprintf(" (%d updates)", len(updates)))
	b.WriteString("\n")

	limit := len(updates)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		item := updates[i]
		b.WriteString(fmt.Sprintf("- %s/%s (%s)\n", item.TargetName, item.ServiceName, item.Image))
	}

	if len(updates) > limit {
		b.WriteString(fmt.Sprintf("...and %d more\n", len(updates)-limit))
	}

	b.WriteString(fmt.Sprintf("\nGenerated at %s", plan.GeneratedAt.Format(time.RFC3339)))
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
