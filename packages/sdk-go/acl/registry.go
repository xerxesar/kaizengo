package acl

import (
	"sort"
	"strings"
	"sync"
)

// ResourceKind classifies a securable surface in the platform.
type ResourceKind string

const (
	KindModel    ResourceKind = "model"
	KindMenu     ResourceKind = "menu"
	KindView     ResourceKind = "view"
	KindQuery    ResourceKind = "query"
	KindMutation ResourceKind = "mutation"
	KindEvent    ResourceKind = "event"
	KindNav      ResourceKind = "nav"
	KindApp      ResourceKind = "app"
	KindAPI      ResourceKind = "api"
)

// ResourceDescriptor is one registered securable target for ACL policies.
type ResourceDescriptor struct {
	App         string
	Kind        ResourceKind
	Name        string
	Resource    string
	Label       string
	Description string
	Actions     []string
	Surface     string
}

var defaultRegistry = NewRegistry()

// Registry holds every securable resource registered at runtime.
type Registry struct {
	mu      sync.RWMutex
	byKey   map[string]ResourceDescriptor
	actions map[string]map[string]struct{}
}

// NewRegistry returns an empty resource registry.
func NewRegistry() *Registry {
	return &Registry{
		byKey:   map[string]ResourceDescriptor{},
		actions: map[string]map[string]struct{}{},
	}
}

// DefaultRegistry is the process-wide ACL resource catalog.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register records a securable resource. Duplicate resource ids merge actions.
func Register(desc ResourceDescriptor) {
	defaultRegistry.Register(desc)
}

// RegisterOperation records one enforced resource/action pair (e.g. from RequireAction).
func RegisterOperation(resource, action, surface, operation string) {
	defaultRegistry.RegisterOperation(resource, action, surface, operation)
}

// All returns every registered resource sorted by resource id.
func All() []ResourceDescriptor {
	return defaultRegistry.All()
}

// ByApp returns resources for one app.
func ByApp(app string) []ResourceDescriptor {
	return defaultRegistry.ByApp(app)
}

// Actions returns the sorted list of known action verbs.
func Actions() []string {
	return defaultRegistry.Actions()
}

func (r *Registry) Register(desc ResourceDescriptor) {
	desc.Resource = strings.TrimSpace(desc.Resource)
	if desc.Resource == "" {
		return
	}
	if desc.App == "" {
		desc.App = inferApp(desc.Resource)
	}
	if desc.Kind == "" {
		desc.Kind = inferKind(desc.Resource)
	}
	if desc.Name == "" {
		desc.Name = inferName(desc.Resource, desc.Kind)
	}
	if desc.Label == "" {
		desc.Label = desc.Resource
	}
	desc.Actions = normalizeActions(desc.Actions)
	if desc.Surface == "" {
		desc.Surface = "internal"
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.byKey[desc.Resource]
	if ok {
		desc.Actions = mergeActions(existing.Actions, desc.Actions)
		if desc.Label == desc.Resource && existing.Label != "" {
			desc.Label = existing.Label
		}
		if desc.Description == "" {
			desc.Description = existing.Description
		}
	}
	r.byKey[desc.Resource] = desc
	for _, action := range desc.Actions {
		r.trackAction(action)
	}
}

func (r *Registry) RegisterOperation(resource, action, surface, operation string) {
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" || action == "" {
		return
	}
	desc := ResourceDescriptor{
		App:      inferApp(resource),
		Kind:     inferKind(resource),
		Name:     inferName(resource, inferKind(resource)),
		Resource: resource,
		Label:    resource,
		Actions:  []string{action},
		Surface:  surface,
	}
	if operation != "" {
		desc.Description = operation
	}
	r.Register(desc)
}

func (r *Registry) All() []ResourceDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ResourceDescriptor, 0, len(r.byKey))
	for _, desc := range r.byKey {
		out = append(out, desc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].App != out[j].App {
			return out[i].App < out[j].App
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Resource < out[j].Resource
	})
	return out
}

