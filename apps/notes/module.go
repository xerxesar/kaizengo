package notes

import (
	"net/http"

	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

func init() {
	module.Register(&App{})
}

// App is the notes application.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "notes",
		Version:     "0.1.0",
		Summary:     "Notes app",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	host.GQL.RegisterQuery("notesPing", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return "notes ok", nil
		},
	})
	host.RegisterNav(module.NavEntry{
		ID:        "notes",
		Title:     "Notes",
		Route:     "notes",
		ModuleURL: "/app-assets/notes/spa.js",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/notes/*",
		http.StripPrefix("/app-assets/notes/", http.FileServer(http.Dir("apps/notes/spa/dist"))),
	)
	return nil
}
