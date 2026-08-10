package settings

import (
	"fmt"
	"net/http"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	"kaizengo/internal/platform/i18n"
	ptime "kaizengo/internal/platform/time"

	"github.com/graphql-go/graphql"
)

func init() {
	module.Register(&App{})
}

// App configures platform + core settings (locale, default calendar, shell title).
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "settings",
		Version:     "0.1.0",
		Summary:     "Configure platform locale, calendar, and shell title",
		Depends:     []string{"core"},
		Installable: true,
	}
}

type snapshot struct {
	Locale          string
	Locales         []string
	DefaultCalendar string
	ShellTitle      string
	Calendars       []ptime.Calendar
}

func (a *App) Setup(host *module.Host) error {
	calType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SettingsCalendar",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(ptime.Calendar).ID(), nil
				},
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(ptime.Calendar).Name(), nil
				},
			},
		},
	})

	settingsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformSettings",
		Fields: graphql.Fields{
			"locale": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).Locale, nil
				},
			},
			"locales": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).Locales, nil
				},
			},
			"defaultCalendar": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).DefaultCalendar, nil
				},
			},
			"shellTitle": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).ShellTitle, nil
				},
			},
			"calendars": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(calType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).Calendars, nil
				},
			},
			"labels": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewObject(graphql.ObjectConfig{
					Name: "SettingsLabels",
					Fields: graphql.Fields{
						"title":    labelField("settings.title"),
						"locale":   labelField("settings.locale"),
						"calendar": labelField("settings.calendar"),
						"shell":    labelField("settings.shell"),
						"save":     labelField("settings.save"),
						"saved":    labelField("settings.saved"),
					},
				})),
				Resolve: func(graphql.ResolveParams) (any, error) {
					return struct{}{}, nil
				},
			},
		},
	})

	host.GQL.RegisterQuery("settings", &graphql.Field{
		Type: graphql.NewNonNull(settingsType),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return snapshot{
				Locale:          i18n.Locale(),
				Locales:         i18n.Locales(),
				DefaultCalendar: config.DefaultCalendar(),
				ShellTitle:      config.ShellTitle(),
				Calendars:       ptime.List(),
			}, nil
		},
	})

	host.GQL.RegisterMutation("updateSettings", &graphql.Field{
		Type: graphql.NewNonNull(settingsType),
		Args: graphql.FieldConfigArgument{
			"locale":          &graphql.ArgumentConfig{Type: graphql.String},
			"defaultCalendar": &graphql.ArgumentConfig{Type: graphql.String},
			"shellTitle":      &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if v, ok := p.Args["locale"].(string); ok && v != "" {
				found := false
				for _, id := range i18n.Locales() {
					if id == v {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("unknown locale %q", v)
				}
				i18n.SetLocale(v)
			}
			if v, ok := p.Args["defaultCalendar"].(string); ok && v != "" {
				if _, ok := ptime.Get(v); !ok {
					return nil, fmt.Errorf("unknown calendar %q", v)
				}
				config.SetDefaultCalendar(v)
			}
			if v, ok := p.Args["shellTitle"].(string); ok && v != "" {
				config.SetShellTitle(v)
			}
			return snapshot{
				Locale:          i18n.Locale(),
				Locales:         i18n.Locales(),
				DefaultCalendar: config.DefaultCalendar(),
				ShellTitle:      config.ShellTitle(),
				Calendars:       ptime.List(),
			}, nil
		},
	})

	host.RegisterNav(module.NavEntry{
		ID:        "settings",
		Title:     "Settings",
		Route:     "settings",
		ModuleURL: "/app-assets/settings/spa.js",
	})
	return nil
}

func labelField(key string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return i18n.T(key), nil
		},
	}
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/settings/*",
		http.StripPrefix("/app-assets/settings/", http.FileServer(http.Dir("apps/settings/spa"))),
	)
	return nil
}
