package engine

import (
	"encoding/json"
	"net/http"

	"kaizengo/internal/module"
	"kaizengo/internal/app"
)

// KeymapBinding is one shortcut exposed to the SPA.
type KeymapBinding struct {
	ID       string `json:"id"`
	App      string `json:"app"`
	Action   string `json:"action"`
	Keys     string `json:"keys"`
	Label    string `json:"label,omitempty"`
	LabelKey string `json:"labelKey,omitempty"`
	Scope    string `json:"scope"`
	Hint     bool   `json:"hint"`
	InForm   bool   `json:"inForm"`
}

// KeymapCatalog is the merged keymap from all loaded apps.
type KeymapCatalog struct {
	Disable  []string        `json:"disable"`
	Bindings []KeymapBinding `json:"bindings"`
}

// KeymapCatalogHandler serves GET /api/keymap with bindings from loaded app specs.
func KeymapCatalogHandler(host *module.Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		catalog := BuildKeymapCatalog(host)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(catalog)
	}
}

// BuildKeymapCatalog merges keymap seeds from every loaded app.
func BuildKeymapCatalog(host *module.Host) KeymapCatalog {
	var out KeymapCatalog
	seenDisable := map[string]struct{}{}
	seenBinding := map[string]struct{}{}

	for _, manifest := range host.Loaded {
		spec, err := app.LoadAppSpec(manifest.Name)
		if err != nil {
			continue
		}
		km := spec.Keymap
		for _, id := range km.Disable {
			key := spec.Name + "." + id
			if _, ok := seenDisable[key]; ok {
				continue
			}
			seenDisable[key] = struct{}{}
			out.Disable = append(out.Disable, key)
		}
		for _, b := range km.Bindings {
			fullID := spec.Name + "." + b.ID
			if _, ok := seenBinding[fullID]; ok {
				continue
			}
			seenBinding[fullID] = struct{}{}
			scope := b.Scope
			if scope == "" {
				scope = "global"
			}
			hint := true
			if b.Hint != nil {
				hint = *b.Hint
			}
			inForm := false
			if b.InForm != nil {
				inForm = *b.InForm
			}
			out.Bindings = append(out.Bindings, KeymapBinding{
				ID:       fullID,
				App:      spec.Name,
				Action:   b.Action,
				Keys:     b.Keys,
				Label:    b.Label,
				LabelKey: b.LabelKey,
				Scope:    scope,
				Hint:     hint,
				InForm:   inForm,
			})
		}
	}
	return out
}
