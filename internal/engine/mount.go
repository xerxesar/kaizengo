package engine

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/internal/app"
	"kaizengo/packages/sdk-go/appspec"
)

// Options configures how an app is mounted from its declarative spec.
type Options struct {
	// AppName is the directory under apps/ (and default route).
	AppName string
	// SpecPath overrides apps/<name>/app.yaml when set.
	SpecPath string
	// Spec is a preloaded spec; wins over SpecPath and apps/<name>/app.yaml.
	Spec *appspec.AppSpec
	// AutoSchema creates every model table/column from the spec (additive DDL)
	// instead of relying on apps/<name>/migrations. Without it only tables and
	// columns added by stored domain layers are created.
	AutoSchema bool
	// SchemaEnv is the env var for schema override (default KaizenGo_<NAME>_SCHEMA).
	SchemaEnv string
	// Version shown in the module manifest.
	Version string
	// Hooks registers Go lifecycle callbacks per model.
	Hooks *HookRegistry
	// Handlers registers custom CQRS query/command implementations.
	Handlers *handlerRegistry
	// Setup runs after locales, nav, catalog queries, and model registration.
	Setup func(host *module.Host, events *EventsSetup) error
	// Mount registers HTTP routes after every app has completed Setup.
	Mount func(host *module.Host) error

	// managed apps are mounted by Domains with an already resolved spec.
	managed bool
}

// App is a thin module.App implementation driven by app.yaml.
// Hybrid apps add custom services or HTTP routes via Options.Setup / Mount.
type App struct {
	opts Options
	base appspec.AppSpec // as read from code (or Options.Spec)
	spec appspec.AppSpec // effective: base plus stored domain layers

	models *ModelRegistry
	// Root GraphQL fields registered from the spec (not by Options.Setup);
	// these are swapped when the domain is remounted.
	ownedQueries   []string
	ownedMutations []string
}

// New returns a module.App that loads apps/<name>/app.yaml and wires
// locales, nav, SPA, GraphQL, and model CRUD from the spec.
func New(opts Options) *App {
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}
	a := &App{opts: opts}
	applyRegisteredHooks(a)
	applyRegisteredHandlers(a)
	return a
}

func (a *App) Manifest() module.Manifest {
	a.ensureSpec()
	m := app.ManifestFromSpec(a.spec, a.opts.Version)
	if m.Name == "" {
		m.Name = a.opts.AppName
	}
	if m.Summary == "" {
		m.Summary = a.opts.AppName + " app"
	}
	if m.Version == "" {
		m.Version = a.opts.Version
	}
	return m
}

func (a *App) ensureSpec() {
	if a.spec.Name != "" {
		return
	}
	if spec, err := a.loadSpec(); err == nil {
		a.spec = spec
	}
}

func (a *App) Setup(host *module.Host) error {
	base, err := a.loadSpec()
	if err != nil {
		return err
	}
	spec := base
	if !a.opts.managed {
		if d, err := DomainsFromHost(host); err == nil {
			spec = d.resolveCompiled(base)
		}
	}
	// The same registered App is set up again when the platform is rebuilt for another database.
	a.base, a.spec, a.models = base, spec, nil

	if spec.EnableI18n && len(spec.Locales) > 0 {
		app.MustLoadLocales(a.opts.AppName)
	}
	if hasNav(spec.Nav) {
		app.RegisterNavFromSpec(host, a.opts.AppName, spec)
	}
	q0, m0 := host.GQL.FieldNames()
	events, err := a.mountModels(host, spec)
	if err != nil {
		return err
	}
	q1, m1 := host.GQL.FieldNames()
	host.Provide(ModelsKey(a.opts.AppName), events.Models)
	host.Provide(a.opts.AppName, a)
	host.Provide(AppKey(a.opts.AppName), a)
	// Defer until after every app's Setup so permissions (and identity/auth for users) exist.
	// identity loads before permissions in the dependency graph, so inline ApplySecurity would skip.
	if !spec.Security.Empty() && !a.opts.managed {
		sec := spec.Security
		appName := a.opts.AppName
		host.OnStart(func(context.Context) error {
			if err := ApplySecurity(host, sec); err != nil {
				return fmt.Errorf("%s security: %w", appName, err)
			}
			return nil
		})
	}
	// Setup may RegisterModel / handlers needed by CQRS list bindings.
	if a.opts.Setup != nil {
		if err := a.opts.Setup(host, events); err != nil {
			return err
		}
	}
	q2, m2 := host.GQL.FieldNames()
	if err := a.mountCQRS(host, spec); err != nil {
		return err
	}
	q3, m3 := host.GQL.FieldNames()
	a.ownedQueries = append(added(q0, q1), added(q2, q3)...)
	a.ownedMutations = append(added(m0, m1), added(m2, m3)...)
	return nil
}

