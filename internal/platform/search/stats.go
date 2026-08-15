package search

// Counter backends expose collection document counts.
type Counter interface {
	CollectionCounts() map[string]int
}

// CollectionCounts returns document totals per collection from the active backend.
func CollectionCounts() map[string]int {
	mu.RLock()
	b := backend
	mu.RUnlock()
	if c, ok := b.(Counter); ok {
		return c.CollectionCounts()
	}
	return map[string]int{}
}
