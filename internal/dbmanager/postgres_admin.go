package dbmanager

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Admin runs CREATE/DROP/TEMPLATE ops over pgx using the platform DSN credentials.
type Admin struct {
	baseDSN string // KaizenGo_POSTGRES_DSN (path is ignored; admin always uses "postgres")
}

// NewAdmin creates an admin client. baseDSN should be KaizenGo_POSTGRES_DSN.
func NewAdmin(baseDSN string) *Admin {
	return &Admin{baseDSN: baseDSN}
}

func (a *Admin) withPool(ctx context.Context, fn func(*pgxpool.Pool) error) error {
	dsn, err := a.maintenanceDSN()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return fn(pool)
}

// maintenanceDSN returns a DSN connected to the always-present "postgres" database.
// Application DB names in KaizenGo_POSTGRES_DSN are for seeding only and may not exist.
func (a *Admin) maintenanceDSN() (string, error) {
	if strings.TrimSpace(a.baseDSN) == "" {
		return "", fmt.Errorf("KaizenGo_POSTGRES_DSN is not set")
	}
	u, err := url.Parse(a.baseDSN)
	if err != nil {
		return "", fmt.Errorf("parse DSN: %w", err)
	}
	cloned := *u
	cloned.Path = "/postgres"
	cloned.RawPath = ""
	return cloned.String(), nil
}

// DatabaseExists reports whether a database exists on the server.
func (a *Admin) DatabaseExists(ctx context.Context, name string) (bool, error) {
	if err := ValidateDBName(name); err != nil {
		return false, err
	}
	var exists bool
	err := a.withPool(ctx, func(pool *pgxpool.Pool) error {
		return pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists)
	})
	return exists, err
}

// CreateDatabase creates an empty database.
func (a *Admin) CreateDatabase(ctx context.Context, name string) error {
	if err := ValidateDBName(name); err != nil {
		return err
	}
	return a.withPool(ctx, func(pool *pgxpool.Pool) error {
		exists, err := a.exists(ctx, pool, name)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("database %q already exists on server", name)
		}
		_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, quoteIdent(name)))
		if err != nil {
			return fmt.Errorf("create database: %w", err)
		}
		return nil
	})
}

// DropDatabase drops a database after terminating backends.
// If the database is already absent on the server, DropDatabase succeeds (idempotent).
func (a *Admin) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateDBName(name); err != nil {
		return err
	}
	return a.withPool(ctx, func(pool *pgxpool.Pool) error {
		exists, err := a.exists(ctx, pool, name)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		if err := a.isolate(ctx, pool, name); err != nil {
			return err
		}
		_, err = pool.Exec(ctx, fmt.Sprintf(`DROP DATABASE %s`, quoteIdent(name)))
		if err != nil {
			_ = a.allowConnections(ctx, pool, name)
			return fmt.Errorf("drop database: %w", err)
		}
		return nil
	})
}

// DuplicateDatabase creates a new database using TEMPLATE source.
func (a *Admin) DuplicateDatabase(ctx context.Context, source, name string) error {
	if err := ValidateDBName(source); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if err := ValidateDBName(name); err != nil {
		return err
	}
	if source == name {
		return fmt.Errorf("source and target names must differ")
	}
	return a.withPool(ctx, func(pool *pgxpool.Pool) error {
		srcExists, err := a.exists(ctx, pool, source)
		if err != nil {
			return err
		}
		if !srcExists {
			return fmt.Errorf("source database %q does not exist", source)
		}
		dstExists, err := a.exists(ctx, pool, name)
		if err != nil {
			return err
		}
		if dstExists {
			return fmt.Errorf("database %q already exists on server", name)
		}
		// TEMPLATE requires exclusive access to the source.
		if err := a.isolate(ctx, pool, source); err != nil {
			return err
		}
		_, err = pool.Exec(ctx, fmt.Sprintf(
			`CREATE DATABASE %s TEMPLATE %s`,
			quoteIdent(name), quoteIdent(source),
		))
		// Always re-enable connections on source.
		_ = a.allowConnections(ctx, pool, source)
		if err != nil {
			return fmt.Errorf("duplicate database: %w", err)
		}
		return nil
	})
}

func (a *Admin) isolate(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, `UPDATE pg_database SET datallowconn = false WHERE datname = $1`, name)
	if err != nil {
		return fmt.Errorf("disable connections to %q: %w", name, err)
	}
	_, err = pool.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, name)
	if err != nil {
		_ = a.allowConnections(ctx, pool, name)
		return fmt.Errorf("terminate backends on %q: %w", name, err)
	}
	return nil
}

func (a *Admin) allowConnections(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, `UPDATE pg_database SET datallowconn = true WHERE datname = $1`, name)
	return err
}

func (a *Admin) exists(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists)
	return exists, err
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
