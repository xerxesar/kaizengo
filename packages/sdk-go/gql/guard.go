package gql

import (
	"context"
	"fmt"

	"kaizengo/internal/auth"
	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

type Authorizer interface {
	MustAllow(ctx context.Context, orgID, resource, action string) error
}

func RequirePrincipal(p graphql.ResolveParams) (*auth.Principal, error) {
	return auth.MustPrincipal(p.Context)
}

func LookupAuthorizer(host *module.Host, serviceName string) (Authorizer, error) {
	raw, ok := host.Lookup(serviceName)
	if !ok {
		return nil, fmt.Errorf("authorizer service %q not found", serviceName)
	}
	authz, ok := raw.(Authorizer)
	if !ok {
		return nil, fmt.Errorf("service %q does not implement Authorizer", serviceName)
	}
	return authz, nil
}

func RequireAction(host *module.Host, serviceName string, p graphql.ResolveParams, resource, action string) (*auth.Principal, error) {
	pr, err := RequirePrincipal(p)
	if err != nil {
		return nil, err
	}
	authz, err := LookupAuthorizer(host, serviceName)
	if err != nil {
		return nil, err
	}
	if err := authz.MustAllow(p.Context, pr.OrgID, resource, action); err != nil {
		return nil, err
	}
	return pr, nil
}
