package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kaizengo/internal/app"
	sdkgql "kaizengo/internal/gql"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"

	"github.com/graphql-go/graphql"
)

// DomainResource guards the domain API (admins hold it through the "*" seed).
const DomainResource = "platform.api.domain"

// RegisterPlatformAPI adds the domain and catalog fields that make every app's
// domain manageable over GraphQL (CLI, tools, external non-Go apps).
func RegisterPlatformAPI(host *module.Host, doms *Domains) {
	acl.Register(acl.ResourceDescriptor{
		App: "platform", Kind: acl.KindAPI, Name: "domain", Resource: DomainResource,
		Label: "Domains", Description: "Read, apply, revert, back up, and restore app domain layers",
		Actions: acl.CRUDActions(), Surface: "graphql",
	})

	domainType := platformDomainType()
	revisionType := platformRevisionType()
	strList := graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))
	resultType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformDomainApplyResult",
		Fields: graphql.Fields{
			"domain":  &graphql.Field{Type: graphql.NewNonNull(domainType)},
			"changes": &graphql.Field{Type: strList},
			"dryRun":  &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	})
	require := func(p graphql.ResolveParams, action string) error {
		_, err := sdkgql.RequireAction(host, acl.ServiceName, p, DomainResource, action)
		return err
	}
	appArg := func(p graphql.ResolveParams) string {
		name, _ := p.Args["app"].(string)
		return strings.TrimSpace(name)
	}
	writeAction := func(dryRun bool, action string) string {
		if dryRun {
			return acl.ActRead
		}
		return action
	}

	host.GQL.RegisterQuery("platformDomains", &graphql.Field{
		Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(domainType))),
		Description: "Every app's domain; kind filters to \"compiled\" or \"dynamic\"",
		Args: graphql.FieldConfigArgument{
			"kind": &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			kind, _ := p.Args["kind"].(string)
			kind = strings.TrimSpace(kind)
			if kind != "" && kind != DomainDynamic && kind != DomainCompiled {
				return nil, fmt.Errorf("kind must be %q or %q", DomainCompiled, DomainDynamic)
			}
			list, err := doms.List(p.Context)
			if err != nil {
				return nil, err
			}
			out := make([]any, 0, len(list))
			for _, d := range list {
				if kind == "" || d.Kind == kind {
					out = append(out, domainMap(d))
				}
			}
			return out, nil
		},
	})

	host.GQL.RegisterQuery("platformDomain", &graphql.Field{
		Type:        domainType,
		Description: "One app's domain: the effective document plus its stored layer",
		Args: graphql.FieldConfigArgument{
			"app": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			d, err := doms.Get(p.Context, appArg(p))
			if err != nil || d == nil {
				return nil, err
			}
			return domainMap(*d), nil
		},
	})

	host.GQL.RegisterQuery("platformDomainRevisions", &graphql.Field{
		Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(revisionType))),
		Description: "Stored base and layer revisions of an app, newest first",
		Args: graphql.FieldConfigArgument{
			"app": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			recs, err := doms.History(p.Context, appArg(p))
			if err != nil {
				return nil, err
			}
			out := make([]any, 0, len(recs))
			for _, r := range recs {
				out = append(out, revisionMap(r))
			}
			return out, nil
		},
	})

	host.GQL.RegisterQuery("platformDomainRevision", &graphql.Field{
		Type: revisionType,
		Args: graphql.FieldConfigArgument{
			"app":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"revision": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			rev, _ := p.Args["revision"].(int)
			r, err := doms.Revision(p.Context, appArg(p), int64(rev))
			if err != nil || r == nil {
				return nil, err
			}
			return revisionMap(*r), nil
		},
	})

	host.GQL.RegisterQuery("platformDomainDiff", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewObject(graphql.ObjectConfig{
			Name: "PlatformDomainDiff",
			Fields: graphql.Fields{
				"app":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"changes": &graphql.Field{Type: strList},
			},
		})),
		Args: graphql.FieldConfigArgument{
			"spec": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			doc, _ := p.Args["spec"].(string)
			app, changes, err := doms.Diff(p.Context, doc)
			if err != nil {
				return nil, err
			}
			return map[string]any{"app": app, "changes": changes}, nil
		},
	})

	host.GQL.RegisterQuery("platformExportDomains", &graphql.Field{
		Type:        graphql.NewNonNull(graphql.String),
		Description: "YAML backup of the current layer of every app (or only apps); restore with platformImportDomains",
		Args: graphql.FieldConfigArgument{
			"apps": &graphql.ArgumentConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if err := require(p, acl.ActRead); err != nil {
				return nil, err
			}
			var names []string
			raw, _ := p.Args["apps"].([]any)
			for _, v := range raw {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					names = append(names, strings.TrimSpace(s))
				}
			}
			return doms.Export(p.Context, names)
		},
	})

	host.GQL.RegisterMutation("platformApplyDomain", &graphql.Field{
		Type:        graphql.NewNonNull(resultType),
		Description: "Apply a complete domain document for any app; what it adds to a compiled app's base is stored as a new layer revision",
		Args: graphql.FieldConfigArgument{
			"spec":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"dryRun": &graphql.ArgumentConfig{Type: graphql.Boolean},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			doc, _ := p.Args["spec"].(string)
			dryRun, _ := p.Args["dryRun"].(bool)
			action, err := applyAction(p.Context, doms, doc, dryRun)
			if err != nil {
				return nil, err
			}
			if err := require(p, action); err != nil {
				return nil, err
			}
			res, err := doms.Apply(p.Context, doc, SourceGraphQL, dryRun)
			if err != nil {
				return nil, err
			}
			return resultMap(*res), nil
		},
	})

	host.GQL.RegisterMutation("platformRevertDomain", &graphql.Field{
		Type:        graphql.NewNonNull(resultType),
		Description: "Store an app's layer as it was at revision as a new revision and mount it",
		Args: graphql.FieldConfigArgument{
			"app":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"revision": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
			"dryRun":   &graphql.ArgumentConfig{Type: graphql.Boolean},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			dryRun, _ := p.Args["dryRun"].(bool)
			if err := require(p, writeAction(dryRun, acl.ActUpdate)); err != nil {
				return nil, err
			}
			rev, _ := p.Args["revision"].(int)
			res, err := doms.Revert(p.Context, appArg(p), int64(rev), dryRun)
			if err != nil {
				return nil, err
			}
			return resultMap(*res), nil
		},
	})

	host.GQL.RegisterMutation("platformDeleteDomain", &graphql.Field{
		Type:        graphql.NewNonNull(resultType),
		Description: "Store an empty layer: resets a compiled app to its base, unmounts a dynamic app (data and history kept)",
		Args: graphql.FieldConfigArgument{
			"app":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"dryRun": &graphql.ArgumentConfig{Type: graphql.Boolean},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			dryRun, _ := p.Args["dryRun"].(bool)
			if err := require(p, writeAction(dryRun, acl.ActDelete)); err != nil {
				return nil, err
			}
			res, err := doms.Delete(p.Context, appArg(p), dryRun)
			if err != nil {
				return nil, err
			}
			return resultMap(*res), nil
		},
	})

	host.GQL.RegisterMutation("platformImportDomains", &graphql.Field{
		Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(resultType))),
		Description: "Restore a platformExportDomains bundle; each app's layer becomes a new revision",
		Args: graphql.FieldConfigArgument{
			"bundle": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"dryRun": &graphql.ArgumentConfig{Type: graphql.Boolean},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			dryRun, _ := p.Args["dryRun"].(bool)
			if err := require(p, writeAction(dryRun, acl.ActUpdate)); err != nil {
				return nil, err
			}
			if !dryRun {
				if err := require(p, acl.ActCreate); err != nil {
					return nil, err
				}
			}
			bundle, _ := p.Args["bundle"].(string)
			list, err := doms.Import(p.Context, bundle, dryRun)
			if err != nil {
				return nil, err
			}
			out := make([]any, 0, len(list))
			for _, r := range list {
				out = append(out, resultMap(r))
			}
			return out, nil
		},
	})

	host.GQL.RegisterQuery("platformCatalog", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(platformOperationType()))),
		Args: graphql.FieldConfigArgument{
			"app": &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			appFilter, _ := p.Args["app"].(string)
			ops, err := Catalog(p.Context, host, strings.TrimSpace(appFilter))
			if err != nil {
				return nil, err
			}
			out := make([]any, 0, len(ops))
			for _, op := range ops {
				out = append(out, operationMap(op))
			}
			return out, nil
		},
	})
}

