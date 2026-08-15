package engine

import (
	"context"

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

// Hooks registers lifecycle callbacks for a model. Returns a for chaining.
func (a *App) Hooks(model string, h Hooks) *App {
	if a.opts.Hooks == nil {
		a.opts.Hooks = NewHookRegistry()
	}
	a.opts.Hooks.Register(model, h)
	return a
}
