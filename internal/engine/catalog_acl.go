package engine

import (
	"context"
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	sdkgql "kaizengo/internal/gql"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/views"
)

// FilterMenuCatalog keeps menu items whose linked view the principal can access.
// Folders stay if any child remains. Component-only leaves stay visible (no view to check).
// Menu ACL resources are not used — visibility is implied from view access.
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

			view := strings.TrimSpace(item.View)
			if view == "" && strings.TrimSpace(item.Component) == "" {
				if len(children) > 0 {
					out = append(out, item)
				}
				continue
			}
			if view == "" {
				// Addon component without a view name — keep if parent path is otherwise open.
				out = append(out, item)
				continue
			}

			allowed, err := authz.CanCatalog(ctx, acl.Check{
				OrgID:    pr.OrgID,
				Resource: acl.ViewResource(app, view),
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

// FilterShellNav keeps Apps dropdown entries when the principal can access any view in that app.
// Nav ACL resources are not used — visibility is implied from view access.
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
		views := acl.ViewsForApp(entry.ID)
		if len(views) == 0 {
			// No tracked views (custom SPA) — leave visible.
			out = append(out, entry)
			continue
		}
		any := false
		for _, view := range views {
			allowed, err := authz.CanCatalog(ctx, acl.Check{
				OrgID:    pr.OrgID,
				Resource: acl.ViewResource(entry.ID, view),
				Action:   acl.ActRead,
			})
			if err != nil {
				return nil, err
			}
			if allowed {
				any = true
				break
			}
		}
		if any {
			out = append(out, entry)
		}
	}
	return out, nil
}
