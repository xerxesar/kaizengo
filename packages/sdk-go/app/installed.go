package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const installedSchema = "kaizengo"
const installedTable = "installed_apps"

// InstalledApp is a row in the platform install table.
type InstalledApp struct {
	Name        string
	Version     string
	InstalledAt time.Time
	UpgradedAt  time.Time
}

// InstalledStore persists which compiled-in apps are installed and at which version.
type InstalledStore struct {
	pool *pgxpool.Pool
}

// OpenInstalledStore creates the platform schema/table if needed.
func OpenInstalledStore(ctx context.Context, pool *pgxpool.Pool) (*InstalledStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	s := &InstalledStore{pool: pool}
	if err := s.ensure(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *InstalledStore) ensure(ctx context.Context) error {
	stmts := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, quoteIdent(installedSchema)),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			name TEXT PRIMARY KEY,
			version TEXT NOT NULL,
			installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			upgraded_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`, s.table()),
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("installed apps bootstrap: %w", err)
		}
	}
	return nil
}

func (s *InstalledStore) table() string {
	return quoteIdent(installedSchema) + "." + quoteIdent(installedTable)
}

// Get returns the install row, or nil if the app is not installed.
func (s *InstalledStore) Get(ctx context.Context, name string) (*InstalledApp, error) {
	var rec InstalledApp
	err := s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT name, version, installed_at, upgraded_at FROM %s WHERE name = $1`, s.table(),
	), name).Scan(&rec.Name, &rec.Version, &rec.InstalledAt, &rec.UpgradedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// List returns every installed app, ordered by name.
func (s *InstalledStore) List(ctx context.Context) ([]InstalledApp, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(
		`SELECT name, version, installed_at, upgraded_at FROM %s ORDER BY name`, s.table(),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstalledApp
	for rows.Next() {
		var rec InstalledApp
		if err := rows.Scan(&rec.Name, &rec.Version, &rec.InstalledAt, &rec.UpgradedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Upsert records an install or version stamp. installed_at is kept on conflict.
func (s *InstalledStore) Upsert(ctx context.Context, name, version string) error {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" {
		return fmt.Errorf("installed app name is empty")
	}
	if version == "" {
		version = "0.0.0"
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (name, version, installed_at, upgraded_at)
		VALUES ($1, $2, now(), now())
		ON CONFLICT (name) DO UPDATE SET version = EXCLUDED.version, upgraded_at = now()
	`, s.table()), name, version)
	return err
}

// Delete removes an install row. Missing names are ignored.
func (s *InstalledStore) Delete(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("installed app name is empty")
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE name = $1`, s.table()), name)
	return err
}
