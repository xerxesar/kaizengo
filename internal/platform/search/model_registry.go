package search

import (
	"strings"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
)

var (
	defaultsMu       sync.RWMutex
	modelDefaults    = map[string]ModelIndexConfig{}
	modelFieldSpecs  = map[string][]appspec.FieldSpec{}
)

// RegisterModelIndex declares default search settings for a code-registered model.
func RegisterModelIndex(app, model string, spec *appspec.SearchSpec, fields []appspec.FieldSpec) {
	if spec == nil {
		return
	}
	key := configKey(app, model)
	indexed := append([]string(nil), spec.Fields...)
	if len(indexed) == 0 {
		for _, f := range fields {
			switch strings.ToLower(f.Type) {
			case "string", "text":
				indexed = append(indexed, f.Name)
			}
		}
	}
	defaultsMu.Lock()
	defer defaultsMu.Unlock()
	modelDefaults[key] = ModelIndexConfig{
		App:        app,
		Model:      model,
		Collection: defaultCollection(app, model, spec),
		Enabled:    true,
		Fields:     indexed,
	}
	modelFieldSpecs[key] = append([]appspec.FieldSpec(nil), fields...)
}

func getModelDefault(app, model string) (ModelIndexConfig, bool) {
	defaultsMu.RLock()
	defer defaultsMu.RUnlock()
	cfg, ok := modelDefaults[configKey(app, model)]
	return cfg, ok
}

func registeredCatalogEntries(counts map[string]int) []ModelCatalogEntry {
	defaultsMu.RLock()
	defer defaultsMu.RUnlock()
	out := make([]ModelCatalogEntry, 0, len(modelDefaults))
	for key, cfg := range modelDefaults {
		if _, overridden := overrides[key]; overridden {
			continue
		}
		fields := append([]appspec.FieldSpec(nil), modelFieldSpecs[key]...)
		out = append(out, ModelCatalogEntry{
			App:           cfg.App,
			Model:         cfg.Model,
			Collection:    cfg.Collection,
			Enabled:       cfg.Enabled,
			Fields:        indexableFieldOptions(appspec.ModelSpec{Fields: fields}, cfg.Fields),
			Source:        "registered",
			DocumentCount: counts[cfg.Collection],
		})
	}
	return out
}
