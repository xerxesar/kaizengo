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

// Legacy layer labels kept for rows written before snapshot revisions.
// New writes use LayerRevision ("revision"): the tip is the full document.
const (
	LayerBase     = "base"
	LayerOverlay  = "overlay"
	LayerRevision = "revision"
)

// DomainRevision is one immutable snapshot (or legacy overlay) of an app's domain.
type DomainRevision struct {
	App       string
	Revision  int64
	Layer     string
	Spec      string
	Source    string // compiled | graphql | plugin:<name> | revert:<n> | restore
	Changes   []string
	CreatedAt time.Time
}

// DomainStore persists domain revisions in kaizengo.domain_revisions.
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
// New callers should set Layer to LayerRevision and Spec to the full document.
func (s *DomainStore) Append(ctx context.Context, rec DomainRevision) (DomainRevision, error) {
	rec.App = strings.TrimSpace(rec.App)
	if rec.App == "" {
		return DomainRevision{}, fmt.Errorf("domain app name is empty")
	}
	switch rec.Layer {
	case LayerBase, LayerOverlay, LayerRevision:
	case "":
		rec.Layer = LayerRevision
	default:
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

// Tip returns the newest revision for app (any layer label), or nil.
func (s *DomainStore) Tip(ctx context.Context, appName string) (*DomainRevision, error) {
	return s.TipAt(ctx, appName, 0)
}

// TipAt returns the newest revision with revision <= asOf (0 = latest).
func (s *DomainStore) TipAt(ctx context.Context, appName string, asOf int64) (*DomainRevision, error) {
	rec, err := scanRevision(s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE app = $1 AND ($2 = 0 OR revision <= $2)
		ORDER BY revision DESC
		LIMIT 1
	`, revisionCols, s.table()), appName, asOf))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Tips returns the tip revision of every app that has at least one revision.
func (s *DomainStore) Tips(ctx context.Context) (map[string]DomainRevision, error) {
	recs, err := s.queryRevisions(ctx, fmt.Sprintf(`
		SELECT DISTINCT ON (app) %s FROM %s ORDER BY app, revision DESC
	`, revisionCols, s.table()))
	if err != nil {
		return nil, err
	}
	out := map[string]DomainRevision{}
	for _, r := range recs {
		out[r.App] = r
	}
	return out, nil
}

// DomainHeads is retained for callers during the snapshot migration.
// Tip is preferred; Base/Overlay are filled from history when present.
type DomainHeads struct {
	Base    *DomainRevision
	Overlay *DomainRevision
	TipRev  *DomainRevision
}

// Revision is the tip revision number (0 when none).
func (h DomainHeads) Revision() int64 {
	if h.TipRev != nil {
		return h.TipRev.Revision
	}
	var n int64
	if h.Base != nil {
		n = h.Base.Revision
	}
	if h.Overlay != nil && h.Overlay.Revision > n {
		n = h.Overlay.Revision
	}
	return n
}

// Heads returns tips keyed by app (legacy shape: TipRev set; Base/Overlay best-effort).
func (s *DomainStore) Heads(ctx context.Context) (map[string]DomainHeads, error) {
	tips, err := s.Tips(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]DomainHeads{}
	for appName, tip := range tips {
		t := tip
		h := DomainHeads{TipRev: &t}
		switch tip.Layer {
		case LayerBase:
			h.Base = &t
		case LayerOverlay:
			h.Overlay = &t
		default:
			h.Overlay = &t
		}
		out[appName] = h
	}
	return out, nil
}

// Head returns the tip for app in DomainHeads form.
func (s *DomainStore) Head(ctx context.Context, appName string) (DomainHeads, error) {
	return s.HeadAt(ctx, appName, 0)
}

// HeadAt returns the tip as of revision.
func (s *DomainStore) HeadAt(ctx context.Context, appName string, revision int64) (DomainHeads, error) {
	tip, err := s.TipAt(ctx, appName, revision)
	if err != nil {
		return DomainHeads{}, err
	}
	if tip == nil {
		return DomainHeads{}, nil
	}
	h := DomainHeads{TipRev: tip}
	switch tip.Layer {
	case LayerBase:
		h.Base = tip
	default:
		h.Overlay = tip
	}
	return h, nil
}

// History returns every revision of app, newest first.
func (s *DomainStore) History(ctx context.Context, appName string) ([]DomainRevision, error) {
	return s.queryRevisions(ctx, fmt.Sprintf(`SELECT %s FROM %s WHERE app = $1 ORDER BY revision DESC`, revisionCols, s.table()), appName)
}

// Get returns one revision, or nil when it does not exist.
func (s *DomainStore) Get(ctx context.Context, appName string, revision int64) (*DomainRevision, error) {
	rec, err := scanRevision(s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE app = $1 AND revision = $2`, revisionCols, s.table()), appName, revision))
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
