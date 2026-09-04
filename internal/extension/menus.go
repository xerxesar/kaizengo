package extension

import (
	"sort"
	"strings"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/i18n"
	"kaizengo/packages/sdk-go/views"
)

// MenuContribution is a menu item an addon injects into another app.
type MenuContribution struct {
	SourceApp string
	TargetApp string
	Parent    string
	ID        string
	Label     string
	LabelKey  string
	View      string
	Route     string
	Order     int
	Component string
}

var (
	menuMu            sync.RWMutex
	menuContributions []MenuContribution
)

// RegisterMenuContribution records a cross-app menu injection.
func RegisterMenuContribution(sourceApp string, spec appspec.MenuExtendSpec) {
	if spec.App == "" || spec.ID == "" {
		return
	}
	if strings.TrimSpace(spec.Label) == "" && strings.TrimSpace(spec.LabelKey) == "" {
		return
	}
	if spec.View == "" && spec.Component == "" {
		return
	}
	view := strings.TrimSpace(spec.View)
	if view == "" {
		view = defaultSlotID(spec.Component)
	}
	menuMu.Lock()
	defer menuMu.Unlock()
	menuContributions = append(menuContributions, MenuContribution{
		SourceApp: sourceApp,
		TargetApp: spec.App,
		Parent:    strings.TrimSpace(spec.Parent),
		ID:        spec.ID,
		Label:     strings.TrimSpace(spec.Label),
		LabelKey:  strings.TrimSpace(spec.LabelKey),
		View:      view,
		Route:     strings.TrimSpace(spec.Route),
		Order:     spec.Order,
		Component: strings.TrimSpace(spec.Component),
	})
}

// MenuContributionsFor returns addon menu items targeting app.
func MenuContributionsFor(app string) []MenuContribution {
	menuMu.RLock()
	defer menuMu.RUnlock()
	out := make([]MenuContribution, 0)
	for _, c := range menuContributions {
		if c.TargetApp == app {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ID < out[j].ID
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// BuildMenuCatalog builds the in-app menu tree from the app's own menus
// plus any exports.menus contributions from other apps.
func BuildMenuCatalog(app string, base []appspec.MenuSpec) []views.Menu {
	tree := buildMenusFromSpec(base)
	for _, c := range MenuContributionsFor(app) {
		item := views.Menu{
			ID:        c.ID,
			Label:     resolveContribLabel(c),
			LabelKey:  c.LabelKey,
			View:      c.View,
			Route:     c.Route,
			Component: c.Component,
			SourceApp: c.SourceApp,
		}
		if item.Route == "" {
			item.Route = menuDefaultRoute(item.ID, item.View)
		}
		if c.Component != "" && item.Component == "" {
			item.Component = c.Component
		}
		if c.Parent == "" {
			tree = append(tree, item)
			continue
		}
		if !insertMenuChild(tree, c.Parent, item) {
			// Parent missing — fall back to top-level so the item is still reachable.
			tree = append(tree, item)
		}
	}
	return tree
}

func buildMenusFromSpec(items []appspec.MenuSpec) []views.Menu {
	out := make([]views.Menu, 0, len(items))
	for _, m := range items {
		item := views.Menu{
			ID:       m.ID,
			Label:    resolveSpecLabel(m),
			LabelKey: m.LabelKey,
			View:     m.View,
			Route:    m.Route,
		}
		if item.Route == "" {
			item.Route = menuDefaultRoute(m.ID, m.View)
		}
		if len(m.Children) > 0 {
			item.Children = buildMenusFromSpec(m.Children)
		}
		out = append(out, item)
	}
	return out
}

func menuDefaultRoute(id, view string) string {
	if strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return strings.TrimSpace(view)
}

func insertMenuChild(items []views.Menu, parentID string, child views.Menu) bool {
	for i := range items {
		if items[i].ID == parentID {
			items[i].Children = append(items[i].Children, child)
			return true
		}
		if insertMenuChild(items[i].Children, parentID, child) {
			return true
		}
	}
	return false
}

func resolveSpecLabel(m appspec.MenuSpec) string {
	if m.LabelKey != "" {
		if translated := i18n.T(m.LabelKey); translated != m.LabelKey {
			return translated
		}
	}
	if m.Label != "" {
		return m.Label
	}
	return m.ID
}

func resolveContribLabel(c MenuContribution) string {
	if c.LabelKey != "" {
		if translated := i18n.T(c.LabelKey); translated != c.LabelKey {
			return translated
		}
	}
	if c.Label != "" {
		return c.Label
	}
	return c.ID
}
