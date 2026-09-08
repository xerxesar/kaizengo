package engine

import (
	"strings"

	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"
)

// RegisterAppResources catalogs every securable surface declared by an app spec.
func RegisterAppResources(spec appspec.AppSpec) {
	app := spec.Name
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindApp,
		Name:        app,
		Resource:    acl.AppResource(app),
		Label:       spec.Title,
		Description: "App-level access for " + app,
		Actions:     acl.AppActions(),
		Surface:     "app",
	})

	// Models, menus, and nav are not client ACL resources.
	// Menus/nav visibility is implied from view access.
	for _, view := range spec.Views {
		registerViewResource(app, view)
	}
	for _, view := range viewCatalog(spec) {
		registerCatalogViewResource(app, view)
	}
	trackMenuViews(app, menuCatalog(spec))

	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Ping",
		Resource: acl.QueryResource(app, camel(app)+"Ping"),
		Label: app + " health", Actions: acl.CallActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Views",
		Resource: acl.QueryResource(app, camel(app)+"Views"),
		Label: app + " view catalog", Actions: acl.CallActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Menus",
		Resource: acl.QueryResource(app, camel(app)+"Menus"),
		Label: app + " menu catalog", Actions: acl.CallActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "ViewSlots",
		Resource: acl.QueryResource(app, camel(app)+"ViewSlots"),
		Label: app + " view slots", Actions: acl.CallActions(), Surface: "graphql",
	})
}

func trackMenuViews(app string, menus []views.Menu) {
	for _, menu := range menus {
		if v := strings.TrimSpace(menu.View); v != "" {
			acl.TrackView(app, v)
			// Ensure page views referenced by menus are cataloged even when not model-derived.
			acl.Register(acl.ResourceDescriptor{
				App: app, Kind: acl.KindView, Name: v,
				Resource: acl.ViewResource(app, v), Label: v,
				Description: "Menu page view " + v, Actions: acl.CallActions(), Surface: "view",
			})
		}
		if len(menu.Children) > 0 {
			trackMenuViews(app, menu.Children)
		}
	}
}

func registerViewResource(app string, view appspec.ViewSpec) {
	name := strings.TrimSpace(view.Name)
	if name == "" {
		return
	}
	acl.TrackView(app, name)
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindView,
		Name:        name,
		Resource:    acl.ViewResource(app, name),
		Label:       name,
		Description: "Page view " + name,
		Actions:     acl.CallActions(),
		Surface:     "view",
	})
}

func registerCatalogViewResource(app string, view views.View) {
	name := strings.TrimSpace(view.Name)
	if name == "" {
		return
	}
	acl.TrackView(app, name)
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindView,
		Name:        name,
		Resource:    acl.ViewResource(app, name),
		Label:       name,
		Description: string(view.Kind) + " view for " + view.Model,
		Actions:     acl.CallActions(),
		Surface:     "view",
	})
}
