package engine

import (
	"context"
	"fmt"

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

// ensureTables applies additive DDL. Tables and columns are created when
// missing; nothing is ever dropped or retyped, so business data survives any
// change to the domain.
func ensureTables(ctx context.Context, pool *pgxpool.Pool, schema string, ddl []tableDDL) error {
	for _, t := range ddl {
		for _, stmt := range modelDDL(schema, t.model, t.create) {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("model %s: %w", t.model.Name, err)
			}
		}
	}
	return nil
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
	for _, f := range storedFields(m) {
		col := colName(f.Name)
		sqlType, def := columnType(f)
		stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s NOT NULL DEFAULT %s`,
			qt, quoteIdent(col), sqlType, def))
		if f.Indexed {
			stmts = append(stmts, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s)`,
				quoteIdent("idx_"+table+"_"+col), qt, quoteIdent(col)))
		}
	}
	return stmts
}

// columnType mirrors scanPtr/projectValue: int → *int, number → *float64,
// bool → *bool, everything else (incl. many2many JSON ids) → *string.
func columnType(f appspec.FieldSpec) (sqlType, defaultExpr string) {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		return "BIGINT", "0"
	case appspec.TypeNumber:
		return "DOUBLE PRECISION", "0"
	case appspec.TypeBool:
		return "BOOLEAN", "false"
	case appspec.TypeMany2Many:
		return "TEXT", "'[]'"
	default:
		return "TEXT", "''"
	}
}
