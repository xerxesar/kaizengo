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
	// SchemaEnv is the env var for schema override (default KaizenGo_<NAME>_SCHEMA).
	SchemaEnv string
	// Version shown in the module manifest.
	Version string
	// Hooks registers Go lifecycle callbacks per model.
	Hooks *HookRegistry
	// Setup runs after locales, nav, catalog queries, and event-sourced models.
	Setup func(host *module.Host, events *EventsSetup) error
	// Mount registers HTTP routes after every app has completed Setup.
	Mount func(host *module.Host) error
}

// App is a thin module.App implementation driven by app.yaml.
// Hybrid apps add custom services or HTTP routes via Options.Setup / Mount.
type App struct {
	opts Options
	spec appspec.AppSpec
}

// New returns a module.App that loads apps/<name>/app.yaml and wires
// locales, nav, SPA, GraphQL, and event-sourced CRUD from the spec.
func New(opts Options) *App {
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}
	a := &App{opts: opts}
	applyRegisteredHooks(a)
	return a
}

func (a *App) Manifest() module.Manifest {
	a.ensureSpec()
	depends := a.spec.Depends
	if len(depends) == 0 {
		depends = []string{"core", "identity", "auth", "permissions"}
	}
	summary := a.spec.Summary
	if summary == "" {
		summary = a.opts.AppName + " app"
	}
	return module.Manifest{
		Name:        a.opts.AppName,
		Version:     a.opts.Version,
		Summary:     summary,
		Depends:     depends,
		Installable: true,
	}
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
	spec, err := a.loadSpec()
	if err != nil {
		return err
	}
	a.spec = spec

	if spec.EnableI18n {
		app.MustLoadLocales(a.opts.AppName)
	}
	if spec.EnableSPA {
		app.RegisterNavFromSpec(host, a.opts.AppName, spec)
	}
	registerBasics(host, spec)
	RegisterAppResources(spec)

	events, err := SetupEvents(host, a.opts.AppName, spec, a.opts.Hooks)
	if err != nil {
		return err
	}
	host.Provide(ModelsKey(a.opts.AppName), events.Models)
	host.Provide(a.opts.AppName, a)
	// Defer until after every app's Setup so permissions (and identity/auth for users) exist.
	// identity loads before permissions in the dependency graph, so inline ApplySecurity would skip.
	if !spec.Security.Empty() {
		sec := spec.Security
		appName := a.opts.AppName
		host.OnStart(func(context.Context) error {
			if err := ApplySecurity(host, sec); err != nil {
				return fmt.Errorf("%s security: %w", appName, err)
			}
			return nil
		})
	}
	if a.opts.Setup != nil {
		return a.opts.Setup(host, events)
	}
	return nil
}

func (a *App) Mount(host *module.Host) error {
	if a.opts.Mount != nil {
		return a.opts.Mount(host)
	}
	return nil
}

func (a *App) loadSpec() (appspec.AppSpec, error) {
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

// Mount is a one-liner helper: module.Register(engine.New(engine.Options{AppName: "hello"}))
// Prefer embedding via New in init().
func Mount(host *module.Host, appName string) error {
	a := New(Options{AppName: appName})
	if err := a.Setup(host); err != nil {
		return err
	}
	return a.Mount(host)
}
