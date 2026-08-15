package appspec

import (
	"fmt"
	"strings"
)

// ExtendSpec wires a named handler to an extension point (addon apps).
type ExtendSpec struct {
	Point    string `yaml:"point"`
	Handler  string `yaml:"handler"`
	Priority int    `yaml:"priority"`
}

// ViewExtendSpec injects a component into another app's view slot.
type ViewExtendSpec struct {
	App       string `yaml:"app"`
	Match     string `yaml:"match"`
	Slot      string `yaml:"slot"`
	Component string `yaml:"component"`
	ID        string `yaml:"id"`       // optional tab/menu id (defaults from component)
	LabelKey  string `yaml:"labelKey"` // optional i18n key for contributed menus/tabs
	Label     string `yaml:"label"`    // optional literal label fallback
}

// MenuExtendSpec injects a menu item into another app's menus tree.
type MenuExtendSpec struct {
	App       string `yaml:"app"`    // target app
	Parent    string `yaml:"parent"` // optional parent menu id (empty = top-level)
	ID        string `yaml:"id"`
	Label     string `yaml:"label"`
	LabelKey  string `yaml:"labelKey"`
	View      string `yaml:"view"`
	Route     string `yaml:"route"`
	Order     int    `yaml:"order"`
	Component string `yaml:"component"` // optional SPA component id
}

// ExportSpec lists SDK components an app publishes.
type ExportSpec struct {
	Components []ComponentExportSpec `yaml:"components"`
	Views      []ViewExtendSpec      `yaml:"views"`
	Menus      []MenuExtendSpec      `yaml:"menus"`
}

// ComponentExportSpec is a stable component id → module path mapping.
type ComponentExportSpec struct {
	ID     string `yaml:"id"`
	Module string `yaml:"module"`
}

func validateExtends(extends []ExtendSpec) error {
	for _, e := range extends {
		if e.Point == "" {
			return fmt.Errorf("extends entry requires point")
		}
		if e.Handler == "" {
			return fmt.Errorf("extends entry for point %q requires handler", e.Point)
		}
	}
	return nil
}

func validateExports(exports ExportSpec) error {
	seen := map[string]struct{}{}
	for _, c := range exports.Components {
		if c.ID == "" {
			return fmt.Errorf("exports.components entry requires id")
		}
		if c.Module == "" {
			return fmt.Errorf("exports.components %q requires module", c.ID)
		}
		if _, ok := seen[c.ID]; ok {
			return fmt.Errorf("duplicate component export id %q", c.ID)
		}
		seen[c.ID] = struct{}{}
	}
	for _, v := range exports.Views {
		if v.App == "" || v.Match == "" || v.Slot == "" || v.Component == "" {
			return fmt.Errorf("exports.views entry requires app, match, slot, component")
		}
	}
	seenMenus := map[string]struct{}{}
	for _, m := range exports.Menus {
		if m.App == "" {
			return fmt.Errorf("exports.menus entry requires app")
		}
		if m.ID == "" {
			return fmt.Errorf("exports.menus entry for app %q requires id", m.App)
		}
		key := m.App + "/" + m.ID
		if _, ok := seenMenus[key]; ok {
			return fmt.Errorf("duplicate exports.menus id %q for app %q", m.ID, m.App)
		}
		seenMenus[key] = struct{}{}
		if strings.TrimSpace(m.Label) == "" && strings.TrimSpace(m.LabelKey) == "" {
			return fmt.Errorf("exports.menus %q requires label or labelKey", m.ID)
		}
		if strings.TrimSpace(m.View) == "" && strings.TrimSpace(m.Component) == "" {
			return fmt.Errorf("exports.menus %q requires view or component", m.ID)
		}
	}
	return nil
}
