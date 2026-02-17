package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/itsmrshow/bulwark/internal/state"
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

// RunManager stores recent runs in memory with optional SQLite write-through.
type RunManager struct {
	mu           sync.RWMutex
	runs         map[string]*Run
	order        []string
	maxRuns      int
	maxEvents    int
	recentEvents []RunEvent
	maxRecent    int
	store        state.Store
}

// NewRunManager creates a run manager with optional store for persistence.
func NewRunManager(maxRuns, maxEvents, maxRecent int, store state.Store) *RunManager {
	rm := &RunManager{
		runs:         make(map[string]*Run),
		order:        make([]string, 0, maxRuns),
		maxRuns:      maxRuns,
		maxEvents:    maxEvents,
		recentEvents: make([]RunEvent, 0, maxRecent),
		maxRecent:    maxRecent,
		store:        store,
	}

	// Load recent runs from store on startup
	if store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if runs, err := store.ListRecentRuns(ctx, maxRuns); err == nil {
			for i := len(runs) - 1; i >= 0; i-- {
				r := runs[i]
				apiRun := &Run{
					ID:          r.ID,
					Mode:        r.Mode,
					Status:      r.Status,
					CreatedAt:   r.CreatedAt,
					StartedAt:   r.StartedAt,
					CompletedAt: r.CompletedAt,
					Events:      []RunEvent{},
				}
				if r.SummaryJSON != "" {
					_ = json.Unmarshal([]byte(r.SummaryJSON), &apiRun.Summary)
				}
				// Load events
				if events, err := store.GetRunEvents(ctx, r.ID); err == nil {
					for _, e := range events {
						re := RunEvent{
							Timestamp: e.Timestamp,
							Level:     e.Level,
							Target:    e.Target,
							Service:   e.Service,
							Step:      e.Step,
							Message:   e.Message,
						}
						if e.DataJSON != "" {
							_ = json.Unmarshal([]byte(e.DataJSON), &re.Data)
						}
						apiRun.Events = append(apiRun.Events, re)
					}
				}
				rm.runs[apiRun.ID] = apiRun
				rm.order = append(rm.order, apiRun.ID)
			}
		}
	}

	return rm
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

	// Write-through to store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = m.store.SaveRun(ctx, &state.Run{
			ID:        run.ID,
			Mode:      run.Mode,
			Status:    run.Status,
			CreatedAt: run.CreatedAt,
			StartedAt: run.StartedAt,
		})
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
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}
	run.Events = append(run.Events, event)
	if len(run.Events) > m.maxEvents {
		run.Events = run.Events[len(run.Events)-m.maxEvents:]
	}

	m.recentEvents = append(m.recentEvents, event)
	if len(m.recentEvents) > m.maxRecent {
		m.recentEvents = m.recentEvents[len(m.recentEvents)-m.maxRecent:]
	}

	// Write-through to store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var dataJSON string
		if event.Data != nil {
			if b, err := json.Marshal(event.Data); err == nil {
				dataJSON = string(b)
			}
		}
		_ = m.store.SaveRunEvent(ctx, &state.RunEvent{
			RunID:     runID,
			Timestamp: event.Timestamp,
			Level:     event.Level,
			Target:    event.Target,
			Service:   event.Service,
			Step:      event.Step,
			Message:   event.Message,
			DataJSON:  dataJSON,
		})
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

	run, ok := m.runs[runID]
	if !ok {
		return
	}
	now := time.Now().UTC()
	run.Status = status
	run.CompletedAt = &now

	// Write-through to store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var summaryJSON string
		if b, err := json.Marshal(run.Summary); err == nil {
			summaryJSON = string(b)
		}
		_ = m.store.SaveRun(ctx, &state.Run{
			ID:          run.ID,
			Mode:        run.Mode,
			Status:      run.Status,
			CreatedAt:   run.CreatedAt,
			StartedAt:   run.StartedAt,
			CompletedAt: run.CompletedAt,
			SummaryJSON: summaryJSON,
		})
	}
}

// Get returns a run by ID. Falls back to store if not in memory.
func (m *RunManager) Get(runID string) (*Run, bool) {
	m.mu.RLock()
	run, ok := m.runs[runID]
	m.mu.RUnlock()

	if ok {
		// Return a shallow copy to avoid data races on events.
		clone := *run
		clone.Events = append([]RunEvent(nil), run.Events...)
		return &clone, true
	}

	// Fallback to store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		storedRun, err := m.store.GetRun(ctx, runID)
		if err != nil {
			return nil, false
		}
		apiRun := &Run{
			ID:          storedRun.ID,
			Mode:        storedRun.Mode,
			Status:      storedRun.Status,
			CreatedAt:   storedRun.CreatedAt,
			StartedAt:   storedRun.StartedAt,
			CompletedAt: storedRun.CompletedAt,
			Events:      []RunEvent{},
		}
		if storedRun.SummaryJSON != "" {
			_ = json.Unmarshal([]byte(storedRun.SummaryJSON), &apiRun.Summary)
		}
		if events, err := m.store.GetRunEvents(ctx, runID); err == nil {
			for _, e := range events {
				re := RunEvent{
					Timestamp: e.Timestamp,
					Level:     e.Level,
					Target:    e.Target,
					Service:   e.Service,
					Step:      e.Step,
					Message:   e.Message,
				}
				if e.DataJSON != "" {
					_ = json.Unmarshal([]byte(e.DataJSON), &re.Data)
				}
				apiRun.Events = append(apiRun.Events, re)
			}
		}
		return apiRun, true
	}

	return nil, false
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
