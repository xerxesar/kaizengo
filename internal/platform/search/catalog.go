package search

import (
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

// FieldOption describes an indexable model field.
type FieldOption struct {
	Name     string
	Type     string
	Selected bool
}

// ModelCatalogEntry is a searchable model discovered from app specs.
type ModelCatalogEntry struct {
	App           string
	Model         string
	Collection    string
	Enabled       bool
	Fields        []FieldOption
	Source        string // yaml | runtime
	DocumentCount int
}

func defaultCollection(app, model string, spec *appspec.SearchSpec) string {
	if spec != nil && strings.TrimSpace(spec.Collection) != "" {
		return strings.TrimSpace(spec.Collection)
	}
	return app + "." + model
}

func defaultFieldsFromSpec(m appspec.ModelSpec) []string {
	if m.Search != nil && len(m.Search.Fields) > 0 {
		return append([]string(nil), m.Search.Fields...)
	}
	out := make([]string, 0)
	for _, f := range m.Fields {
		switch strings.ToLower(f.Type) {
		case "string", "text":
			out = append(out, f.Name)
		}
	}
	return out
}

func indexableFieldOptions(m appspec.ModelSpec, selected []string) []FieldOption {
	sel := map[string]struct{}{}
	for _, f := range selected {
		sel[f] = struct{}{}
	}
	out := make([]FieldOption, 0, len(m.Fields))
	for _, f := range m.Fields {
		switch strings.ToLower(f.Type) {
		case "string", "text", "int", "bool":
			_, picked := sel[f.Name]
			out = append(out, FieldOption{
				Name:     f.Name,
				Type:     f.Type,
				Selected: picked,
			})
		}
	}
	return out
}

// EffectiveIndex returns collection, indexed fields, and whether search is enabled.
func EffectiveIndex(appName, modelName string) (collection string, fields []string, enabled bool) {
	if o, ok := GetOverride(appName, modelName); ok {
		return o.Collection, append([]string(nil), o.Fields...), o.Enabled
	}
	if d, ok := getModelDefault(appName, modelName); ok {
		return d.Collection, append([]string(nil), d.Fields...), d.Enabled
	}
	spec, err := appspec.LoadApp(appName)
	if err != nil {
		return appName + "." + modelName, nil, false
	}
	for _, m := range spec.Models {
		if m.Name != modelName {
			continue
		}
		if m.Search != nil {
			return defaultCollection(appName, modelName, m.Search),
				defaultFieldsFromSpec(m),
				true
		}
		return defaultCollection(appName, modelName, nil), nil, false
	}
	return appName + "." + modelName, nil, false
}

// Catalog builds the admin catalog for loaded apps.
func Catalog(appNames []string) []ModelCatalogEntry {
	counts := CollectionCounts()
	out := registeredCatalogEntries(counts)
	seen := map[string]struct{}{}
	for _, e := range out {
		seen[configKey(e.App, e.Model)] = struct{}{}
	}
	for _, appName := range appNames {
		spec, err := appspec.LoadApp(appName)
		if err != nil || len(spec.Models) == 0 {
			continue
		}
		for _, m := range spec.Models {
			key := configKey(appName, m.Name)
			if _, ok := seen[key]; ok {
				continue
			}
			collection, fields, enabled := EffectiveIndex(appName, m.Name)
			source := "yaml"
			if m.Search != nil {
				source = "yaml"
			}
			if _, ok := GetOverride(appName, m.Name); ok {
				source = "runtime"
			} else if m.Search == nil {
				source = "runtime"
			}
			entry := ModelCatalogEntry{
				App:           appName,
				Model:         m.Name,
				Collection:    collection,
				Enabled:       enabled,
				Fields:        indexableFieldOptions(m, fields),
				Source:        source,
				DocumentCount: counts[collection],
			}
			out = append(out, entry)
			seen[key] = struct{}{}
		}
	}
	return out
}

// DocumentCount returns indexed documents for a collection.
func DocumentCount(collection string) int {
	return CollectionCounts()[collection]
}
