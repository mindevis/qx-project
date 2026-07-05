package ttlcache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// Cache stores values with a fixed TTL and simple FIFO eviction.
type Cache[V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	items   map[string]entry[V]
	order   []string
	inflight map[string]*loadCall[V]
}

type loadCall[V any] struct {
	done chan struct{}
	val  V
	err  error
}

// New creates a TTL cache. maxEntries <= 0 defaults to 500.
func New[V any](ttl time.Duration, maxEntries int) *Cache[V] {
	if maxEntries <= 0 {
		maxEntries = 500
	}
	return &Cache[V]{
		ttl:      ttl,
		max:      maxEntries,
		items:    make(map[string]entry[V]),
		inflight: make(map[string]*loadCall[V]),
	}
}

func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			delete(c.items, key)
		}
		var zero V
		return zero, false
	}
	return item.value, true
}

func (c *Cache[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; !exists {
		c.order = append(c.order, key)
	}
	c.items[key] = entry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.evictLocked()
}

func (c *Cache[V]) GetOrLoad(key string, load func() (V, error)) (V, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	c.mu.Lock()
	if value, ok := c.items[key]; ok && time.Now().Before(value.expiresAt) {
		c.mu.Unlock()
		return value.value, nil
	}
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.val, call.err
	}
	call := &loadCall[V]{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	call.val, call.err = load()
	if call.err == nil {
		c.Set(key, call.val)
	}

	c.mu.Lock()
	delete(c.inflight, key)
	close(call.done)
	c.mu.Unlock()
	return call.val, call.err
}

func (c *Cache[V]) evictLocked() {
	for len(c.items) > c.max && len(c.order) > 0 {
		key := c.order[0]
		c.order = c.order[1:]
		delete(c.items, key)
	}
}
