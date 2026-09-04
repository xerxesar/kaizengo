package engine

import (
	"context"

	"kaizengo/internal/platform/search"
	"kaizengo/packages/sdk-go/appspec"
)

func syncSearchIndex(ctx context.Context, spec appspec.AppSpec, model appspec.ModelSpec, orgID string, record Record, delete bool) {
	if !search.ShouldIndex(spec.Name, model.Name) {
		return
	}
	collection, _, _ := search.EffectiveIndex(spec.Name, model.Name)
	id, _ := record["id"].(string)
	if id == "" {
		return
	}
	if delete {
		_ = search.Delete(ctx, collection, orgID, id)
		return
	}
	doc, ok := search.BuildDocument(spec.Name, model.Name, orgID, record)
	if !ok {
		return
	}
	_ = search.Upsert(ctx, doc)
}
