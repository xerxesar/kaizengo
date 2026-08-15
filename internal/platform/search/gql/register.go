package gql

import (
	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/search"

	"github.com/graphql-go/graphql"
)

// Register adds platform search queries to the host GraphQL schema.
func Register(host *module.Host) {
	hitType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchHit",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.Hit).ID, nil
				},
			},
			"collection": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.Hit).Collection, nil
				},
			},
			"title": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.Hit).Title, nil
				},
			},
			"snippet": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.Hit).Snippet, nil
				},
			},
			"score": &graphql.Field{
				Type: graphql.Float,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(search.Hit).Score, nil
				},
			},
		},
	})

	host.GQL.RegisterQuery("search", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(hitType))),
		Args: graphql.FieldConfigArgument{
			"q": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"collections": &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.NewNonNull(graphql.String)),
			},
			"limit": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := auth.MustPrincipal(p.Context)
			if err != nil {
				return nil, err
			}
			q, _ := p.Args["q"].(string)
			limit, _ := p.Args["limit"].(int)
			var collections []string
			if raw, ok := p.Args["collections"].([]any); ok {
				for _, v := range raw {
					if s, ok := v.(string); ok && s != "" {
						collections = append(collections, s)
					}
				}
			}
			return search.Query(p.Context, pr.OrgID, q, collections, limit)
		},
	})

	host.GQL.RegisterQuery("searchBackend", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := auth.MustPrincipal(p.Context); err != nil {
				return nil, err
			}
			return search.BackendName(), nil
		},
	})
}
