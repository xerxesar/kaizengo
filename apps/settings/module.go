package settings

//go:generate go run ../../cmd/kaizengo gen-types settings

import (
	"fmt"

	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	ptime "kaizengo/internal/platform/time"
	"kaizengo/packages/sdk-go/i18n"

	"github.com/graphql-go/graphql"
)

const appName = "settings"
const appVersion = "0.1.0"

func init() {
	module.Register(engine.New(engine.Options{
		AppName: appName,
		Version: appVersion,
		Setup:   setup,
	}))
}

type snapshot struct {
	Locale          string
	Locales         []i18n.LocaleInfo
	DefaultCalendar string
	ShellTitle      string
	Calendars       []ptime.Calendar
}

func setup(host *module.Host, _ *engine.EventsSetup) error {
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
