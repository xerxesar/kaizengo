package core

//go:generate go run ../../cmd/kaizengo gen-types core

import (
	"fmt"
	"net/http"
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	i18ngql "kaizengo/internal/platform/i18n/gql"
	chartpresetgql "kaizengo/internal/platform/chartpresets/gql"
	listpresetgql "kaizengo/internal/platform/listpresets/gql"
	searchgql "kaizengo/internal/platform/search/gql"

	"github.com/go-chi/chi/v5"
)

const appName = "core"
const appVersion = "0.1.0"

func init() {
	module.Register(engine.New(engine.Options{
		AppName: appName,
		Version: appVersion,
		Setup: func(host *module.Host, _ *engine.EventsSetup) error {
			i18ngql.Register(host)
			listpresetgql.Register(host)
			chartpresetgql.Register(host)
			searchgql.Register(host)
			return nil
		},
		Mount: mount,
	}))
}

func mount(host *module.Host) error {
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
