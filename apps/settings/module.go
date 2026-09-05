package settings

//go:generate go run ../../cmd/godino gen-types settings

import (
	"fmt"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	ptime "kaizengo/internal/platform/time"
	"kaizengo/internal/app"
	"kaizengo/internal/engine"
	"kaizengo/internal/extension"
	"kaizengo/packages/sdk-go/i18n"
	"kaizengo/packages/sdk-go/views"

	"github.com/graphql-go/graphql"
)

func init() {
	module.Register(&App{})
}

const appName = "settings"
const appVersion = "0.1.0"

// App configures platform + core settings (locale, default calendar, shell title).
type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

type snapshot struct {
	Locale          string
	Locales         []i18n.LocaleInfo
	DefaultCalendar string
	ShellTitle      string
	Calendars       []ptime.Calendar
}

func (a *App) Setup(host *module.Host) error {
	spec := app.MustAppSpec(appName)
	if spec.EnableI18n {
		app.MustLoadLocales(appName)
	}
	if _, err := engine.SetupEvents(host, appName, spec, nil); err != nil {
		return err
	}

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

	localeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "LocaleInfo",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(i18n.LocaleInfo).ID, nil
				},
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(i18n.LocaleInfo).Name, nil
				},
			},
			"dir": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return string(p.Source.(i18n.LocaleInfo).Dir), nil
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
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(localeType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(snapshot).Locales, nil
				},
			},
			"dir": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return string(i18n.Info(p.Source.(snapshot).Locale).Dir), nil
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
						"subtitle": labelField("settings.subtitle"),
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

	snap := func() snapshot {
		return snapshot{
			Locale:          i18n.Locale(),
			Locales:         i18n.LocaleInfos(),
			DefaultCalendar: config.DefaultCalendar(),
			ShellTitle:      config.ShellTitle(),
			Calendars:       ptime.List(),
		}
	}

	host.GQL.RegisterQuery("settings", &graphql.Field{
		Type: graphql.NewNonNull(settingsType),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return snap(), nil
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
				for _, info := range i18n.LocaleInfos() {
					if info.ID == v {
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
			return snap(), nil
		},
	})

	app.RegisterNavFromSpec(host, appName, spec)
	engine.RegisterPing(host, spec)

	menuType := graphql.NewObject(graphql.ObjectConfig{Name: "SettingsMenu", Fields: graphql.Fields{}})
	menuType.AddFieldConfig("id", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).ID, nil
		},
	})
	menuType.AddFieldConfig("label", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Label, nil
		},
	})
	menuType.AddFieldConfig("labelKey", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).LabelKey, nil
		},
	})
	menuType.AddFieldConfig("view", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).View, nil
		},
	})
	menuType.AddFieldConfig("route", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Route, nil
		},
	})
	menuType.AddFieldConfig("component", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Component, nil
		},
	})
	menuType.AddFieldConfig("sourceApp", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).SourceApp, nil
		},
	})
	menuType.AddFieldConfig("children", &graphql.Field{
		Type: graphql.NewList(graphql.NewNonNull(menuType)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Children, nil
		},
	})
	host.GQL.RegisterQuery("settingsMenus", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(menuType))),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return extension.BuildMenuCatalog(appName, spec.Menus), nil
		},
	})

	slotType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SettingsViewSlot",
		Fields: graphql.Fields{
			"slot": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Slot, nil
				},
			},
			"component": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Component, nil
				},
			},
			"module": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Module, nil
				},
			},
			"sourceApp": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).SourceApp, nil
				},
			},
			"id": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).ID, nil
				},
			},
			"labelKey": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).LabelKey, nil
				},
			},
			"label": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Label, nil
				},
			},
		},
	})
	host.GQL.RegisterQuery("settingsViewSlots", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(slotType))),
		Args: graphql.FieldConfigArgument{
			"view": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			view, _ := p.Args["view"].(string)
			return extension.ViewSlotsFor(appName, view), nil
		},
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
	return nil
}
