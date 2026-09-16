package listpresets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kaizengo/internal/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = "kaizengo"
const table = "list_view_presets"

// Preset is a saved list view search/filter/group configuration.
type Preset struct {
	ID         string
	OrgID      string
	Model      string
	Name       string
	OwnerID    string
	Shared     bool
	IsDefault  bool
	Predefined bool
	PageSize   int
	Q          string
	SearchIn   []string
	Domain     string
	GroupBy    []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type SaveInput struct {
	ID         string
	OrgID      string
	Model      string
	Name       string
	OwnerID    string
	Shared     bool
	IsDefault  bool
	Predefined bool
	PageSize   int
	Q          string
	SearchIn   []string
	Domain     string
	GroupBy    []string
}

func tableName() string {
	return schema + "." + table
}

// EnsureBootstrap creates the platform table if missing.
func EnsureBootstrap(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schema))
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id UUID PRIMARY KEY,
		org_id TEXT NOT NULL,
		model TEXT NOT NULL,
		name TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		shared BOOLEAN NOT NULL DEFAULT false,
		is_default BOOLEAN NOT NULL DEFAULT false,
		predefined BOOLEAN NOT NULL DEFAULT false,
		page_size INT NOT NULL DEFAULT 0,
		q TEXT NOT NULL DEFAULT '',
		search_in TEXT NOT NULL DEFAULT '',
		domain TEXT NOT NULL DEFAULT '',
		group_by TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`, tableName()))
	if err != nil {
		return fmt.Errorf("list_view_presets bootstrap: %w", err)
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS list_view_presets_org_model_idx ON %s (org_id, model)`,
		tableName(),
	))
	return err
}

func poolFrom(ctx context.Context) (*pgxpool.Pool, error) {
	db, ok := postgres.FromContext(ctx)
	if !ok || db == nil {
		return nil, fmt.Errorf("postgres not available")
	}
	pool := db.Pool()
	if pool == nil {
		return nil, fmt.Errorf("postgres pool not available")
	}
	return pool, nil
}

func encodeCSV(items []string) string {
	return strings.Join(items, ",")
}

func decodeCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := strings.Split(raw, ",")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func scanPreset(row pgx.Row) (Preset, error) {
	var rec Preset
	var searchIn, groupBy string
	err := row.Scan(
		&rec.ID, &rec.OrgID, &rec.Model, &rec.Name, &rec.OwnerID,
		&rec.Shared, &rec.IsDefault, &rec.Predefined, &rec.PageSize,
		&rec.Q, &searchIn, &rec.Domain, &groupBy,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return Preset{}, err
	}
	rec.SearchIn = decodeCSV(searchIn)
	rec.GroupBy = decodeCSV(groupBy)
	return rec, nil
}

const selectCols = `id, org_id, model, name, owner_id, shared, is_default, predefined, page_size, q, search_in, domain, group_by, created_at, updated_at`

// List returns presets visible to the user for a model (own + shared in org).
func List(ctx context.Context, orgID, model, ownerID string) ([]Preset, error) {
	pool, err := poolFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := EnsureBootstrap(ctx, pool); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE org_id = $1 AND model = $2 AND (owner_id = $3 OR shared = true)
		ORDER BY name ASC`, selectCols, tableName()), orgID, model, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Preset
	for rows.Next() {
		rec, err := scanPreset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// DefaultForUser returns the user's default preset for a model, if any.
func DefaultForUser(ctx context.Context, orgID, model, ownerID string) (*Preset, error) {
	pool, err := poolFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := EnsureBootstrap(ctx, pool); err != nil {
		return nil, err
	}
	row := pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE org_id = $1 AND model = $2 AND owner_id = $3 AND is_default = true
		LIMIT 1`, selectCols, tableName()), orgID, model, ownerID)
	rec, err := scanPreset(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Save creates or updates a preset. Clears other defaults when isDefault is set.
func Save(ctx context.Context, in SaveInput) (Preset, error) {
	pool, err := poolFrom(ctx)
	if err != nil {
		return Preset{}, err
	}
	if err := EnsureBootstrap(ctx, pool); err != nil {
		return Preset{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Preset{}, fmt.Errorf("name is required")
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		return Preset{}, fmt.Errorf("model is required")
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = uuid.NewString()
	}
	domain := strings.TrimSpace(in.Domain)
	if domain != "" && domain != "[]" {
		var probe any
		if err := json.Unmarshal([]byte(domain), &probe); err != nil {
			return Preset{}, fmt.Errorf("domain: invalid JSON")
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Preset{}, err
	}
	defer tx.Rollback(ctx)

	if in.IsDefault {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			UPDATE %s SET is_default = false, updated_at = now()
			WHERE org_id = $1 AND model = $2 AND owner_id = $3 AND is_default = true AND id <> $4`,
			tableName()), in.OrgID, model, in.OwnerID, id); err != nil {
			return Preset{}, err
		}
	}

	now := time.Now().UTC()
	var existingOwner string
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT owner_id FROM %s WHERE id = $1 AND org_id = $2`, tableName()), id, in.OrgID).Scan(&existingOwner)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Preset{}, err
	}
	if existingOwner != "" && existingOwner != in.OwnerID {
		return Preset{}, fmt.Errorf("preset not owned by you")
	}

	var rec Preset
	if existingOwner == "" {
		rec, err = scanPreset(tx.QueryRow(ctx, fmt.Sprintf(`
			INSERT INTO %s (
				id, org_id, model, name, owner_id, shared, is_default, predefined, page_size,
				q, search_in, domain, group_by, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			RETURNING %s`, tableName(), selectCols),
			id, in.OrgID, model, name, in.OwnerID, in.Shared, in.IsDefault, in.Predefined, in.PageSize,
			strings.TrimSpace(in.Q), encodeCSV(in.SearchIn), domain, encodeCSV(in.GroupBy), now, now,
		))
	} else {
		rec, err = scanPreset(tx.QueryRow(ctx, fmt.Sprintf(`
			UPDATE %s SET
				name = $4, shared = $5, is_default = $6, predefined = $7, page_size = $8,
				q = $9, search_in = $10, domain = $11, group_by = $12, updated_at = $13
			WHERE id = $1 AND org_id = $2 AND owner_id = $3
			RETURNING %s`, tableName(), selectCols),
			id, in.OrgID, in.OwnerID, name, in.Shared, in.IsDefault, in.Predefined, in.PageSize,
			strings.TrimSpace(in.Q), encodeCSV(in.SearchIn), domain, encodeCSV(in.GroupBy), now,
		))
	}
	if err != nil {
		return Preset{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Preset{}, err
	}
	return rec, nil
}

// Delete removes a preset owned by ownerID.
func Delete(ctx context.Context, orgID, id, ownerID string) error {
	pool, err := poolFrom(ctx)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, fmt.Sprintf(`
		DELETE FROM %s WHERE org_id = $1 AND id = $2 AND owner_id = $3`, tableName()), orgID, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("preset not found or not owned by you")
	}
	return nil
}
