// Package acl is the Enterprise Runtime authorization primitive:
// unified policy evaluation (model, field, record, and future workflow actions).
// Policy rows are stored by apps/permissions; this package has no database.
package acl

import "context"

const (
	ServiceName = "permissions"

	EffectAllow = "allow"
	EffectDeny  = "deny"

	ActRead    = "read"
	ActCreate  = "create"
	ActUpdate  = "update"
	ActDelete  = "delete"
	ActExecute = "execute"
	ActAll     = "*"

	ResAll = "*"

	FieldsAll = "*"
)

// Entry is one unified ACL policy row (same shape for model, field, and record rules).
type Entry struct {
	ID       string
	RoleID   string
	RoleName string // denormalized slug for $user.roles context
	Effect   string // allow|deny
	Resource string
	Actions  []string // empty or ["*"] = all actions
	Fields   []string // empty or ["*"] = all fields
	Domain   Domain   // empty = all records
	Priority int
	Active   bool
}

// Check is an authorization request.
type Check struct {
	OrgID    string
	Resource string
	Action   string
	Record   map[string]any // optional; enables domain evaluation
	Fields   []string       // fields touched on write; empty = whole-record / read
}

// Decision is the result of evaluating ACL entries.
// FieldsRead / FieldsWrite: nil means all fields; non-nil empty means none.
type Decision struct {
	Allowed     bool
	FieldsRead  []string
	FieldsWrite []string
}

// PrincipalContext is substituted into domains ($user.*).
type PrincipalContext struct {
	UserID string
	OrgID  string
	Roles  []string
}

// Authorizer is implemented by apps/permissions and consumed by the engine.
type Authorizer interface {
	Can(ctx context.Context, check Check) (Decision, error)
	CanCatalog(ctx context.Context, check Check) (bool, error)
	MustAllow(ctx context.Context, check Check) error
	ListDomain(ctx context.Context, orgID, resource, action string) (ListFilter, error)
	DeniedFields(ctx context.Context, orgID, resource, action string, record map[string]any) ([]string, error)
}

// ListFilter describes how to restrict List queries.
type ListFilter struct {
	// DenyAll means no rows are visible.
	DenyAll bool
	// Unrestricted means no domain filter (all org rows allowed by grants).
	Unrestricted bool
	// Allow domains are OR'd; Deny domains are OR'd and negated.
	Allow []Domain
	Deny  []Domain
}
