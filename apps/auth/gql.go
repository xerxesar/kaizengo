package auth

import (
	"fmt"

	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

func registerGQL(host *module.Host, svc *Service) {
	host.GQL.RegisterQuery("me", &graphql.Field{
		Type: graphql.NewNonNull(newAuthUserType()),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return svc.Me(p.Context)
		},
	})
}

func newAuthUserType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "AuthUser",
		Fields: graphql.Fields{
			"id":    nonNullString(func(p graphql.ResolveParams) string { return sourceAuthUser(p).ID }),
			"orgId": nonNullString(func(p graphql.ResolveParams) string { return sourceAuthUser(p).OrgID }),
			"email": nonNullString(func(p graphql.ResolveParams) string { return sourceAuthUser(p).Email }),
			"name":  nonNullString(func(p graphql.ResolveParams) string { return sourceAuthUser(p).Name }),
			"roles": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return sourceAuthUser(p).Roles, nil
				},
			},
		},
	})
}

func nonNullString(fn func(graphql.ResolveParams) string) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return fn(p), nil
		},
	}
}

func sourceAuthUser(p graphql.ResolveParams) AuthUser {
	switch v := p.Source.(type) {
	case AuthUser:
		return v
	case *AuthUser:
		return *v
	default:
		panic(fmt.Sprintf("expected AuthUser, got %T", p.Source))
	}
}
