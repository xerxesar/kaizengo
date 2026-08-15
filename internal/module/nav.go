package module

import (
	"sort"

	"kaizengo/internal/platform/i18n"
)

// NavEntry is shown in the core SPA apps dropdown.
type NavEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`    // resolved label (from TitleKey at serve time when set)
	TitleKey string `json:"-"`        // i18n key; preferred source of truth for the menu label
	Route    string `json:"route"`    // path under /app/
	Order    int    `json:"order,omitempty"`
}

// RegisterNav adds an entry to the core SPA apps menu.
// Provide TitleKey (preferred) and/or Title. TitleKey is translated when serving /api/apps.
func (h *Host) RegisterNav(e NavEntry) {
	if e.ID == "" || e.Route == "" {
		panic("module: NavEntry requires id and route")
	}
	if e.TitleKey == "" && e.Title == "" {
		panic("module: NavEntry requires titleKey or title")
	}
	if e.Title == "" {
		e.Title = i18n.T(e.TitleKey)
	}
	for _, existing := range h.nav {
		if existing.ID == e.ID {
			panic("module: duplicate nav id: " + e.ID)
		}
		if existing.Route == e.Route {
			panic("module: duplicate nav route: " + e.Route)
		}
	}
	h.nav = append(h.nav, e)
}

// NavEntries returns registered menu entries (copy) with titles resolved for the active locale.
func (h *Host) NavEntries() []NavEntry {
	out := make([]NavEntry, len(h.nav))
	copy(out, h.nav)
	for i := range out {
		if out[i].TitleKey == "" {
			continue
		}
		translated := i18n.T(out[i].TitleKey)
		if translated != out[i].TitleKey || out[i].Title == "" {
			out[i].Title = translated
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Title < out[j].Title
	})
	return out
}
