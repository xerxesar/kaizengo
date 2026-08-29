package engine

import (
	"context"
	"fmt"
	"sync"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"

	"github.com/graphql-go/graphql"
)

// RegisteredModel declares a code-backed model for KForm/KTable and {app}Views.
type RegisteredModel struct {
	Name        string
	Fields      []appspec.FieldSpec
	ListColumns []views.Column
	// Resource is the permissions resource (e.g. identity.users). Defaults to spec.Resource.
	Resource string
	// ObjectType is the GraphQL output type for CRUD operations. Required.
	ObjectType *graphql.Object
	// Search enables platform search indexing for this model.
	Search *appspec.SearchSpec
	// ToRecord maps a handler result to a flat record for search indexing.
	ToRecord func(any) map[string]any
	// Reindex rebuilds the search index (optional; used from search settings).
	Reindex func(ctx context.Context) (int, error)

	List   ModelListFunc
	Get    ModelGetFunc
	Create ModelCreateFunc
	Update ModelUpdateFunc
	Delete ModelDeleteFunc
}

// RequestContext carries auth scope for registered model handlers.
type RequestContext struct {
	Context context.Context
	OrgID   string
	UserID  string
}

type ModelListFunc func(ctx RequestContext) ([]any, error)
type ModelGetFunc func(ctx RequestContext, id string) (any, error)
type ModelCreateFunc func(ctx RequestContext, args map[string]any) (any, error)
type ModelUpdateFunc func(ctx RequestContext, id string, args map[string]any) (any, error)
type ModelDeleteFunc func(ctx RequestContext, id string) error

type modelRegistry struct {
	mu     sync.RWMutex
	byApp  map[string][]RegisteredModel
	byName map[string]map[string]RegisteredModel
}

var registry modelRegistry = modelRegistry{
	byApp:  map[string][]RegisteredModel{},
	byName: map[string]map[string]RegisteredModel{},
}

// RegisterModel wires GraphQL CRUD for a code-backed model.
// When the model is also declared in app.yaml, fields/search default from the
// YAML entry so handlers only need to supply persistence logic.
func RegisterModel(host *module.Host, spec appspec.AppSpec, m RegisteredModel) error {
	if m.Name == "" {
		return fmt.Errorf("registered model name is required")
	}
	if m.ObjectType == nil {
		return fmt.Errorf("registered model %q requires ObjectType", m.Name)
	}
	if m.List == nil {
		return fmt.Errorf("registered model %q requires List handler", m.Name)
	}
	m = mergeRegisteredModel(spec, m)
	if len(m.Fields) == 0 {
		return fmt.Errorf("registered model %q has no fields (declare them in app.yaml or RegisterModel)", m.Name)
	}

	registry.mu.Lock()
	if registry.byName[spec.Name] == nil {
		registry.byName[spec.Name] = map[string]RegisteredModel{}
	}
	if _, ok := registry.byName[spec.Name][m.Name]; ok {
		registry.mu.Unlock()
		return fmt.Errorf("registered model %q already exists for app %q", m.Name, spec.Name)
	}
	registry.byName[spec.Name][m.Name] = m
	registry.byApp[spec.Name] = append(registry.byApp[spec.Name], m)
	registry.mu.Unlock()

	registerModelSearch(spec.Name, m)
	registerRegisteredModelResource(spec.Name, m)
	registerRegisteredModelGQL(host, spec, m)
	return nil
}

func mergeRegisteredModel(spec appspec.AppSpec, m RegisteredModel) RegisteredModel {
	for _, existing := range spec.Models {
		if existing.Name != m.Name {
			continue
		}
		if len(m.Fields) == 0 {
			m.Fields = existing.Fields
		}
		if m.Search == nil {
			m.Search = existing.Search
		}
		break
	}
	return m
}

func registeredModels(app string) []RegisteredModel {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	out := make([]RegisteredModel, len(registry.byApp[app]))
	copy(out, registry.byApp[app])
	return out
}

func registeredModelByName(app, name string) (RegisteredModel, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	m, ok := registry.byName[app][name]
	return m, ok
}

func registeredModelSpec(m RegisteredModel) appspec.ModelSpec {
	return appspec.ModelSpec{Name: m.Name, Fields: m.Fields, Search: m.Search}
}
