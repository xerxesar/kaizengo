package plugin

import (
	sdkgql "kaizengo/internal/gql"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"

	"github.com/graphql-go/graphql"
)

// PluginsResource guards platformPlugins (admins hold it through the "*" seed).
const PluginsResource = "platform.api.plugins"

// Register exposes m as the platformPlugins query.
func Register(host *module.Host, m *Manager) {
	acl.Register(acl.ResourceDescriptor{
		App: "platform", Kind: acl.KindAPI, Name: "plugins", Resource: PluginsResource,
		Label: "External Go apps", Description: "List plugin processes and what they bind",
		Actions: []string{acl.ActRead}, Surface: "graphql",
	})

	strList := graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))
	pluginType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformPlugin",
		Fields: graphql.Fields{
			"name":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"version":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"path":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"apps":     &graphql.Field{Type: strList},
			"hooks":    &graphql.Field{Type: strList},
			"handlers": &graphql.Field{Type: strList},
			"error":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	host.GQL.RegisterQuery("platformPlugins", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(pluginType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, PluginsResource, acl.ActRead); err != nil {
				return nil, err
			}
			list := m.List()
			out := make([]any, 0, len(list))
			for _, info := range list {
				out = append(out, map[string]any{
					"name": info.Name, "version": info.Version, "path": info.Path,
					"apps": nonNil(info.Apps), "hooks": nonNil(info.Hooks), "handlers": nonNil(info.Handlers),
					"error": info.Error,
				})
			}
			return out, nil
		},
	})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
