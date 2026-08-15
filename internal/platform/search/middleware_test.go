package search

import (
	"context"
	"testing"
)

func TestQueryMiddlewareIntercepts(t *testing.T) {
	prev := backend
	t.Cleanup(func() {
		mu.Lock()
		backend = prev
		mu.Unlock()
		mwMu.Lock()
		queryMiddlewares = nil
		mwMu.Unlock()
	})

	mu.Lock()
	backend = newMemoryBackend()
	mu.Unlock()
	mwMu.Lock()
	queryMiddlewares = nil
	mwMu.Unlock()

	_ = Upsert(context.Background(), Document{
		ID: "1", Collection: "demo.item", OrgID: "org", Title: "alpha", Body: "memory only",
	})

	UseQuery(func(next QueryFunc) QueryFunc {
		return func(ctx context.Context, orgID, q string, collections []string, limit int) ([]Hit, error) {
			if q == "typesense" {
				return []Hit{{ID: "ts-1", Collection: "demo.item", Title: "from typesense", Score: 10}}, nil
			}
			return next(ctx, orgID, q, collections, limit)
		}
	})

	hits, err := Query(context.Background(), "org", "typesense", []string{"demo.item"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "ts-1" {
		t.Fatalf("expected typesense middleware hit, got %#v", hits)
	}

	hits, err = Query(context.Background(), "org", "alpha", []string{"demo.item"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "1" {
		t.Fatalf("expected fallback memory hit, got %#v", hits)
	}
}
