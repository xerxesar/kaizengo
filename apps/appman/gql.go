package appman

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/internal/engine"
	sdkgql "kaizengo/internal/gql"

	"github.com/graphql-go/graphql"
)

const resource = "appman"

func registerGQL(host *module.Host, mgr *engine.Manager) {
	appType := newAppType()

	acl.Register(acl.ResourceDescriptor{
		App:         "appman",
		Kind:        acl.KindApp,
		Name:        "appman",
		Resource:    resource,
		Label:       "App Manager",
		Description: "Install and upgrade platform apps",
		Actions:     acl.AppActions(),
		Surface:     "graphql",
	})
	acl.RegisterOperation(resource, acl.ActRead, "graphql", "apps")
	acl.RegisterOperation(resource, acl.ActCreate, "graphql", "installApp")
	acl.RegisterOperation(resource, acl.ActExecute, "graphql", "installApp")
	acl.RegisterOperation(resource, acl.ActUpdate, "graphql", "upgradeApp")
	acl.RegisterOperation(resource, acl.ActExecute, "graphql", "upgradeApp")

	host.GQL.RegisterQuery("apps", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(appType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead); err != nil {
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
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActCreate); err != nil {
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
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActUpdate); err != nil {
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
