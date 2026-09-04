package pgread

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckpointStore struct {
	pool   *pgxpool.Pool
	schema string
}

func NewCheckpointStore(pool *pgxpool.Pool, schema string) *CheckpointStore {
	if strings.TrimSpace(schema) == "" {
		schema = "public"
	}
	return &CheckpointStore{pool: pool, schema: schema}
}

func (c *CheckpointStore) Ensure(ctx context.Context) error {
	var exists bool
	row := c.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = 'projection_checkpoints')`,
		c.schema,
	)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("projection_checkpoints table missing in schema %q (add a migration)", c.schema)
	}
	return nil
}

func (c *CheckpointStore) GetCheckpoint(ctx context.Context, name string) (int64, error) {
	var last int64
	row := c.pool.QueryRow(ctx, fmt.Sprintf(`SELECT last_row_id FROM %s.projection_checkpoints WHERE name = $1`, quoteIdent(c.schema)), name)
	if err := row.Scan(&last); err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return last, nil
}

func (c *CheckpointStore) SaveCheckpoint(ctx context.Context, name string, rowID int64) error {
	_, err := c.pool.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.projection_checkpoints(name,last_row_id,updated_at)
		VALUES ($1,$2,now())
		ON CONFLICT(name) DO UPDATE SET last_row_id = excluded.last_row_id, updated_at = excluded.updated_at`,
		quoteIdent(c.schema)), name, rowID)
	return err
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
