package app

import (
	"context"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/postgres"
	"kaizengo/internal/events/pgstore"
)

// SchemaStore returns a schema-scoped event store on the platform Postgres pool.
func SchemaStore(ctx context.Context, host *module.Host, schema string) (*pgstore.Store, error) {
	db, err := postgres.FromHost(host)
	if err != nil {
		return nil, err
	}
	return pgstore.FromPool(ctx, db.Pool(), pgstore.Config{Schema: schema})
}
