package engine

import (
	"fmt"
	"sort"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
	sdkgql "kaizengo/internal/gql"

	"github.com/graphql-go/graphql"
)

func registerCQRS(host *module.Host, spec appspec.AppSpec, models *ModelRegistry, handlers *handlerRegistry) error {
	objs := map[string]*graphql.Object{}
	for _, m := range spec.Models {
		if m.Virtual {
			obj, err := virtualModelObject(spec, m)
			if err != nil {
				return err
			}
			objs[m.Name] = obj
			continue
		}
		svc, err := findModelService(models, m.Name)
		if err != nil {
			return err
		}
		objs[m.Name] = newRecordType(spec, svc)
	}

	for _, q := range spec.Queries {
		fieldName := cqrsFieldName(spec.Name, q.Name)
		resource := acl.QueryResource(spec.Name, fieldName)
		field, err := buildQueryField(host, spec, models, handlers, objs, q, resource)
		if err != nil {
			return err
		}
		// Catalog fields = GraphQL return-type fields (selection set).
		acl.Register(acl.ResourceDescriptor{
			App: spec.Name, Kind: acl.KindQuery, Name: fieldName,
			Resource: resource, Label: q.Name, Actions: acl.CallActions(),
			Fields: graphqlOutputFieldNames(field.Type), Surface: "graphql",
		})
		host.GQL.RegisterQuery(fieldName, field)
		acl.RegisterOperation(resource, acl.ActRead, "graphql", fieldName)

		if model := strings.TrimSpace(q.List); model != "" {
			modelName := model // capture for closures
			countName := fieldName + "Count"
			countResource := acl.QueryResource(spec.Name, countName)
			host.GQL.RegisterQuery(countName, &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Args: listFilterArgs(),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
					if err != nil {
						return nil, err
					}
					rc := RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}
					page, err := listCQRSPage(spec.Name, models, modelName, rc, parseListPageOpts(p.Args))
					if err != nil {
						return nil, err
					}
					return page.Total, nil
				},
			})
			acl.Register(acl.ResourceDescriptor{
				App: spec.Name, Kind: acl.KindQuery, Name: countName,
				Resource: countResource, Label: q.Name + " count", Actions: acl.CallActions(),
				Surface: "graphql",
			})
			acl.RegisterOperation(countResource, acl.ActRead, "graphql", countName)

			groupsName := fieldName + "Groups"
			groupsResource := acl.QueryResource(spec.Name, groupsName)
			host.GQL.RegisterQuery(groupsName, &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(groupBucketType(spec.Name)))),
				Args: listPageArgs(),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
					if err != nil {
						return nil, err
					}
					rc := RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}
					return groupsCQRS(spec.Name, models, modelName, rc, parseListPageOpts(p.Args))
				},
			})
			acl.Register(acl.ResourceDescriptor{
				App: spec.Name, Kind: acl.KindQuery, Name: groupsName,
				Resource: groupsResource, Label: q.Name + " groups", Actions: acl.CallActions(),
				Surface: "graphql",
			})
			acl.RegisterOperation(groupsResource, acl.ActRead, "graphql", groupsName)
		}
	}

	for _, c := range spec.Commands {
		fieldName := cqrsFieldName(spec.Name, c.Name)
		resource := acl.CommandResource(spec.Name, fieldName)
		field, err := buildCommandField(host, spec, models, handlers, objs, c, resource)
		if err != nil {
			return err
		}
		// Catalog fields = GraphQL input arguments for the mutation.
		acl.Register(acl.ResourceDescriptor{
			App: spec.Name, Kind: acl.KindCommand, Name: fieldName,
			Resource: resource, Label: c.Name, Actions: acl.CallActions(),
			Fields: graphqlArgNames(field.Args), Surface: "graphql",
		})
		host.GQL.RegisterMutation(fieldName, field)
		acl.RegisterOperation(resource, acl.ActExecute, "graphql", fieldName)
	}
	return nil
}

// graphqlOutputFieldNames returns object field names for a GraphQL output type
// (unwrapping NonNull / List). Scalars and other non-object types yield nil.
func graphqlOutputFieldNames(t graphql.Output) []string {
	for t != nil {
		switch typed := t.(type) {
		case *graphql.NonNull:
			t = typed.OfType
		case *graphql.List:
			t = typed.OfType
		case *graphql.Object:
			fields := typed.Fields()
			out := make([]string, 0, len(fields))
			for name := range fields {
				if n := strings.TrimSpace(name); n != "" {
					out = append(out, n)
				}
			}
			sort.Strings(out)
			return out
		default:
			return nil
		}
	}
	return nil
}

