package api

import (
	"testing"
	"time"
)

func TestRunManager_CreateRun(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	if run.ID == "" {
		t.Error("expected non-empty run ID")
	}
	if run.Mode != "apply" {
		t.Errorf("expected mode apply, got %s", run.Mode)
	}
	if run.Status != "running" {
		t.Errorf("expected status running, got %s", run.Status)
	}
	if len(run.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(run.Events))
	}
}

func TestRunManager_Get(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	got, ok := rm.Get(run.ID)
	if !ok {
		t.Fatal("expected to find run")
	}
	if got.ID != run.ID {
		t.Errorf("expected ID %s, got %s", run.ID, got.ID)
	}
}

func TestRunManager_GetNotFound(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	_, ok := rm.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestRunManager_AddEvent(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	rm.AddEvent(run.ID, RunEvent{
		Level:   "info",
		Step:    "start",
		Message: "test message",
	})

	got, _ := rm.Get(run.ID)
	if len(got.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got.Events))
	}
	if got.Events[0].Message != "test message" {
		t.Errorf("expected test message, got %s", got.Events[0].Message)
	}
	if got.Events[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestRunManager_AddEvent_NonexistentRun(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	// Should not panic
	rm.AddEvent("nonexistent", RunEvent{Message: "test"})
}

func TestRunManager_EventCapacity(t *testing.T) {
	rm := NewRunManager(10, 5, 50, nil)
	run := rm.CreateRun("apply")

	for i := 0; i < 10; i++ {
		rm.AddEvent(run.ID, RunEvent{
			Level:   "info",
			Message: "event",
		})
	}

	got, _ := rm.Get(run.ID)
	if len(got.Events) != 5 {
		t.Errorf("expected 5 events (capacity), got %d", len(got.Events))
	}
}

func TestRunManager_RunCapacity(t *testing.T) {
	rm := NewRunManager(3, 100, 50, nil)

	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		run := rm.CreateRun("apply")
		ids[i] = run.ID
	}

	// Oldest runs should be evicted
	_, ok := rm.Get(ids[0])
	if ok {
		t.Error("expected oldest run to be evicted")
	}
	_, ok = rm.Get(ids[1])
	if ok {
		t.Error("expected second oldest run to be evicted")
	}

	// Newest should still exist
	_, ok = rm.Get(ids[4])
	if !ok {
		t.Error("expected newest run to exist")
	}
}

func TestRunManager_Complete(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	rm.Complete(run.ID, "completed")

	got, _ := rm.Get(run.ID)
	if got.Status != "completed" {
		t.Errorf("expected status completed, got %s", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("expected non-nil completed_at")
	}
}

func TestRunManager_Complete_NonexistentRun(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	// Should not panic
	rm.Complete("nonexistent", "completed")
}

func TestRunManager_UpdateSummary(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	rm.UpdateSummary(run.ID, RunSummary{
		UpdatesApplied: 2,
		UpdatesSkipped: 1,
		UpdatesFailed:  0,
		Rollbacks:      0,
	})

	got, _ := rm.Get(run.ID)
	if got.Summary.UpdatesApplied != 2 {
		t.Errorf("expected 2 applied, got %d", got.Summary.UpdatesApplied)
	}
	if got.Summary.UpdatesSkipped != 1 {
		t.Errorf("expected 1 skipped, got %d", got.Summary.UpdatesSkipped)
	}
}

func TestRunManager_RecentEvents(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	for i := 0; i < 5; i++ {
		rm.AddEvent(run.ID, RunEvent{
			Level:   "info",
			Message: "event",
		})
	}

	events := rm.RecentEvents(3)
	if len(events) != 3 {
		t.Errorf("expected 3 recent events, got %d", len(events))
	}
}

func TestRunManager_RecentEvents_Empty(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	events := rm.RecentEvents(10)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestRunManager_RecentEventsCapacity(t *testing.T) {
	rm := NewRunManager(10, 100, 3, nil)
	run := rm.CreateRun("apply")

	for i := 0; i < 10; i++ {
		rm.AddEvent(run.ID, RunEvent{
			Level:   "info",
			Message: "event",
		})
	}

	events := rm.RecentEvents(0)
	if len(events) != 3 {
		t.Errorf("expected 3 recent events (capacity), got %d", len(events))
	}
}

func TestRunManager_GetReturnsClone(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")
	rm.AddEvent(run.ID, RunEvent{Message: "original"})

	got1, _ := rm.Get(run.ID)
	got1.Events = append(got1.Events, RunEvent{Message: "injected"})

	got2, _ := rm.Get(run.ID)
	if len(got2.Events) != 1 {
		t.Errorf("expected 1 event (mutation should not affect original), got %d", len(got2.Events))
	}
}

func TestRunManager_EventTimestamp(t *testing.T) {
	rm := NewRunManager(10, 100, 50, nil)
	run := rm.CreateRun("apply")

	// Event with explicit timestamp
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	rm.AddEvent(run.ID, RunEvent{
		Timestamp: ts,
		Message:   "timed",
	})

	got, _ := rm.Get(run.ID)
	if !got.Events[0].Timestamp.Equal(ts) {
		t.Errorf("expected timestamp %v, got %v", ts, got.Events[0].Timestamp)
	}
}
