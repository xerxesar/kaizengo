package gql

import (
	"sort"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/i18n"

	"github.com/graphql-go/graphql"
)

// Register adds platform localization queries to the host GraphQL schema.
func Register(host *module.Host) {
	entryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "I18nEntry",
		Fields: graphql.Fields{
			"key": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(entry).Key, nil
				},
			},
			"value": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(entry).Value, nil
				},
			},
		},
	})

	bundleType := graphql.NewObject(graphql.ObjectConfig{
		Name: "I18nBundle",
		Fields: graphql.Fields{
			"locale": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(bundle).Locale, nil
				},
			},
			"dir": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(bundle).Dir, nil
				},
			},
			"entries": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(entryType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(bundle).Entries, nil
				},
			},
		},
	})

	host.GQL.RegisterQuery("i18n", &graphql.Field{
		Type: graphql.NewNonNull(bundleType),
		Args: graphql.FieldConfigArgument{
			"keys": &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
			},
			"prefix": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"prefixes": &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
			},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			var keys []string
			if raw, ok := p.Args["keys"].([]any); ok {
				for _, v := range raw {
					if s, ok := v.(string); ok {
						keys = append(keys, s)
					}
				}
			}
			var prefixes []string
			if prefix, _ := p.Args["prefix"].(string); prefix != "" {
				prefixes = append(prefixes, prefix)
			}
			if raw, ok := p.Args["prefixes"].([]any); ok {
				for _, v := range raw {
					if s, ok := v.(string); ok && s != "" {
						prefixes = append(prefixes, s)
					}
				}
			}
			locale, messages := i18n.Bundle(keys, prefixes...)
			info := i18n.Info(locale)
			entries := make([]entry, 0, len(messages))
			for k, v := range messages {
				entries = append(entries, entry{Key: k, Value: v})
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
			return bundle{Locale: locale, Dir: string(info.Dir), Entries: entries}, nil
		},
	})
}
type entry struct {
	Key   string
	Value string
}

type bundle struct {
	Locale  string
	Dir     string
	Entries []entry
}