func (r *Registry) ByApp(app string) []ResourceDescriptor {
	app = strings.TrimSpace(app)
	all := r.All()
	if app == "" {
		return all
	}
	out := make([]ResourceDescriptor, 0)
	for _, desc := range all {
		if desc.App == app {
			out = append(out, desc)
		}
	}
	return out
}

func (r *Registry) Actions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.actions))
	for action := range r.actions {
		out = append(out, action)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) trackAction(action string) {
	action = strings.TrimSpace(action)
	if action == "" {
		return
	}
	if r.actions[action] == nil {
		r.actions[action] = map[string]struct{}{}
	}
	r.actions[action][action] = struct{}{}
}

func mergeActions(existing, extra []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existing)+len(extra))
	for _, list := range [][]string{existing, extra} {
		for _, action := range list {
			action = strings.TrimSpace(action)
			if action == "" {
				continue
			}
			if _, ok := seen[action]; ok {
				continue
			}
			seen[action] = struct{}{}
			out = append(out, action)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeActions(actions []string) []string {
	if len(actions) == 0 {
		return CRUDActions()
	}
	return mergeActions(nil, actions)
}

func inferApp(resource string) string {
	parts := strings.Split(resource, ".")
	if len(parts) == 0 {
		return resource
	}
	return parts[0]
}

func inferKind(resource string) ResourceKind {
	parts := strings.Split(resource, ".")
	if len(parts) == 1 {
		return KindApp
	}
	switch parts[1] {
	case "menu":
		return KindMenu
	case "view":
		return KindView
	case "query":
		return KindQuery
	case "mutation":
		return KindMutation
	case "event":
		return KindEvent
	case "nav":
		return KindNav
	case "api":
		return KindAPI
	default:
		if len(parts) == 2 {
			return KindModel
		}
		return KindAPI
	}
}

func inferName(resource string, kind ResourceKind) string {
	parts := strings.Split(resource, ".")
	switch kind {
	case KindApp:
		return parts[0]
	case KindModel:
		if len(parts) >= 2 {
			return parts[1]
		}
	case KindMenu, KindView, KindQuery, KindMutation, KindEvent, KindNav, KindAPI:
		if len(parts) >= 3 {
			return strings.Join(parts[2:], ".")
		}
	}
	if len(parts) == 0 {
		return resource
	}
	return parts[len(parts)-1]
}

// ModelResource is the default ACL id for a model: "{app}.{model}".
func ModelResource(app, model string) string {
	return strings.TrimSpace(app) + "." + strings.TrimSpace(model)
}

// AppResource is the app-level ACL id.
func AppResource(app string) string {
	return strings.TrimSpace(app)
}

// MenuResource identifies an in-app menubar entry (app.yaml menus:).
func MenuResource(app, menuID string) string {
	return strings.TrimSpace(app) + ".menu." + strings.TrimSpace(menuID)
}

// NavResource identifies the shell Apps dropdown entry (app.yaml nav:).
func NavResource(app string) string {
	return strings.TrimSpace(app) + ".nav"
}

// ViewResource identifies a UI view/page.
func ViewResource(app, viewName string) string {
	return strings.TrimSpace(app) + ".view." + strings.TrimSpace(viewName)
}

// QueryResource identifies a GraphQL query surface.
func QueryResource(app, queryName string) string {
	return strings.TrimSpace(app) + ".query." + strings.TrimSpace(queryName)
}

// MutationResource identifies a GraphQL mutation surface.
func MutationResource(app, mutationName string) string {
	return strings.TrimSpace(app) + ".mutation." + strings.TrimSpace(mutationName)
}

// EventResource identifies an event stream/type.
func EventResource(app, eventName string) string {
	return strings.TrimSpace(app) + ".event." + strings.TrimSpace(eventName)
}
