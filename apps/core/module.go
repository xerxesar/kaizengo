package core

import (
	"fmt"
	"net/http"
	"strings"

	"kaizengo/apps/core/handlers"
	"kaizengo/apps/core/service"
	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

const ServiceGreeting = "core.greeting"

func init() {
	module.Register(&App{})
}

// App is the base kaizengo shell (SPA launcher + shared services).
type App struct {
	greeting *service.Greeting
}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "core",
		Version:     "0.1.0",
		Summary:     "Core SPA shell that hosts other apps",
		Depends:     nil,
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	a.greeting = service.NewGreeting()
	host.Provide(ServiceGreeting, a.greeting)

	// Apps register their own GraphQL fields on host.GQL — core only owns hello.
	host.GQL.RegisterQuery("hello", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			name, _ := p.Args["name"].(string)
			return a.greeting.Hello(name), nil
		},
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	r := host.Router

	r.Get("/", http.RedirectHandler("/app/", http.StatusFound).ServeHTTP)
	r.Get("/health", handlers.Health)

	gql, err := host.GQL.Handler()
	if err != nil {
		return fmt.Errorf("graphql schema: %w", err)
	}
	r.Handle("/graphql", gql)
	r.Get("/api/apps", module.NavHandler(host))

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Get("/favicon.ico", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "static/icon.ico")
	})

	spa := handlers.SPA("apps/core/spa/dist")
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

// GreetingFromHost looks up the core greeting service (for other apps).
func GreetingFromHost(host *module.Host) (*service.Greeting, error) {
	svc, ok := host.Lookup(ServiceGreeting)
	if !ok {
		return nil, fmt.Errorf("core greeting service not registered")
	}
	g, ok := svc.(*service.Greeting)
	if !ok {
		return nil, fmt.Errorf("core greeting service has unexpected type")
	}
	return g, nil
}
