package search

import (
	"fmt"
	"strings"
)

// BuildDocument creates a search document from a projected record using effective index config.
func BuildDocument(app, model, orgID string, record map[string]any) (Document, bool) {
	collection, fields, enabled := EffectiveIndex(app, model)
	if !enabled || record == nil {
		return Document{}, false
	}
	id, _ := record["id"].(string)
	if id == "" {
		return Document{}, false
	}
	doc := Document{
		ID:         id,
		Collection: collection,
		OrgID:      orgID,
		Fields:     map[string]string{},
	}
	fieldSet := map[string]struct{}{}
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}
	var parts []string
	for key, v := range record {
		if len(fieldSet) > 0 {
			if _, ok := fieldSet[key]; !ok {
				continue
			}
		}
		if v == nil {
			continue
		}
		s := fmt.Sprint(v)
		doc.Fields[key] = s
		parts = append(parts, s)
	}
	if len(parts) > 0 {
		doc.Title = parts[0]
		if len(parts) > 1 {
			doc.Body = strings.Join(parts[1:], " ")
		}
	}
	return doc, true
}

// ShouldIndex reports whether a model has search indexing enabled.
func ShouldIndex(app, model string) bool {
	_, _, enabled := EffectiveIndex(app, model)
	return enabled
}
