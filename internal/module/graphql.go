package module

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/graphql-go/graphql"
	gqlhandler "github.com/graphql-go/handler"
)

// GraphQLRegistry lets apps contribute Query/Mutation fields without
// modifying core. Fields are registered in Setup; the handler is rebuilt
// after a live install so new fields appear without a process restart.
type GraphQLRegistry struct {
	mu       sync.Mutex
	query    graphql.Fields
	mutation graphql.Fields
	built    http.Handler
	buildErr error
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
	g.built = nil
	g.buildErr = nil
}

// RegisterMutation adds a root Mutation field (call from App.Setup).
func (g *GraphQLRegistry) RegisterMutation(name string, field *graphql.Field) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.mutation[name]; exists {
		panic(fmt.Sprintf("graphql: duplicate Mutation field %q", name))
	}
	g.mutation[name] = field
	g.built = nil
	g.buildErr = nil
}

// Invalidate drops the cached schema so the next request rebuilds it.
func (g *GraphQLRegistry) Invalidate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.built = nil
	g.buildErr = nil
}

// Handler returns a live handler that rebuilds the schema after Invalidate.
func (g *GraphQLRegistry) Handler() (http.Handler, error) {
	g.mu.Lock()
	err := g.rebuildLocked()
	g.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(g.serve), nil
}

func (g *GraphQLRegistry) serve(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	err := g.rebuildLocked()
	h := g.built
	g.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.ServeHTTP(w, r)
}

func (g *GraphQLRegistry) rebuildLocked() error {
	if g.built != nil {
		return g.buildErr
	}

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
		g.built = nil
		return err
	}
	g.buildErr = nil
	g.built = gqlhandler.New(&gqlhandler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	return nil
}