// mountModels registers the spec-driven catalog, resources, tables, and model
// services (including model CRUD GraphQL for apps without CQRS).
func (a *App) mountModels(host *module.Host, spec appspec.AppSpec) (*EventsSetup, error) {
	registerBasics(host, spec)
	RegisterAppResources(spec)
	events, err := setupEvents(host, a.opts.AppName, spec, a.opts.Hooks, schemaDDL(a.base, spec, a.opts.AutoSchema), a.models)
	if err != nil {
		return nil, err
	}
	a.models = events.Models
	return events, nil
}

func (a *App) mountCQRS(host *module.Host, spec appspec.AppSpec) error {
	if !spec.HasCQRS() {
		return nil
	}
	if err := registerCQRS(host, spec, a.models, a.handlersFor(host, spec)); err != nil {
		return fmt.Errorf("%s cqrs: %w", a.opts.AppName, err)
	}
	return nil
}

// remount re-registers the spec-driven surface for next. The caller removes
// the previously owned fields first and rolls back on error (Domains.swapLocked).
func (a *App) remount(host *module.Host, next appspec.AppSpec) error {
	q0, m0 := host.GQL.FieldNames()
	if _, err := a.mountModels(host, next); err != nil {
		return err
	}
	if err := a.mountCQRS(host, next); err != nil {
		return err
	}
	q1, m1 := host.GQL.FieldNames()
	a.ownedQueries, a.ownedMutations = added(q0, q1), added(m0, m1)
	a.spec = next
	return nil
}

// handlersFor returns the Go handlers plus a call-time dispatcher (Domains
// BindHandler, used by external Go apps) for every query/command without
// shorthand that has no Go handler and is not part of the compiled base.
func (a *App) handlersFor(host *module.Host, spec appspec.AppSpec) *handlerRegistry {
	d, err := DomainsFromHost(host)
	if err != nil {
		return a.opts.Handlers
	}
	inBase := map[string]bool{}
	if !a.opts.managed {
		for _, q := range a.base.Queries {
			inBase["q:"+q.Name] = true
		}
		for _, c := range a.base.Commands {
			inBase["c:"+c.Name] = true
		}
	}
	reg := newHandlerRegistry()
	if a.opts.Handlers != nil {
		for n, fn := range a.opts.Handlers.queries {
			reg.registerQuery(n, fn)
		}
		for n, fn := range a.opts.Handlers.commands {
			reg.registerCommand(n, fn)
		}
	}
	appName := spec.Name
	for _, q := range spec.Queries {
		if q.List != "" || q.Get != "" || reg.query(q.Name) != nil || inBase["q:"+q.Name] {
			continue
		}
		name := q.Name
		reg.registerQuery(name, func(c HandlerCtx, args map[string]any) (any, error) {
			return d.dispatch(c, args, appName, KindQuery, name)
		})
	}
	for _, c := range spec.Commands {
		if c.Create != "" || c.Update != "" || c.Delete != "" || reg.command(c.Name) != nil || inBase["c:"+c.Name] {
			continue
		}
		name := c.Name
		reg.registerCommand(name, func(hc HandlerCtx, args map[string]any) (any, error) {
			return d.dispatch(hc, args, appName, KindCommand, name)
		})
	}
	return reg
}

func (a *App) Mount(host *module.Host) error {
	if a.opts.Mount != nil {
		return a.opts.Mount(host)
	}
	return nil
}

// Spec returns the effective spec: the code spec plus stored domain layers
// (empty before Setup).
func (a *App) Spec() appspec.AppSpec {
	return a.spec
}

// BaseSpec returns the spec as read from code, without domain layers.
func (a *App) BaseSpec() appspec.AppSpec {
	return a.base
}

func (a *App) loadSpec() (appspec.AppSpec, error) {
	if a.opts.Spec != nil {
		return *a.opts.Spec, nil
	}
	if a.opts.SpecPath != "" {
		return appspec.LoadFile(a.opts.SpecPath)
	}
	if a.opts.AppName == "" {
		return appspec.AppSpec{}, fmt.Errorf("engine: AppName is required")
	}
	return appspec.LoadApp(a.opts.AppName)
}

func (a *App) schemaEnv() string {
	if a.opts.SchemaEnv != "" {
		return a.opts.SchemaEnv
	}
	return "KaizenGo_" + strings.ToUpper(a.opts.AppName) + "_SCHEMA"
}

func hasNav(nav appspec.NavSpec) bool {
	return strings.TrimSpace(nav.Route) != "" ||
		strings.TrimSpace(nav.LabelKey) != "" ||
		strings.TrimSpace(nav.Label) != ""
}

// Mount is a one-liner helper: module.Register(engine.New(engine.Options{AppName: "hello"}))
// Prefer embedding via New in init().
func Mount(host *module.Host, appName string) error {
	a := New(Options{AppName: appName})
	if err := a.Setup(host); err != nil {
		return err
	}
	return a.Mount(host)
}
