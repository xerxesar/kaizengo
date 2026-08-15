package search

import (
	"context"
	"strings"
	"sync"
)

type memKey struct {
	collection string
	orgID      string
	id         string
}

type memoryBackend struct {
	mu   sync.RWMutex
	docs map[memKey]Document
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{docs: map[memKey]Document{}}
}

// NewMemoryBackend returns an in-memory search backend (for tests and Typesense fallback).
func NewMemoryBackend() Backend {
	return newMemoryBackend()
}

func (m *memoryBackend) Query(_ context.Context, orgID, q string, collections []string, limit int) ([]Hit, error) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return []Hit{}, nil
	}
	allowed := map[string]struct{}{}
	for _, c := range collections {
		c = strings.TrimSpace(c)
		if c != "" {
			allowed[c] = struct{}{}
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Hit, 0)
	for k, doc := range m.docs {
		if doc.OrgID != orgID {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[k.collection]; !ok {
				continue
			}
		}
		text := strings.ToLower(doc.Title + " " + doc.Body)
		for _, v := range doc.Fields {
			text += " " + strings.ToLower(v)
		}
		if !strings.Contains(text, q) {
			continue
		}
		snippet := doc.Title
		if doc.Body != "" {
			snippet = doc.Title + " — " + truncate(doc.Body, 120)
		}
		out = append(out, Hit{
			ID:         doc.ID,
			Collection: doc.Collection,
			Title:      doc.Title,
			Snippet:    snippet,
			Score:      1,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memoryBackend) Upsert(_ context.Context, doc Document) error {
	if doc.ID == "" || doc.Collection == "" || doc.OrgID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs[memKey{doc.Collection, doc.OrgID, doc.ID}] = doc
	return nil
}

func (m *memoryBackend) Delete(_ context.Context, collection, orgID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.docs, memKey{collection, orgID, id})
	return nil
}

func (m *memoryBackend) CollectionCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]int{}
	for k := range m.docs {
		out[k.collection]++
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
