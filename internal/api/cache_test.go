package api

import (
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/planner"
)

func TestPlanCache_DefaultTTL(t *testing.T) {
	c := newPlanCache(0)
	if c.ttl != 15*time.Second {
		t.Errorf("expected default TTL 15s, got %v", c.ttl)
	}
}

func TestPlanCache_CustomTTL(t *testing.T) {
	c := newPlanCache(30 * time.Second)
	if c.ttl != 30*time.Second {
		t.Errorf("expected TTL 30s, got %v", c.ttl)
	}
}

func TestPlanCache_GetEmpty(t *testing.T) {
	c := newPlanCache(time.Minute)
	plan, ok := c.Get()
	if ok {
		t.Error("expected miss for empty cache")
	}
	if plan != nil {
		t.Error("expected nil plan for empty cache")
	}
}

func TestPlanCache_SetAndGet(t *testing.T) {
	c := newPlanCache(time.Minute)
	p := &planner.Plan{
		TargetCount:  2,
		ServiceCount: 5,
		UpdateCount:  1,
	}

	c.Set(p)
	got, ok := c.Get()
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.TargetCount != 2 {
		t.Errorf("expected target count 2, got %d", got.TargetCount)
	}
	if got.ServiceCount != 5 {
		t.Errorf("expected service count 5, got %d", got.ServiceCount)
	}
}

func TestPlanCache_Expiry(t *testing.T) {
	c := newPlanCache(10 * time.Millisecond)
	p := &planner.Plan{UpdateCount: 3}

	c.Set(p)

	// Should be cached immediately
	_, ok := c.Get()
	if !ok {
		t.Fatal("expected cache hit before expiry")
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	_, ok = c.Get()
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestPlanCache_Overwrite(t *testing.T) {
	c := newPlanCache(time.Minute)
	c.Set(&planner.Plan{UpdateCount: 1})
	c.Set(&planner.Plan{UpdateCount: 5})

	got, ok := c.Get()
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.UpdateCount != 5 {
		t.Errorf("expected update count 5, got %d", got.UpdateCount)
	}
}
