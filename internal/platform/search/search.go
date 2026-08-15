package search

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Document is a normalized search document.
type Document struct {
	ID         string
	Collection string
	OrgID      string
	Title      string
	Body       string
	Fields     map[string]string
}

// Hit is a search result row.
type Hit struct {
	ID         string
	Collection string
	Title      string
	Snippet    string
	Score      float64
}

// Backend indexes and queries documents (Typesense, memory, noop, …).
type Backend interface {
	Query(ctx context.Context, orgID, q string, collections []string, limit int) ([]Hit, error)
	Upsert(ctx context.Context, doc Document) error
	Delete(ctx context.Context, collection, orgID, id string) error
}

var (
	mu      sync.RWMutex
	backend Backend = &noopBackend{}
)

// Register replaces the active search backend (e.g. Typesense driver at startup).
func Register(b Backend) {
	if b == nil {
		panic("platform/search: nil backend")
	}
	mu.Lock()
	defer mu.Unlock()
	backend = b
}

// BackendName returns a short id for the active backend (for diagnostics).
func BackendName() string {
	mu.RLock()
	defer mu.RUnlock()
	if n, ok := backend.(interface{ Name() string }); ok {
		return n.Name()
	}
	switch backend.(type) {
	case *memoryBackend:
		return "memory"
	case *noopBackend:
		return "noop"
	default:
		return fmt.Sprintf("%T", backend)
	}
}

// Query searches indexed documents for an org.
// Registered QueryMiddleware runs first (e.g. Typesense), then the active backend.
func Query(ctx context.Context, orgID, q string, collections []string, limit int) ([]Hit, error) {
	mu.RLock()
	b := backend
	mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	chain := buildQueryChain(b.Query)
	hits, err := chain(ctx, orgID, q, collections, limit)
	if err != nil {
		return nil, err
	}
	if hits == nil {
		return []Hit{}, nil
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Title < hits[j].Title
	})
	return hits, nil
}

// Upsert indexes or updates a document.
func Upsert(ctx context.Context, doc Document) error {
	mu.RLock()
	b := backend
	mu.RUnlock()
	return b.Upsert(ctx, doc)
}

// Delete removes a document from an index.
func Delete(ctx context.Context, collection, orgID, id string) error {
	mu.RLock()
	b := backend
	mu.RUnlock()
	return b.Delete(ctx, collection, orgID, id)
}
