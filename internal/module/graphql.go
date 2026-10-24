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
	schema   *graphql.Schema
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

// RemoveQuery drops a root Query field (dynamic apps re-registering on re-apply).
func (g *GraphQLRegistry) RemoveQuery(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.query, name)
	g.built = nil
	g.buildErr = nil
}

// RemoveMutation drops a root Mutation field.
func (g *GraphQLRegistry) RemoveMutation(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.mutation, name)
	g.built = nil
	g.buildErr = nil
}

// QueryField returns a registered root Query field.
func (g *GraphQLRegistry) QueryField(name string) (*graphql.Field, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f, ok := g.query[name]
	return f, ok
}

// MutationField returns a registered root Mutation field.
func (g *GraphQLRegistry) MutationField(name string) (*graphql.Field, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f, ok := g.mutation[name]
	return f, ok
}

// FieldNames returns the registered root Query and Mutation field names.
func (g *GraphQLRegistry) FieldNames() (queries, mutations []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k := range g.query {
		queries = append(queries, k)
	}
	for k := range g.mutation {
		mutations = append(mutations, k)
	}
	return queries, mutations
}

// Schema returns the current schema, rebuilding it after Invalidate.
func (g *GraphQLRegistry) Schema() (*graphql.Schema, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.rebuildLocked(); err != nil {
		return nil, err
	}
	return g.schema, nil
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
		g.schema = nil
		return err
	}
	g.buildErr = nil
	g.schema = &schema
	g.built = gqlhandler.New(&gqlhandler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	return nil
}
