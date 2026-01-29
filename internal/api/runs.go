package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// RunEvent is a structured event emitted during plan/apply.
type RunEvent struct {
	Timestamp time.Time              `json:"ts"`
	Level     string                 `json:"level"`
	Target    string                 `json:"target,omitempty"`
	Service   string                 `json:"service,omitempty"`
	Step      string                 `json:"step,omitempty"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// RunSummary summarizes a run.
type RunSummary struct {
	UpdatesApplied int `json:"updates_applied"`
	UpdatesSkipped int `json:"updates_skipped"`
	UpdatesFailed  int `json:"updates_failed"`
	Rollbacks      int `json:"rollbacks"`
}

// Run represents an apply or plan run.
type Run struct {
	ID          string     `json:"id"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Summary     RunSummary `json:"summary"`
	Events      []RunEvent `json:"events"`
}

// RunManager stores recent runs in memory.
type RunManager struct {
	mu           sync.RWMutex
	runs         map[string]*Run
	order        []string
	maxRuns      int
	maxEvents    int
	recentEvents []RunEvent
	maxRecent    int
}

// NewRunManager creates a run manager.
func NewRunManager(maxRuns, maxEvents, maxRecent int) *RunManager {
	return &RunManager{
		runs:         make(map[string]*Run),
		order:        make([]string, 0, maxRuns),
		maxRuns:      maxRuns,
		maxEvents:    maxEvents,
		recentEvents: make([]RunEvent, 0, maxRecent),
		maxRecent:    maxRecent,
	}
}

// CreateRun creates a new run entry.
func (m *RunManager) CreateRun(mode string) *Run {
	run := &Run{
		ID:        newRunID(),
		Mode:      mode,
		Status:    "running",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
		Events:    []RunEvent{},
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.runs[run.ID] = run
	m.order = append(m.order, run.ID)
	if len(m.order) > m.maxRuns {
		oldest := m.order[0]
		delete(m.runs, oldest)
		m.order = m.order[1:]
	}

	return run
}

// AddEvent appends an event to a run.
func (m *RunManager) AddEvent(runID string, event RunEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run, ok := m.runs[runID]
	if !ok {
		return
	}
	event.Timestamp = event.Timestamp.UTC()
	run.Events = append(run.Events, event)
	if len(run.Events) > m.maxEvents {
		run.Events = run.Events[len(run.Events)-m.maxEvents:]
	}

	m.recentEvents = append(m.recentEvents, event)
	if len(m.recentEvents) > m.maxRecent {
		m.recentEvents = m.recentEvents[len(m.recentEvents)-m.maxRecent:]
	}
}

// UpdateSummary updates the run summary.
func (m *RunManager) UpdateSummary(runID string, summary RunSummary) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if run, ok := m.runs[runID]; ok {
		run.Summary = summary
	}
}

// Complete marks a run as complete.
func (m *RunManager) Complete(runID string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if run, ok := m.runs[runID]; ok {
		now := time.Now().UTC()
		run.Status = status
		run.CompletedAt = &now
	}
}

// Get returns a run by ID.
func (m *RunManager) Get(runID string) (*Run, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	run, ok := m.runs[runID]
	if !ok {
		return nil, false
	}

	// Return a shallow copy to avoid data races on events.
	clone := *run
	clone.Events = append([]RunEvent(nil), run.Events...)
	return &clone, true
}

// RecentEvents returns recent events across runs.
func (m *RunManager) RecentEvents(limit int) []RunEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.recentEvents) {
		limit = len(m.recentEvents)
	}
	start := len(m.recentEvents) - limit
	return append([]RunEvent(nil), m.recentEvents[start:]...)
}

func newRunID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
