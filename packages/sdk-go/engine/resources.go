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

	for _, model := range spec.Models {
		registerModelResource(app, model)
	}
	for _, model := range registeredModels(app) {
		registerRegisteredModelResource(app, model)
	}

	registerMenuResources(app, spec.Menus)
	registerExportedMenus(app, spec.Exports.Menus)
	registerCatalogMenuResources(app, menuCatalog(spec))
	for _, view := range spec.Views {
		registerViewResource(app, view)
	}
	for _, view := range viewCatalog(spec) {
		registerCatalogViewResource(app, view)
	}

	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Ping",
		Resource: acl.QueryResource(app, camel(app)+"Ping"),
		Label: app + " health", Actions: acl.ReadActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Views",
		Resource: acl.QueryResource(app, camel(app)+"Views"),
		Label: app + " view catalog", Actions: acl.CatalogActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "Menus",
		Resource: acl.QueryResource(app, camel(app)+"Menus"),
		Label: app + " menu catalog", Actions: acl.CatalogActions(), Surface: "graphql",
	})
	acl.Register(acl.ResourceDescriptor{
		App: app, Kind: acl.KindQuery, Name: camel(app) + "ViewSlots",
		Resource: acl.QueryResource(app, camel(app)+"ViewSlots"),
		Label: app + " view slots", Actions: acl.CatalogActions(), Surface: "graphql",
	})
}

func registerModelResource(app string, model appspec.ModelSpec) {
	actions := acl.CRUDActions()
	if model.Internal {
		actions = acl.ReadActions()
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindModel,
		Name:        model.Name,
		Resource:    acl.ModelResource(app, model.Name),
		Label:       pascal(model.Name),
		Description: "Model " + model.Name,
		Actions:     actions,
		Surface:     "model",
	})
	stream := strings.TrimSpace(model.Stream)
	if stream == "" {
		stream = model.Name
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindEvent,
		Name:        model.Name,
		Resource:    acl.EventResource(app, model.Name),
		Label:       model.Name + " events",
		Description: "Event stream " + stream + " for model " + model.Name,
		Actions:     append(acl.ReadActions(), acl.ActExecute),
		Surface:     "event",
	})
}

func registerRegisteredModelResource(app string, model RegisteredModel) {
	resource := model.Resource
	if resource == "" {
		resource = acl.ModelResource(app, model.Name)
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindModel,
		Name:        model.Name,
		Resource:    resource,
		Label:       pascal(model.Name),
		Description: "Registered model " + model.Name,
		Actions:     acl.CRUDActions(),
		Surface:     "model",
	})
}

func registerExportedMenus(sourceApp string, menus []appspec.MenuExtendSpec) {
	for _, menu := range menus {
		target := strings.TrimSpace(menu.App)
		id := strings.TrimSpace(menu.ID)
		if target == "" || id == "" {
			continue
		}
		label := strings.TrimSpace(menu.Label)
		if label == "" {
			label = id
		}
		acl.Register(acl.ResourceDescriptor{
			App:         target,
			Kind:        acl.KindMenu,
			Name:        id,
			Resource:    acl.MenuResource(target, id),
			Label:       label,
			Description: "Menu exported by " + sourceApp,
			Actions:     acl.CatalogActions(),
			Surface:     "menu",
		})
	}
}

func registerCatalogMenuResources(app string, menus []views.Menu) {
	for _, menu := range menus {
		registerCatalogMenuResource(app, menu)
		if len(menu.Children) > 0 {
			registerCatalogMenuResources(app, menu.Children)
		}
	}
}

func registerCatalogMenuResource(app string, menu views.Menu) {
	id := strings.TrimSpace(menu.ID)
	if id == "" {
		return
	}
	label := strings.TrimSpace(menu.Label)
	if label == "" {
		label = id
	}
	desc := "Menu " + id
	if menu.SourceApp != "" {
		desc += " (from " + menu.SourceApp + ")"
	}
	if route := strings.TrimSpace(menu.Route); route != "" {
		desc += " route " + route
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindMenu,
		Name:        id,
		Resource:    acl.MenuResource(app, id),
		Label:       label,
		Description: desc,
		Actions:     acl.CatalogActions(),
		Surface:     "menu",
	})
}

func registerMenuResources(app string, menus []appspec.MenuSpec) {
	for _, menu := range menus {
		registerMenuResource(app, menu)
	}
}

func registerMenuResource(app string, menu appspec.MenuSpec) {
	id := strings.TrimSpace(menu.ID)
	if id == "" {
		return
	}
	label := strings.TrimSpace(menu.Label)
	if label == "" {
		label = id
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindMenu,
		Name:        id,
		Resource:    acl.MenuResource(app, id),
		Label:       label,
		Description: menuRouteDescription(menu),
		Actions:     acl.CatalogActions(),
		Surface:     "menu",
	})
	for _, child := range menu.Children {
		registerMenuResource(app, child)
	}
}

func registerViewResource(app string, view appspec.ViewSpec) {
	name := strings.TrimSpace(view.Name)
	if name == "" {
		return
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindView,
		Name:        name,
		Resource:    acl.ViewResource(app, name),
		Label:       name,
		Description: "Page view " + name,
		Actions:     acl.CatalogActions(),
		Surface:     "view",
	})
}

func registerCatalogViewResource(app string, view views.View) {
	name := strings.TrimSpace(view.Name)
	if name == "" {
		return
	}
	acl.Register(acl.ResourceDescriptor{
		App:         app,
		Kind:        acl.KindView,
		Name:        name,
		Resource:    acl.ViewResource(app, name),
		Label:       name,
		Description: string(view.Kind) + " view for " + view.Model,
		Actions:     acl.CatalogActions(),
		Surface:     "view",
	})
}

func menuRouteDescription(menu appspec.MenuSpec) string {
	route := strings.TrimSpace(menu.Route)
	view := strings.TrimSpace(menu.View)
	switch {
	case route != "" && view != "":
		return "Menu " + menu.ID + " → " + view + " (" + route + ")"
	case view != "":
		return "Menu " + menu.ID + " → " + view
	case route != "":
		return "Menu " + menu.ID + " (" + route + ")"
	default:
		return "Menu " + menu.ID
	}
}

func registerModelOperations(spec appspec.AppSpec, model appspec.ModelSpec, resource string) {
	acl.RegisterOperation(resource, acl.ActRead, "graphql", listName(spec, model))
	acl.RegisterOperation(resource, acl.ActRead, "graphql", getName(spec, model))
	if model.Internal {
		return
	}
	acl.RegisterOperation(resource, acl.ActCreate, "graphql", createName(spec, model))
	acl.RegisterOperation(resource, acl.ActUpdate, "graphql", updateName(spec, model))
	acl.RegisterOperation(resource, acl.ActDelete, "graphql", deleteName(spec, model))
}