func graphqlArgNames(args graphql.FieldConfigArgument) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for name := range args {
		if n := strings.TrimSpace(name); n != "" {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func buildQueryField(
	host *module.Host,
	spec appspec.AppSpec,
	models *ModelRegistry,
	handlers *handlerRegistry,
	objs map[string]*graphql.Object,
	q appspec.QuerySpec,
	resource string,
) (*graphql.Field, error) {
	if model := strings.TrimSpace(q.List); model != "" {
		obj := objs[model]
		if obj == nil {
			return nil, fmt.Errorf("query %q: missing type for model %q", q.Name, model)
		}
		return &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))),
			Args: listPageArgs(),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
				if err != nil {
					return nil, err
				}
				rc := RequestContext{Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID}
				page, err := listCQRSPage(spec.Name, models, model, rc, parseListPageOpts(p.Args))
				if err != nil {
					return nil, err
				}
				if page.Items == nil {
					page.Items = []Record{}
				}
				return page.Items, nil
			},
		}, nil
	}
	if model := strings.TrimSpace(q.Get); model != "" {
		obj := objs[model]
		if obj == nil {
			return nil, fmt.Errorf("query %q: missing type for model %q", q.Name, model)
		}
		return &graphql.Field{
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
				return models.Get(p.Context, pr.OrgID, model, id)
			},
		}, nil
	}

	fn := handlers.query(q.Name)
	if fn == nil {
		return nil, fmt.Errorf("query %q: no list/get shorthand and no Go handler registered", q.Name)
	}
	outType, err := cqrsReturnType(spec, objs, q.Returns, q.Args)
	if err != nil {
		return nil, fmt.Errorf("query %q: %w", q.Name, err)
	}
	return &graphql.Field{
		Type: outType,
		Args: fieldSpecsArgs(q.Args, true),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
			if err != nil {
				return nil, err
			}
			return fn(HandlerCtx{
				Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID,
				Models: models, Host: host, Spec: spec,
			}, p.Args)
		},
	}, nil
}

func buildCommandField(
	host *module.Host,
	spec appspec.AppSpec,
	models *ModelRegistry,
	handlers *handlerRegistry,
	objs map[string]*graphql.Object,
	c appspec.CommandSpec,
	resource string,
) (*graphql.Field, error) {
	if model := strings.TrimSpace(c.Create); model != "" {
		obj := objs[model]
		m, ok := modelByName(spec, model)
		if !ok || obj == nil {
			return nil, fmt.Errorf("command %q: unknown model %q", c.Name, model)
		}
		return &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: fieldArgs(m, true),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActExecute)
				if err != nil {
					return nil, err
				}
				ctx := WithInternalWrite(p.Context)
				return models.Create(ctx, pr.OrgID, pr.UserID, model, p.Args)
			},
		}, nil
	}
	if model := strings.TrimSpace(c.Update); model != "" {
		obj := objs[model]
		m, ok := modelByName(spec, model)
		if !ok || obj == nil {
			return nil, fmt.Errorf("command %q: unknown model %q", c.Name, model)
		}
		return &graphql.Field{
			Type: graphql.NewNonNull(obj),
			Args: withIDArgs(fieldArgs(m, false)),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActExecute)
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
				ctx := WithInternalWrite(p.Context)
				return models.Update(ctx, pr.OrgID, model, id, fields)
			},
		}, nil
	}
	if model := strings.TrimSpace(c.Delete); model != "" {
		if _, ok := modelByName(spec, model); !ok {
			return nil, fmt.Errorf("command %q: unknown model %q", c.Name, model)
		}
		return &graphql.Field{
			Type: graphql.NewNonNull(graphql.Boolean),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActExecute)
				if err != nil {
					return nil, err
				}
				id, _ := p.Args["id"].(string)
				ctx := WithInternalWrite(p.Context)
				if err := models.Delete(ctx, pr.OrgID, model, id); err != nil {
					return nil, err
				}
				return true, nil
			},
		}, nil
	}

	fn := handlers.command(c.Name)
	if fn == nil {
		return nil, fmt.Errorf("command %q: no create/update/delete shorthand and no Go handler registered", c.Name)
	}
	outType, err := cqrsReturnType(spec, objs, c.Returns, c.Args)
	if err != nil {
		return nil, fmt.Errorf("command %q: %w", c.Name, err)
	}
	return &graphql.Field{
		Type: outType,
		Args: fieldSpecsArgs(c.Args, true),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActExecute)
			if err != nil {
				return nil, err
			}
			return fn(HandlerCtx{
				Context: p.Context, OrgID: pr.OrgID, UserID: pr.UserID,
				Models: models, Host: host, Spec: spec,
			}, p.Args)
		},
	}, nil
}

func fieldSpecsArgs(fields []appspec.FieldSpec, requiredOnly bool) graphql.FieldConfigArgument {
	args := graphql.FieldConfigArgument{}
	for _, f := range fields {
		gqlType := gqlInputType(f)
		if requiredOnly && f.Required && f.Default == nil {
			args[f.Name] = &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlType)}
		} else {
			args[f.Name] = &graphql.ArgumentConfig{Type: gqlType}
		}
	}
	return args
}

