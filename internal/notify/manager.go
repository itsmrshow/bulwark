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

type AutoUpdateRunReport struct {
	RunID       string
	Mode        string
	Status      string
	StartedAt   time.Time
	CompletedAt time.Time
	Summary     AutoUpdateRunSummary
	Items       []AutoUpdateRunItem
}

type AutoUpdateRunSummary struct {
	UpdatesApplied int
	UpdatesSkipped int
	UpdatesFailed  int
	Rollbacks      int
}

type AutoUpdateRunItem struct {
	Target      string
	Service     string
	Image       string
	Result      string
	CompletedAt time.Time
	Details     string
}

// ApplyFunc triggers an automatic update run. safe and unsafe correspond to
// which risk tiers should be updated.
type ApplyFunc func(ctx context.Context, safe bool, unsafe bool)

// Manager orchestrates notification settings and scheduled jobs.
type Manager struct {
	logger   *logging.Logger
	store    Store
	planFn   PlanFunc
	applyFn  ApplyFunc
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

// WithApplyFunc attaches an apply function for scheduled auto-updates.
func (m *Manager) WithApplyFunc(fn ApplyFunc) *Manager {
	m.applyFn = fn
	return m
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
	autoUpdateActive := settings.AutoUpdateEnabled && m.applyFn != nil
	if !settings.NotifyOnFind && !settings.DigestEnabled && !autoUpdateActive {
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
	if autoUpdateActive {
		if err := sched.AddJob(settings.AutoUpdateCron, &autoUpdateJob{
			manager: m,
			safe:    settings.AutoUpdateSafe,
			unsafe:  settings.AutoUpdateUnsafe,
		}); err != nil {
			m.logger.Error().Err(err).Msg("failed to schedule auto-update")
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
	autoUpdateEnabled, autoUpdateEnabledSet := readEnvBool("BULWARK_AUTO_UPDATE_ENABLED")
	autoUpdateSafe, autoUpdateSafeSet := readEnvBool("BULWARK_AUTO_UPDATE_SAFE")
	autoUpdateUnsafe, autoUpdateUnsafeSet := readEnvBool("BULWARK_AUTO_UPDATE_UNSAFE")
	autoUpdateCron := strings.TrimSpace(os.Getenv("BULWARK_AUTO_UPDATE_CRON"))

	if discord == "" && slack == "" &&
		!notifyOnFindSet && !digestEnabledSet && checkCron == "" && digestCron == "" &&
		!autoUpdateEnabledSet && !autoUpdateSafeSet && !autoUpdateUnsafeSet && autoUpdateCron == "" {
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
	if autoUpdateEnabledSet {
		m.config.AutoUpdateEnabled = autoUpdateEnabled
		m.envLock.AutoUpdateEnabled = autoUpdateEnabled
	}
	if autoUpdateSafeSet {
		m.config.AutoUpdateSafe = autoUpdateSafe
		m.envLock.AutoUpdateSafe = autoUpdateSafe
	}
	if autoUpdateUnsafeSet {
		m.config.AutoUpdateUnsafe = autoUpdateUnsafe
		m.envLock.AutoUpdateUnsafe = autoUpdateUnsafe
	}
	if autoUpdateCron != "" {
		m.config.AutoUpdateCron = autoUpdateCron
		m.envLock.AutoUpdateCron = autoUpdateCron
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
		slack := SlackNotifier{WebhookURL: settings.SlackWebhook}
		if err := slack.SendRich(ctx, embedToSlackPayload(embed)); err != nil {
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
	probeSummary := summarizeProbeResults(result.ProbeResults)

	description := fmt.Sprintf("%s finished for `%s` on `%s`.", notificationOutcomeLabel(result), image, result.TargetID)
	if result.TargetID == "" {
		description = fmt.Sprintf("%s finished for `%s`.", notificationOutcomeLabel(result), image)
	}

	embed := discordEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields: []discordEmbedField{
			{Name: "Target", Value: nonEmpty(result.TargetID, "unknown"), Inline: true},
			{Name: "Service", Value: result.ServiceName, Inline: true},
			{Name: "Image", Value: image, Inline: false},
			{Name: "Old Digest", Value: "`" + oldDigest + "`", Inline: true},
			{Name: "New Digest", Value: "`" + newDigest + "`", Inline: true},
			{Name: "Duration", Value: duration, Inline: true},
			{Name: "Probes", Value: probeSummary, Inline: true},
		},
		Footer:    &discordEmbedFooter{Text: "Bulwark"},
		Timestamp: result.CompletedAt.UTC().Format(time.RFC3339),
	}
	if result.RollbackPerformed {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:  "Rollback",
			Value: fmt.Sprintf("Returned to `%s`", shortDigest(result.RollbackDigest)),
		})
	}
	if result.Error != nil {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:  "Details",
			Value: truncateNotificationText(result.Error.Error(), 300),
		})
	}

	if err := m.sendDiscordEmbed(ctx, settings, embed); err != nil {
		m.logger.Warn().Err(err).Msg("failed to send result notification")
	}
}

func (m *Manager) NotifyAutoUpdateRun(ctx context.Context, report AutoUpdateRunReport) {
	settings := m.Settings()
	if !settings.DiscordEnabled || settings.DiscordWebhook == "" {
		return
	}

	embed := formatAutoUpdateRunEmbed(report)
	discord := DiscordNotifier{WebhookURL: settings.DiscordWebhook}
	if err := discord.sendEmbed(ctx, embed); err != nil {
		m.logger.Warn().Err(err).Str("run_id", report.RunID).Msg("failed to send auto-update completion notification")
	}
}

func formatDiscoveryEmbed(mode string, updates []planner.PlanItem, plan *planner.Plan) discordEmbed {
	title := "🔄 Updates Available"
	if mode == "digest" {
		title = "📋 Scheduled Digest"
	}

	limit := len(updates)
	if limit > 8 {
		limit = 8
	}

	blockedCount := 0
	for _, item := range updates {
		if !item.Allowed {
			blockedCount++
		}
	}

	fields := []discordEmbedField{
		{Name: "Updates", Value: fmt.Sprintf("`%d`", len(updates)), Inline: true},
		{Name: "Allowed", Value: fmt.Sprintf("`%d`", plan.AllowedCount), Inline: true},
		{Name: "Blocked", Value: fmt.Sprintf("`%d`", blockedCount), Inline: true},
	}
	for i := 0; i < limit; i++ {
		item := updates[i]
		cur := shortDigest(item.CurrentDigest)
		rem := shortDigest(item.RemoteDigest)
		value := fmt.Sprintf(
			"`%s` -> `%s`\n%s\n%s · %s",
			cur,
			rem,
			item.Image,
			notificationDecision(item),
			truncateNotificationText(item.Reason, 96),
		)
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
		Description: fmt.Sprintf("%d update(s) detected across %d target(s) and %d tracked service(s).", len(updates), plan.TargetCount, plan.ServiceCount),
		Color:       0xF4A22C,
		Fields:      fields,
		Footer:      &discordEmbedFooter{Text: fmt.Sprintf("Bulwark • %s", mode)},
		Timestamp:   plan.GeneratedAt.UTC().Format(time.RFC3339),
	}
}

func formatAutoUpdateRunEmbed(report AutoUpdateRunReport) discordEmbed {
	title := "✅ Auto-Update Completed"
	color := 0x57F287
	if report.Status == "failed" {
		title = "❌ Auto-Update Completed With Failures"
		color = 0xED4245
	} else if report.Summary.Rollbacks > 0 {
		title = "⚠️ Auto-Update Completed With Rollbacks"
		color = 0xFEE75C
	}

	duration := report.CompletedAt.Sub(report.StartedAt).Round(time.Millisecond).String()
	fields := []discordEmbedField{
		{Name: "Run ID", Value: report.RunID, Inline: false},
		{Name: "Applied", Value: fmt.Sprintf("`%d`", report.Summary.UpdatesApplied), Inline: true},
		{Name: "Skipped", Value: fmt.Sprintf("`%d`", report.Summary.UpdatesSkipped), Inline: true},
		{Name: "Failed", Value: fmt.Sprintf("`%d`", report.Summary.UpdatesFailed), Inline: true},
		{Name: "Rollbacks", Value: fmt.Sprintf("`%d`", report.Summary.Rollbacks), Inline: true},
		{Name: "Mode", Value: fmt.Sprintf("`%s`", nonEmpty(report.Mode, "safe")), Inline: true},
		{Name: "Duration", Value: duration, Inline: true},
	}

	limit := len(report.Items)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		item := report.Items[i]
		details := strings.TrimSpace(item.Details)
		if details == "" {
			details = "No details"
		}
		fields = append(fields, discordEmbedField{
			Name: fmt.Sprintf("%s/%s", nonEmpty(item.Target, "unknown"), nonEmpty(item.Service, "unknown")),
			Value: fmt.Sprintf(
				"%s\n`%s`\n%s\n%s",
				item.Image,
				item.Result,
				item.CompletedAt.UTC().Format(time.RFC3339),
				truncateNotificationText(details, 140),
			),
		})
	}
	if len(report.Items) > limit {
		fields = append(fields, discordEmbedField{
			Name:  "…",
			Value: fmt.Sprintf("and %d more result(s)", len(report.Items)-limit),
		})
	}

	description := fmt.Sprintf(
		"Scheduled auto-update finished with %d applied, %d skipped, %d failed, and %d rollback(s).",
		report.Summary.UpdatesApplied,
		report.Summary.UpdatesSkipped,
		report.Summary.UpdatesFailed,
		report.Summary.Rollbacks,
	)
	if len(report.Items) == 0 {
		description = "Scheduled auto-update finished with no container changes."
	}

	return discordEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      fields,
		Footer:      &discordEmbedFooter{Text: "Bulwark • auto-update"},
		Timestamp:   report.CompletedAt.UTC().Format(time.RFC3339),
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

func notificationDecision(item planner.PlanItem) string {
	if item.Allowed {
		return fmt.Sprintf("%s • allowed", item.Risk)
	}
	return fmt.Sprintf("%s • blocked", item.Risk)
}

func notificationOutcomeLabel(result *state.UpdateResult) string {
	switch {
	case result.RollbackPerformed:
		return "Rollback"
	case result.Success:
		return "Update"
	default:
		return "Failure"
	}
}

func summarizeProbeResults(results []state.ProbeResult) string {
	if len(results) == 0 {
		return "No probes"
	}

	passed := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			passed++
			continue
		}
		failed++
	}

	if failed == 0 {
		return fmt.Sprintf("%d passed", passed)
	}
	return fmt.Sprintf("%d passed, %d failed", passed, failed)
}

func truncateNotificationText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit || limit <= 3 {
		return value
	}
	return value[:limit-3] + "..."
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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

type autoUpdateJob struct {
	manager *Manager
	safe    bool
	unsafe  bool
}

func (j *autoUpdateJob) Name() string { return "auto-update" }

func (j *autoUpdateJob) Execute(ctx context.Context) error {
	if j.manager.applyFn == nil {
		return fmt.Errorf("auto-update: apply function not configured")
	}
	j.manager.logger.Info().Bool("safe", j.safe).Bool("unsafe", j.unsafe).Msg("auto-update triggered by scheduler")
	j.manager.applyFn(ctx, j.safe, j.unsafe)
	return nil
}
