package typesense

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/search"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
)

type configSnapshot struct {
	Backend   string
	Connected bool
	Models    []search.ModelCatalogEntry
}

func RegisterGQL(host *module.Host) {
	fieldType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchFieldOption",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.FieldOption).Name, nil
				},
			},
			"type": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.FieldOption).Type, nil
				},
			},
			"selected": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.FieldOption).Selected, nil
				},
			},
		},
	})

	modelType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchModelConfig",
		Fields: graphql.Fields{
			"app": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).App, nil
				},
			},
			"model": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).Model, nil
				},
			},
			"collection": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).Collection, nil
				},
			},
			"enabled": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).Enabled, nil
				},
			},
			"source": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).Source, nil
				},
			},
			"documentCount": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).DocumentCount, nil
				},
			},
			"fields": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(fieldType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.ModelCatalogEntry).Fields, nil
				},
			},
		},
	})

	configType := buildConfigType(modelType)

	snap := func() configSnapshot {
		names := make([]string, 0, len(host.Loaded))
		for _, m := range host.Loaded {
			names = append(names, m.Name)
		}
		return configSnapshot{
			Backend:   search.BackendName(),
			Connected: typesenseConnected(),
			Models:    search.Catalog(names),
		}
	}

	host.GQL.RegisterQuery("searchConfig", &graphql.Field{
		Type: graphql.NewNonNull(configType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return snap(), nil
		},
	})

	host.GQL.RegisterMutation("updateSearchModelConfig", &graphql.Field{
		Type: graphql.NewNonNull(configType),
		Args: graphql.FieldConfigArgument{
			"app":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"model":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"enabled": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Boolean)},
			"fields":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			appName, _ := p.Args["app"].(string)
			modelName, _ := p.Args["model"].(string)
			enabled, _ := p.Args["enabled"].(bool)
			var fields []string
			if raw, ok := p.Args["fields"].([]any); ok {
				for _, v := range raw {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						fields = append(fields, s)
					}
				}
			}
			if enabled && len(fields) == 0 {
				return nil, fmt.Errorf("select at least one field to index")
			}
			spec, err := appspec.LoadApp(appName)
			if err != nil {
				return nil, err
			}
			var modelSpec appspec.ModelSpec
			found := false
			for _, m := range spec.Models {
				if m.Name == modelName {
					modelSpec = m
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown model %q in app %q", modelName, appName)
			}
			collection, _, _ := search.EffectiveIndex(appName, modelName)
			if err := search.SetOverride(search.ModelIndexConfig{
				App:        appName,
				Model:      modelName,
				Collection: collection,
				Enabled:    enabled,
				Fields:     fields,
			}); err != nil {
				return nil, err
			}
			_ = modelSpec
			return snap(), nil
		},
	})

	reindexResultType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ReindexResult",
		Fields: graphql.Fields{
			"indexed": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(reindexResult).Indexed, nil
				},
			},
			"searchConfig": &graphql.Field{
				Type: graphql.NewNonNull(configType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(reindexResult).Config, nil
				},
			},
		},
	})

	host.GQL.RegisterMutation("reindexSearchModel", &graphql.Field{
		Type: graphql.NewNonNull(reindexResultType),
		Args: graphql.FieldConfigArgument{
			"app":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"model": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"field": &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			appName, _ := p.Args["app"].(string)
			modelName, _ := p.Args["model"].(string)
			field, _ := p.Args["field"].(string)
			field = strings.TrimSpace(field)

			if field != "" {
				if err := search.EnsureFieldIndexed(appName, modelName, field); err != nil {
					return nil, err
				}
			} else if !search.ShouldIndex(appName, modelName) {
				return nil, fmt.Errorf("search is not enabled for %s.%s", appName, modelName)
			}

			ctx := p.Context
			if ctx == nil {
				ctx = context.Background()
			}
			count, err := search.ReindexModel(ctx, appName, modelName)
			if err != nil {
				return nil, err
			}
			return reindexResult{Indexed: count, Config: snap()}, nil
		},
	})
}

type reindexResult struct {
	Indexed int
	Config  configSnapshot
}

func buildConfigType(modelType *graphql.Object) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchConfig",
		Fields: graphql.Fields{
			"backend": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(configSnapshot).Backend, nil
				},
			},
			"connected": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(configSnapshot).Connected, nil
				},
			},
			"models": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(modelType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(configSnapshot).Models, nil
				},
			},
		},
	})
}