func cqrsReturnType(spec appspec.AppSpec, objs map[string]*graphql.Object, returns string, args []appspec.FieldSpec) (graphql.Output, error) {
	returns = strings.TrimSpace(returns)
	switch {
	case returns == "" || returns == "bool" || returns == "boolean":
		if returns == "" && len(args) == 0 {
			return graphql.NewNonNull(graphql.Boolean), nil
		}
		if returns == "bool" || returns == "boolean" {
			return graphql.NewNonNull(graphql.Boolean), nil
		}
	case returns == "int":
		return graphql.NewNonNull(graphql.Int), nil
	case returns == "string":
		return graphql.NewNonNull(graphql.String), nil
	}
	if strings.HasSuffix(returns, "[]") {
		model := strings.TrimSuffix(returns, "[]")
		obj := objs[model]
		if obj == nil {
			return nil, fmt.Errorf("unknown return model %q", model)
		}
		return graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))), nil
	}
	if obj := objs[returns]; obj != nil {
		return obj, nil
	}
	if returns == "" {
		return graphql.NewNonNull(graphql.Boolean), nil
	}
	return nil, fmt.Errorf("unsupported returns %q (use model, model[], int, bool, string)", returns)
}

// cqrsBindingsForModel returns GraphQL field names bound to a model via shorthand.
func cqrsBindingsForModel(spec appspec.AppSpec, model string) (list, get, create, update, del string) {
	for _, q := range spec.Queries {
		if q.List == model {
			list = cqrsFieldName(spec.Name, q.Name)
		}
		if q.Get == model {
			get = cqrsFieldName(spec.Name, q.Name)
		}
	}
	for _, c := range spec.Commands {
		if c.Create == model {
			create = cqrsFieldName(spec.Name, c.Name)
		}
		if c.Update == model {
			update = cqrsFieldName(spec.Name, c.Name)
		}
		if c.Delete == model {
			del = cqrsFieldName(spec.Name, c.Name)
		}
	}
	return
}

// virtualModelObject builds a GraphQL type for virtual (non-Postgres) models.
// Prefer RegisterModel.ObjectType when present.
func virtualModelObject(spec appspec.AppSpec, m appspec.ModelSpec) (*graphql.Object, error) {
	if rm, ok := registeredModelByName(spec.Name, m.Name); ok && rm.ObjectType != nil {
		return rm.ObjectType, nil
	}
	fields := graphql.Fields{
		"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID), Resolve: mapField("id")},
	}
	for _, f := range m.Fields {
		gqlType := gqlInputType(f)
		if f.Required {
			gqlType = graphql.NewNonNull(gqlType)
		}
		fields[f.Name] = &graphql.Field{Type: gqlType, Resolve: mapField(f.Name)}
	}
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   typeName(spec, m),
		Fields: fields,
	}), nil
}

// listCQRSPage prefers a RegisteredModel List (virtual / code-backed), else ModelRegistry.
func listCQRSPage(app string, models *ModelRegistry, model string, rc RequestContext, opts ListPageOpts) (ListPageResult, error) {
	opts = opts.Normalize()
	if rm, ok := registeredModelByName(app, model); ok && rm.List != nil {
		list, err := rm.List(rc)
		if err != nil {
			return ListPageResult{}, err
		}
		if list == nil {
			list = []any{}
		}
		list = filterAnyRecords(list, opts)
		items, total := sliceAnyPage(list, opts)
		out := make([]Record, 0, len(items))
		for _, item := range items {
			if rec := anyToRecord(item); rec != nil {
				out = append(out, rec)
			}
		}
		page := opts.Page
		pageSize := opts.PageSize
		if pageSize <= 0 {
			page, pageSize = 1, total
		}
		return ListPageResult{Items: out, Total: total, Page: page, PageSize: pageSize}, nil
	}
	if models == nil {
		return ListPageResult{}, fmt.Errorf("model %q: not registered (virtual models require RegisterModel)", model)
	}
	return models.ListPage(rc.Context, rc.OrgID, model, opts)
}

func groupsCQRS(app string, models *ModelRegistry, model string, rc RequestContext, opts ListPageOpts) ([]GroupBucket, error) {
	opts = opts.Normalize()
	if len(opts.GroupBy) == 0 {
		return nil, nil
	}
	if rm, ok := registeredModelByName(app, model); ok && rm.List != nil {
		list, err := rm.List(rc)
		if err != nil {
			return nil, err
		}
		list = filterAnyRecords(list, ListPageOpts{
			Domain: opts.Domain, Q: opts.Q, SearchIn: opts.SearchIn,
		})
		return groupAnyRecords(list, opts.GroupBy), nil
	}
	if models == nil {
		return nil, fmt.Errorf("model %q: not registered", model)
	}
	return models.Groups(rc.Context, rc.OrgID, model, opts)
}

func groupAnyRecords(list []any, groupBy []string) []GroupBucket {
	counts := map[string]*GroupBucket{}
	order := []string{}
	for _, item := range list {
		rec := anyToRecord(item)
		if rec == nil {
			continue
		}
		vals := make([]string, len(groupBy))
		for i, f := range groupBy {
			vals[i] = fmt.Sprint(rec[f])
			if vals[i] == "<nil>" {
				vals[i] = ""
			}
		}
		key := strings.Join(vals, "\x1f")
		if b, ok := counts[key]; ok {
			b.Count++
		} else {
			b := &GroupBucket{Values: vals, Count: 1}
			counts[key] = b
			order = append(order, key)
		}
	}
	out := make([]GroupBucket, 0, len(order))
	for _, k := range order {
		out = append(out, *counts[k])
	}
	return out
}
