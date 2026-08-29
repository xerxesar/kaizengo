package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
)

const (
	securitySeedAuthor = "00000000-0000-0000-0000-000000000001"
	authServiceName    = "auth"
)

// securityPerms is satisfied by apps/permissions/service.Service.
type securityPerms interface {
	EnsureRole(ctx context.Context, orgID, name, label string) (string, error)
	EnsureACLEntry(ctx context.Context, orgID, roleName, name, effect, resource, actions, fields, domain string, priority int) error
	DisableEntry(ctx context.Context, orgID, name string) error
	AssignRole(ctx context.Context, userID, orgID, roleName string) error
}

// securityAuth is satisfied by apps/auth.Service.
type securityAuth interface {
	SetPassword(ctx context.Context, userID, password string) error
}

// ApplySecurity seeds roles, ACL entries, and demo users from appspec.SecuritySpec.
// Missing permissions/identity/auth services are skipped with a log when the section needs them.
func ApplySecurity(host *module.Host, sec appspec.SecuritySpec) error {
	if sec.Empty() {
		return nil
	}
	ctx := WithInternal(context.Background())

	needsPerms := len(sec.Roles) > 0 || len(sec.Disable) > 0 || len(sec.Entries) > 0 || hasUserRoles(sec.Users)
	needsIdentity := len(sec.Users) > 0 || needsPerms
	needsAuth := hasUserPasswords(sec.Users)

	var perm securityPerms
	if needsPerms {
		raw, ok := host.Lookup(acl.ServiceName)
		if !ok {
			log.Printf("engine: security.yaml skipped (permissions service not found)")
			return nil
		}
		p, ok := raw.(securityPerms)
		if !ok {
			return fmt.Errorf("security: permissions service has unexpected type %T", raw)
		}
		perm = p
	}

	var users *ModelRegistry
	if needsIdentity {
		m, err := ModelsFromHost(host, "identity")
		if err != nil {
			log.Printf("engine: security.yaml skipped (%v)", err)
			return nil
		}
		users = m
	}

	var authSvc securityAuth
	if needsAuth {
		raw, ok := host.Lookup(authServiceName)
		if !ok {
			return fmt.Errorf("security: auth service not found (required for user passwords)")
		}
		a, ok := raw.(securityAuth)
		if !ok {
			return fmt.Errorf("security: auth service has unexpected type %T", raw)
		}
		authSvc = a
	}

	orgID, err := resolveSecurityOrg(ctx, users)
	if err != nil {
		return err
	}

	for _, r := range sec.Roles {
		if _, err := perm.EnsureRole(ctx, orgID, r.Name, r.Label); err != nil {
			return fmt.Errorf("security role %s: %w", r.Name, err)
		}
	}
	for _, name := range sec.Disable {
		if err := perm.DisableEntry(ctx, orgID, name); err != nil {
			return fmt.Errorf("security disable %s: %w", name, err)
		}
	}
	for _, e := range sec.Entries {
		actions, fields, domain, err := encodeEntryColumns(e)
		if err != nil {
			return fmt.Errorf("security entry %s: %w", e.Name, err)
		}
		effect := strings.ToLower(strings.TrimSpace(e.Effect))
		if effect == "" {
			effect = acl.EffectAllow
		}
		if err := perm.EnsureACLEntry(ctx, orgID, e.Role, e.Name, effect, e.Resource, actions, fields, domain, e.Priority); err != nil {
			return fmt.Errorf("security entry %s: %w", e.Name, err)
		}
	}
	for _, u := range sec.Users {
		if err := seedSecurityUser(ctx, users, perm, authSvc, orgID, u); err != nil {
			return err
		}
	}
	return nil
}

func hasUserRoles(users []appspec.SecurityUserSpec) bool {
	for _, u := range users {
		if len(u.Roles) > 0 {
			return true
		}
	}
	return false
}

func hasUserPasswords(users []appspec.SecurityUserSpec) bool {
	for _, u := range users {
		if strings.TrimSpace(u.Password) != "" {
			return true
		}
	}
	return false
}

func resolveSecurityOrg(ctx context.Context, users *ModelRegistry) (string, error) {
	if users == nil {
		return "", fmt.Errorf("security: identity models required")
	}
	orgs, err := users.ListAll(ctx, "organization")
	if err != nil {
		return "", fmt.Errorf("security: list organizations: %w", err)
	}
	if len(orgs) == 0 {
		return "", fmt.Errorf("security: no organization to seed into")
	}
	return fmt.Sprint(orgs[0]["id"]), nil
}

func encodeEntryColumns(e appspec.SecurityEntrySpec) (actions, fields, domain string, err error) {
	if len(e.Actions) == 0 {
		actions = `["*"]`
	} else {
		b, err := json.Marshal(e.Actions)
		if err != nil {
			return "", "", "", fmt.Errorf("actions: %w", err)
		}
		actions = string(b)
	}
	switch {
	case e.Fields.All:
		fields = `"*"`
	case len(e.Fields.Names) == 0:
		fields = `"*"`
	default:
		b, err := json.Marshal(e.Fields.Names)
		if err != nil {
			return "", "", "", fmt.Errorf("fields: %w", err)
		}
		fields = string(b)
	}
	if e.Domain == nil {
		domain = `[]`
	} else {
		b, err := json.Marshal(e.Domain)
		if err != nil {
			return "", "", "", fmt.Errorf("domain: %w", err)
		}
		domain = string(b)
	}
	return actions, fields, domain, nil
}

func seedSecurityUser(ctx context.Context, users *ModelRegistry, perm securityPerms, authSvc securityAuth, orgID string, u appspec.SecurityUserSpec) error {
	email := strings.TrimSpace(u.Email)
	member, err := users.FindBy(ctx, "user", "email", email)
	if err != nil {
		name := strings.TrimSpace(u.Name)
		if name == "" {
			name = email
		}
		member, err = users.Create(ctx, orgID, securitySeedAuthor, "user", map[string]any{
			"email":  email,
			"name":   name,
			"status": "active",
		})
		if err != nil {
			return fmt.Errorf("security user %s: create: %w", email, err)
		}
		log.Printf("engine: security created user %s", email)
	}
	userID := fmt.Sprint(member["id"])
	userOrg := fmt.Sprint(member["orgId"])
	if userOrg == "" {
		userOrg = orgID
	}

	if perm != nil {
		for _, role := range u.Roles {
			if err := perm.AssignRole(ctx, userID, userOrg, role); err != nil {
				return fmt.Errorf("security user %s: assign role %s: %w", email, role, err)
			}
		}
	}
	if pw := strings.TrimSpace(u.Password); pw != "" {
		if authSvc == nil {
			return fmt.Errorf("security user %s: password set but auth unavailable", email)
		}
		if err := authSvc.SetPassword(ctx, userID, pw); err != nil {
			return fmt.Errorf("security user %s: password: %w", email, err)
		}
	}
	return nil
}
