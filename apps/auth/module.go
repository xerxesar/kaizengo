package auth

//go:generate go run ../../cmd/godino gen-types auth

import (
	"context"

	permsvc "kaizengo/apps/permissions/service"
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	var svc *Service
	module.Register(engine.New(engine.Options{
		AppName: "auth",
		Version: "0.1.0",
		Setup: func(host *module.Host, events *engine.EventsSetup) error {
			users, err := engine.ModelsFromHost(host, "identity")
			if err != nil {
				return err
			}
			store := NewStore(events.Pool, events.Schema)
			svc = New(store, users, permissionRoles(host))
			host.Provide(Name, svc)
			if err := seedAdmin(context.Background(), svc); err != nil {
				return err
			}
			registerGQL(host, svc)
			return nil
		},
		Mount: func(host *module.Host) error {
			h := &Handlers{Auth: svc}
			host.Router.Post("/auth/login", h.Login)
			host.Router.Post("/auth/logout", h.Logout)
			host.Router.Get("/auth/me", h.Me)
			return nil
		},
	}))
}

func permissionRoles(host *module.Host) RoleLookup {
	return func(ctx context.Context, userID, orgID string) ([]string, error) {
		raw, ok := host.Lookup(permsvc.Name)
		if !ok {
			return nil, nil
		}
		svc, ok := raw.(*permsvc.Service)
		if !ok {
			return nil, nil
		}
		return svc.Roles(ctx, userID, orgID)
	}
}
