package engine

import (
	"kaizengo/internal/extension"
)

func toExtensionContext(point string, hc HookContext) extension.Context {
	rec := map[string]any(nil)
	if hc.Record != nil {
		rec = map[string]any(hc.Record)
	}
	return extension.Context{
		Context:  hc.Context,
		Point:    point,
		App:      hc.App,
		Model:    hc.Model,
		OrgID:    hc.OrgID,
		UserID:   hc.UserID,
		RecordID: hc.RecordID,
		Fields:   hc.Fields,
		Record:   rec,
	}
}

func runExtensions(point string, hc HookContext, stopOnError bool) error {
	return extension.Run(point, toExtensionContext(point, hc), stopOnError)
}

func modelPoint(app, model, phase string) string {
	return extension.ModelPoint(app, model, phase)
}
