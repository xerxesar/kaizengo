package clock

import (
	"net/http"
	gotime "time"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	"kaizengo/internal/platform/i18n"
	ptime "kaizengo/internal/platform/time"

	"github.com/graphql-go/graphql"
)

func init() {
	module.Register(&App{})
}

// App is a SPA analog clock that consumes platform/time calendars.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "clock",
		Version:     "0.1.0",
		Summary:     "Realtime analog clock (uses platform calendars)",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	calendarInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CalendarInfo",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					c := p.Source.(ptime.Calendar)
					return c.ID(), nil
				},
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					c := p.Source.(ptime.Calendar)
					return c.Name(), nil
				},
			},
		},
	})

	host.GQL.RegisterQuery("clockCalendars", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(calendarInfoType))),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return ptime.List(), nil
		},
	})

	host.GQL.RegisterQuery("clockNow", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Args: graphql.FieldConfigArgument{
			"calendar": &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id, _ := p.Args["calendar"].(string)
			if id == "" {
				id = config.DefaultCalendar()
			}
			return ptime.Format(id, gotime.Now())
		},
	})

	host.GQL.RegisterQuery("clockCopy", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewObject(graphql.ObjectConfig{
			Name: "ClockCopy",
			Fields: graphql.Fields{
				"title": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(graphql.ResolveParams) (any, error) {
						return i18n.T("clock.title"), nil
					},
				},
				"subtitle": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(graphql.ResolveParams) (any, error) {
						return i18n.T("clock.subtitle"), nil
					},
				},
				"calendarLabel": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(graphql.ResolveParams) (any, error) {
						return i18n.T("clock.calendar"), nil
					},
				},
			},
		})),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return struct{}{}, nil
		},
	})

	host.RegisterNav(module.NavEntry{
		ID:        "clock",
		Title:     "Clock",
		Route:     "clock",
		ModuleURL: "/app-assets/clock/spa.js",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/clock/*",
		http.StripPrefix("/app-assets/clock/", http.FileServer(http.Dir("apps/clock/spa"))),
	)
	return nil
}
