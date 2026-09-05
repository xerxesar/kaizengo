package permissions

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	sdkgql "kaizengo/internal/gql"

	"github.com/graphql-go/graphql"
)

const catalogResource = "permissions.catalog"

func registerGQL(host *module.Host) {
	resourceType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SecuredResource",
		Fields: graphql.Fields{
			"app":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"kind":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"name":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"resource":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"label":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"actions":     &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
			"surface":     &graphql.Field{Type: graphql.String},
		},
	})

	acl.Register(acl.ResourceDescriptor{
		App:         "permissions",
		Kind:        acl.KindQuery,
		Name:        "resources",
		Resource:    catalogResource,
		Label:       "Resource catalog",
		Description: "List registered securable resources",
		Actions:     acl.ReadActions(),
		Surface:     "graphql",
	})

	host.GQL.RegisterQuery("resources", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(resourceType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, catalogResource, acl.ActRead); err != nil {
				return nil, err
			}
			return mapResources(acl.All()), nil
		},
	})

	host.GQL.RegisterQuery("aclActions", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, catalogResource, acl.ActRead); err != nil {
				return nil, err
			}
			return acl.StandardActions(), nil
		},
	})
}

type resourceRow struct {
	App         string   `json:"app"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Resource    string   `json:"resource"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Surface     string   `json:"surface"`
}

func mapResources(items []acl.ResourceDescriptor) []resourceRow {
	out := make([]resourceRow, 0, len(items))
	for _, item := range items {
		out = append(out, resourceRow{
			App:         item.App,
			Kind:        string(item.Kind),
			Name:        item.Name,
			Resource:    item.Resource,
			Label:       item.Label,
			Description: item.Description,
			Actions:     item.Actions,
			Surface:     item.Surface,
		})
	}
	return out
}
