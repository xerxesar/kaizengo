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
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pr, err := sdkgql.RequireAction(host, acl.ServiceName, p, resource, acl.ActRead)
				if err != nil {
					return nil, err
				}
				list, err := models.List(p.Context, pr.OrgID, model)
				if err != nil {
					return nil, err
				}
				if list == nil {
					list = []Record{}
				}
				return list, nil
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
