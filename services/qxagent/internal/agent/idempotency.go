package agent

import (
	"sync"

	"github.com/qxproject/qx/pkg/protocol"
)

type requestCache struct {
	mu    sync.Mutex
	items map[string]protocol.Envelope
	order []string
	limit int
}

func newRequestCache(limit int) *requestCache {
	if limit <= 0 {
		limit = 1000
	}
	return &requestCache{items: make(map[string]protocol.Envelope), limit: limit}
}

func (c *requestCache) Get(id string) (protocol.Envelope, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	env, ok := c.items[id]
	return env, ok
}

func (c *requestCache) Set(id string, env protocol.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[id]; exists {
		c.items[id] = env
		return
	}
	for len(c.order) >= c.limit {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}
	c.order = append(c.order, id)
	c.items[id] = env
}
