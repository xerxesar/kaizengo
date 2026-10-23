package engine

import (
	"fmt"
	"strings"

	"kaizengo/internal/module"
	sdkgql "kaizengo/internal/gql"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"

	"github.com/graphql-go/graphql"
)

func registerRegisteredModelGQL(host *module.Host, spec appspec.AppSpec, m RegisteredModel) {
	model := registeredModelSpec(m)
	resource := m.Resource
	if resource == "" {
		resource = modelResource(spec.Name, m.Name)
	}
	obj := m.ObjectType

	listFieldName := listName(spec, model)
	specCRUD := sdkgql.CRUDSpec{ListName: listFieldName}

	specCRUD.ListField = &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))),
		Args: listPageArgs(),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
			if err != nil {
				return nil, err
			}
			list, err := m.List(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID})
			if err != nil {
				return nil, err
			}
			if list == nil {
				list = []any{}
			}
			opts := parseListPageOpts(p.Args)
			list = filterAnyRecords(list, opts)
			items, _ := sliceAnyPage(list, opts)
			return items, nil
		},
	}

	if m.Get != nil {
		specCRUD.GetName = getName(spec, model)
		specCRUD.GetField = &graphql.Field{
			Type: obj,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				return m.Get(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}, id)
			},
		}
	}

	if m.Create != nil {
		specCRUD.CreateName = createName(spec, model)
		specCRUD.CreateField = &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: fieldArgs(model, true),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActCreate)
				if err != nil {
					return nil, err
				}
				rec, err := m.Create(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}, p.Args)
				if err != nil {
					return nil, err
				}
				if m.ToRecord != nil {
					syncRegisteredSearchIndex(p.Context, spec.Name, m.Name, pr.OrgID, m.ToRecord(rec), false)
				}
				return rec, nil
			},
		}
	}

	if m.Update != nil {
		specCRUD.UpdateName = updateName(spec, model)
		specCRUD.UpdateField = &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: withIDArgs(fieldArgs(model, false)),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActUpdate)
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
				rec, err := m.Update(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}, id, fields)
				if err != nil {
					return nil, err
				}
				if m.ToRecord != nil {
					syncRegisteredSearchIndex(p.Context, spec.Name, m.Name, pr.OrgID, m.ToRecord(rec), false)
				}
				return rec, nil
			},
		}
	}

	if m.Delete != nil {
		specCRUD.DeleteName = deleteName(spec, model)
		specCRUD.DeleteField = &graphql.Field{
			Type: graphql.NewNonNull(graphql.Boolean),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActDelete)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				if err := m.Delete(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}, id); err != nil {
					return nil, err
				}
				return true, nil
			},
		}
	}

	sdkgql.RegisterCRUD(host.GQL, specCRUD)
	host.GQL.RegisterQuery(listFieldName+"Count", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Args: listFilterArgs(),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
			if err != nil {
				return nil, err
			}
			list, err := m.List(RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID})
			if err != nil {
				return nil, err
			}
			list = filterAnyRecords(list, parseListPageOpts(p.Args))
			return len(list), nil
		},
	})
}

func filterAnyRecords(list []any, opts ListPageOpts) []any {
	expr, err := acl.ParseDomainExpr(opts.Domain)
	if err != nil || (expr == nil && strings.TrimSpace(opts.Q) == "") {
		if err != nil {
			return list
		}
		if strings.TrimSpace(opts.Q) == "" {
			return list
		}
	}
	q := strings.ToLower(strings.TrimSpace(opts.Q))
	out := make([]any, 0, len(list))
	for _, item := range list {
		rec := anyToRecord(item)
		if rec == nil {
			out = append(out, item)
			continue
		}
		if expr != nil && !expr.Match(rec, acl.PrincipalContext{}) {
			continue
		}
		if q != "" {
			ok := false
			fields := opts.SearchIn
			if len(fields) == 0 {
				for k, v := range rec {
					if strings.Contains(strings.ToLower(fmt.Sprint(v)), q) {
						ok = true
						_ = k
						break
					}
				}
			} else {
				for _, f := range fields {
					if strings.Contains(strings.ToLower(fmt.Sprint(rec[f])), q) {
						ok = true
						break
					}
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func anyToRecord(item any) Record {
	switch t := item.(type) {
	case Record:
		return t
	case map[string]any:
		return Record(t)
	default:
		return nil
	}
}

func buildListViewFromRegistered(m RegisteredModel) views.View {
	item := views.View{
		Name:  modelListViewName(m.Name),
		Model: m.Name,
		Kind:  views.ListView,
	}
	if len(m.ListColumns) > 0 {
		item.Columns = append(item.Columns, m.ListColumns...)
	} else {
		for _, f := range m.Fields {
			item.Columns = append(item.Columns, views.Column{Key: f.Name, Label: pascal(f.Name)})
		}
		item.Columns = append(item.Columns, views.Column{Key: "updatedAt", Label: "Updated", Width: "12rem"})
	}
	return item
}

func buildFormViewFromRegistered(m RegisteredModel) views.View {
	item := views.View{
		Name:  modelFormViewName(m.Name),
		Model: m.Name,
		Kind:  views.FormView,
	}
	for _, f := range m.Fields {
        item.Fields = append(item.Fields, views.Field{
			Key: f.Name, Label: pascal(f.Name), Type: f.CanonicalType(), Required: f.Required,
			Relation: f.Relation, Inverse: f.Inverse, Values: append([]string{}, f.Values...),
		})
	}
	return item
}