// applyAction is create for a new app and update for an existing one (read for dry runs).
func applyAction(ctx context.Context, doms *Domains, doc string, dryRun bool) (string, error) {
	if dryRun {
		return acl.ActRead, nil
	}
	spec, err := parseDoc(doc)
	if err != nil {
		return "", err
	}
	ok, err := doms.Exists(ctx, spec.Name)
	if err != nil {
		return "", err
	}
	if !ok {
		return acl.ActCreate, nil
	}
	return acl.ActUpdate, nil
}

func platformDomainType() *graphql.Object {
	str := graphql.NewNonNull(graphql.String)
	integer := graphql.NewNonNull(graphql.Int)
	strList := graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformDomain",
		Fields: graphql.Fields{
			"app":           &graphql.Field{Type: str},
			"kind":          &graphql.Field{Type: str, Description: "compiled (base layer from code) or dynamic (stored layers only)"},
			"title":         &graphql.Field{Type: str},
			"summary":       &graphql.Field{Type: str},
			"spec":          &graphql.Field{Type: str, Description: "Effective domain document (base plus layer); apply an edited copy"},
			"layer":         &graphql.Field{Type: str, Description: "Stored layer on top of the base; the whole definition for dynamic apps"},
			"revision":      &graphql.Field{Type: integer},
			"baseRevision":  &graphql.Field{Type: integer},
			"layerRevision": &graphql.Field{Type: integer},
			"source":        &graphql.Field{Type: str},
			"updatedAt":     &graphql.Field{Type: graphql.String},
			"loaded":        &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"error":         &graphql.Field{Type: graphql.String},
			"models":        &graphql.Field{Type: strList},
			"queries":       &graphql.Field{Type: strList},
			"commands":      &graphql.Field{Type: strList},
		},
	})
}

