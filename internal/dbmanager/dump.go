package dbmanager

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DumpDatabase writes a plain-SQL backup via pgx (no external pg_dump).
func DumpDatabase(dsn string, w io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect for backup: %w", err)
	}
	defer pool.Close()

	if _, err := io.WriteString(w, "-- KaizenGo SQL backup (pgx)\n\n"); err != nil {
		return err
	}

	schemas, err := listUserSchemas(ctx, pool)
	if err != nil {
		return err
	}
	for _, schema := range schemas {
		if _, err := fmt.Fprintf(w, "CREATE SCHEMA IF NOT EXISTS %s;\n\n", quoteIdent(schema)); err != nil {
			return err
		}
		tables, err := listTables(ctx, pool, schema)
		if err != nil {
			return err
		}
		for _, table := range tables {
			cols, err := tableColumns(ctx, pool, schema, table)
			if err != nil {
				return err
			}
			if len(cols) == 0 {
				continue
			}
			ddl := buildCreateTable(schema, table, cols)
			if _, err := io.WriteString(w, ddl+"\n\n"); err != nil {
				return err
			}
			fq := quoteIdent(schema) + "." + quoteIdent(table)
			colList := quoteColumnList(cols)
			if _, err := fmt.Fprintf(w, "COPY %s (%s) FROM STDIN;\n", fq, colList); err != nil {
				return err
			}
			acquired, err := pool.Acquire(ctx)
			if err != nil {
				return err
			}
			copySQL := fmt.Sprintf("COPY %s (%s) TO STDOUT", fq, colList)
			_, err = acquired.Conn().PgConn().CopyTo(ctx, w, copySQL)
			acquired.Release()
			if err != nil {
				return fmt.Errorf("copy %s.%s: %w", schema, table, err)
			}
			if _, err := io.WriteString(w, "\\.\n\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

// RestoreDatabase applies a DumpDatabase SQL file into an existing database via pgx.
func RestoreDatabase(dsn, dumpPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	f, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer f.Close()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect for restore: %w", err)
	}
	defer pool.Close()

	return execSQLBackup(ctx, pool, f)
}

// WriteTempDump writes r to a temporary file and returns its path.
func WriteTempDump(r io.Reader) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "kaizengo-restore-*.sql")
	if err != nil {
		return "", nil, err
	}
	path = f.Name()
	cleanup = func() { _ = os.Remove(path) }
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

type columnDef struct {
	Name     string
	Type     string
	NotNull  bool
	Default  string
	IsSerial bool
}

func listUserSchemas(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT nspname FROM pg_namespace
		WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		  AND nspname NOT LIKE 'pg_toast%'
		  AND nspname NOT LIKE 'pg_temp%'
		ORDER BY nspname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func listTables(ctx context.Context, pool *pgxpool.Pool, schema string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind = 'r'
		ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func tableColumns(ctx context.Context, pool *pgxpool.Pool, schema, table string) ([]columnDef, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.attname,
		       pg_catalog.format_type(a.atttypid, a.atttypmod),
		       a.attnotnull,
		       COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '')
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
		WHERE n.nspname = $1 AND c.relname = $2
		  AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []columnDef
	for rows.Next() {
		var col columnDef
		if err := rows.Scan(&col.Name, &col.Type, &col.NotNull, &col.Default); err != nil {
			return nil, err
		}
		out = append(out, col)
	}
	return out, rows.Err()
}

func buildCreateTable(schema, table string, cols []columnDef) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s.%s (\n", quoteIdent(schema), quoteIdent(table))
	for i, c := range cols {
		fmt.Fprintf(&b, "  %s %s", quoteIdent(c.Name), c.Type)
		if c.Default != "" {
			fmt.Fprintf(&b, " DEFAULT %s", c.Default)
		}
		if c.NotNull {
			b.WriteString(" NOT NULL")
		}
		if i < len(cols)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString(");")
	return b.String()
}

func quoteColumnList(cols []columnDef) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = quoteIdent(c.Name)
	}
	return strings.Join(parts, ", ")
}

func execSQLBackup(ctx context.Context, pool *pgxpool.Pool, r io.Reader) error {
	sc := bufio.NewScanner(r)
	// Allow large COPY lines.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 16*1024*1024)

	var stmt strings.Builder
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") || trim == "" {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trim), "COPY ") && strings.HasSuffix(strings.ToUpper(trim), "FROM STDIN;") {
			copySQL := strings.TrimSuffix(trim, ";")
			// Read COPY data until \.
			var data strings.Builder
			for sc.Scan() {
				row := sc.Text()
				if row == "\\." {
					break
				}
				data.WriteString(row)
				data.WriteByte('\n')
			}
			acquired, err := pool.Acquire(ctx)
			if err != nil {
				return err
			}
			_, err = acquired.Conn().PgConn().CopyFrom(ctx, strings.NewReader(data.String()), copySQL)
			acquired.Release()
			if err != nil {
				return fmt.Errorf("restore copy: %w", err)
			}
			stmt.Reset()
			continue
		}
		stmt.WriteString(line)
		stmt.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			sql := strings.TrimSpace(stmt.String())
			stmt.Reset()
			if sql == "" {
				continue
			}
			if _, err := pool.Exec(ctx, sql); err != nil {
				return fmt.Errorf("restore exec: %w\nSQL: %s", err, truncate(sql, 200))
			}
		}
	}
	return sc.Err()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
