package engine

import (
	"fmt"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/extension"
	sdkgql "kaizengo/packages/sdk-go/gql"
	"kaizengo/packages/sdk-go/views"

	"github.com/graphql-go/graphql"
)

// RegisterCatalogQueries registers standard {app}Ping / Views / Menus / ViewSlots
// queries from the app spec. Used by the engine and by hybrid apps (e.g. auth).
func RegisterCatalogQueries(host *module.Host, spec appspec.AppSpec) {
	registerBasics(host, spec)
}

// RegisterPing adds the standard {app}Ping health query (ACL-guarded via identity.query.{app}Ping).
func RegisterPing(host *module.Host, spec appspec.AppSpec) {
	app := spec.Name
	queryName := camel(app) + "Ping"
	resource := acl.QueryResource(app, queryName)
	host.GQL.RegisterQuery(queryName, &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead); err != nil {
				return nil, err
			}
			return app + " ok", nil
		},
	})
}

func registerBasics(host *module.Host, spec appspec.AppSpec) {
	RegisterPing(host, spec)

	viewType := newViewType(pascal(spec.Name) + "View")
	host.GQL.RegisterQuery(camel(spec.Name)+"Views", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(viewType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			return viewCatalog(spec), nil
		},
	})

	menuType := newMenuType(pascal(spec.Name) + "Menu")
	host.GQL.RegisterQuery(camel(spec.Name)+"Menus", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(menuType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			menus := menuCatalog(spec)
			return FilterMenuCatalog(p.Context, host, spec.Name, menus)
		},
	})

	slotType := newViewSlotType(pascal(spec.Name) + "ViewSlot")
	host.GQL.RegisterQuery(camel(spec.Name)+"ViewSlots", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(slotType))),
		Args: graphql.FieldConfigArgument{
			"view": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			view, _ := p.Args["view"].(string)
			return extension.ViewSlotsFor(spec.Name, view), nil
		},
	})
}

func registerModelGQL(host *module.Host, spec appspec.AppSpec, svc *modelService) {
	obj := newRecordType(spec, svc)
	res := svc.resourceName()
	registerModelOperations(spec, svc.model, res)
	// ACL is enforced inside modelService; GraphQL only requires a session.
	crud := sdkgql.CRUDSpec{
		ListName: listName(spec, svc.model),
		ListField: &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequirePrincipal(p)
				if err != nil {
					return nil, err
				}
				list, err := svc.List(p.Context, pr.OrgID)
				if err != nil {
					return nil, err
				}
				if list == nil {
					list = []Record{}
				}
				return list, nil
			},
		},
		GetName: getName(spec, svc.model),
		GetField: &graphql.Field{
			Type: obj,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequirePrincipal(p)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				return svc.Get(p.Context, pr.OrgID, id)
			},
		},
	}
	if !svc.model.Internal {
		crud.CreateName = createName(spec, svc.model)
		crud.CreateField = &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: fieldArgs(svc.model, true),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequirePrincipal(p)
				if err != nil {
					return nil, err
				}
				return svc.Create(p.Context, pr.OrgID, pr.UserID, p.Args)
			},
		}
		crud.UpdateName = updateName(spec, svc.model)
		crud.UpdateField = &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: withIDArgs(fieldArgs(svc.model, false)),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequirePrincipal(p)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				fields := map[string]any{}
				for k, v := range p.Args {
					if k == "id" {
						continue
					}
					fields[k] = v
				}
				return svc.Update(p.Context, pr.OrgID, id, fields)
			},
		}
		crud.DeleteName = deleteName(spec, svc.model)
		crud.DeleteField = &graphql.Field{
			Type: graphql.NewNonNull(graphql.Boolean),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequirePrincipal(p)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				if err := svc.Delete(p.Context, pr.OrgID, id); err != nil {
					return nil, err
				}
				return true, nil
			},
		}
	}
	sdkgql.RegisterCRUD(host.GQL, crud)
}

func fieldArgs(model appspec.ModelSpec, requiredOnly bool) graphql.FieldConfigArgument {
	args := graphql.FieldConfigArgument{}
	for _, f := range model.Fields {
		if f.Readonly || !f.Stored() {
			continue
		}
		gqlType := gqlInputType(f)
		if requiredOnly {
			if f.Required && f.Default == nil {
				args[f.Name] = &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlType)}
			} else {
				args[f.Name] = &graphql.ArgumentConfig{Type: gqlType}
			}
		} else {
			args[f.Name] = &graphql.ArgumentConfig{Type: gqlType}
		}
	}
	return args
}

