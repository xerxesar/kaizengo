package engine

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/appspec"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tableDDL is one model whose table (create) or columns the engine ensures.
type tableDDL struct {
	model  appspec.ModelSpec
	create bool
}

// schemaDDL lists what to create for spec. With autoSchema every stored model
// is ensured; otherwise only what layers add on top of base (base tables come
// from the app's migrations).
func schemaDDL(base, spec appspec.AppSpec, autoSchema bool) []tableDDL {
	var out []tableDDL
	baseModels := map[string]appspec.ModelSpec{}
	if !autoSchema {
		for _, m := range base.Models {
			baseModels[m.Name] = m
		}
	}
	for _, m := range spec.Models {
		if m.Virtual {
			continue
		}
		bm, ok := baseModels[m.Name]
		if !ok {
			out = append(out, tableDDL{model: m, create: true})
			continue
		}
		have := map[string]struct{}{}
		for _, f := range bm.Fields {
			have[f.Name] = struct{}{}
		}
		added := appspec.ModelSpec{Name: m.Name}
		for _, f := range m.Fields {
			if _, ok := have[f.Name]; !ok {
				added.Fields = append(added.Fields, f)
			}
		}
		if len(added.Fields) > 0 {
			out = append(out, tableDDL{model: added})
		}
	}
	return out
}

// ensureSchema applies additive DDL. Tables and columns are created when
// missing; nothing is ever dropped or retyped, so business data survives any
// change to the domain. Indexes are ensured for the whole spec. Foreign keys
// are added only for many2one columns this call creates.
func ensureSchema(ctx context.Context, pool *pgxpool.Pool, schema string, spec appspec.AppSpec, ddl []tableDDL) error {
	var created []newColumn
	for _, t := range ddl {
		table := modelTable(t.model)
		for _, f := range storedFields(t.model) {
			if f.CanonicalType() != appspec.TypeMany2One {
				continue
			}
			exists, err := columnExists(ctx, pool, schema, table, colName(f.Name))
			if err != nil {
				return fmt.Errorf("model %s: %w", t.model.Name, err)
			}
			if !exists {
				created = append(created, newColumn{Model: t.model.Name, Field: f.Name})
			}
		}
		for _, stmt := range modelDDL(schema, t.model, t.create) {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("model %s: %w", t.model.Name, err)
			}
		}
	}
	for _, stmt := range relationDDL(schema, spec, created) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("relations: %w", err)
		}
	}
	return nil
}

func columnExists(ctx context.Context, pool *pgxpool.Pool, schema, table, col string) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
	`, schema, table, col).Scan(&n)
	return n > 0, err
}

func modelDDL(schema string, m appspec.ModelSpec, create bool) []string {
	table := modelTable(m)
	qt := quoteIdent(schema) + "." + quoteIdent(table)
	var stmts []string
	if create {
		stmts = append(stmts, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id         TEXT PRIMARY KEY,
			org_id     TEXT NOT NULL,
			author_id  TEXT NOT NULL,
			deleted    BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`, qt),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (org_id, updated_at DESC)`,
				quoteIdent("idx_"+table+"_org"), qt))
	}
	// many2many has no column (junction table). many2one is nullable TEXT unless
	// required, so a foreign key can reject the empty string. indexed adds a
	// btree index on (org_id, column); many2one skips it because relationDDL
	// already creates that index. Composite, unique, gin, and foreign keys are
	// emitted by relationDDL.
	for _, f := range storedFields(m) {
		col := colName(f.Name)
		if f.CanonicalType() == appspec.TypeMany2One {
			nullability := "NULL"
			if f.Required {
				nullability = "NOT NULL"
			}
			stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s TEXT %s`,
				qt, quoteIdent(col), nullability))
		} else {
			sqlType, def := columnType(f)
			stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s NOT NULL DEFAULT %s`,
				qt, quoteIdent(col), sqlType, def))
		}
		// many2one already gets (org_id, column) in relationDDL. Other indexed
		// columns lead with org_id so the index matches tenant-scoped reads.
		if f.Indexed && f.CanonicalType() != appspec.TypeMany2One {
			stmts = append(stmts, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s, %s)`,
				quoteIdent(trimIdent("idx_"+table+"_"+col)), qt, quoteIdent("org_id"), quoteIdent(col)))
		}
	}
	return stmts
}

