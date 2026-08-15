package service

import (
	"context"
	"fmt"
	"strings"

	types "kaizengo/apps/permissions/__types__"
	"kaizengo/internal/auth"
)

const Name = "permissions"

const (
	RoleAdmin  = string(types.UserRoleRoleAdmin)
	RoleMember = string(types.UserRoleRoleMember)

	ResOrg      = "identity.organization"
	ResOrgUnits = "identity.org_units"
	ResUsers    = "identity.users"
	ResCounter  = "counter"
	ResNotes    = "notes"
	ResHello    = "hello"
	ResAll      = "*"

	ActRead   = "read"
	ActCreate = "create"
	ActUpdate = "update"
	ActDelete = "delete"
	ActAll    = "*"
)

// Store lists and assigns roles (implemented with engine models in the app module).
type Store interface {
	ListRoles(ctx context.Context, userID, orgID string) ([]string, error)
	AssignUserRole(ctx context.Context, orgID, userID, role string) error
}

// Service evaluates role-based access control.
type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Roles(ctx context.Context, userID, orgID string) ([]string, error) {
	return s.store.ListRoles(ctx, userID, orgID)
}

func (s *Service) Can(ctx context.Context, orgID, resource, action string) (bool, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return false, nil
	}
	if p.OrgID != orgID {
		return false, nil
	}
	roles, err := s.store.ListRoles(ctx, p.UserID, orgID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if roleAllows(role, resource, action) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) MustAllow(ctx context.Context, orgID, resource, action string) error {
	ok, err := s.Can(ctx, orgID, resource, action)
	if err != nil {
		return err
	}
	if !ok {
		return auth.ErrForbidden
	}
	return nil
}

func (s *Service) AssignRole(ctx context.Context, userID, orgID, role string) error {
	if !validRole(role) {
		return fmt.Errorf("unknown role %q", role)
	}
	return s.store.AssignUserRole(ctx, orgID, userID, role)
}

func (s *Service) SeedAdmin(ctx context.Context, userID, orgID string) error {
	return s.AssignRole(ctx, userID, orgID, RoleAdmin)
}

func roleAllows(role, resource, action string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if match(p.resource, resource) && match(p.action, action) {
			return true
		}
	}
	return false
}

func match(pattern, value string) bool {
	return pattern == ActAll || strings.EqualFold(pattern, value)
}

func validRole(role string) bool {
	return types.UserRoleRole(role).Valid()
}

type permission struct {
	resource string
	action   string
}

var rolePermissions = map[string][]permission{
	RoleAdmin: {
		{ResAll, ActAll},
	},
	RoleMember: {
		{ResOrg, ActRead},
		{ResOrgUnits, ActRead},
		{ResUsers, ActRead},
		{ResCounter, ActRead},
		{ResNotes, ActRead},
		{ResNotes, ActCreate},
		{ResNotes, ActUpdate},
		{ResNotes, ActDelete},
		{ResHello, ActRead},
		{ResHello, ActCreate},
		{ResHello, ActUpdate},
		{ResHello, ActDelete},
		{"appman", ActRead},
	},
}
