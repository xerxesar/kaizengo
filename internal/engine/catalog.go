package engine

import (
	"context"
	"sort"
	"strings"

	"kaizengo/internal/auth"
	sdkgql "kaizengo/internal/gql"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
)

// Operation is one callable GraphQL field, described for CLI/palette discovery.
type Operation struct {
	App          string
	Kind         string // query | command
	Name         string // spec name (CQRS) or listX/getX/createX/updateX/deleteX (model CRUD)
	Field        string // GraphQL root field
	Operation    string // query | mutation
	Resource     string
	Action       string
	Args         []OperationArg
	Returns      string
	ReturnFields []string
	Dynamic      bool
}

// OperationArg is one GraphQL argument with its printed type (e.g. "String!").
type OperationArg struct {
	Name     string
	Type     string
	Required bool
}

// Catalog lists callable operations of loaded engine apps (compiled-in and dynamic)
// that the caller in ctx may invoke. appFilter limits to one app when set.
func Catalog(ctx context.Context, host *module.Host, appFilter string) ([]Operation, error) {
	authz, _ := sdkgql.LookupAuthorizer(host, acl.ServiceName)
	pr, _ := auth.PrincipalFrom(ctx)
	dyn, _ := DomainsFromHost(host)

	var out []Operation
	claimed := map[string]bool{}
	for _, mf := range host.Loaded {
		a, ok := AppFromHost(host, mf.Name)
		if !ok {
			continue
		}
		spec := a.Spec()
		isDynamic := dyn != nil && dyn.IsDynamic(spec.Name)
		for _, op := range specOperations(spec) {
			field, ok := lookupField(host, op)
			if !ok {
				continue
			}
			claimed[op.Operation+":"+op.Field] = true
			if appFilter != "" && mf.Name != appFilter {
				continue
			}
			describeField(&op, field)
			op.Dynamic = isDynamic
			if authz != nil && pr != nil {
				allowed, err := authz.Can(ctx, acl.Check{OrgID: pr.OrgID, Resource: op.Resource, Action: op.Action})
				if err != nil || !allowed.Allowed {
					continue
				}
			}
			out = append(out, op)
		}
	}
	for _, op := range unclaimedOperations(host, claimed) {
		if appFilter == "" || op.App == appFilter {
			op.Dynamic = dyn != nil && dyn.IsDynamic(op.App)
			out = append(out, op)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].App != out[j].App {
			return out[i].App < out[j].App
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind > out[j].Kind // queries before commands
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// GraphQLApp is the catalog app for hand-registered root fields that no loaded
// app prefix claims (settings, search, view presets, platform meta, …).
// Unclaimed fields enforce access in their resolvers, so they are listed
// without a catalog ACL check.
const GraphQLApp = "graphql"

var hiddenRootFields = map[string]bool{
	"query:platformShellCommands": true,
	"mutation:platformShell":      true,
}

func unclaimedOperations(host *module.Host, claimed map[string]bool) []Operation {
	queries, mutations := host.GQL.FieldNames()
	prefixes := make([]string, 0, len(host.Loaded))
	for _, mf := range host.Loaded {
		prefixes = append(prefixes, mf.Name)
	}
	sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
	owner := func(field string) (string, string) {
		for _, app := range prefixes {
			p := camel(app)
			if len(field) > len(p) && strings.HasPrefix(field, p) && field[len(p)] >= 'A' && field[len(p)] <= 'Z' {
				return app, strings.ToLower(field[len(p):len(p)+1]) + field[len(p)+1:]
			}
		}
		return GraphQLApp, field
	}
	var out []Operation
	add := func(kind, operation, field string, f *graphql.Field) {
		key := operation + ":" + field
		if claimed[key] || hiddenRootFields[key] {
			return
		}
		app, name := owner(field)
		op := Operation{App: app, Kind: kind, Name: name, Field: field, Operation: operation}
		describeField(&op, f)
		out = append(out, op)
	}
	for _, name := range queries {
		if f, ok := host.GQL.QueryField(name); ok {
			add(KindQuery, "query", name, f)
		}
	}
	for _, name := range mutations {
		if f, ok := host.GQL.MutationField(name); ok {
			add(KindCommand, "mutation", name, f)
		}
	}
	return out
}

func specOperations(spec appspec.AppSpec) []Operation {
	var ops []Operation
	if spec.HasCQRS() {
		for _, q := range spec.Queries {
			field := cqrsFieldName(spec.Name, q.Name)
			ops = append(ops, Operation{
				App: spec.Name, Kind: KindQuery, Name: q.Name, Field: field, Operation: "query",
				Resource: acl.QueryResource(spec.Name, field), Action: acl.ActRead,
			})
		}
		for _, c := range spec.Commands {
			field := cqrsFieldName(spec.Name, c.Name)
			ops = append(ops, Operation{
				App: spec.Name, Kind: KindCommand, Name: c.Name, Field: field, Operation: "mutation",
				Resource: acl.CommandResource(spec.Name, field), Action: acl.ActExecute,
			})
		}
		return ops
	}
	for _, m := range spec.Models {
		if m.Virtual {
			continue
		}
		res := modelResource(spec.Name, m.Name)
		model := pascal(m.Name)
		ops = append(ops,
			Operation{App: spec.Name, Kind: KindQuery, Name: "list" + model, Field: listName(spec, m), Operation: "query", Resource: res, Action: acl.ActRead},
			Operation{App: spec.Name, Kind: KindQuery, Name: "get" + model, Field: getName(spec, m), Operation: "query", Resource: res, Action: acl.ActRead},
		)
		if m.Internal {
			continue
		}
		ops = append(ops,
			Operation{App: spec.Name, Kind: KindCommand, Name: "create" + model, Field: createName(spec, m), Operation: "mutation", Resource: res, Action: acl.ActCreate},
			Operation{App: spec.Name, Kind: KindCommand, Name: "update" + model, Field: updateName(spec, m), Operation: "mutation", Resource: res, Action: acl.ActUpdate},
			Operation{App: spec.Name, Kind: KindCommand, Name: "delete" + model, Field: deleteName(spec, m), Operation: "mutation", Resource: res, Action: acl.ActDelete},
		)
	}
	return ops
}

func lookupField(host *module.Host, op Operation) (*graphql.Field, bool) {
	if op.Operation == "mutation" {
		return host.GQL.MutationField(op.Field)
	}
	return host.GQL.QueryField(op.Field)
}

func describeField(op *Operation, f *graphql.Field) {
	names := make([]string, 0, len(f.Args))
	for name := range f.Args {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t := f.Args[name].Type
		_, required := t.(*graphql.NonNull)
		op.Args = append(op.Args, OperationArg{Name: name, Type: t.String(), Required: required})
	}
	if f.Type != nil {
		op.Returns = f.Type.String()
		op.ReturnFields = leafFieldNames(f.Type)
	}
}

// leafFieldNames returns scalar/enum fields of an object return type (the
// default selection set a CLI client can request without nesting).
func leafFieldNames(t graphql.Output) []string {
	obj, ok := unwrapType(t).(*graphql.Object)
	if !ok {
		return nil
	}
	var out []string
	for name, def := range obj.Fields() {
		switch unwrapType(def.Type).(type) {
		case *graphql.Scalar, *graphql.Enum:
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return []string{"__typename"}
	}
	sort.Slice(out, func(i, j int) bool { return fieldOrder(out[i]) < fieldOrder(out[j]) })
	return out
}

func unwrapType(t graphql.Type) graphql.Type {
	for {
		switch typed := t.(type) {
		case *graphql.NonNull:
			t = typed.OfType
		case *graphql.List:
			t = typed.OfType
		default:
			return t
		}
	}
}

// fieldOrder puts id first and audit columns last so CLI tables read naturally.
func fieldOrder(name string) string {
	switch name {
	case "id":
		return "0"
	case "orgId", "authorId", "deleted", "createdAt", "updatedAt":
		return "2" + name
	default:
		return "1" + strings.ToLower(name)
	}
}
