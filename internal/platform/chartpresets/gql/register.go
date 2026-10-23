package gql

import (
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/chartpresets"

	"github.com/graphql-go/graphql"
)

type presetSource struct{ chartpresets.Preset }

// Register adds chart view preset queries/mutations to the host GraphQL schema.
func Register(host *module.Host) {
	presetType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ChartViewPreset",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).ID, nil
				},
			},
			"model": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Model, nil
				},
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Name, nil
				},
			},
			"ownerId": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).OwnerID, nil
				},
			},
			"shared": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Shared, nil
				},
			},
			"isDefault": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).IsDefault, nil
				},
			},
			"type": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Type, nil
				},
			},
			"types": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Types, nil
				},
			},
			"xField": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).XField, nil
				},
			},
			"yField": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					v := p.Source.(presetSource).YField
					if v == "" {
						return nil, nil
					}
					return v, nil
				},
			},
			"seriesField": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					v := p.Source.(presetSource).SeriesField
					if v == "" {
						return nil, nil
					}
					return v, nil
				},
			},
			"measure": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Measure, nil
				},
			},
			"createdAt": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).CreatedAt.Format("2006-01-02T15:04:05Z07:00"), nil
				},
			},
			"updatedAt": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), nil
				},
			},
		},
	})

	wrap := func(items []chartpresets.Preset) []presetSource {
		out := make([]presetSource, len(items))
		for i, item := range items {
			out[i] = presetSource{item}
		}
		return out
	}

	host.GQL.RegisterQuery("chartViewPresets", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(presetType))),
		Args: graphql.FieldConfigArgument{
			"model": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := auth.MustPrincipal(p.Context)
			if err != nil {
				return nil, err
			}
			model, _ := p.Args["model"].(string)
			items, err := chartpresets.List(p.Context, pr.OrgID, strings.TrimSpace(model), pr.UserID)
			if err != nil {
				return nil, err
			}
			return wrap(items), nil
		},
	})

	host.GQL.RegisterQuery("defaultChartViewPreset", &graphql.Field{
		Type: presetType,
		Args: graphql.FieldConfigArgument{
			"model": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := auth.MustPrincipal(p.Context)
			if err != nil {
				return nil, err
			}
			model, _ := p.Args["model"].(string)
			rec, err := chartpresets.DefaultForUser(p.Context, pr.OrgID, strings.TrimSpace(model), pr.UserID)
			if err != nil || rec == nil {
				return nil, err
			}
			return presetSource{*rec}, nil
		},
	})

	saveInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "SaveChartViewPresetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"id":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"model":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"name":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"shared":      &graphql.InputObjectFieldConfig{Type: graphql.Boolean},
			"isDefault":   &graphql.InputObjectFieldConfig{Type: graphql.Boolean},
			"type":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"types":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
			"xField":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"yField":      &graphql.InputObjectFieldConfig{Type: graphql.String},
			"seriesField": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"measure":     &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	})

	host.GQL.RegisterMutation("saveChartViewPreset", &graphql.Field{
		Type: graphql.NewNonNull(presetType),
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(saveInput)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := auth.MustPrincipal(p.Context)
			if err != nil {
				return nil, err
			}
			raw, _ := p.Args["input"].(map[string]any)
			in := chartpresets.SaveInput{
				OrgID:   pr.OrgID,
				OwnerID: pr.UserID,
			}
			if v, ok := raw["id"].(string); ok {
				in.ID = v
			}
			if v, ok := raw["model"].(string); ok {
				in.Model = v
			}
			if v, ok := raw["name"].(string); ok {
				in.Name = v
			}
			if v, ok := raw["shared"].(bool); ok {
				in.Shared = v
			}
			if v, ok := raw["isDefault"].(bool); ok {
				in.IsDefault = v
			}
			if v, ok := raw["type"].(string); ok {
				in.Type = v
			}
			if v, ok := raw["xField"].(string); ok {
				in.XField = v
			}
			if v, ok := raw["yField"].(string); ok {
				in.YField = v
			}
			if v, ok := raw["seriesField"].(string); ok {
				in.SeriesField = v
			}
			if v, ok := raw["measure"].(string); ok {
				in.Measure = v
			}
			in.Types = stringSliceArg(raw["types"])
			rec, err := chartpresets.Save(p.Context, in)
			if err != nil {
				return nil, err
			}
			return presetSource{rec}, nil
		},
	})

	host.GQL.RegisterMutation("deleteChartViewPreset", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Boolean),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := auth.MustPrincipal(p.Context)
			if err != nil {
				return nil, err
			}
			id, _ := p.Args["id"].(string)
			if err := chartpresets.Delete(p.Context, pr.OrgID, id, pr.UserID); err != nil {
				return nil, err
			}
			return true, nil
		},
	})
}

func stringSliceArg(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}
