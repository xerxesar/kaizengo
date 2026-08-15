package engine

import (
	"context"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/i18n"
)

// HookContext is passed to lifecycle hooks. Fields is mutable on Before* hooks.
type HookContext struct {
	Context  context.Context
	App      appspec.AppSpec
	Model    appspec.ModelSpec
	OrgID    string
	UserID   string
	RecordID string
	Fields   map[string]any
	Record   Record
}

// T translates a catalog key (same catalogs as the SPA `t()` helper).
func (hc HookContext) T(key string) string {
	return i18n.T(key)
}

// Tf is T with fmt.Sprintf-style formatting.
func (hc HookContext) Tf(key string, args ...any) string {
	return i18n.Tf(key, args...)
}

// Hooks are optional Go callbacks for custom controller logic.
type Hooks struct {
	BeforeCreate func(HookContext) error
	AfterCreate  func(HookContext) error
	BeforeUpdate func(HookContext) error
	AfterUpdate  func(HookContext) error
	BeforeDelete func(HookContext) error
	AfterDelete  func(HookContext) error
}

// HookRegistry maps model name → hooks.
type HookRegistry struct {
	byModel map[string]Hooks
}

func NewHookRegistry() *HookRegistry {
	return &HookRegistry{byModel: make(map[string]Hooks)}
}

func (r *HookRegistry) Register(model string, h Hooks) {
	if r == nil {
		return
	}
	if r.byModel == nil {
		r.byModel = make(map[string]Hooks)
	}
	r.byModel[model] = h
}

func (r *HookRegistry) forModel(model string) Hooks {
	if r == nil {
		return Hooks{}
	}
	return r.byModel[model]
}

var (
	globalModelHooksMu sync.Mutex
	globalModelHooks   = map[string]map[string]Hooks{}
)

// RegisterModelHooks records lifecycle callbacks for a model in an app.
// Call from a model package init() so engine.New picks them up.
func RegisterModelHooks(app, model string, h Hooks) {
	if app == "" || model == "" {
		return
	}
	globalModelHooksMu.Lock()
	defer globalModelHooksMu.Unlock()
	byModel, ok := globalModelHooks[app]
	if !ok {
		byModel = map[string]Hooks{}
		globalModelHooks[app] = byModel
	}
	byModel[model] = h
}

func applyRegisteredHooks(a *App) {
	if a == nil || a.opts.AppName == "" {
		return
	}
	globalModelHooksMu.Lock()
	defer globalModelHooksMu.Unlock()
	for model, h := range globalModelHooks[a.opts.AppName] {
		a.Hooks(model, h)
	}
}

// Hooks registers lifecycle callbacks for a model. Returns a for chaining.
func (a *App) Hooks(model string, h Hooks) *App {
	if a.opts.Hooks == nil {
		a.opts.Hooks = NewHookRegistry()
	}
	a.opts.Hooks.Register(model, h)
	return a
}
