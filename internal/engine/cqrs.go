package engine

import (
	"context"
	"fmt"
	"sync"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
)

// HandlerCtx is passed to custom query/command handlers.
type HandlerCtx struct {
	Context context.Context
	OrgID   string
	UserID  string
	Models  *ModelRegistry
	Host    *module.Host
	Spec    appspec.AppSpec
}

// QueryHandler implements a named query declared without list/get shorthand.
type QueryHandler func(ctx HandlerCtx, args map[string]any) (any, error)

// CommandHandler implements a named command declared without create/update/delete shorthand.
type CommandHandler func(ctx HandlerCtx, args map[string]any) (any, error)

type handlerRegistry struct {
	queries  map[string]QueryHandler
	commands map[string]CommandHandler
}

func newHandlerRegistry() *handlerRegistry {
	return &handlerRegistry{
		queries:  map[string]QueryHandler{},
		commands: map[string]CommandHandler{},
	}
}

func (r *handlerRegistry) registerQuery(name string, fn QueryHandler) {
	if r == nil || name == "" || fn == nil {
		return
	}
	if r.queries == nil {
		r.queries = map[string]QueryHandler{}
	}
	r.queries[name] = fn
}

func (r *handlerRegistry) registerCommand(name string, fn CommandHandler) {
	if r == nil || name == "" || fn == nil {
		return
	}
	if r.commands == nil {
		r.commands = map[string]CommandHandler{}
	}
	r.commands[name] = fn
}

func (r *handlerRegistry) query(name string) QueryHandler {
	if r == nil {
		return nil
	}
	return r.queries[name]
}

func (r *handlerRegistry) command(name string) CommandHandler {
	if r == nil {
		return nil
	}
	return r.commands[name]
}

var (
	globalHandlersMu sync.Mutex
	globalHandlers   = map[string]*handlerRegistry{}
)

// RegisterQueryHandler records a custom query handler for an app (from package init).
func RegisterQueryHandler(app, name string, fn QueryHandler) {
	if app == "" || name == "" || fn == nil {
		return
	}
	globalHandlersMu.Lock()
	defer globalHandlersMu.Unlock()
	reg, ok := globalHandlers[app]
	if !ok {
		reg = newHandlerRegistry()
		globalHandlers[app] = reg
	}
	reg.registerQuery(name, fn)
}

// RegisterCommandHandler records a custom command handler for an app (from package init).
func RegisterCommandHandler(app, name string, fn CommandHandler) {
	if app == "" || name == "" || fn == nil {
		return
	}
	globalHandlersMu.Lock()
	defer globalHandlersMu.Unlock()
	reg, ok := globalHandlers[app]
	if !ok {
		reg = newHandlerRegistry()
		globalHandlers[app] = reg
	}
	reg.registerCommand(name, fn)
}

func applyRegisteredHandlers(a *App) {
	if a == nil || a.opts.AppName == "" {
		return
	}
	globalHandlersMu.Lock()
	defer globalHandlersMu.Unlock()
	reg := globalHandlers[a.opts.AppName]
	if reg == nil {
		return
	}
	if a.opts.Handlers == nil {
		a.opts.Handlers = newHandlerRegistry()
	}
	for name, fn := range reg.queries {
		a.opts.Handlers.registerQuery(name, fn)
	}
	for name, fn := range reg.commands {
		a.opts.Handlers.registerCommand(name, fn)
	}
}

// Query registers a custom query handler. Returns a for chaining.
func (a *App) Query(name string, fn QueryHandler) *App {
	if a.opts.Handlers == nil {
		a.opts.Handlers = newHandlerRegistry()
	}
	a.opts.Handlers.registerQuery(name, fn)
	return a
}

// Command registers a custom command handler. Returns a for chaining.
func (a *App) Command(name string, fn CommandHandler) *App {
	if a.opts.Handlers == nil {
		a.opts.Handlers = newHandlerRegistry()
	}
	a.opts.Handlers.registerCommand(name, fn)
	return a
}

func cqrsFieldName(app, name string) string {
	return camel(app) + pascal(name)
}

func modelByName(spec appspec.AppSpec, name string) (appspec.ModelSpec, bool) {
	for _, m := range spec.Models {
		if m.Name == name {
			return m, true
		}
	}
	return appspec.ModelSpec{}, false
}

func findModelService(reg *ModelRegistry, name string) (*modelService, error) {
	if reg == nil {
		return nil, fmt.Errorf("model registry not initialized")
	}
	return reg.require(name)
}
