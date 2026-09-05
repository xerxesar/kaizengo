package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"kaizengo/internal/events"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Schema string
}

type Store struct {
	pool     *pgxpool.Pool
	schema   string
	ownsPool bool
}

func Open(ctx context.Context, dsn string, cfg Config) (*Store, error) {
	if strings.TrimSpace(cfg.Schema) == "" {
		cfg.Schema = "public"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{pool: pool, schema: cfg.Schema, ownsPool: true}
	if err := s.EnsureBootstrap(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// FromPool builds a schema-scoped store on a shared platform pool.
// Close does not close the pool — the platform connector owns it.
func FromPool(ctx context.Context, pool *pgxpool.Pool, cfg Config) (*Store, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	if strings.TrimSpace(cfg.Schema) == "" {
		cfg.Schema = "public"
	}
	s := &Store{pool: pool, schema: cfg.Schema, ownsPool: false}
	if err := s.EnsureBootstrap(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.ownsPool && s.pool != nil {
		s.pool.Close()
	}
}

// EnsureBootstrap creates the app schema and schema_migrations tracking table.
// Model and event-store DDL must live in explicit SQL migrations, not here.
func (s *Store) EnsureBootstrap(ctx context.Context) error {
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

func (s *Store) Append(ctx context.Context, streamID, streamType string, expectedVersion int64, newEvents ...events.NewEvent) ([]events.Event, error) {
	if len(newEvents) == 0 {
		return nil, nil
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var version int64
	row := tx.QueryRow(ctx, fmt.Sprintf(`SELECT version FROM %s WHERE stream_id = $1 FOR UPDATE`, s.t("streams")), streamID)
	switch err := row.Scan(&version); {
	case errors.Is(err, pgx.ErrNoRows):
		version = 0
		if expectedVersion != 0 {
			return nil, events.ErrConcurrency
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s (stream_id, stream_type, version) VALUES ($1,$2,0)`, s.t("streams")), streamID, streamType)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if version != expectedVersion {
			return nil, events.ErrConcurrency
		}
	}

	out := make([]events.Event, 0, len(newEvents))
	now := time.Now().UTC()
	for i, e := range newEvents {
		version++
		payload := events.MustJSON(e.Payload)
		metadata := events.MustJSON(e.Metadata)
		ev := events.Event{
			ID:         uuid.NewString(),
			StreamID:   streamID,
			StreamType: streamType,
			Version:    version,
			Type:       e.Type,
			Payload:    payload,
			Metadata:   metadata,
			OccurredAt: now.Add(time.Duration(i) * time.Microsecond),
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s
			(event_id, stream_id, stream_type, version, event_type, payload, metadata, occurred_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, s.t("events")),
			ev.ID, ev.StreamID, ev.StreamType, ev.Version, ev.Type, ev.Payload, nullableJSON(ev.Metadata), ev.OccurredAt,
		); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET version = $1, updated_at = now() WHERE stream_id = $2`, s.t("streams")), version, streamID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) LoadStream(ctx context.Context, streamID string) ([]events.Event, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT id, event_id, stream_id, stream_type, version, event_type, payload, metadata, occurred_at
		FROM %s WHERE stream_id = $1 ORDER BY version ASC`, s.t("events")), streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]events.Event, 0)
	for rows.Next() {
		ev, _, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	if len(out) == 0 {
		return nil, events.ErrNotFound
	}
	return out, rows.Err()
}

func (s *Store) LoadSince(ctx context.Context, fromID int64, limit int) ([]events.PersistedEvent, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT id, event_id, stream_id, stream_type, version, event_type, payload, metadata, occurred_at
		FROM %s WHERE id > $1 ORDER BY id ASC LIMIT $2`, s.t("events")), fromID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]events.PersistedEvent, 0, limit)
	for rows.Next() {
		ev, rowID, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, events.PersistedEvent{RowID: rowID, Event: ev})
	}
	return out, rows.Err()
}

func (s *Store) ApplyMigrations(ctx context.Context, migrationSQL map[string]string) error {
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

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(s rowScanner) (events.Event, int64, error) {
	var rowID int64
	var ev events.Event
	var metadata []byte
	if err := s.Scan(&rowID, &ev.ID, &ev.StreamID, &ev.StreamType, &ev.Version, &ev.Type, &ev.Payload, &metadata, &ev.OccurredAt); err != nil {
		return ev, 0, err
	}
	if len(metadata) > 0 {
		ev.Metadata = json.RawMessage(metadata)
	}
	return ev, rowID, nil
}

func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func (s *Store) t(name string) string {
	return quoteIdent(s.schema) + "." + quoteIdent(name)
}
