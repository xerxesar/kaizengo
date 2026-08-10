package counter

import (
	"net/http"

	"kaizengo/apps/counter/service"
	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

func init() {
	module.Register(&App{})
}

// App mounts a Svelte counter UI backed by GraphQL server state.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "counter",
		Version:     "0.1.0",
		Summary:     "Svelte counter with GraphQL server state",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	ctr := service.New()
	host.Provide(service.Name, ctr)

	host.GQL.RegisterQuery("counter", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return ctr.Value(), nil
		},
	})
	host.GQL.RegisterMutation("addCounter", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Args: graphql.FieldConfigArgument{
			"by": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			by, _ := p.Args["by"].(int)
			return ctr.Add(by), nil
		},
	})
	host.GQL.RegisterMutation("resetCounter", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return ctr.Reset(), nil
		},
	})

	host.RegisterNav(module.NavEntry{
		ID:        "counter",
		Title:     "Counter",
		Route:     "counter",
		ModuleURL: "/app-assets/counter/spa.js",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/counter/*",
		http.StripPrefix("/app-assets/counter/", http.FileServer(http.Dir("apps/counter/spa/dist"))),
	)
	return nil
}
