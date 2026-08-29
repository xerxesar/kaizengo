package identity

import (
	"context"
	"fmt"
	"os"
	"strings"

	idtypes "kaizengo/apps/identity/__types__"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"

	"github.com/google/uuid"
)

const (
	defaultAdminEmail = "admin@kaizengo.local"
	seedAuthor        = "00000000-0000-0000-0000-000000000001"
)

func seed(_ *module.Host, events *engine.EventsSetup) error {
	ctx := context.Background()
	if err := seedDemo(ctx, events.Models); err != nil {
		return err
	}
	return seedAdminUser(ctx, events.Models)
}

func seedDemo(ctx context.Context, m *engine.ModelRegistry) error {
	orgs, err := m.ListAll(ctx, "organization")
	if err != nil {
		return err
	}
	if len(orgs) == 0 {
		orgID := uuid.NewString()
		if _, err := m.Create(ctx, orgID, seedAuthor, "organization", map[string]any{
			"id": orgID, "name": "Acme Mining Corp", "slug": "acme-mining",
		}); err != nil {
			return err
		}

		exploration, err := createOrgUnit(ctx, m, orgID, nil, idtypes.OrgUnitTypeBusinessUnit, "Exploration Division")
		if err != nil {
			return err
		}
		operations, err := createOrgUnit(ctx, m, orgID, nil, idtypes.OrgUnitTypeBusinessUnit, "Operations Division")
		if err != nil {
			return err
		}
		centralGeo, err := createOrgUnit(ctx, m, orgID, &exploration, idtypes.OrgUnitTypeDepartment, "Central Geology")
		if err != nil {
			return err
		}
		projectAlpha, err := createOrgUnit(ctx, m, orgID, &exploration, idtypes.OrgUnitTypeTeam, "Project Alpha")
		if err != nil {
			return err
		}
		if _, err := createOrgUnit(ctx, m, orgID, &projectAlpha, idtypes.OrgUnitTypeLocation, "Northern Block"); err != nil {
			return err
		}
		if _, err := createOrgUnit(ctx, m, orgID, &operations, idtypes.OrgUnitTypeLocation, "Processing Plant"); err != nil {
			return err
		}
		if _, err := createOrgUnit(ctx, m, orgID, &operations, idtypes.OrgUnitTypeDepartment, "Maintenance"); err != nil {
			return err
		}

		jahan, err := createUser(ctx, m, orgID, "jahan.doran@acme.example", "Jahan Doran")
		if err != nil {
			return err
		}
		pos, err := createOrgUnit(ctx, m, orgID, &centralGeo, idtypes.OrgUnitTypePosition, "Senior Geologist")
		if err != nil {
			return err
		}
		if err := assignUser(ctx, m, orgID, jahan, pos, "holder"); err != nil {
			return err
		}
		if err := assignUser(ctx, m, orgID, jahan, projectAlpha, "member"); err != nil {
			return err
		}

		ali, err := createUser(ctx, m, orgID, "ali.rezaei@acme.example", "Ali Rezaei")
		if err != nil {
			return err
		}
		maintPos, err := createOrgUnit(ctx, m, orgID, nil, idtypes.OrgUnitTypePosition, "Maintenance Supervisor")
		if err != nil {
			return err
		}
		return assignUser(ctx, m, orgID, ali, maintPos, "holder")
	}

	orgID := fmt.Sprint(orgs[0]["id"])
	return ensureDemoUsers(ctx, m, orgID)
}

// ensureDemoUsers creates Jahan/Ali when the org already existed (idempotent re-seed).
func ensureDemoUsers(ctx context.Context, m *engine.ModelRegistry, orgID string) error {
	for _, u := range []struct{ email, name string }{
		{"jahan.doran@acme.example", "Jahan Doran"},
		{"ali.rezaei@acme.example", "Ali Rezaei"},
	} {
		if _, err := m.FindBy(ctx, "user", "email", u.email); err == nil {
			continue
		}
		if _, err := createUser(ctx, m, orgID, u.email, u.name); err != nil {
			return err
		}
	}
	return nil
}

func seedAdminUser(ctx context.Context, m *engine.ModelRegistry) error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("KaizenGo_ADMIN_EMAIL")))
	if email == "" {
		email = defaultAdminEmail
	}
	if _, err := m.FindBy(ctx, "user", "email", email); err == nil {
		return nil
	}
	orgs, err := m.ListAll(ctx, "organization")
	if err != nil {
		return err
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no organization to attach admin user")
	}
	_, err = createUser(ctx, m, fmt.Sprint(orgs[0]["id"]), email, "Platform Admin")
	return err
}

func createUser(ctx context.Context, m *engine.ModelRegistry, orgID, email, name string) (string, error) {
	rec, err := m.Create(ctx, orgID, seedAuthor, "user", map[string]any{
		"email": strings.ToLower(strings.TrimSpace(email)),
		"name":  strings.TrimSpace(name),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprint(rec["id"]), nil
}

func createOrgUnit(ctx context.Context, m *engine.ModelRegistry, orgID string, parentID *string, unitType idtypes.OrgUnitType, name string) (string, error) {
	fields := map[string]any{"name": name, "type": string(unitType)}
	if parentID != nil {
		fields["parentId"] = *parentID
	}
	rec, err := m.Create(ctx, orgID, seedAuthor, "org_unit", fields)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(rec["id"]), nil
}

func assignUser(ctx context.Context, m *engine.ModelRegistry, orgID, userID, orgUnitID, role string) error {
	_, err := m.Create(ctx, orgID, seedAuthor, "membership", map[string]any{
		"userId":    userID,
		"orgUnitId": orgUnitID,
		"role":      role,
	})
	return err
}
