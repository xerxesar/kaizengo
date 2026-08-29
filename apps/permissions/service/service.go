package service

import (
	"context"
	"fmt"

	"kaizengo/internal/auth"
	"kaizengo/packages/sdk-go/acl"
)

const Name = acl.ServiceName

const (
	RoleAdmin  = "admin"
	RoleMember = "member"

	ActRead   = acl.ActRead
	ActCreate = acl.ActCreate
	ActUpdate = acl.ActUpdate
	ActDelete = acl.ActDelete
	ActAll    = acl.ActAll
)

// EntrySpec is a declarative acl_entry used by apps to seed policies.
type EntrySpec struct {
	Name     string
	Effect   string // allow|deny
	Resource string
	Actions  string // JSON array
	Fields   string // JSON or "*"
	Domain   string // JSON domain
	Priority int
}

// Store loads and mutates policy data (implemented with engine + WithInternal).
type Store interface {
	ListRoleNames(ctx context.Context, userID, orgID string) ([]string, error)
	ListEntriesForUser(ctx context.Context, userID, orgID string) ([]acl.Entry, error)
	EnsureRole(ctx context.Context, orgID, authorID, name, label string) (roleID string, err error)
	EnsureACL(ctx context.Context, orgID, authorID, roleID, name, effect, resource, actions, fields, domain string, priority int) error
	DisableACLByName(ctx context.Context, orgID, name string) error
	AssignUserRoleID(ctx context.Context, orgID, authorID, userID, roleID string) error
	FindRoleID(ctx context.Context, orgID, name string) (string, error)
	UserHasRole(ctx context.Context, orgID, userID, roleID string) (bool, error)
}

// Service evaluates unified ACL policies.
type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Roles(ctx context.Context, userID, orgID string) ([]string, error) {
	return s.store.ListRoleNames(ctx, userID, orgID)
}

func (s *Service) Can(ctx context.Context, check acl.Check) (acl.Decision, error) {
	return s.evaluate(ctx, check, acl.Evaluate)
}

func (s *Service) CanCatalog(ctx context.Context, check acl.Check) (bool, error) {
	d, err := s.evaluate(ctx, check, acl.EvaluateCatalog)
	if err != nil {
		return false, err
	}
	return d.Allowed, nil
}

func (s *Service) evaluate(ctx context.Context, check acl.Check, eval func([]acl.Entry, acl.Check, acl.PrincipalContext) acl.Decision) (acl.Decision, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return acl.Decision{}, nil
	}
	orgID := check.OrgID
	if orgID == "" {
		orgID = p.OrgID
	}
	if p.OrgID != orgID {
		return acl.Decision{}, nil
	}
	entries, err := s.store.ListEntriesForUser(ctx, p.UserID, orgID)
	if err != nil {
		return acl.Decision{}, err
	}
	roles, err := s.store.ListRoleNames(ctx, p.UserID, orgID)
	if err != nil {
		return acl.Decision{}, err
	}
	pc := acl.PrincipalContext{UserID: p.UserID, OrgID: orgID, Roles: roles}
	return eval(entries, check, pc), nil
}

// CanResource is a thin wrapper for callers that only pass org/resource/action.
func (s *Service) CanResource(ctx context.Context, orgID, resource, action string) (bool, error) {
	d, err := s.Can(ctx, acl.Check{OrgID: orgID, Resource: resource, Action: action})
	if err != nil {
		return false, err
	}
	return d.Allowed, nil
}

func (s *Service) MustAllow(ctx context.Context, check acl.Check) error {
	d, err := s.Can(ctx, check)
	if err != nil {
		return err
	}
	if !d.Allowed {
		return auth.ErrForbidden
	}
	return nil
}

// MustAllowResource keeps the old three-string signature for appman and custom resolvers.
func (s *Service) MustAllowResource(ctx context.Context, orgID, resource, action string) error {
	return s.MustAllow(ctx, acl.Check{OrgID: orgID, Resource: resource, Action: action})
}

func (s *Service) ListDomain(ctx context.Context, orgID, resource, action string) (acl.ListFilter, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return acl.ListFilter{DenyAll: true}, nil
	}
	if p.OrgID != orgID {
		return acl.ListFilter{DenyAll: true}, nil
	}
	entries, err := s.store.ListEntriesForUser(ctx, p.UserID, orgID)
	if err != nil {
		return acl.ListFilter{}, err
	}
	roles, err := s.store.ListRoleNames(ctx, p.UserID, orgID)
	if err != nil {
		return acl.ListFilter{}, err
	}
	pc := acl.PrincipalContext{UserID: p.UserID, OrgID: orgID, Roles: roles}
	return acl.BuildListFilter(entries, resource, action, pc), nil
}

// DeniedFields returns explicit field denies when * is allowed (for masking).
func (s *Service) DeniedFields(ctx context.Context, orgID, resource, action string, record map[string]any) ([]string, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, nil
	}
	entries, err := s.store.ListEntriesForUser(ctx, p.UserID, orgID)
	if err != nil {
		return nil, err
	}
	roles, err := s.store.ListRoleNames(ctx, p.UserID, orgID)
	if err != nil {
		return nil, err
	}
	pc := acl.PrincipalContext{UserID: p.UserID, OrgID: orgID, Roles: roles}
	return acl.DeniedFields(entries, resource, action, pc, record), nil
}

