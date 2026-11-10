package engine

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/internal/platform/postgres"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DomainMigrateFunc is a compiled-in Go data migration for an app tip change.
// schema is the Postgres schema (search_path already set when SQL helpers run).
type DomainMigrateFunc func(ctx context.Context, pool *pgxpool.Pool, schema string, prev, next appspec.AppSpec) error

// RegisterDomainMigration registers a named Go migration for app (e.g. "backfill_ok").
func (d *Domains) RegisterDomainMigration(app, name string, fn DomainMigrateFunc) {
	if d == nil || fn == nil {
		return
	}
	app, name = strings.TrimSpace(app), strings.TrimSpace(name)
	if app == "" || name == "" {
		return
	}
	d.mmu.Lock()
	defer d.mmu.Unlock()
	if d.migrations == nil {
		d.migrations = map[string]DomainMigrateFunc{}
	}
	d.migrations[app+"/"+name] = fn
}

func (d *Domains) lookupMigration(app, name string) DomainMigrateFunc {
	d.mmu.Lock()
	defer d.mmu.Unlock()
	if d.migrations == nil {
		return nil
	}
	return d.migrations[app+"/"+name]
}

// MigrationHint describes one data-impacting tip change and a stub SQL suggestion.
type MigrationHint struct {
	Path string
	Hint string
	Stub string
}

// dataMigrationNeeded lists model/field changes that may need row data SQL.
func dataMigrationNeeded(prev *appspec.AppSpec, next appspec.AppSpec) []MigrationHint {
	if prev == nil {
		return nil
	}
	var out []MigrationHint
	prevModels := map[string]appspec.ModelSpec{}
	for _, m := range prev.Models {
		prevModels[m.Name] = m
	}
	nextModels := map[string]struct{}{}
	for _, m := range next.Models {
		nextModels[m.Name] = struct{}{}
		pm, ok := prevModels[m.Name]
		if !ok {
			if !m.Virtual {
				out = append(out, MigrationHint{
					Path: "models." + m.Name,
					Hint: "new model table (rows start empty)",
					Stub: fmt.Sprintf("-- new table %s (additive DDL creates it)\n-- INSERT INTO %s (...)", pluralize(m.Name), pluralize(m.Name)),
				})
			}
			continue
		}
		prevFields := map[string]appspec.FieldSpec{}
		for _, f := range pm.Fields {
			prevFields[f.Name] = f
		}
		nextFields := map[string]struct{}{}
		for _, f := range m.Fields {
			nextFields[f.Name] = struct{}{}
			pf, ok := prevFields[f.Name]
			if !ok {
				if f.Stored() || f.CanonicalType() == appspec.TypeMany2One {
					col := colName(f.Name)
					out = append(out, MigrationHint{
						Path: fmt.Sprintf("models.%s.fields.%s", m.Name, f.Name),
						Hint: "new column (default empty/zero unless you backfill)",
						Stub: fmt.Sprintf("-- UPDATE %s SET %s = … WHERE …;", pluralize(m.Name), quoteIdent(col)),
					})
				}
				continue
			}
			if pf.CanonicalType() != f.CanonicalType() && pf.Stored() && f.Stored() {
				out = append(out, MigrationHint{
					Path: fmt.Sprintf("models.%s.fields.%s", m.Name, f.Name),
					Hint: fmt.Sprintf("type change %s → %s", pf.CanonicalType(), f.CanonicalType()),
					Stub: fmt.Sprintf("-- ALTER TABLE %s ALTER COLUMN %s TYPE … USING …;", pluralize(m.Name), quoteIdent(colName(f.Name))),
				})
			}
		}
		for _, f := range pm.Fields {
			if _, ok := nextFields[f.Name]; ok {
				continue
			}
			if !f.Stored() && f.CanonicalType() != appspec.TypeMany2One && f.CanonicalType() != appspec.TypeMany2Many {
				continue
			}
			stub := fmt.Sprintf("-- column %s kept (soft remove). To rename:\n-- ALTER TABLE %s RENAME COLUMN %s TO new_name;\n-- To copy then drop: UPDATE …; then apply with --force",
				colName(f.Name), pluralize(m.Name), quoteIdent(colName(f.Name)))
			if f.CanonicalType() == appspec.TypeMany2Many {
				stub = fmt.Sprintf("-- junction %s (soft-kept). DROP only with --force", junctionTable(m.Name, f.Name))
			}
			out = append(out, MigrationHint{
				Path: fmt.Sprintf("models.%s.fields.%s", m.Name, f.Name),
				Hint: "field removed from API (column/junction kept unless --force)",
				Stub: stub,
			})
		}
	}
	for _, m := range prev.Models {
		if _, ok := nextModels[m.Name]; ok || m.Virtual {
			continue
		}
		out = append(out, MigrationHint{
			Path: "models." + m.Name,
			Hint: "model removed from API (table kept)",
			Stub: fmt.Sprintf("-- table %s kept; DROP only with a manual migrate + --force path later", pluralize(m.Name)),
		})
	}
	return out
}

