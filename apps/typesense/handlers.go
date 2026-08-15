package typesense

import (
	"kaizengo/internal/platform/search"
	"kaizengo/packages/sdk-go/extension"
)

func indexDocument(ctx extension.Context) error {
	if ctx.Record == nil || !search.ShouldIndex(ctx.App.Name, ctx.Model.Name) {
		return nil
	}
	doc, ok := search.BuildDocument(ctx.App.Name, ctx.Model.Name, ctx.OrgID, ctx.Record)
	if !ok {
		return nil
	}
	return search.Upsert(ctx.Context, doc)
}

func deleteDocument(ctx extension.Context) error {
	if ctx.RecordID == "" || !search.ShouldIndex(ctx.App.Name, ctx.Model.Name) {
		return nil
	}
	collection, _, _ := search.EffectiveIndex(ctx.App.Name, ctx.Model.Name)
	return search.Delete(ctx.Context, collection, ctx.OrgID, ctx.RecordID)
}

// queryDocuments is a named extension handler for platform.search.query.
// Actual interception is via search.UseQuery middleware registered at Setup;
// this named hook documents the extension point for app.yaml extends.
func queryDocuments(ctx extension.Context) error {
	_ = ctx
	return nil
}
