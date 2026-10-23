package app

import (
	"context"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/postgres"
)

// SchemaStoreFromHost returns a schema-scoped migrator on the platform Postgres pool.
func SchemaStoreFromHost(ctx context.Context, host *module.Host, schema string) (*SchemaStore, error) {
	db, err := postgres.FromHost(host)
	if err != nil {
		return nil, err
	}
	return NewSchemaStoreFromPool(ctx, db.Pool(), SchemaStoreConfig{Schema: schema})
}
