package engine

import (
	"fmt"
	"regexp"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

// newColumn is a many2one column this ensure created, so it can take a foreign key.
type newColumn struct {
	Model string
	Field string
}

func trimIdent(name string) string {
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

// junctionTable is schema-local "{model}_{field}_rel".
func junctionTable(model, field string) string {
	return trimIdent(model + "_" + colName(field) + "_rel")
}

// relationDDL is the second pass: indexes for the whole spec, junction tables
// for many2many fields the engine owns, and foreign keys for columns created
// in this pass. Statements are additive.
func relationDDL(schema string, spec appspec.AppSpec, created []newColumn) []string {
	var stmts []string
	fresh := map[string]struct{}{}
	for _, c := range created {
		fresh[c.Model+"."+c.Field] = struct{}{}
	}
	for _, m := range spec.Models {
		if m.Virtual {
			continue
		}
		table := modelTable(m)
		qt := quoteIdent(schema) + "." + quoteIdent(table)
		for _, f := range m.Fields {
			col := colName(f.Name)
			if f.CanonicalType() == appspec.TypeMany2One {
				stmts = append(stmts, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s, %s)`,
					quoteIdent(trimIdent("idx_"+table+"_"+col+"_org")), qt, quoteIdent("org_id"), quoteIdent(col)))
			}
			if f.Unique && f.Column() {
				stmts = append(stmts, uniqueIndexSQL(schema, table, trimIdent("uidx_"+table+"_"+col), []string{col}, appspec.IndexSpec{}, nil, ""))
			}
			if f.CanonicalType() == appspec.TypeOne2Many {
				if ix := inverseIndexSQL(schema, spec.Name, f); ix != "" {
					stmts = append(stmts, ix)
				}
			}
			if f.CanonicalType() == appspec.TypeMany2Many {
				stmts = append(stmts, junctionDDL(schema, spec.Name, m, f)...)
			}
		}
		for _, ix := range m.Indexes {
			method := ix.Method
			if method == "" {
				method = "btree"
			}
			name := ix.Name
			if name == "" {
				name = trimIdent("idx_" + m.Name + "_" + strings.Join(ix.Fields, "_"))
			}
			cols := make([]string, len(ix.Fields))
			for i, field := range ix.Fields {
				cols[i] = colName(field)
			}
			where, err := ix.WhereSQL(func(col string) string {
				switch col {
				case "id", "org_id", "deleted":
					return col
				}
				if f, ok := fieldByName(m, col); ok {
					return colName(f.Name)
				}
				return col
			})
			if err != nil {
				where = ""
			}
			include := make([]string, len(ix.Include))
			for i, field := range ix.Include {
				include[i] = colName(field)
			}
			if ix.Unique {
				stmts = append(stmts, uniqueIndexSQL(schema, table, name, cols, ix, include, where))
				continue
			}
			stmts = append(stmts, indexSQL(schema, table, name, cols, ix, include, where, method))
		}
	}
	for _, c := range created {
		m, ok := modelByName(spec, c.Model)
		if !ok {
			continue
		}
		f, ok := fieldByName(m, c.Field)
		if !ok || f.CanonicalType() != appspec.TypeMany2One {
			continue
		}
		if _, freshOK := fresh[c.Model+"."+c.Field]; !freshOK {
			continue
		}
		ref, ok := f.ResolveRelation(spec.Name)
		if !ok {
			continue
		}
		stmts = append(stmts, foreignKeySQL(
			schema, modelTable(m), colName(f.Name),
			targetSchema(schema, spec.Name, ref), pluralize(ref.Model),
			f.OnDeleteAction(),
		))
	}
	return stmts
}

func fieldByName(m appspec.ModelSpec, name string) (appspec.FieldSpec, bool) {
	for _, f := range m.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return appspec.FieldSpec{}, false
}

func targetSchema(localSchema, localApp string, ref appspec.RelationRef) string {
	if ref.App == "" || ref.App == localApp {
		return localSchema
	}
	return ref.App
}

func indexSQL(schema, table, name string, cols []string, ix appspec.IndexSpec, include []string, where, method string) string {
	qt := quoteIdent(schema) + "." + quoteIdent(table)
	switch method {
	case "hash":
		return fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s USING hash (%s)`,
			quoteIdent(name), qt, quoteIdent(cols[0]))
	case "gin":
		return fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s USING gin (to_tsvector('simple', %s))`,
			quoteIdent(name), qt, quoteIdent(cols[0]))
	default:
		return "CREATE INDEX IF NOT EXISTS " + quoteIdent(name) + " ON " + qt + " (" + quoteOrdered(cols, ix) + ")" + includeSQL(include) + whereSQL(where)
	}
}

func quoteOrdered(cols []string, ix appspec.IndexSpec) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
		if ix.OrderAt(i) == "desc" {
			quoted[i] += " DESC"
		}
	}
	return strings.Join(quoted, ", ")
}

func includeSQL(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
	}
	return " INCLUDE (" + strings.Join(quoted, ", ") + ")"
}

func whereSQL(where string) string {
	if strings.TrimSpace(where) == "" {
		return ""
	}
	return " WHERE " + where
}

func uniqueIndexSQL(schema, table, name string, cols []string, ix appspec.IndexSpec, include []string, where string) string {
	qt := quoteIdent(schema) + "." + quoteIdent(table)
	parts := []string{quoteIdent("org_id")}
	for i, c := range cols {
		if c == "org_id" {
			continue
		}
		part := quoteIdent(c)
		if ix.OrderAt(i) == "desc" {
			part += " DESC"
		}
		parts = append(parts, part)
	}
	pred := "deleted = false"
	if strings.TrimSpace(where) != "" {
		pred += " AND " + where
	}
	return fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)%s WHERE %s`,
		quoteIdent(trimIdent(name)), qt, strings.Join(parts, ", "), includeSQL(include), pred)
}

func inverseIndexSQL(schema, app string, f appspec.FieldSpec) string {
	ref, ok := f.ResolveRelation(app)
	if !ok || strings.TrimSpace(f.Inverse) == "" {
		return ""
	}
	table := pluralize(ref.Model)
	col := colName(f.Inverse)
	qt := quoteIdent(targetSchema(schema, app, ref)) + "." + quoteIdent(table)
	return fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s, %s)`,
		quoteIdent(trimIdent("idx_"+table+"_"+col+"_org")), qt, quoteIdent("org_id"), quoteIdent(col))
}

func junctionDDL(schema, app string, m appspec.ModelSpec, f appspec.FieldSpec) []string {
	ref, ok := f.ResolveRelation(app)
	if !ok {
		return nil
	}
	name := junctionTable(m.Name, f.Name)
	qt := quoteIdent(schema) + "." + quoteIdent(name)
	return []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			org_id   TEXT NOT NULL,
			left_id  TEXT NOT NULL,
			right_id TEXT NOT NULL,
			PRIMARY KEY (org_id, left_id, right_id)
		)`, qt),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s, %s)`,
			quoteIdent(trimIdent("idx_"+name+"_right")), qt, quoteIdent("org_id"), quoteIdent("right_id")),
		foreignKeySQL(schema, name, "left_id", schema, modelTable(m), "CASCADE"),
		foreignKeySQL(schema, name, "right_id", targetSchema(schema, app, ref), pluralize(ref.Model), "CASCADE"),
	}
}

func foreignKeySQL(schema, table, col, targetSchema, targetTable, action string) string {
	return fmt.Sprintf(`DO $$
BEGIN
  ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON DELETE %s;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$`,
		quoteIdent(schema)+"."+quoteIdent(table),
		quoteIdent(trimIdent("fk_"+table+"_"+col)),
		quoteIdent(col),
		quoteIdent(targetSchema)+"."+quoteIdent(targetTable),
		quoteIdent("id"),
		action,
	)
}

var m2mPred = regexp.MustCompile(`m2mfilter_([a-z0-9_]+) (IS DISTINCT FROM|=) (\$\d+)|m2mfilter_([a-z0-9_]+) (NOT IN|IN) (\([^)]*\))`)

func (s *modelService) rewriteRelationSQL(clause string) string {
	if !strings.Contains(clause, "m2mfilter_") {
		return clause
	}
	return m2mPred.ReplaceAllStringFunc(clause, func(m string) string {
		p := m2mPred.FindStringSubmatch(m)
		var col, expr string
		not := false
		if p[1] != "" {
			col = p[1]
			expr = "r.right_id = " + p[3]
			not = p[2] != "="
		} else {
			col = p[4]
			not = strings.HasPrefix(p[5], "NOT")
			expr = "r.right_id IN " + p[6]
		}
		jt := quoteIdent(s.schema) + "." + quoteIdent(junctionTable(s.model.Name, col))
		q := fmt.Sprintf(`EXISTS (SELECT 1 FROM %s r WHERE r.org_id = %s.org_id AND r.left_id = %s.id AND %s)`,
			jt, s.qtable(), s.qtable(), expr)
		if not {
			return "NOT (" + q + ")"
		}
		return q
	})
}
