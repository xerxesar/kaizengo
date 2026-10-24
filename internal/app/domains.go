package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	domainRevisionsTable = "domain_revisions"
	legacyDomainsTable   = "domains"
)

// Domain layers. Base is the compiled-in spec as the engine last read it from
// code; overlay is everything applied at runtime on top of it (for apps without
// code, the whole definition). An empty overlay spec means "no overlay".
const (
	LayerBase    = "base"
	LayerOverlay = "overlay"
)

// DomainRevision is one stored layer revision of an app's domain. Revisions
// are numbered per app across both layers and never rewritten.
type DomainRevision struct {
	App       string
	Revision  int64
	Layer     string
	Spec      string
	Source    string // compiled | graphql | plugin:<name> | revert:<n> | restore
	Changes   []string
	CreatedAt time.Time
}

// DomainHeads are the latest revision of each layer (nil when absent).
type DomainHeads struct {
	Base    *DomainRevision
	Overlay *DomainRevision
}

// Revision is the newest of the two heads (0 when there is none).
func (h DomainHeads) Revision() int64 {
	var n int64
	if h.Base != nil {
		n = h.Base.Revision
	}
	if h.Overlay != nil && h.Overlay.Revision > n {
		n = h.Overlay.Revision
	}
	return n
}

// DomainStore persists layered domain revisions in kaizengo.domain_revisions.
type DomainStore struct {
	pool *pgxpool.Pool
}

// OpenDomainStore creates the revisions table if needed and imports rows from
// the earlier single-revision kaizengo.domains table once.
func OpenDomainStore(ctx context.Context, pool *pgxpool.Pool) (*DomainStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	s := &DomainStore{pool: pool}
	stmts := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, quoteIdent(installedSchema)),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			app TEXT NOT NULL,
			revision BIGINT NOT NULL,
			layer TEXT NOT NULL,
			spec TEXT NOT NULL,
			source TEXT NOT NULL,
			changes JSONB NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (app, revision)
		)`, s.table()),
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return nil, fmt.Errorf("domains bootstrap: %w", err)
		}
	}
	var legacy *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass($1)::text`, installedSchema+"."+legacyDomainsTable).Scan(&legacy); err != nil {
		return nil, fmt.Errorf("domains bootstrap: %w", err)
	}
	if legacy != nil {
		if _, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s (app, revision, layer, spec, source, created_at)
			SELECT d.app, d.revision, $1, d.spec, d.source, d.updated_at FROM %s d
			WHERE NOT EXISTS (SELECT 1 FROM %s r WHERE r.app = d.app)
		`, s.table(), quoteIdent(installedSchema)+"."+quoteIdent(legacyDomainsTable), s.table()), LayerOverlay); err != nil {
			return nil, fmt.Errorf("domains import: %w", err)
		}
	}
	return s, nil
}

func (s *DomainStore) table() string {
	return quoteIdent(installedSchema) + "." + quoteIdent(domainRevisionsTable)
}

const revisionCols = `app, revision, layer, spec, source, changes, created_at`

func scanRevision(row pgx.Row) (DomainRevision, error) {
	var rec DomainRevision
	var changes []byte
	if err := row.Scan(&rec.App, &rec.Revision, &rec.Layer, &rec.Spec, &rec.Source, &changes, &rec.CreatedAt); err != nil {
		return rec, err
	}
	_ = json.Unmarshal(changes, &rec.Changes)
	return rec, nil
}

func (s *DomainStore) queryRevisions(ctx context.Context, sql string, args ...any) ([]DomainRevision, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DomainRevision
	for rows.Next() {
		rec, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Append stores the next revision for rec.App and returns it.
func (s *DomainStore) Append(ctx context.Context, rec DomainRevision) (DomainRevision, error) {
	rec.App = strings.TrimSpace(rec.App)
	if rec.App == "" {
		return DomainRevision{}, fmt.Errorf("domain app name is empty")
	}
	if rec.Layer != LayerBase && rec.Layer != LayerOverlay {
		return DomainRevision{}, fmt.Errorf("unknown domain layer %q", rec.Layer)
	}
	changes, err := json.Marshal(nonNilStrings(rec.Changes))
	if err != nil {
		return DomainRevision{}, err
	}
	return scanRevision(s.pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %[1]s (app, revision, layer, spec, source, changes, created_at)
		VALUES ($1, (SELECT COALESCE(MAX(revision), 0) + 1 FROM %[1]s WHERE app = $1), $2, $3, $4, $5, now())
		RETURNING %[2]s
	`, s.table(), revisionCols), rec.App, rec.Layer, rec.Spec, rec.Source, changes))
}

// Heads returns the latest base and overlay revision of every app.
func (s *DomainStore) Heads(ctx context.Context) (map[string]DomainHeads, error) {
	recs, err := s.queryRevisions(ctx, fmt.Sprintf(`
		SELECT DISTINCT ON (app, layer) %s FROM %s ORDER BY app, layer, revision DESC
	`, revisionCols, s.table()))
	if err != nil {
		return nil, err
	}
	out := map[string]DomainHeads{}
	for i := range recs {
		h := out[recs[i].App]
		if recs[i].Layer == LayerBase {
			h.Base = &recs[i]
		} else {
			h.Overlay = &recs[i]
		}
		out[recs[i].App] = h
	}
	return out, nil
}

// Head returns the latest revision of each layer for app.
func (s *DomainStore) Head(ctx context.Context, app string) (DomainHeads, error) {
	return s.HeadAt(ctx, app, 0)
}

// HeadAt is Head as of revision (0 = latest).
func (s *DomainStore) HeadAt(ctx context.Context, app string, revision int64) (DomainHeads, error) {
	recs, err := s.queryRevisions(ctx, fmt.Sprintf(`
		SELECT DISTINCT ON (layer) %s FROM %s
		WHERE app = $1 AND ($2 = 0 OR revision <= $2)
		ORDER BY layer, revision DESC
	`, revisionCols, s.table()), app, revision)
	if err != nil {
		return DomainHeads{}, err
	}
	var h DomainHeads
	for i := range recs {
		if recs[i].Layer == LayerBase {
			h.Base = &recs[i]
		} else {
			h.Overlay = &recs[i]
		}
	}
	return h, nil
}

// History returns every revision of app, newest first.
func (s *DomainStore) History(ctx context.Context, app string) ([]DomainRevision, error) {
	return s.queryRevisions(ctx, fmt.Sprintf(`SELECT %s FROM %s WHERE app = $1 ORDER BY revision DESC`, revisionCols, s.table()), app)
}

// Get returns one revision, or nil when it does not exist.
func (s *DomainStore) Get(ctx context.Context, app string, revision int64) (*DomainRevision, error) {
	rec, err := scanRevision(s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE app = $1 AND revision = $2`, revisionCols, s.table()), app, revision))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