func platformRevisionType() *graphql.Object {
	str := graphql.NewNonNull(graphql.String)
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformDomainRevision",
		Fields: graphql.Fields{
			"app":       &graphql.Field{Type: str},
			"revision":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"layer":     &graphql.Field{Type: str, Description: "base (compiled spec from code) or overlay (applied at runtime)"},
			"spec":      &graphql.Field{Type: str, Description: "Stored document; empty for an overlay that removes the layer"},
			"source":    &graphql.Field{Type: str},
			"changes":   &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
			"createdAt": &graphql.Field{Type: str},
		},
	})
}

func platformOperationType() *graphql.Object {
	str := graphql.NewNonNull(graphql.String)
	arg := graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformOperationArg",
		Fields: graphql.Fields{
			"name":     &graphql.Field{Type: str},
			"type":     &graphql.Field{Type: str},
			"required": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	})
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "PlatformOperation",
		Fields: graphql.Fields{
			"app":          &graphql.Field{Type: str},
			"kind":         &graphql.Field{Type: str},
			"name":         &graphql.Field{Type: str},
			"field":        &graphql.Field{Type: str},
			"operation":    &graphql.Field{Type: str},
			"resource":     &graphql.Field{Type: str},
			"args":         &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(arg)))},
			"returns":      &graphql.Field{Type: str},
			"returnFields": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
			"dynamic":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	})
}

func domainMap(d DomainInfo) map[string]any {
	var updated any
	if !d.UpdatedAt.IsZero() {
		updated = d.UpdatedAt.UTC().Format(time.RFC3339)
	}
	var errMsg any
	if d.Error != "" {
		errMsg = d.Error
	}
	return map[string]any{
		"app": d.App, "kind": d.Kind, "title": d.Title, "summary": d.Summary,
		"spec": d.Spec, "layer": d.Layer, "source": d.Source,
		"revision": int(d.Revision), "baseRevision": int(d.BaseRevision), "layerRevision": int(d.LayerRevision),
		"updatedAt": updated, "loaded": d.Loaded, "error": errMsg,
		"models": nonNil(d.Models), "queries": nonNil(d.Queries), "commands": nonNil(d.Commands),
	}
}

func resultMap(r ApplyResult) map[string]any {
	return map[string]any{"domain": domainMap(r.Domain), "changes": nonNil(r.Changes), "dryRun": r.DryRun}
}

func revisionMap(r app.DomainRevision) map[string]any {
	return map[string]any{
		"app": r.App, "revision": int(r.Revision), "layer": r.Layer, "spec": r.Spec,
		"source": r.Source, "changes": nonNil(r.Changes), "createdAt": r.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func operationMap(op Operation) map[string]any {
	args := make([]any, 0, len(op.Args))
	for _, a := range op.Args {
		args = append(args, map[string]any{"name": a.Name, "type": a.Type, "required": a.Required})
	}
	return map[string]any{
		"app": op.App, "kind": op.Kind, "name": op.Name, "field": op.Field,
		"operation": op.Operation, "resource": op.Resource, "args": args,
		"returns": op.Returns, "returnFields": nonNil(op.ReturnFields), "dynamic": op.Dynamic,
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