// columnType mirrors scanPtr/projectValue: int → *int, number → *float64,
// bool → *bool, many2one → nullable text, everything else → *string.
func columnType(f appspec.FieldSpec) (sqlType, defaultExpr string) {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		return "BIGINT", "0"
	case appspec.TypeNumber:
		return "DOUBLE PRECISION", "0"
	case appspec.TypeBool:
		return "BOOLEAN", "false"
	default:
		return "TEXT", "''"
	}
}

// forceDropRemoved emits destructive DDL for fields/indexes/junctions present
// in prev but absent from next. Only called when ApplyWith Force is set.
func forceDropRemoved(ctx context.Context, pool *pgxpool.Pool, schema string, prev, next appspec.AppSpec) error {
	for _, stmt := range forceDropDDL(schema, prev, next) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func forceDropDDL(schema string, prev, next appspec.AppSpec) []string {
	nextModels := map[string]appspec.ModelSpec{}
	for _, m := range next.Models {
		nextModels[m.Name] = m
	}
	var stmts []string
	for _, pm := range prev.Models {
		if pm.Virtual {
			continue
		}
		table := modelTable(pm)
		qt := quoteIdent(schema) + "." + quoteIdent(table)
		nm, keepModel := nextModels[pm.Name]
		nextFields := map[string]appspec.FieldSpec{}
		if keepModel {
			for _, f := range nm.Fields {
				nextFields[f.Name] = f
			}
		}
		nextIx := map[string]struct{}{}
		if keepModel {
			for _, ix := range nm.Indexes {
				nextIx[indexKey(ix)] = struct{}{}
			}
		}
		for _, f := range pm.Fields {
			nf, ok := nextFields[f.Name]
			if !ok {
				stmts = append(stmts, forceRemoveFieldDDL(schema, pm, f)...)
				continue
			}
			if f.Stored() && nf.Stored() {
				prevSQL, _ := columnType(f)
				nextSQL, _ := columnType(nf)
				if prevSQL != nextSQL {
					col := colName(f.Name)
					stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, qt, quoteIdent(col)))
					// Column is re-added by ensureSchema on the next mount.
				}
			}
		}
		for _, ix := range pm.Indexes {
			if _, ok := nextIx[indexKey(ix)]; ok {
				continue
			}
			name := ix.Name
			if name == "" {
				name = trimIdent("idx_" + pm.Name + "_" + strings.Join(ix.Fields, "_"))
			}
			stmts = append(stmts, fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`, quoteIdent(schema), quoteIdent(name)))
		}
	}
	return stmts
}

func indexKey(ix appspec.IndexSpec) string {
	if ix.Name != "" {
		return "n:" + ix.Name
	}
	return "f:" + strings.Join(ix.Fields, ",") + "|u:" + fmt.Sprint(ix.Unique) + "|m:" + ix.Method
}

func forceRemoveFieldDDL(schema string, m appspec.ModelSpec, f appspec.FieldSpec) []string {
	table := modelTable(m)
	qt := quoteIdent(schema) + "." + quoteIdent(table)
	col := colName(f.Name)
	switch f.CanonicalType() {
	case appspec.TypeMany2Many:
		jt := quoteIdent(schema) + "." + quoteIdent(junctionTable(m.Name, f.Name))
		return []string{fmt.Sprintf(`DROP TABLE IF EXISTS %s`, jt)}
	case appspec.TypeOne2Many:
		return nil
	default:
		var stmts []string
		if f.Indexed {
			stmts = append(stmts, fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`,
				quoteIdent(schema), quoteIdent(trimIdent("idx_"+table+"_"+col))))
		}
		if f.Unique && f.Column() {
			stmts = append(stmts, fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`,
				quoteIdent(schema), quoteIdent(trimIdent("uidx_"+table+"_"+col))))
		}
		if f.CanonicalType() == appspec.TypeMany2One {
			stmts = append(stmts, fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`,
				quoteIdent(schema), quoteIdent(trimIdent("idx_"+table+"_"+col+"_org"))))
		}
		if f.Column() || f.CanonicalType() == appspec.TypeMany2One {
			stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, qt, quoteIdent(col)))
		}
		return stmts
	}
}
