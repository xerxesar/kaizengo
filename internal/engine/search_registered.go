package engine

import (
	"context"

	"kaizengo/internal/platform/search"
)

func syncRegisteredSearchIndex(ctx context.Context, app, model, orgID string, record map[string]any, delete bool) {
	if !search.ShouldIndex(app, model) {
		return
	}
	collection, _, _ := search.EffectiveIndex(app, model)
	if delete {
		id, _ := record["id"].(string)
		if id != "" {
			_ = search.Delete(ctx, collection, orgID, id)
		}
		return
	}
	doc, ok := search.BuildDocument(app, model, orgID, record)
	if !ok {
		return
	}
	_ = search.Upsert(ctx, doc)
}

func registerModelSearch(app string, m RegisteredModel) {
	if m.Search == nil {
		return
	}
	search.RegisterModelIndex(app, m.Name, m.Search, m.Fields)
	if m.Reindex != nil {
		search.RegisterReindexer(app, m.Name, m.Reindex)
	}
}