func withIDArgs(args graphql.FieldConfigArgument) graphql.FieldConfigArgument {
	out := graphql.FieldConfigArgument{
		"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	}
	for k, v := range args {
		out[k] = v
	}
	return out
}

func gqlInputType(f appspec.FieldSpec) graphql.Output {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		return graphql.Int
	case appspec.TypeNumber:
		return graphql.Float
	case appspec.TypeBool:
		return graphql.Boolean
	case appspec.TypeMany2One:
		return graphql.ID
	case appspec.TypeMany2Many, appspec.TypeOne2Many:
		return graphql.NewList(graphql.NewNonNull(graphql.ID))
	default:
		return graphql.String
	}
}

func newRecordType(spec appspec.AppSpec, svc *modelService) *graphql.Object {
	model := svc.model
	fields := graphql.Fields{
		"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID), Resolve: mapField("id")},
		"orgId":     &graphql.Field{Type: graphql.NewNonNull(graphql.ID), Resolve: mapField("orgId")},
		"authorId":  &graphql.Field{Type: graphql.NewNonNull(graphql.ID), Resolve: mapField("authorId")},
		"deleted":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean), Resolve: mapField("deleted")},
		"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String), Resolve: mapField("createdAt")},
		"updatedAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String), Resolve: mapField("updatedAt")},
	}
	for _, f := range model.Fields {
		fields[f.Name] = recordField(spec, svc, f)
	}
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   typeName(spec, model),
		Fields: fields,
	})
}

func recordField(spec appspec.AppSpec, svc *modelService, f appspec.FieldSpec) *graphql.Field {
	gqlType := gqlInputType(f)
	if f.Required || f.CanonicalType() == appspec.TypeOne2Many || f.CanonicalType() == appspec.TypeMany2Many {
		gqlType = graphql.NewNonNull(gqlType)
	}
	if f.CanonicalType() == appspec.TypeOne2Many {
		return &graphql.Field{
			Type: gqlType,
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return resolveOne2Many(p, spec, svc, f)
			},
		}
	}
	return &graphql.Field{
		Type:    gqlType,
		Resolve: mapField(f.Name),
	}
}

func resolveOne2Many(p graphql.ResolveParams, spec appspec.AppSpec, svc *modelService, f appspec.FieldSpec) (any, error) {
	rec, ok := p.Source.(Record)
	if !ok {
		if m, ok := p.Source.(map[string]any); ok {
			rec = Record(m)
		} else {
			return []string{}, nil
		}
	}
	ref, ok := f.ResolveRelation(spec.Name)
	if !ok || svc == nil {
		return []string{}, nil
	}
	orgID := str(rec["orgId"])
	id := str(rec["id"])
	var (
		rows []Record
		err  error
	)
	if ref.App == spec.Name && svc.registry != nil {
		rows, err = svc.registry.ListBy(p.Context, orgID, ref.Model, f.Inverse, id)
	} else if svc.host != nil {
		other, lookupErr := ModelsFromHost(svc.host, ref.App)
		if lookupErr != nil {
			return nil, lookupErr
		}
		rows, err = other.ListBy(p.Context, orgID, ref.Model, f.Inverse, id)
	}
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, str(row["id"]))
	}
	return ids, nil
}

func mapField(key string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		rec, ok := p.Source.(Record)
		if !ok {
			if m, ok := p.Source.(map[string]any); ok {
				rec = Record(m)
			} else {
				return nil, fmt.Errorf("unexpected source type %T", p.Source)
			}
		}
		return rec[key], nil
	}
}

func viewCatalog(spec appspec.AppSpec) []views.View {
	out := make([]views.View, 0, (len(spec.Models)+len(registeredModels(spec.Name)))*2)
	seen := map[string]struct{}{}
	for _, m := range spec.Models {
		list := buildListView(m)
		if rm, ok := registeredModelByName(spec.Name, m.Name); ok {
			if len(rm.ListColumns) > 0 {
				list = buildListViewFromRegistered(rm)
			}
		}
		out = append(out, list)
		if !m.Internal {
			form := buildFormView(m)
			if rm, ok := registeredModelByName(spec.Name, m.Name); ok {
				if len(rm.Fields) > 0 {
					form = buildFormViewFromRegistered(rm)
				}
			}
			out = append(out, form)
		}
		seen[m.Name] = struct{}{}
	}
	for _, m := range registeredModels(spec.Name) {
		if _, ok := seen[m.Name]; ok {
			continue
		}
		out = append(out, buildListViewFromRegistered(m), buildFormViewFromRegistered(m))
	}
	return out
}