// EnsureRole upserts a role by name (apps call this from security.yaml / Setup).
func (s *Service) EnsureRole(ctx context.Context, orgID, name, label string) (string, error) {
	if label == "" {
		label = name
	}
	return s.store.EnsureRole(ctx, orgID, seedAuthor, name, label)
}

// EnsureACLEntry upserts an acl_entry using JSON-string columns (engine security seed).
func (s *Service) EnsureACLEntry(ctx context.Context, orgID, roleName, name, effect, resource, actions, fields, domain string, priority int) error {
	return s.EnsureEntry(ctx, orgID, roleName, EntrySpec{
		Name:     name,
		Effect:   effect,
		Resource: resource,
		Actions:  actions,
		Fields:   fields,
		Domain:   domain,
		Priority: priority,
	})
}

// EnsureEntry upserts an acl_entry for a role by name (apps call this from Setup).
func (s *Service) EnsureEntry(ctx context.Context, orgID, roleName string, spec EntrySpec) error {
	roleID, err := s.store.FindRoleID(ctx, orgID, roleName)
	if err != nil {
		return err
	}
	effect := spec.Effect
	if effect == "" {
		effect = acl.EffectAllow
	}
	actions := spec.Actions
	if actions == "" {
		actions = `["*"]`
	}
	fields := spec.Fields
	if fields == "" {
		fields = `"*"`
	}
	domain := spec.Domain
	if domain == "" {
		domain = `[]`
	}
	return s.store.EnsureACL(ctx, orgID, seedAuthor, roleID, spec.Name, effect, spec.Resource, actions, fields, domain, spec.Priority)
}

// DisableEntry deactivates a seeded acl_entry by name (e.g. replace a broad grant).
func (s *Service) DisableEntry(ctx context.Context, orgID, name string) error {
	return s.store.DisableACLByName(ctx, orgID, name)
}

func (s *Service) AssignRole(ctx context.Context, userID, orgID, roleName string) error {
	p, _ := auth.PrincipalFrom(ctx)
	author := seedAuthor
	if p != nil {
		author = p.UserID
	}
	roleID, err := s.store.FindRoleID(ctx, orgID, roleName)
	if err != nil {
		return err
	}
	ok, err := s.store.UserHasRole(ctx, orgID, userID, roleID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return s.store.AssignUserRoleID(ctx, orgID, author, userID, roleID)
}

func (s *Service) SeedAdmin(ctx context.Context, userID, orgID string) error {
	return s.SeedDefaults(ctx, orgID, userID)
}

// SeedDefaults ensures admin/member roles, baseline acl_entry rows, and admin assignment.
func (s *Service) SeedDefaults(ctx context.Context, orgID, adminUserID string) error {
	adminID, err := s.store.EnsureRole(ctx, orgID, seedAuthor, RoleAdmin, "Administrator")
	if err != nil {
		return err
	}
	memberID, err := s.store.EnsureRole(ctx, orgID, seedAuthor, RoleMember, "Member")
	if err != nil {
		return err
	}

	type seedACL struct {
		name, effect, resource, actions, fields, domain string
		priority                                        int
		roleID                                          string
	}
	seeds := []seedACL{
		{name: "admin-all", effect: acl.EffectAllow, resource: "*", actions: `["*"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: adminID},
		// member: identity read
		{name: "member-identity-user-read", effect: acl.EffectAllow, resource: "identity.user", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		{name: "member-identity-organization-read", effect: acl.EffectAllow, resource: "identity.organization", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		{name: "member-identity-org-unit-read", effect: acl.EffectAllow, resource: "identity.org_unit", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		{name: "member-identity-membership-read", effect: acl.EffectAllow, resource: "identity.membership", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		// hellospec policies are seeded by apps/hellospec (own-record + field deny demo)
		// member: inventory read (app wildcard)
		{name: "member-inventory-read", effect: acl.EffectAllow, resource: "inventory.*", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		// member: appman read
		{name: "member-appman-read", effect: acl.EffectAllow, resource: "appman", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		// member: settings read
		{name: "member-settings-read", effect: acl.EffectAllow, resource: "settings.*", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
		{name: "member-permissions-catalog-read", effect: acl.EffectAllow, resource: "permissions.catalog", actions: `["read"]`, fields: `"*"`, domain: `[]`, priority: 0, roleID: memberID},
	}
	for _, row := range seeds {
		if err := s.store.EnsureACL(ctx, orgID, seedAuthor, row.roleID, row.name, row.effect, row.resource, row.actions, row.fields, row.domain, row.priority); err != nil {
			return fmt.Errorf("seed acl %s: %w", row.name, err)
		}
	}

	if adminUserID != "" {
		ok, err := s.store.UserHasRole(ctx, orgID, adminUserID, adminID)
		if err != nil {
			return err
		}
		if !ok {
			if err := s.store.AssignUserRoleID(ctx, orgID, seedAuthor, adminUserID, adminID); err != nil {
				return err
			}
		}
	}
	return nil
}

const seedAuthor = "00000000-0000-0000-0000-000000000001"

// Ensure Authorizer interface compliance.
var _ acl.Authorizer = (*Service)(nil)
