package api

import (
	"sync"
	"time"

	"github.com/itsmrshow/bulwark/internal/planner"
)

type planCache struct {
	mu      sync.RWMutex
	plan    *planner.Plan
	expires time.Time
	ttl     time.Duration
}

func newPlanCache(ttl time.Duration) *planCache {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	return &planCache{ttl: ttl}
}

func (c *planCache) Get() (*planner.Plan, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.plan == nil || time.Now().After(c.expires) {
		return nil, false
	}
	return c.plan, true
}

func (c *planCache) Set(plan *planner.Plan) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.plan = plan
	c.expires = time.Now().Add(c.ttl)
}
