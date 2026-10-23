package appman

import (
	"fmt"
	"strings"

	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
)

const resource = "appman"

func registerAppModel(host *module.Host, spec appspec.AppSpec, mgr *engine.Manager) error {
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
	acl.RegisterOperation(resource, acl.ActRead, "graphql", "appmanApps")
	acl.RegisterOperation(resource, acl.ActCreate, "graphql", "appmanInstallApp")
	acl.RegisterOperation(resource, acl.ActExecute, "graphql", "appmanInstallApp")
	acl.RegisterOperation(resource, acl.ActUpdate, "graphql", "appmanUpgradeApp")
	acl.RegisterOperation(resource, acl.ActExecute, "graphql", "appmanUpgradeApp")

	return engine.RegisterModel(host, spec, engine.RegisteredModel{
		Name:       "app",
		Resource:   resource,
		ObjectType: appRecordType(),
		List: func(ctx engine.RequestContext) ([]any, error) {
			apps, err := mgr.Apps(ctx.Context)
			if err != nil {
				return nil, err
			}
			out := make([]any, 0, len(apps))
			for _, a := range apps {
				out = append(out, appToRecord(a))
			}
			return out, nil
		},
	})
}

func registerCommands(app *engine.App) {
	app.Command("installApp", func(c engine.HandlerCtx, args map[string]any) (any, error) {
		mgr, err := engine.ManagerFromHost(c.Host)
		if err != nil {
			return nil, err
		}
		name, _ := args["name"].(string)
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("name is required")
		}
		info, err := mgr.Install(c.Context, name)
		if err != nil {
			return nil, err
		}
		if info == nil {
			return nil, fmt.Errorf("install returned no app")
		}
		return appToRecord(*info), nil
	})
	app.Command("upgradeApp", func(c engine.HandlerCtx, args map[string]any) (any, error) {
		mgr, err := engine.ManagerFromHost(c.Host)
		if err != nil {
			return nil, err
		}
		name, _ := args["name"].(string)
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("name is required")
		}
		info, err := mgr.Upgrade(c.Context, name)
		if err != nil {
			return nil, err
		}
		if info == nil {
			return nil, fmt.Errorf("upgrade returned no app")
		}
		return appToRecord(*info), nil
	})
}

func appLane(a engine.AppInfo) string {
	if a.AutoInstall {
		return "system"
	}
	if a.Upgrade {
		return "upgrade"
	}
	if a.Installed {
		return "installed"
	}
	return "available"
}

func appToRecord(a engine.AppInfo) engine.Record {
	deps := a.Depends
	if deps == nil {
		deps = []string{}
	}
	return engine.Record{
		"id":               a.Name,
		"name":             a.Name,
		"title":            a.Title,
		"summary":          a.Summary,
		"version":          a.Version,
		"installedVersion": a.InstalledVersion,
		"installed":        a.Installed,
		"loaded":           a.Loaded,
		"autoInstall":      a.AutoInstall,
		"upgrade":          a.Upgrade,
		"status":           appLane(a),
		"depends":          strings.Join(deps, ", "),
	}
}

func appRecordType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "AppmanAppRecord",
		Fields: graphql.Fields{
			"id":               mapStringID("id"),
			"name":             mapNonNullString("name"),
			"title":            mapNonNullString("title"),
			"summary":          mapNonNullString("summary"),
			"version":          mapNonNullString("version"),
			"installedVersion": mapString("installedVersion"),
			"installed":        mapNonNullBool("installed"),
			"loaded":           mapNonNullBool("loaded"),
			"autoInstall":      mapNonNullBool("autoInstall"),
			"upgrade":          mapNonNullBool("upgrade"),
			"status":           mapNonNullString("status"),
			"depends":          mapNonNullString("depends"),
		},
	})
}

func sourceRecord(p graphql.ResolveParams) engine.Record {
	switch v := p.Source.(type) {
	case engine.Record:
		return v
	case map[string]any:
		return engine.Record(v)
	default:
		return engine.Record{}
	}
}

func mapStringID(key string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.ID),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return fmt.Sprint(sourceRecord(p)[key]), nil
		},
	}
}

func mapNonNullString(key string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return fmt.Sprint(sourceRecord(p)[key]), nil
		},
	}
}

func mapString(key string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			v := sourceRecord(p)[key]
			if v == nil || fmt.Sprint(v) == "" {
				return nil, nil
			}
			return fmt.Sprint(v), nil
		},
	}
}

func mapNonNullBool(key string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.Boolean),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			v := sourceRecord(p)[key]
			switch t := v.(type) {
			case bool:
				return t, nil
			default:
				return false, nil
			}
		},
	}
}
