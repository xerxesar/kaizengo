package postgres

import (
	"context"
	"fmt"
	"net/http"

	"kaizengo/internal/module"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ServiceName is the host bag key for the shared Postgres connector.
const ServiceName = "platform.postgres"

type ctxKey struct{}

// DB is the platform-wide Postgres connector (one pool, many app schemas).
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a shared connection pool from dsn.
func Connect(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres DSN not set (KaizenGo_POSTGRES_DSN)")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Pool returns the underlying pgx pool.
func (db *DB) Pool() *pgxpool.Pool {
	if db == nil {
		return nil
	}
	return db.pool
}

// Ping checks connectivity.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.pool == nil {
		return fmt.Errorf("postgres not connected")
	}
	return db.pool.Ping(ctx)
}

// Close closes the shared pool.
func (db *DB) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}

// Attach registers db on the host bag and closes it on shutdown.
func Attach(host *module.Host, db *DB) {
	host.Provide(ServiceName, db)
	host.OnStop(func(context.Context) error {
		db.Close()
		return nil
	})
}

// FromHost returns the shared Postgres connector from the host bag.
func FromHost(host *module.Host) (*DB, error) {
	if host == nil {
		return nil, fmt.Errorf("host is nil")
	}
	raw, ok := host.Lookup(ServiceName)
	if !ok {
		return nil, fmt.Errorf("%s not registered (server must connect Postgres before loading apps)", ServiceName)
	}
	db, ok := raw.(*DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("%s has unexpected type %T", ServiceName, raw)
	}
	return db, nil
}

// WithContext stores db in ctx for request handlers.
func WithContext(ctx context.Context, db *DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, db)
}

// FromContext returns the Postgres connector injected by Middleware.
func FromContext(ctx context.Context) (*DB, bool) {
	db, ok := ctx.Value(ctxKey{}).(*DB)
	return db, ok && db != nil
}

// Middleware injects the shared DB into every request context.
func Middleware(db *DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithContext(r.Context(), db)))
		})
	}
}
