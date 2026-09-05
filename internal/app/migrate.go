package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/internal/events/pgstore"
)

// ApplyMigrationsFromDir loads apps/<appName>/migrations/*.sql in sorted order
// and applies any that have not been recorded in schema_migrations.
func ApplyMigrationsFromDir(ctx context.Context, store *pgstore.Store, appName, schema string) error {
	dir := filepath.Join("apps", appName, "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("app %q requires migrations at %s", appName, dir)
		}
		return err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("app %q: no .sql files in %s", appName, dir)
	}
	migrations := make(map[string]string, len(names))
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		ver := strings.TrimSuffix(name, ".sql")
		migrations[ver] = withSearchPath(schema, string(b))
	}
	return store.ApplyMigrations(ctx, migrations)
}

// ApplyMigrationsFromFS applies embedded SQL files from migrations/*.sql.
// versionPrefix is prepended to each filename stem for schema_migrations.version
// (e.g. "identity_" → "identity_001_init").
func ApplyMigrationsFromFS(ctx context.Context, store *pgstore.Store, migrations fs.FS, schema, versionPrefix string) error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("no .sql migrations found")
	}
	files := make(map[string]string, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(migrations, filepath.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		ver := versionPrefix + strings.TrimSuffix(name, ".sql")
		files[ver] = withSearchPath(schema, string(b))
	}
	return store.ApplyMigrations(ctx, files)
}

// Migrate applies pending SQL files for an app without registering GraphQL or nav.
func Migrate(host *module.Host, appName string) error {
	spec, err := appspec.LoadApp(appName)
	if err != nil {
		return err
	}
	schema := Env("KaizenGo_"+strings.ToUpper(appName)+"_SCHEMA", spec.Schema)
	ctx := context.Background()
	store, err := SchemaStore(ctx, host, schema)
	if err != nil {
		return fmt.Errorf("%s: %w", appName, err)
	}
	return ApplyMigrationsFromDir(ctx, store, appName, schema)
}

func withSearchPath(schema, sql string) string {
	return fmt.Sprintf("SET LOCAL search_path TO %s, public;\n%s", quoteIdent(schema), sql)
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
