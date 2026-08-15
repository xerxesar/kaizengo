package search

import (
	"context"
	"fmt"
	"sync"
)

// ReindexFunc rebuilds the search index for a model from its read store.
type ReindexFunc func(ctx context.Context) (int, error)

type reindexEntry struct {
	app   string
	model string
	fn    ReindexFunc
}

var (
	reindexMu sync.RWMutex
	reindexers []reindexEntry
)

// RegisterReindexer registers a handler that rebuilds search docs for app.model.
func RegisterReindexer(app, model string, fn ReindexFunc) {
	if app == "" || model == "" || fn == nil {
		return
	}
	reindexMu.Lock()
	defer reindexMu.Unlock()
	reindexers = append(reindexers, reindexEntry{app: app, model: model, fn: fn})
}

// ReindexModel runs the registered reindex handler for app.model.
func ReindexModel(ctx context.Context, app, model string) (int, error) {
	reindexMu.RLock()
	defer reindexMu.RUnlock()
	for _, e := range reindexers {
		if e.app == app && e.model == model {
			return e.fn(ctx)
		}
	}
	return 0, fmt.Errorf("no reindex handler for %s.%s", app, model)
}

// EnsureFieldIndexed adds field to runtime config when missing, enabling search if needed.
func EnsureFieldIndexed(app, model, field string) error {
	if field == "" {
		return nil
	}
	collection, fields, _ := EffectiveIndex(app, model)
	for _, f := range fields {
		if f == field {
			return nil
		}
	}
	fields = append(fields, field)
	return SetOverride(ModelIndexConfig{
		App:        app,
		Model:      model,
		Collection: collection,
		Enabled:    true,
		Fields:     fields,
	})
}
