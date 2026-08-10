package module

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/graphql-go/graphql"
	gqlhandler "github.com/graphql-go/handler"
)

// GraphQLRegistry lets apps contribute Query/Mutation fields without
// modifying core. Fields are registered in Setup; the handler is built later.
type GraphQLRegistry struct {
	mu        sync.Mutex
	query     graphql.Fields
	mutation  graphql.Fields
	built     http.Handler
	buildErr  error
	buildOnce sync.Once
}

func newGraphQLRegistry() *GraphQLRegistry {
	return &GraphQLRegistry{
		query:    graphql.Fields{},
		mutation: graphql.Fields{},
	}
}

// RegisterQuery adds a root Query field (call from App.Setup).
func (g *GraphQLRegistry) RegisterQuery(name string, field *graphql.Field) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.query[name]; exists {
		panic(fmt.Sprintf("graphql: duplicate Query field %q", name))
	}
	g.query[name] = field
}

// RegisterMutation adds a root Mutation field (call from App.Setup).
func (g *GraphQLRegistry) RegisterMutation(name string, field *graphql.Field) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.mutation[name]; exists {
		panic(fmt.Sprintf("graphql: duplicate Mutation field %q", name))
	}
	g.mutation[name] = field
}

// Handler builds (once) an HTTP handler for the combined schema.
func (g *GraphQLRegistry) Handler() (http.Handler, error) {
	g.buildOnce.Do(func() {
		g.mu.Lock()
		defer g.mu.Unlock()

		queryFields := graphql.Fields{}
		for k, v := range g.query {
			queryFields[k] = v
		}
		mutationFields := graphql.Fields{}
		for k, v := range g.mutation {
			mutationFields[k] = v
		}

		root := graphql.ObjectConfig{Name: "Query", Fields: queryFields}
		if len(queryFields) == 0 {
			// GraphQL requires at least one query field.
			root.Fields = graphql.Fields{
				"_empty": &graphql.Field{
					Type: graphql.Boolean,
					Resolve: func(graphql.ResolveParams) (any, error) {
						return true, nil
					},
				},
			}
		}

		schemaConfig := graphql.SchemaConfig{
			Query: graphql.NewObject(root),
		}
		if len(mutationFields) > 0 {
			schemaConfig.Mutation = graphql.NewObject(graphql.ObjectConfig{
				Name:   "Mutation",
				Fields: mutationFields,
			})
		}

		schema, err := graphql.NewSchema(schemaConfig)
		if err != nil {
			g.buildErr = err
			return
		}
		g.built = gqlhandler.New(&gqlhandler.Config{
			Schema:   &schema,
			Pretty:   true,
			GraphiQL: true,
		})
	})
	return g.built, g.buildErr
}
