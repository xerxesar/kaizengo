package engine

import (
	"context"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	sdkgql "kaizengo/packages/sdk-go/gql"
	"kaizengo/packages/sdk-go/views"
)

// FilterMenuCatalog removes menu items the principal cannot read.
// Catalog surfaces default to visible unless a matching policy denies (or allows) them.
func FilterMenuCatalog(ctx context.Context, host *module.Host, app string, menus []views.Menu) ([]views.Menu, error) {
	authz, err := sdkgql.LookupAuthorizer(host, acl.ServiceName)
	if err != nil {
		return menus, nil
	}
	pr, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}

	var filter func([]views.Menu) ([]views.Menu, error)
	filter = func(items []views.Menu) ([]views.Menu, error) {
		out := make([]views.Menu, 0, len(items))
		for _, item := range items {
			children, err := filter(item.Children)
			if err != nil {
				return nil, err
			}
			item.Children = children

			if item.View == "" && item.Component == "" {
				if len(children) > 0 {
					out = append(out, item)
				}
				continue
			}

			allowed, err := authz.CanCatalog(ctx, acl.Check{
				OrgID:    pr.OrgID,
				Resource: acl.MenuResource(app, item.ID),
				Action:   acl.ActRead,
			})
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
			out = append(out, item)
		}
		return out, nil
	}
	return filter(menus)
}

// FilterShellNav removes Apps dropdown entries the principal cannot read.
func FilterShellNav(ctx context.Context, host *module.Host, entries []module.NavEntry) ([]module.NavEntry, error) {
	authz, err := sdkgql.LookupAuthorizer(host, acl.ServiceName)
	if err != nil {
		return entries, nil
	}
	pr, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}

	out := make([]module.NavEntry, 0, len(entries))
	for _, entry := range entries {
		allowed, err := authz.CanCatalog(ctx, acl.Check{
			OrgID:    pr.OrgID,
			Resource: acl.NavResource(entry.ID),
			Action:   acl.ActRead,
		})
		if err != nil {
			return nil, err
		}
		if allowed {
			out = append(out, entry)
		}
	}
	return out, nil
}
