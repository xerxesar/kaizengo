package core

//go:generate go run ../../cmd/godino gen-types core

import (
	"fmt"
	"net/http"
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	i18ngql "kaizengo/internal/platform/i18n/gql"
	searchgql "kaizengo/internal/platform/search/gql"
	"kaizengo/internal/app"
	"kaizengo/internal/engine"

	"github.com/go-chi/chi/v5"
)

const appName = "core"
const appVersion = "0.1.0"

func init() {
	module.Register(&App{})
}

// App is the base kaizengo shell (SPA launcher + platform GraphQL fields).
type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

func (a *App) Setup(host *module.Host) error {
	spec := app.MustAppSpec(appName)
	if spec.EnableI18n {
		app.MustLoadLocales(appName)
	}

	if _, err := engine.SetupEvents(host, appName, spec, nil); err != nil {
		return err
	}

	i18ngql.Register(host)
	searchgql.Register(host)
	return nil
}

func (a *App) Mount(host *module.Host) error {
	r := host.Router

	r.Get("/", http.RedirectHandler("/app/", http.StatusFound).ServeHTTP)
	r.Get("/health", module.Health)

	gql, err := host.GQL.Handler()
	if err != nil {
		return fmt.Errorf("graphql schema: %w", err)
	}

	r.Group(func(protected chi.Router) {
		protected.Use(auth.RequireAuth)
		protected.Handle("/graphql", gql)
		protected.Get("/api/apps", engine.NavCatalogHandler(host))
		protected.Get("/api/keymap", engine.KeymapCatalogHandler(host))
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Get("/favicon.ico", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "static/icon.ico")
	})

	spa := module.SPA("apps/core/spa/dist")
	r.Handle("/app", http.RedirectHandler("/app/", http.StatusMovedPermanently))
	r.Handle("/app/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/app")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		spa.ServeHTTP(w, req)
	}))

	return nil
}
