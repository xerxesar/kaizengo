package engine

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/postgres"
	"kaizengo/internal/platform/search"
	"kaizengo/internal/app"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/internal/events/pgstore"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ModelRegistry exposes event-sourced CRUD for models declared in app.yaml.
type ModelRegistry struct {
	byName map[string]*modelService
}

func (r *ModelRegistry) Create(ctx context.Context, orgID, authorID, model string, fields map[string]any) (Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.Create(ctx, orgID, authorID, fields)
}

func (r *ModelRegistry) Update(ctx context.Context, orgID, model, id string, fields map[string]any) (Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.Update(ctx, orgID, id, fields)
}

func (r *ModelRegistry) Delete(ctx context.Context, orgID, model, id string) error {
	svc, err := r.require(model)
	if err != nil {
		return err
	}
	return svc.Delete(ctx, orgID, id)
}

func (r *ModelRegistry) List(ctx context.Context, orgID, model string) ([]Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.List(ctx, orgID)
}

func (r *ModelRegistry) Get(ctx context.Context, orgID, model, id string) (Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.Get(ctx, orgID, id)
}

// GetByID loads a record by primary key without an org scope (for cross-lookups).
func (r *ModelRegistry) GetByID(ctx context.Context, model, id string) (Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	rec, err := svc.getRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if boolVal(rec["deleted"]) {
		return nil, errNotFound
	}
	return rec, nil
}

// FindBy returns the first non-deleted row matching field=value (not org-scoped).
func (r *ModelRegistry) FindBy(ctx context.Context, model, field, value string) (Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.findBy(ctx, field, value)
}

// ListBy returns non-deleted rows for an org where field=value.
func (r *ModelRegistry) ListBy(ctx context.Context, orgID, model, field, value string) ([]Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.listBy(ctx, orgID, field, value)
}

// ListAll returns every non-deleted row for a model (not org-scoped).
func (r *ModelRegistry) ListAll(ctx context.Context, model string) ([]Record, error) {
	svc, err := r.require(model)
	if err != nil {
		return nil, err
	}
	return svc.listAll(ctx)
}

func (r *ModelRegistry) require(model string) (*modelService, error) {
	if r == nil || r.byName == nil {
		return nil, fmt.Errorf("model registry not initialized")
	}
	svc, ok := r.byName[model]
	if !ok {
		return nil, fmt.Errorf("unknown model %q", model)
	}
	return svc, nil
}

// EventsSetup is the result of wiring Postgres schema migrations and event-sourced models.
type EventsSetup struct {
	Store   *pgstore.Store
	Models  *ModelRegistry
	Schema  string
	AppName string
	Pool    *pgxpool.Pool
}

// SetupEvents uses the platform Postgres connector, applies apps/<app>/migrations,
// and registers model CRUD for the app schema.
func SetupEvents(host *module.Host, appName string, spec appspec.AppSpec, hooks *HookRegistry) (*EventsSetup, error) {
	db, err := postgres.FromHost(host)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", appName, err)
	}
	schema := app.Env(schemaEnv(appName), spec.Schema)
	ctx := context.Background()
	store, err := pgstore.FromPool(ctx, db.Pool(), pgstore.Config{Schema: schema})
	if err != nil {
		return nil, fmt.Errorf("%s postgres schema %q: %w", appName, schema, err)
	}
	spec.Schema = schema

	if err := app.ApplyMigrationsFromDir(ctx, store, appName, schema); err != nil {
		return nil, err
	}

	reg := &ModelRegistry{byName: map[string]*modelService{}}
	for _, model := range spec.Models {
		svc := newModelService(store, spec, model, hooks)
		svc.registry = reg
		svc.host = host
		reg.byName[model.Name] = svc
		registerModelResource(spec.Name, model)
		registerModelGQL(host, spec, svc)
		search.RegisterReindexer(spec.Name, model.Name, svc.reindexAll)
	}

	return &EventsSetup{Store: store, Models: reg, Schema: schema, AppName: appName, Pool: db.Pool()}, nil
}

func schemaEnv(appName string) string {
	return "KaizenGo_" + strings.ToUpper(appName) + "_SCHEMA"
}

// ModelsKey is the host bag key for an app's ModelRegistry.
func ModelsKey(appName string) string {
	return appName + ".models"
}

// ModelsFromHost returns the ModelRegistry provided during another app's Setup.
func ModelsFromHost(host *module.Host, appName string) (*ModelRegistry, error) {
	if host == nil {
		return nil, fmt.Errorf("host is nil")
	}
	raw, ok := host.Lookup(ModelsKey(appName))
	if !ok {
		return nil, fmt.Errorf("%s models not registered", appName)
	}
	reg, ok := raw.(*ModelRegistry)
	if !ok || reg == nil {
		return nil, fmt.Errorf("%s models have unexpected type %T", appName, raw)
	}
	return reg, nil
}
