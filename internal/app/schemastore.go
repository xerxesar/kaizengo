package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaStore owns schema bootstrap and versioned SQL migrations for one
// Postgres schema. It does not store domain events.
type SchemaStore struct {
	pool     *pgxpool.Pool
	schema   string
	ownsPool bool
}

// SchemaStoreConfig configures a schema-scoped migrator.
type SchemaStoreConfig struct {
	Schema string
}

// NewSchemaStoreFromPool builds a schema-scoped store on a shared platform pool.
// Close does not close the pool — the platform connector owns it.
func NewSchemaStoreFromPool(ctx context.Context, pool *pgxpool.Pool, cfg SchemaStoreConfig) (*SchemaStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	if strings.TrimSpace(cfg.Schema) == "" {
		cfg.Schema = "public"
	}
	s := &SchemaStore{pool: pool, schema: cfg.Schema, ownsPool: false}
	if err := s.EnsureBootstrap(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the pool only when this store created it.
func (s *SchemaStore) Close() {
	if s != nil && s.ownsPool && s.pool != nil {
		s.pool.Close()
	}
}

// EnsureBootstrap creates the app schema and schema_migrations tracking table.
func (s *SchemaStore) EnsureBootstrap(ctx context.Context) error {
	stmts := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, quoteIdent(s.schema)),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`, s.t("schema_migrations")),
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ApplyMigrations applies pending SQL files keyed by version string.
func (s *SchemaStore) ApplyMigrations(ctx context.Context, migrationSQL map[string]string) error {
	keys := make([]string, 0, len(migrationSQL))
	for k := range migrationSQL {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, ver := range keys {
		var exists bool
		row := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE version = $1)`, s.t("schema_migrations")), ver)
		if err := row.Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, migrationSQL[ver]); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s(version) VALUES($1)`, s.t("schema_migrations")), ver); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Pool returns the underlying connection pool.
func (s *SchemaStore) Pool() *pgxpool.Pool { return s.pool }

// Schema returns the Postgres schema name.
func (s *SchemaStore) Schema() string { return s.schema }

func (s *SchemaStore) t(name string) string {
	return quoteIdent(s.schema) + "." + quoteIdent(name)
}