func modelListViewName(model string) string {
	return pascal(model) + "List"
}

func modelFormViewName(model string) string {
	return pascal(model) + "Form"
}

func buildListView(m appspec.ModelSpec) views.View {
	item := views.View{
		Name:  modelListViewName(m.Name),
		Model: m.Name,
		Kind:  views.ListView,
	}
	for _, f := range m.Fields {
		if f.CanonicalType() == appspec.TypeOne2Many {
			continue
		}
		item.Columns = append(item.Columns, views.Column{Key: f.Name, Label: pascal(f.Name)})
	}
	item.Columns = append(item.Columns, views.Column{Key: "updatedAt", Label: "Updated", Width: "12rem"})
	return item
}

func buildFormView(m appspec.ModelSpec) views.View {
	item := views.View{
		Name:  modelFormViewName(m.Name),
		Model: m.Name,
		Kind:  views.FormView,
	}
	for _, f := range m.Fields {
		if f.CanonicalType() == appspec.TypeOne2Many {
			continue
		}
		item.Fields = append(item.Fields, views.Field{
			Key: f.Name, Label: pascal(f.Name), Type: f.CanonicalType(), Required: f.Required,
			Relation: f.Relation, Inverse: f.Inverse,
		})
	}
	return item
}

func newViewType(name string) *graphql.Object {
	columnType := graphql.NewObject(graphql.ObjectConfig{
		Name: name + "Column",
		Fields: graphql.Fields{
			"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"label":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"width":  &graphql.Field{Type: graphql.String},
			"align":  &graphql.Field{Type: graphql.String},
			"hidden": &graphql.Field{Type: graphql.Boolean},
		},
	})
	fieldType := graphql.NewObject(graphql.ObjectConfig{
		Name: name + "Field",
		Fields: graphql.Fields{
			"key":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"label":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"type":     &graphql.Field{Type: graphql.String},
			"required": &graphql.Field{Type: graphql.Boolean},
			"relation": &graphql.Field{Type: graphql.String},
			"inverse":  &graphql.Field{Type: graphql.String},
		},
	})
	return graphql.NewObject(graphql.ObjectConfig{
		Name: name,
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(views.View).Name, nil
				},
			},
			"model": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(views.View).Model, nil
				},
			},
			"kind": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return string(p.Source.(views.View).Kind), nil
				},
			},
			"columns": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(columnType)),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(views.View).Columns, nil
				},
			},
			"fields": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(fieldType)),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(views.View).Fields, nil
				},
			},
		},
	})
}

func newMenuType(name string) *graphql.Object {
	menuType := graphql.NewObject(graphql.ObjectConfig{Name: name, Fields: graphql.Fields{}})
	menuType.AddFieldConfig("id", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).ID, nil
		},
	})
	menuType.AddFieldConfig("label", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Label, nil
		},
	})
	menuType.AddFieldConfig("labelKey", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).LabelKey, nil
		},
	})
	menuType.AddFieldConfig("view", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).View, nil
		},
	})
	menuType.AddFieldConfig("route", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Route, nil
		},
	})
	menuType.AddFieldConfig("component", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Component, nil
		},
	})
	menuType.AddFieldConfig("sourceApp", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).SourceApp, nil
		},
	})
	menuType.AddFieldConfig("children", &graphql.Field{
		Type: graphql.NewList(graphql.NewNonNull(menuType)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(views.Menu).Children, nil
		},
	})
	return menuType
}

func newViewSlotType(name string) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: name,
		Fields: graphql.Fields{
			"slot": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Slot, nil
				},
			},
			"component": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Component, nil
				},
			},
			"module": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Module, nil
				},
			},
			"sourceApp": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).SourceApp, nil
				},
			},
			"id": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).ID, nil
				},
			},
			"labelKey": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).LabelKey, nil
				},
			},
			"label": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(extension.ViewSlot).Label, nil
				},
			},
		},
	})
}
