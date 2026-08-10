package status

import (
	"net/http"

	"kaizengo/internal/module"
)

func init() {
	module.Register(&App{})
}

// App is a minimal sample SPA mounted in the core shell.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "status",
		Version:     "0.1.0",
		Summary:     "Example SPA app in the core shell",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	host.RegisterNav(module.NavEntry{
		ID:        "status",
		Title:     "Status",
		Route:     "status",
		ModuleURL: "/app-assets/status/spa.js",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/status/*",
		http.StripPrefix("/app-assets/status/", http.FileServer(http.Dir("apps/status/spa"))),
	)
	return nil
}
