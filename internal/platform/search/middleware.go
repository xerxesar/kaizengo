package search

import (
	"context"
	"sync"
)

// QueryFunc is the core search query signature used by backends and middleware.
type QueryFunc func(ctx context.Context, orgID, q string, collections []string, limit int) ([]Hit, error)

// QueryMiddleware wraps a QueryFunc (onion style: outer middleware runs first).
// Call next to fall through to the next middleware or the active backend.
type QueryMiddleware func(next QueryFunc) QueryFunc

var (
	mwMu         sync.RWMutex
	queryMiddlewares []QueryMiddleware
)

// UseQuery registers middleware that sits between callers and the search backend.
// Middleware is applied in registration order (first registered = outermost).
func UseQuery(mw QueryMiddleware) {
	if mw == nil {
		return
	}
	mwMu.Lock()
	defer mwMu.Unlock()
	queryMiddlewares = append(queryMiddlewares, mw)
}

func buildQueryChain(backend QueryFunc) QueryFunc {
	mwMu.RLock()
	mws := append([]QueryMiddleware(nil), queryMiddlewares...)
	mwMu.RUnlock()
	h := backend
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
