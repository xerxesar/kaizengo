package permissions

//go:generate go run ../../cmd/godino gen-types permissions

import (
	"context"
	"fmt"
	"log"
	"strings"

	"kaizengo/apps/permissions/service"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/app"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	module.Register(&App{})
}

const appName = "permissions"
const appVersion = "0.1.0"
const seedAuthor = "00000000-0000-0000-0000-000000000001"

type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

// engineStore adapts ModelRegistry to service.Store (keeps service free of engine imports).
type engineStore struct {
	models *engine.ModelRegistry
}

func (s engineStore) ListRoles(ctx context.Context, userID, orgID string) ([]string, error) {
	recs, err := s.models.List(ctx, orgID, "user_role")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var roles []string
	for _, rec := range recs {
		if fmt.Sprint(rec["userId"]) != userID {
			continue
		}
		role := fmt.Sprint(rec["role"])
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles, nil
}

func (s engineStore) AssignUserRole(ctx context.Context, orgID, userID, role string) error {
	_, err := s.models.Create(ctx, orgID, seedAuthor, "user_role", map[string]any{
		"userId": userID,
		"role":   role,
	})
	return err
}

func (a *App) Setup(host *module.Host) error {
	spec := app.MustAppSpec(appName)
	events, err := engine.SetupEvents(host, appName, spec, nil)
	if err != nil {
		return err
	}

	svc := service.New(engineStore{models: events.Models})
	host.Provide(service.Name, svc)

	if err := a.seedAdminRole(host, svc); err != nil {
		log.Printf("permissions: admin seed: %v", err)
	}
	return nil
}

func (a *App) seedAdminRole(host *module.Host, perm *service.Service) error {
	users, err := engine.ModelsFromHost(host, "identity")
	if err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(app.Env("KaizenGo_ADMIN_EMAIL", "admin@kaizengo.local")))
	ctx := context.Background()
	rec, err := users.FindBy(ctx, "user", "email", email)
	if err != nil {
		return err
	}
	userID := fmt.Sprint(rec["id"])
	orgID := fmt.Sprint(rec["orgId"])

	roles, _ := perm.Roles(ctx, userID, orgID)
	for _, r := range roles {
		if r == service.RoleAdmin {
			return nil
		}
	}
	return perm.SeedAdmin(ctx, userID, orgID)
}

func (a *App) Mount(host *module.Host) error {
	return nil
}
