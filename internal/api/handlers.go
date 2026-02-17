package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itsmrshow/bulwark/internal/discovery"
	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/executor"
	"github.com/itsmrshow/bulwark/internal/planner"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/registry"
	"github.com/itsmrshow/bulwark/internal/state"
)

type healthResponse struct {
	Status    string `json:"status"`
	ReadOnly  bool   `json:"read_only"`
	UIEnabled bool   `json:"ui_enabled"`
}

type overviewResponse struct {
	GeneratedAt      time.Time      `json:"generated_at"`
	ReadOnly         bool           `json:"read_only"`
	ManagedTargets   int            `json:"managed_targets"`
	ManagedServices  int            `json:"managed_services"`
	UpdatesAvailable int            `json:"updates_available"`
	LastRun          *overviewRun   `json:"last_run,omitempty"`
	Failures         int            `json:"failures"`
	Rollbacks        int            `json:"rollbacks"`
	Activity         []activityItem `json:"activity"`
}

type overviewRun struct {
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"`
}

type activityItem struct {
	Timestamp time.Time `json:"ts"`
	Action    string    `json:"action"`
	Target    string    `json:"target,omitempty"`
	Service   string    `json:"service,omitempty"`
	Message   string    `json:"message"`
}

type planRequest struct {
	Target          string `json:"target,omitempty"`
	IncludeDisabled bool   `json:"include_disabled,omitempty"`
}

type applyRequest struct {
	Mode       string   `json:"mode"`
	Target     string   `json:"target,omitempty"`
	ServiceIDs []string `json:"service_ids,omitempty"`
	Force      bool     `json:"force,omitempty"`
}

type applyResponse struct {
	RunID string `json:"run_id"`
}

