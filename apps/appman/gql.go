package appman

import (
	permsvc "kaizengo/apps/permissions/service"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
	sdkgql "kaizengo/packages/sdk-go/gql"

	"github.com/graphql-go/graphql"
)

const resource = "appman"

func registerGQL(host *module.Host, mgr *engine.Manager) {
	appType := newAppType()

	host.GQL.RegisterQuery("apps", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(appType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, permsvc.Name, p, resource, permsvc.ActRead); err != nil {
				return nil, err
			}
			return mgr.Apps(p.Context)
		},
	})
	host.GQL.RegisterMutation("installApp", &graphql.Field{
		Type: graphql.NewNonNull(appType),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, permsvc.Name, p, resource, permsvc.ActCreate); err != nil {
				return nil, err
			}
			name, _ := p.Args["name"].(string)
			return mgr.Install(p.Context, name)
		},
	})
	host.GQL.RegisterMutation("upgradeApp", &graphql.Field{
		Type: graphql.NewNonNull(appType),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, permsvc.Name, p, resource, permsvc.ActUpdate); err != nil {
				return nil, err
			}
			name, _ := p.Args["name"].(string)
			return mgr.Upgrade(p.Context, name)
		},
	})
}

func newAppType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "App",
		Fields: graphql.Fields{
			"name":             nonNullString(func(a engine.AppInfo) string { return a.Name }),
			"title":            nonNullString(func(a engine.AppInfo) string { return a.Title }),
			"summary":          nonNullString(func(a engine.AppInfo) string { return a.Summary }),
			"version":          nonNullString(func(a engine.AppInfo) string { return a.Version }),
			"installedVersion": stringField(func(a engine.AppInfo) string { return a.InstalledVersion }),
			"installed":        nonNullBool(func(a engine.AppInfo) bool { return a.Installed }),
			"loaded":           nonNullBool(func(a engine.AppInfo) bool { return a.Loaded }),
			"autoInstall":      nonNullBool(func(a engine.AppInfo) bool { return a.AutoInstall }),
			"upgrade":          nonNullBool(func(a engine.AppInfo) bool { return a.Upgrade }),
			"depends": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					deps := sourceApp(p).Depends
					if deps == nil {
						deps = []string{}
					}
					return deps, nil
				},
			},
		},
	})
}

func sourceApp(p graphql.ResolveParams) engine.AppInfo {
	switch v := p.Source.(type) {
	case engine.AppInfo:
		return v
	case *engine.AppInfo:
		return *v
	default:
		return engine.AppInfo{}
	}
}

func nonNullString(fn func(engine.AppInfo) string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return fn(sourceApp(p)), nil
		},
	}
}

func stringField(fn func(engine.AppInfo) string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			v := fn(sourceApp(p))
			if v == "" {
				return nil, nil
			}
			return v, nil
		},
	}
}

func nonNullBool(fn func(engine.AppInfo) bool) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.Boolean),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return fn(sourceApp(p)), nil
		},
	}
}
