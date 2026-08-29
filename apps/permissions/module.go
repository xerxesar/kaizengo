package permissions

//go:generate go run ../../cmd/godino gen-types permissions

import (
	"context"
	"fmt"
	"log"
	"strings"

	"kaizengo/apps/permissions/service"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/app"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	module.Register(&App{})
}

const appName = "permissions"
const appVersion = "0.2.0"
const seedAuthor = "00000000-0000-0000-0000-000000000001"

type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

type engineStore struct {
	models *engine.ModelRegistry
}

func (s engineStore) ctx() context.Context {
	return engine.WithInternal(context.Background())
}

func (s engineStore) ListRoleNames(ctx context.Context, userID, orgID string) ([]string, error) {
	ictx := engine.WithInternal(ctx)
	urs, err := s.models.List(ictx, orgID, "user_role")
	if err != nil {
		return nil, err
	}
	roles, err := s.models.List(ictx, orgID, "role")
	if err != nil {
		return nil, err
	}
	byID := map[string]string{}
	for _, r := range roles {
		byID[fmt.Sprint(r["id"])] = fmt.Sprint(r["name"])
	}
	seen := map[string]struct{}{}
	var out []string
	for _, ur := range urs {
		if fmt.Sprint(ur["userId"]) != userID {
			continue
		}
		name := byID[fmt.Sprint(ur["roleId"])]
		// legacy column during migration
		if name == "" {
			name = fmt.Sprint(ur["role"])
		}
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

func (s engineStore) ListEntriesForUser(ctx context.Context, userID, orgID string) ([]acl.Entry, error) {
	ictx := engine.WithInternal(ctx)
	urs, err := s.models.List(ictx, orgID, "user_role")
	if err != nil {
		return nil, err
	}
	roleIDs := map[string]struct{}{}
	for _, ur := range urs {
		if fmt.Sprint(ur["userId"]) != userID {
			continue
		}
		rid := fmt.Sprint(ur["roleId"])
		if rid != "" {
			roleIDs[rid] = struct{}{}
		}
	}
	roles, err := s.models.List(ictx, orgID, "role")
	if err != nil {
		return nil, err
	}
	roleName := map[string]string{}
	for _, r := range roles {
		id := fmt.Sprint(r["id"])
		roleName[id] = fmt.Sprint(r["name"])
		// also match legacy slug assignments via name lookup later if needed
	}

	recs, err := s.models.List(ictx, orgID, "acl_entry")
	if err != nil {
		return nil, err
	}
	out := make([]acl.Entry, 0, len(recs))
	for _, rec := range recs {
		rid := fmt.Sprint(rec["roleId"])
		if _, ok := roleIDs[rid]; !ok {
			continue
		}
		actions, allAct, err := acl.ParseStringList(fmt.Sprint(rec["actions"]))
		if err != nil {
			return nil, err
		}
		if allAct {
			actions = nil
		}
		fields, allFields, err := acl.ParseStringList(fmt.Sprint(rec["fields"]))
		if err != nil {
			return nil, err
		}
		if allFields {
			fields = nil
		}
		domain, err := acl.ParseDomain(fmt.Sprint(rec["domain"]))
		if err != nil {
			return nil, err
		}
		active := true
		if v, ok := rec["active"].(bool); ok {
			active = v
		}
		prio := 0
		switch v := rec["priority"].(type) {
		case int:
			prio = v
		case int32:
			prio = int(v)
		case int64:
			prio = int(v)
		case float64:
			prio = int(v)
		}
		out = append(out, acl.Entry{
			ID:       fmt.Sprint(rec["id"]),
			RoleID:   rid,
			RoleName: roleName[rid],
			Effect:   fmt.Sprint(rec["effect"]),
			Resource: fmt.Sprint(rec["resource"]),
			Actions:  actions,
			Fields:   fields,
			Domain:   domain,
			Priority: prio,
			Active:   active,
		})
	}
	return out, nil
}

func (s engineStore) FindRoleID(ctx context.Context, orgID, name string) (string, error) {
	ictx := engine.WithInternal(ctx)
	recs, err := s.models.List(ictx, orgID, "role")
	if err != nil {
		return "", err
	}
	want := strings.ToLower(strings.TrimSpace(name))
	for _, r := range recs {
		if strings.ToLower(fmt.Sprint(r["name"])) == want {
			return fmt.Sprint(r["id"]), nil
		}
	}
	return "", fmt.Errorf("role %q not found", name)
}

func (s engineStore) EnsureRole(ctx context.Context, orgID, authorID, name, label string) (string, error) {
	ictx := engine.WithInternal(ctx)
	if id, err := s.FindRoleID(ictx, orgID, name); err == nil {
		return id, nil
	}
	rec, err := s.models.Create(ictx, orgID, authorID, "role", map[string]any{
		"name":        name,
		"label":       label,
		"description": "",
		"active":      true,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprint(rec["id"]), nil
}

func (s engineStore) EnsureACL(ctx context.Context, orgID, authorID, roleID, name, effect, resource, actions, fields, domain string, priority int) error {
	ictx := engine.WithInternal(ctx)
	recs, err := s.models.List(ictx, orgID, "acl_entry")
	if err != nil {
		return err
	}
	for _, r := range recs {
		if fmt.Sprint(r["name"]) == name && fmt.Sprint(r["roleId"]) == roleID {
			_, err := s.models.Update(ictx, orgID, "acl_entry", fmt.Sprint(r["id"]), map[string]any{
				"effect":   effect,
				"resource": resource,
				"actions":  actions,
				"fields":   fields,
				"domain":   domain,
				"priority": priority,
				"active":   true,
			})
			return err
		}
	}
	_, err = s.models.Create(ictx, orgID, authorID, "acl_entry", map[string]any{
		"name":     name,
		"roleId":   roleID,
		"effect":   effect,
		"resource": resource,
		"actions":  actions,
		"fields":   fields,
		"domain":   domain,
		"priority": priority,
		"active":   true,
	})
	return err
}

func (s engineStore) DisableACLByName(ctx context.Context, orgID, name string) error {
	ictx := engine.WithInternal(ctx)
	recs, err := s.models.List(ictx, orgID, "acl_entry")
	if err != nil {
		return err
	}
	for _, r := range recs {
		if fmt.Sprint(r["name"]) != name {
			continue
		}
		if active, ok := r["active"].(bool); ok && !active {
			continue
		}
		if _, err := s.models.Update(ictx, orgID, "acl_entry", fmt.Sprint(r["id"]), map[string]any{
			"active": false,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s engineStore) AssignUserRoleID(ctx context.Context, orgID, authorID, userID, roleID string) error {
	ictx := engine.WithInternal(ctx)
	_, err := s.models.Create(ictx, orgID, authorID, "user_role", map[string]any{
		"userId": userID,
		"roleId": roleID,
	})
	return err
}

func (s engineStore) UserHasRole(ctx context.Context, orgID, userID, roleID string) (bool, error) {
	ictx := engine.WithInternal(ctx)
	urs, err := s.models.List(ictx, orgID, "user_role")
	if err != nil {
		return false, err
	}
	for _, ur := range urs {
		if fmt.Sprint(ur["userId"]) == userID && fmt.Sprint(ur["roleId"]) == roleID {
			return true, nil
		}
	}
	return false, nil
}

func (a *App) Setup(host *module.Host) error {
	spec := app.MustAppSpec(appName)
	if spec.EnableI18n {
		app.MustLoadLocales(appName)
	}
	events, err := engine.SetupEvents(host, appName, spec, nil)
	if err != nil {
		return err
	}
	host.Provide(engine.ModelsKey(appName), events.Models)

	svc := service.New(engineStore{models: events.Models})
	host.Provide(service.Name, svc)
	engine.RegisterAppResources(spec)
	registerGQL(host)

	if err := a.seed(host, svc); err != nil {
		log.Printf("permissions: seed: %v", err)
	}
	return nil
}

func (a *App) seed(host *module.Host, perm *service.Service) error {
	users, err := engine.ModelsFromHost(host, "identity")
	if err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(app.Env("KaizenGo_ADMIN_EMAIL", "admin@kaizengo.local")))
	ctx := engine.WithInternal(context.Background())
	rec, err := users.FindBy(ctx, "user", "email", email)
	if err != nil {
		return err
	}
	userID := fmt.Sprint(rec["id"])
	orgID := fmt.Sprint(rec["orgId"])
	return perm.SeedDefaults(ctx, orgID, userID)
}

func (a *App) Mount(host *module.Host) error {
	return nil
}
