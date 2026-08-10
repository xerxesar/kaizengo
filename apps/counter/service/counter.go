package service

import "sync"

const Name = "counter"

// Counter holds shared counter state on the server.
type Counter struct {
	mu    sync.Mutex
	value int
}

func New() *Counter { return &Counter{} }

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func (c *Counter) Add(delta int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
	return c.value
}

func (c *Counter) Reset() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
	return c.value
}
