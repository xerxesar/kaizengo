package gql

import (
	"fmt"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"

	"github.com/graphql-go/graphql"
)

// Authorizer is the host-bag RBAC service (apps/permissions).
type Authorizer = acl.Authorizer

func RequirePrincipal(p graphql.ResolveParams) (*auth.Principal, error) {
	return auth.MustPrincipal(p.Context)
}

func LookupAuthorizer(host *module.Host, serviceName string) (acl.Authorizer, error) {
	if serviceName == "" {
		serviceName = acl.ServiceName
	}
	raw, ok := host.Lookup(serviceName)
	if !ok {
		return nil, fmt.Errorf("authorizer service %q not found", serviceName)
	}
	authz, ok := raw.(acl.Authorizer)
	if !ok {
		return nil, fmt.Errorf("service %q does not implement acl.Authorizer", serviceName)
	}
	return authz, nil
}

// RequireAction ensures a session and checks ACL for resource/action (non-model APIs).
func RequireAction(host *module.Host, serviceName string, p graphql.ResolveParams, resource, action string) (*auth.Principal, error) {
	operation := ""
	if p.Info.FieldName != "" {
		operation = p.Info.FieldName
	}
	acl.RegisterOperation(resource, action, "graphql", operation)

	pr, err := RequirePrincipal(p)
	if err != nil {
		return nil, err
	}
	authz, err := LookupAuthorizer(host, serviceName)
	if err != nil {
		return nil, err
	}
	if err := authz.MustAllow(p.Context, acl.Check{
		OrgID:    pr.OrgID,
		Resource: resource,
		Action:   action,
	}); err != nil {
		return nil, err
	}
	return pr, nil
}