type historyResponse struct {
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Items    []planner.HistoryItem `json:"items"`
	HasMore  bool                  `json:"has_more"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		ReadOnly:  s.cfg.ReadOnly,
		UIEnabled: s.cfg.UIEnabled,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	if s.cfg.WebToken == "" {
		writeError(w, http.StatusServiceUnavailable, "authentication disabled", "BULWARK_WEB_TOKEN is not configured")
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if req.Token != s.cfg.WebToken {
		writeError(w, http.StatusUnauthorized, "invalid token", "")
		return
	}

	// Create session
	sessionID := s.sessions.create()

	// Set httpOnly cookie (secure in production with HTTPS)
	http.SetCookie(w, &http.Cookie{
		Name:     "bulwark_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged in successfully",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	// Get and delete session
	if cookie, err := r.Cookie("bulwark_session"); err == nil {
		s.sessions.delete(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "bulwark_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (s *Server) handleEnableWrites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	if s.cfg.ReadOnly {
		writeError(w, http.StatusForbidden, "read-only mode", "Bulwark is in global read-only mode (BULWARK_UI_READONLY=true)")
		return
	}

	if s.cfg.WebToken == "" {
		writeError(w, http.StatusServiceUnavailable, "authentication disabled", "BULWARK_WEB_TOKEN is not configured")
		return
	}

	// Auto-authenticate using the backend token (no user token required)
	sessionID := s.sessions.create()

	// Set httpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "bulwark_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Write mode enabled",
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	ctx := r.Context()
	plan, planErr := s.getPlan(ctx, planRequest{})

	var managedTargets int
	var managedServices int
	if plan != nil {
		managedTargets = plan.TargetCount
		managedServices = plan.ServiceCount
	}

	updatesAvailable := 0
	if plan != nil {
		updatesAvailable = plan.UpdateCount
	}

	var failures, rollbacks int
	var lastRun *overviewRun
	activity := s.activityFromRuns(20)

	if s.store != nil {
		history, err := s.store.GetUpdateHistory(ctx, 50)
		if err == nil {
			for i, item := range history {
				if !item.Success {
					failures++
				}
				if item.RollbackPerformed {
					rollbacks++
				}
				if i == 0 {
					lastRun = &overviewRun{CompletedAt: item.CompletedAt, Status: statusFromResult(item)}
				}
			}

			if len(activity) < 20 {
				activity = append(activity, activityFromHistory(history, 20-len(activity))...)
			}
		}
	}

	resp := overviewResponse{
		GeneratedAt:      time.Now().UTC(),
		ReadOnly:         s.cfg.ReadOnly,
		ManagedTargets:   managedTargets,
		ManagedServices:  managedServices,
		UpdatesAvailable: updatesAvailable,
		LastRun:          lastRun,
		Failures:         failures,
		Rollbacks:        rollbacks,
		Activity:         activity,
	}

	if planErr != nil {
		s.logger.Warn().Err(planErr).Msg("overview plan failed")
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	ctx := r.Context()
	targets, err := s.discoverTargets(ctx, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "discovery failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"targets": targets})
}

func (s *Server) handleTargetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/targets/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing target id", "")
		return
	}

	ctx := r.Context()
	target, err := s.discoverTarget(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "target not found", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, target)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req planRequest
	if r.Body != nil {
		if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid request", err.Error())
			return
		}
	}

	plan, err := s.getPlan(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "plan failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req applyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "safe"
	}
	if mode != "safe" && mode != "selected" && mode != "all" {
		writeError(w, http.StatusBadRequest, "invalid mode", "use safe, selected, or all")
		return
	}

	run := s.runs.CreateRun("apply")
	writeJSON(w, http.StatusAccepted, applyResponse{RunID: run.ID})

	go s.executeApply(run.ID, req, mode)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing run id", "")
		return
	}

	run, ok := s.runs.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found", "")
		return
	}

	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	if s.store == nil {
		writeJSON(w, http.StatusOK, historyResponse{Page: 1, PageSize: 0, Items: []planner.HistoryItem{}})
		return
	}

	page := parseIntQuery(r, "page", 1)
	pageSize := parseIntQuery(r, "page_size", 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	filters := planner.HistoryFilter{
		TargetID:  r.URL.Query().Get("target_id"),
		ServiceID: r.URL.Query().Get("service_id"),
		Result:    r.URL.Query().Get("result"),
	}

	items, hasMore, err := s.getHistory(r.Context(), filters, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "history failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, historyResponse{
		Page:     page,
		PageSize: pageSize,
		Items:    items,
		HasMore:  hasMore,
	})
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	// Parse request
	target := r.URL.Query().Get("target")
	service := r.URL.Query().Get("service")

	if target == "" || service == "" {
		writeError(w, http.StatusBadRequest, "missing required parameters: target and service", "")
		return
	}

	// Discover the target
	ctx := r.Context()
	targets, err := s.discoverTargets(ctx, target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "discovery failed", err.Error())
		return
	}

	if len(targets) == 0 {
		writeError(w, http.StatusNotFound, "target not found", target)
		return
	}

	discoveredTarget := &targets[0]

	// Find the service
	var discoveredService *state.Service
	for i := range discoveredTarget.Services {
		if discoveredTarget.Services[i].Name == service {
			discoveredService = &discoveredTarget.Services[i]
			break
		}
	}

	if discoveredService == nil {
		writeError(w, http.StatusNotFound, "service not found", service)
		return
	}

	// Get last successful update from history
	if s.store == nil {
		writeError(w, http.StatusInternalServerError, "state store not configured", "")
		return
	}

	history, err := s.store.GetUpdateHistoryByService(ctx, discoveredService.ID, 1)
	if err != nil || len(history) == 0 {
		writeError(w, http.StatusNotFound, "no update history found for service", service)
		return
	}

	lastUpdate := history[0]

	// Create executor and perform rollback
	dockerClient, err := docker.NewClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Docker client", err.Error())
		return
	}
	defer func() { _ = dockerClient.Close() }()

	policyEngine := policy.NewEngine(s.logger)
	exec := executor.NewExecutor(dockerClient, policyEngine, s.store, s.logger, false).WithLockTimeout(s.cfg.LockTimeout)

	// Create a fake update result to pass to rollback
	result := &state.UpdateResult{
		TargetID:    discoveredTarget.ID,
		ServiceID:   discoveredService.ID,
		ServiceName: discoveredService.Name,
		OldDigest:   lastUpdate.OldDigest,
		NewDigest:   discoveredService.CurrentDigest,
	}

	// Execute rollback
	err = exec.ExecuteRollback(ctx, discoveredTarget, discoveredService, result)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rollback failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"target":         target,
		"service":        service,
		"rolled_back_to": lastUpdate.OldDigest[:12],
		"message":        "Successfully rolled back to previous version",
	})
}

func (s *Server) discoverTargets(ctx context.Context, target string) ([]state.Target, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	defer func() { _ = dockerClient.Close() }()

	discoverer := discovery.NewDiscoverer(s.logger, dockerClient)
	if s.store != nil {
		discoverer = discoverer.WithStore(s.store)
	}

	if target != "" {
		found, err := discoverer.DiscoverTarget(ctx, s.cfg.Root, target)
		if err != nil {
			return nil, err
		}
		return []state.Target{*found}, nil
	}

	return discoverer.Discover(ctx, s.cfg.Root)
}

func (s *Server) discoverTarget(ctx context.Context, targetID string) (*state.Target, error) {
	targets, err := s.discoverTargets(ctx, targetID)
	if err == nil && len(targets) > 0 {
		return &targets[0], nil
	}
	if s.store != nil {
		stored, storeErr := s.store.GetTarget(ctx, targetID)
		if storeErr == nil {
			return stored, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return nil, errors.New("target not found")
}

func (s *Server) getPlan(ctx context.Context, req planRequest) (*planner.Plan, error) {
	if req.Target == "" && !req.IncludeDisabled {
		if cached, ok := s.planCache.Get(); ok {
			return cached, nil
		}
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	defer func() { _ = dockerClient.Close() }()

	discoverer := discovery.NewDiscoverer(s.logger, dockerClient)
	if s.store != nil {
		discoverer = discoverer.WithStore(s.store)
	}

	registryClient := registry.NewClient(s.logger)
	policyEngine := policy.NewEngine(s.logger)

	plannerSvc := planner.NewPlanner(s.logger, discoverer, registryClient, policyEngine)
	plan, err := plannerSvc.BuildPlan(ctx, planner.PlanOptions{
		Root:            s.cfg.Root,
		TargetFilter:    req.Target,
		IncludeDisabled: req.IncludeDisabled,
	})
	if err != nil {
		return nil, err
	}

	if req.Target == "" && !req.IncludeDisabled {
		s.planCache.Set(plan)
	}

	return plan, nil
}

func (s *Server) executeApply(runID string, req applyRequest, mode string) {
	ctx := context.Background()

	logger := s.logger.WithComponent("apply")
	s.runs.AddEvent(runID, RunEvent{Level: "info", Step: "start", Message: "Apply run started"})
	s.runs.AddEvent(runID, RunEvent{Level: "info", Step: "plan", Message: "Building update plan"})

	dockerClient, err := docker.NewClient()
	if err != nil {
		s.runs.AddEvent(runID, RunEvent{Level: "error", Step: "docker", Message: "Failed to create Docker client", Data: map[string]interface{}{"error": err.Error()}})
		s.runs.Complete(runID, "failed")
		return
	}
	defer func() { _ = dockerClient.Close() }()

	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if s.store != nil {
		discoverer = discoverer.WithStore(s.store)
	}

	registryClient := registry.NewClient(logger)
	policyEngine := policy.NewEngine(logger)
	plannerSvc := planner.NewPlanner(logger, discoverer, registryClient, policyEngine)

	var plan *planner.Plan
	if req.Target == "" {
		if cached, ok := s.planCache.Get(); ok {
			plan = cached
			s.runs.AddEvent(runID, RunEvent{Level: "info", Step: "plan", Message: "Using cached plan"})
		}
	}

	if plan == nil {
		var planErr error
		plan, planErr = plannerSvc.BuildPlan(ctx, planner.PlanOptions{
			Root:            s.cfg.Root,
			TargetFilter:    req.Target,
			IncludeDisabled: false,
		})
		if planErr != nil {
			s.runs.AddEvent(runID, RunEvent{Level: "error", Step: "plan", Message: "Failed to build plan", Data: map[string]interface{}{"error": planErr.Error()}})
			s.runs.Complete(runID, "failed")
			return
		}
		if req.Target == "" {
			s.planCache.Set(plan)
		}
	}

	summary := RunSummary{}
	updateSummary := func() {
		s.runs.UpdateSummary(runID, summary)
	}

	s.runs.AddEvent(runID, RunEvent{
		Level:   "info",
		Step:    "plan",
		Message: fmt.Sprintf("Plan ready: %d updates across %d services", plan.UpdateCount, plan.ServiceCount),
		Data: map[string]interface{}{
			"target_count":  plan.TargetCount,
			"service_count": plan.ServiceCount,
			"update_count":  plan.UpdateCount,
			"allowed_count": plan.AllowedCount,
		},
	})
	if plan.UpdateCount == 0 {
		s.runs.AddEvent(runID, RunEvent{Level: "info", Step: "complete", Message: "No updates available; nothing to apply"})
		updateSummary()
		s.runs.Complete(runID, "completed")
		return
	}

	serviceFilter := make(map[string]bool)
	for _, id := range req.ServiceIDs {
		serviceFilter[id] = true
	}

	exec := executor.NewExecutor(dockerClient, policyEngine, s.store, logger, false).WithLockTimeout(s.cfg.LockTimeout)

	for _, item := range plan.Items {
		if !item.UpdateAvailable {
			continue
		}

		// Track if this service is explicitly selected
		isExplicitlySelected := false
		if mode == "selected" && len(serviceFilter) > 0 {
			if !serviceFilter[item.ServiceID] {
				continue
			}
			isExplicitlySelected = true
		}

		if mode == "safe" && item.Risk != planner.RiskSafe {
			summary.UpdatesSkipped++
			s.runs.AddEvent(runID, RunEvent{Level: "info", Target: item.TargetName, Service: item.ServiceName, Step: "skip", Message: "Skipped (not safe)"})
			updateSummary()
			continue
		}

		// Allow manual override: if user explicitly selected services, treat as forced
		forceUpdate := req.Force || isExplicitlySelected
		if !item.Allowed && !forceUpdate {
			summary.UpdatesSkipped++
			s.runs.AddEvent(runID, RunEvent{Level: "warn", Target: item.TargetName, Service: item.ServiceName, Step: "skip", Message: item.Reason})
			updateSummary()
			continue
		}

		s.runs.AddEvent(runID, RunEvent{Level: "info", Target: item.TargetName, Service: item.ServiceName, Step: "update", Message: "Applying update"})

		result := exec.ExecuteUpdate(ctx, item.Target, item.Service, item.RemoteDigest)

		if result.Success {
			summary.UpdatesApplied++
			s.runs.AddEvent(runID, RunEvent{Level: "info", Target: item.TargetName, Service: item.ServiceName, Step: "complete", Message: "Update applied"})
			updateSummary()
			continue
		}

		if executor.IsSkipError(result.Error) {
			summary.UpdatesSkipped++
			s.runs.AddEvent(runID, RunEvent{Level: "warn", Target: item.TargetName, Service: item.ServiceName, Step: "skip", Message: executor.SkipReason(result.Error)})
			updateSummary()
			continue
		}

		summary.UpdatesFailed++
		s.runs.AddEvent(runID, RunEvent{Level: "error", Target: item.TargetName, Service: item.ServiceName, Step: "failed", Message: fmt.Sprintf("Update failed: %v", result.Error)})
		updateSummary()

		if result.RollbackPerformed {
			summary.Rollbacks++
			s.runs.AddEvent(runID, RunEvent{Level: "info", Target: item.TargetName, Service: item.ServiceName, Step: "rollback", Message: "Rollback complete"})
			updateSummary()
		} else if policyEngine.ShouldRollback(ctx, result) {
			s.runs.AddEvent(runID, RunEvent{Level: "warn", Target: item.TargetName, Service: item.ServiceName, Step: "rollback", Message: "Attempting rollback"})
			if err := exec.ExecuteRollback(ctx, item.Target, item.Service, result); err != nil {
				s.runs.AddEvent(runID, RunEvent{Level: "error", Target: item.TargetName, Service: item.ServiceName, Step: "rollback", Message: fmt.Sprintf("Rollback failed: %v", err)})
			} else {
				summary.Rollbacks++
				s.runs.AddEvent(runID, RunEvent{Level: "info", Target: item.TargetName, Service: item.ServiceName, Step: "rollback", Message: "Rollback complete"})
				updateSummary()
			}
		}

		// Update-path failures without probes are not persisted by executor; store once here
		// after rollback handling so history reflects the final outcome.
		if s.store != nil && len(result.ProbeResults) == 0 {
			if err := s.store.SaveUpdateResult(ctx, result); err != nil {
				s.runs.AddEvent(runID, RunEvent{
					Level:   "warn",
					Target:  item.TargetName,
					Service: item.ServiceName,
					Step:    "history",
					Message: fmt.Sprintf("Failed to save update history: %v", err),
				})
			}
		}
	}

	s.runs.UpdateSummary(runID, summary)
	status := "completed"
	if summary.UpdatesFailed > 0 {
		status = "failed"
	}
	if len(plan.Items) == 0 {
		status = "completed"
	}
	s.runs.Complete(runID, status)
}

func (s *Server) uiHandler() http.Handler {
	fileServer := http.FileServer(http.Dir(s.cfg.DistDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)
		if strings.Contains(path, "..") {
			writeError(w, http.StatusBadRequest, "invalid path", "")
			return
		}

		fullPath := filepath.Join(s.cfg.DistDir, path)
		stat, err := os.Stat(fullPath)
		if err == nil && !stat.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexPath := filepath.Join(s.cfg.DistDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			writeError(w, http.StatusServiceUnavailable, "ui not built", "Run web build to generate dist assets")
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}

func (s *Server) activityFromRuns(limit int) []activityItem {
	events := s.runs.RecentEvents(limit)
	items := make([]activityItem, 0, len(events))
	for _, event := range events {
		action := event.Step
		if action == "" {
			action = "event"
		}
		items = append(items, activityItem{
			Timestamp: event.Timestamp,
			Action:    action,
			Target:    event.Target,
			Service:   event.Service,
			Message:   event.Message,
		})
	}
	return items
}

func activityFromHistory(history []state.UpdateResult, limit int) []activityItem {
	items := make([]activityItem, 0, limit)
	for _, result := range history {
		if len(items) >= limit {
			break
		}
		action := statusFromResult(result)
		message := "Update completed"
		if result.Error != nil {
			message = result.Error.Error()
		}
		items = append(items, activityItem{
			Timestamp: result.CompletedAt,
			Action:    action,
			Target:    result.TargetID,
			Service:   result.ServiceName,
			Message:   message,
		})
	}
	return items
}

func statusFromResult(result state.UpdateResult) string {
	if result.RollbackPerformed {
		return "rolled_back"
	}
	if result.Success {
		return "updated"
	}
	return "failed"
}

func (s *Server) getHistory(ctx context.Context, filters planner.HistoryFilter, page, pageSize int) ([]planner.HistoryItem, bool, error) {
	limit := page * pageSize
	var results []state.UpdateResult
	var err error

	switch {
	case filters.ServiceID != "":
		results, err = s.store.GetUpdateHistoryByService(ctx, filters.ServiceID, limit)
	case filters.TargetID != "":
		results, err = s.store.GetUpdateHistoryByTarget(ctx, filters.TargetID, limit)
	default:
		results, err = s.store.GetUpdateHistory(ctx, limit)
	}
	if err != nil {
		return nil, false, err
	}

	items := planner.MapHistory(results)
	filtered := planner.FilterHistory(items, filters)

	start := (page - 1) * pageSize
	if start > len(filtered) {
		return []planner.HistoryItem{}, false, nil
	}

	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	hasMore := end < len(filtered)
	return filtered[start:end], hasMore, nil
}

func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
