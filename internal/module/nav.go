package module

// NavEntry is shown in the core SPA apps dropdown.
// Each app is loaded via dynamic import(moduleUrl) and mount(el).
type NavEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Route     string `json:"route"`     // path under /app/
	ModuleURL string `json:"moduleUrl"` // ESM URL, e.g. /app-assets/clock/spa.js
}

// RegisterNav adds an entry to the core SPA apps menu.
func (h *Host) RegisterNav(e NavEntry) {
	if e.ID == "" || e.Title == "" || e.Route == "" || e.ModuleURL == "" {
		panic("module: NavEntry requires id, title, route, and moduleUrl")
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

// NavEntries returns registered menu entries (copy).
func (h *Host) NavEntries() []NavEntry {
	out := make([]NavEntry, len(h.nav))
	copy(out, h.nav)
	return out
}