func migrationRequiredError(hints []MigrationHint) error {
	var b strings.Builder
	b.WriteString("this tip changes stored data shape; provide a data migration before apply:\n")
	b.WriteString("  domain … --migrate-pre=\"…\"   # SQL before additive DDL (e.g. RENAME COLUMN)\n")
	b.WriteString("  domain … --migrate=\"…\"       # SQL after additive DDL (backfill)\n")
	b.WriteString("  domain … --migrate-go <name>  # compiled Go migration (post phase)\n")
	b.WriteString("  domain … --no-migrate         # skip (accept defaults / soft omit)\n\n")
	b.WriteString("changes:\n")
	for _, h := range hints {
		b.WriteString("  - ")
		b.WriteString(h.Path)
		b.WriteString(": ")
		b.WriteString(h.Hint)
		b.WriteByte('\n')
		for _, line := range strings.Split(h.Stub, "\n") {
			b.WriteString("      ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

func (d *Domains) runDomainMigratePre(ctx context.Context, appName, schema string, prev, next appspec.AppSpec, opts ApplyOptions) error {
	sql := strings.TrimSpace(opts.MigratePre)
	if sql == "" {
		return nil
	}
	if err := d.execDomainMigrateSQL(ctx, appName, schema, sql); err != nil {
		return fmt.Errorf("migrate-pre: %w", err)
	}
	return nil
}

func (d *Domains) runDomainMigratePost(ctx context.Context, appName, schema string, prev, next appspec.AppSpec, opts ApplyOptions) error {
	sql := strings.TrimSpace(opts.MigratePost)
	goName := strings.TrimSpace(opts.MigrateGo)
	if sql == "" && goName == "" {
		return nil
	}
	db, err := postgres.FromHost(d.host)
	if err != nil || db == nil || db.Pool() == nil {
		return fmt.Errorf("migrate: postgres not available: %w", err)
	}
	pool := db.Pool()
	if schema == "" {
		schema = appName
	}
	if goName != "" {
		fn := d.lookupMigration(appName, goName)
		if fn == nil {
			return fmt.Errorf("migrate-go %q not registered for app %q", goName, appName)
		}
		if err := fn(ctx, pool, schema, prev, next); err != nil {
			return fmt.Errorf("migrate-go %s: %w", goName, err)
		}
	}
	if sql != "" {
		if err := execMigrateSQL(ctx, pool, schema, sql); err != nil {
			return fmt.Errorf("migrate-post: %w", err)
		}
	}
	return nil
}

func (d *Domains) execDomainMigrateSQL(ctx context.Context, appName, schema, sql string) error {
	db, err := postgres.FromHost(d.host)
	if err != nil || db == nil || db.Pool() == nil {
		return fmt.Errorf("postgres not available: %w", err)
	}
	if schema == "" {
		schema = appName
	}
	return execMigrateSQL(ctx, db.Pool(), schema, sql)
}

func execMigrateSQL(ctx context.Context, pool *pgxpool.Pool, schema, sql string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path', $1, true)`, schema+", public"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
