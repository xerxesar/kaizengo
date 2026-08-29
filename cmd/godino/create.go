package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type AppOptions struct {
	Name         string
	Type         string
	Title        string
	Summary      string
	WithGraphQL  bool
	EventSourced bool
	Addon        bool
}

type appData struct {
	Name         string
	Pkg          string
	Title        string
	Summary      string
	Route        string
	WithGraphQL  bool
	IsSolid      bool
	EventSourced bool
}

func createApp(opts AppOptions) error {
	name := strings.TrimSpace(opts.Name)
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid name %q (use lowercase letters, digits, underscore; start with a letter)", name)
	}
	if name == "core" || name == "apps" {
		return fmt.Errorf("%q is reserved", name)
	}

	typ := strings.ToLower(strings.TrimSpace(opts.Type))
	if typ != "vanilla" && typ != "solid" && typ != "svelte" {
		return fmt.Errorf("type must be solid, got %q (svelte is deprecated; use solid)", typ)
	}
	if typ == "svelte" {
		typ = "solid"
	}

	root, err := findModuleRoot()
	if err != nil {
		return err
	}

	title := opts.Title
	if title == "" {
		title = strings.ToUpper(name[:1]) + name[1:]
		title = strings.ReplaceAll(title, "_", " ")
	}
	summary := opts.Summary
	if summary == "" {
		summary = title + " app"
	}

	if opts.Addon {
		return createAddon(root, name, title, summary)
	}

	appDir := filepath.Join(root, "apps", name)
	if _, err := os.Stat(appDir); err == nil {
		return fmt.Errorf("app already exists: %s", appDir)
	} else if !os.IsNotExist(err) {
		return err
	}

	data := appData{
		Name:         name,
		Pkg:          name,
		Title:        title,
		Summary:      summary,
		Route:        strings.ReplaceAll(name, "_", "-"),
		WithGraphQL:  opts.WithGraphQL,
		IsSolid:      typ == "solid",
		EventSourced: opts.EventSourced,
	}

	if err := os.MkdirAll(filepath.Join(appDir, "views"), 0o755); err != nil {
		return err
	}

	files := map[string]string{}
	if opts.EventSourced {
		files["app.yaml"] = render(appSpecYAMLTmpl, data)
		files["module.go"] = render(eventSourcedModuleGoTmpl, data)
		files["hooks.go"] = render(eventSourcedHooksGoTmpl, data)
		files["migrations/001_events.sql"] = render(eventsMigrationSQLTmpl, data)
		files["migrations/002_items_read.sql"] = render(readModelMigrationSQLTmpl, data)
		files["locale/en.po"] = render(localeEnTmpl, data)
		files["locale/fa.po"] = render(localeFaTmpl, data)
		files["views/Items.page.tsx"] = render(solidListViewTmpl, data)
		files["views/NewItem.page.tsx"] = render(solidFormViewTmpl, data)
	} else {
		files["module.go"] = render(moduleGoTmpl, data)
	}
	if typ == "vanilla" {
		return fmt.Errorf("vanilla apps are no longer supported; use --type solid")
	}
	if !opts.EventSourced {
		files["views/Index.page.tsx"] = render(solidViewTmpl, data)
	}
	if opts.WithGraphQL {
		if err := os.MkdirAll(filepath.Join(appDir, "lib"), 0o755); err != nil {
			return err
		}
		files["lib/graphql.ts"] = render(solidGraphQLTSTmpl, data)
	}

	for rel, content := range files {
		path := filepath.Join(appDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if strings.HasSuffix(rel, ".go") {
			formatted, err := format.Source([]byte(content))
			if err != nil {
				return fmt.Errorf("format %s: %w\n----\n%s", rel, err, content)
			}
			content = string(formatted)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Println("created", filepath.Join("apps", name, rel))
	}

	if err := registerBlankImport(root, name); err != nil {
		return err
	}
	fmt.Println("updated apps/apps.go")

	if opts.EventSourced {
		if err := runGenTypes([]string{name}); err != nil {
			return fmt.Errorf("gen-types: %w", err)
		}
	}

	fmt.Printf("\nNext steps:\n")
	if typ == "solid" {
		fmt.Printf("  make spa-build   # rebuild the central SPA (includes app views)\n")
	}
	fmt.Printf("  make generate     # refresh __types__ after app.yaml changes\n")
	fmt.Printf("  go run ./cmd/server\n")
	fmt.Printf("  open http://localhost:8080/app/%s\n", data.Route)
	return nil
}

func createAddon(root, name, title, summary string) error {
	appDir := filepath.Join(root, "apps", name)
	if _, err := os.Stat(appDir); err == nil {
		return fmt.Errorf("app already exists: %s", appDir)
	} else if !os.IsNotExist(err) {
		return err
	}
	data := appData{
		Name:    name,
		Pkg:     name,
		Title:   title,
		Summary: summary,
	}
	files := map[string]string{
		"app.yaml":  render(addonAppYamlTmpl, data),
		"module.go": render(addonModuleGoTmpl, data),
	}
	for rel, content := range files {
		path := filepath.Join(appDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if strings.HasSuffix(rel, ".go") {
			formatted, err := format.Source([]byte(content))
			if err != nil {
				return fmt.Errorf("format %s: %w\n----\n%s", rel, err, content)
			}
			content = string(formatted)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Println("created", filepath.Join("apps", name, rel))
	}
	if err := registerBlankImport(root, name); err != nil {
		return err
	}
	fmt.Println("updated apps/apps.go")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Implement named handlers and call extension.RegisterNamed in init()\n")
	fmt.Printf("  Wire handlers in app.yaml extends: entries\n")
	fmt.Printf("  go run ./cmd/server\n")
	return nil
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found; run from inside the kaizengo module")
		}
		dir = parent
	}
}

func registerBlankImport(root, name string) error {
	path := filepath.Join(root, "apps", "apps.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	imp := fmt.Sprintf("\t_ \"kaizengo/apps/%s\"\n", name)
	if bytes.Contains(b, []byte(imp)) || bytes.Contains(b, []byte(fmt.Sprintf("kaizengo/apps/%s\"", name))) {
		return nil
	}

	const marker = "import (\n"
	i := bytes.Index(b, []byte(marker))
	if i < 0 {
		return fmt.Errorf("could not find import block in apps/apps.go")
	}
	insertAt := i + len(marker)
	var out bytes.Buffer
	out.Write(b[:insertAt])
	out.WriteString(imp)
	out.Write(b[insertAt:])
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return fmt.Errorf("format apps.go: %w", err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

func render(tmpl string, data appData) string {
	t := template.Must(template.New("").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}
