package engine

import (
	"encoding/json"
	"net/http"

	"kaizengo/internal/module"
)

// NavCatalogHandler serves GET /api/apps with shell-nav ACL filtering.
func NavCatalogHandler(host *module.Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := host.NavEntries()
		filtered, err := FilterShellNav(r.Context(), host, entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(filtered)
	}
}
