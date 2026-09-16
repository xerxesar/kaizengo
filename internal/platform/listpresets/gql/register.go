package gql

import (
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/listpresets"

	"github.com/graphql-go/graphql"
)

type presetSource struct{ listpresets.Preset }

// Register adds list view preset queries/mutations to the host GraphQL schema.
func Register(host *module.Host) {
	presetType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ListViewPreset",
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
			"predefined": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Predefined, nil
				},
			},
			"pageSize": &graphql.Field{
				Type: graphql.Int,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					v := p.Source.(presetSource).PageSize
					if v <= 0 {
						return nil, nil
					}
					return v, nil
				},
			},
			"q": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Q, nil
				},
			},
			"searchIn": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).SearchIn, nil
				},
			},
			"domain": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).Domain, nil
				},
			},
			"groupBy": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(presetSource).GroupBy, nil
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

	wrap := func(items []listpresets.Preset) []presetSource {
		out := make([]presetSource, len(items))
		for i, item := range items {
			out[i] = presetSource{item}
		}
		return out
	}

	host.GQL.RegisterQuery("listViewPresets", &graphql.Field{
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
			items, err := listpresets.List(p.Context, pr.OrgID, strings.TrimSpace(model), pr.UserID)
			if err != nil {
				return nil, err
			}
			return wrap(items), nil
		},
	})

	host.GQL.RegisterQuery("defaultListViewPreset", &graphql.Field{
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
			rec, err := listpresets.DefaultForUser(p.Context, pr.OrgID, strings.TrimSpace(model), pr.UserID)
			if err != nil || rec == nil {
				return nil, err
			}
			return presetSource{*rec}, nil
		},
	})

	saveInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "SaveListViewPresetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"id":         &graphql.InputObjectFieldConfig{Type: graphql.String},
			"model":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"name":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"shared":     &graphql.InputObjectFieldConfig{Type: graphql.Boolean},
			"isDefault":  &graphql.InputObjectFieldConfig{Type: graphql.Boolean},
			"predefined": &graphql.InputObjectFieldConfig{Type: graphql.Boolean},
			"pageSize":   &graphql.InputObjectFieldConfig{Type: graphql.Int},
			"q":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"searchIn":   &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
			"domain":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"groupBy":    &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	})

	host.GQL.RegisterMutation("saveListViewPreset", &graphql.Field{
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
			in := listpresets.SaveInput{
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
			if v, ok := raw["predefined"].(bool); ok {
				in.Predefined = v
			}
			switch v := raw["pageSize"].(type) {
			case int:
				in.PageSize = v
			case int32:
				in.PageSize = int(v)
			case int64:
				in.PageSize = int(v)
			case float64:
				in.PageSize = int(v)
			}
			if v, ok := raw["q"].(string); ok {
				in.Q = v
			}
			if v, ok := raw["domain"].(string); ok {
				in.Domain = v
			}
			in.SearchIn = stringSliceArg(raw["searchIn"])
			in.GroupBy = stringSliceArg(raw["groupBy"])
			rec, err := listpresets.Save(p.Context, in)
			if err != nil {
				return nil, err
			}
			return presetSource{rec}, nil
		},
	})

	host.GQL.RegisterMutation("deleteListViewPreset", &graphql.Field{
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
			if err := listpresets.Delete(p.Context, pr.OrgID, id, pr.UserID); err != nil {
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
