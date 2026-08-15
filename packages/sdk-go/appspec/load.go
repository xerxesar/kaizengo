package appspec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// PageSuffix marks a Svelte file under views/ as a menu-mountable page.
const PageSuffix = ".page.svelte"

type fileSpec struct {
	Name       string   `yaml:"name"`
	Title      string   `yaml:"title"`
	Summary    string   `yaml:"summary"`
	Depends    []string `yaml:"depends"`
	Provides   []string `yaml:"provides"`
	Uses       []string `yaml:"uses"`
	Extensions *bool    `yaml:"extensions"`
	Schema     string   `yaml:"schema"`
	Resource   string   `yaml:"resource"`
	Events     struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"events"`
	SPA         *bool        `yaml:"spa"`
	I18n        *bool        `yaml:"i18n"`
	AutoInstall *bool        `yaml:"autoInstall"`
	Nav         NavSpec      `yaml:"nav"`
	Menus       []MenuSpec   `yaml:"menus"`
	Models      yaml.Node    `yaml:"models"`
	Views       yaml.Node    `yaml:"views"`
	Locales     []fileLocale `yaml:"locales"`
	Extends     []ExtendSpec `yaml:"extends"`
	Exports     ExportSpec   `yaml:"exports"`
}

type fileModel struct {
	Name      string      `yaml:"name"`
	Stream    string      `yaml:"stream"`
	Aggregate string      `yaml:"aggregate"`
	Internal  bool        `yaml:"internal"`
	Fields    []FieldSpec `yaml:"fields"`
	Search    *SearchSpec `yaml:"search"`
}

type fileLocale struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Dir  string `yaml:"dir"`
}

// LoadFile reads and validates an app.yaml spec from disk, resolving
// relative model includes against the file's directory and discovering
// views/*.page.svelte pages.
func LoadFile(path string) (AppSpec, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return AppSpec{}, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return AppSpec{}, err
	}
	return parse(b, filepath.Dir(abs), abs)
}

// LoadApp reads apps/<name>/app.yaml from the repo root.
func LoadApp(appName string) (AppSpec, error) {
	return LoadFile(filepath.Join("apps", appName, "app.yaml"))
}

// ListNames returns directory names under apps/ that contain app.yaml.
func ListNames() ([]string, error) {
	return ListNamesAt("apps")
}

// ListNamesAt returns app directory names under root that contain app.yaml.
func ListNamesAt(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), "app.yaml")); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

// Parse decodes YAML bytes into a validated AppSpec.
// Model includes (string paths) require LoadFile.
func Parse(b []byte) (AppSpec, error) {
	return parse(b, "", "")
}

func parse(b []byte, appRoot, source string) (AppSpec, error) {
	var raw fileSpec
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return AppSpec{}, fmt.Errorf("parse app spec: %w", err)
	}
	if raw.Views.Kind != 0 && !isNullNode(&raw.Views) {
		return AppSpec{}, fmt.Errorf("views: is not supported; add a %s file under views/ (menus reference the file name)", PageSuffix)
	}
	spec := AppSpec{
		Name:         raw.Name,
		Title:        raw.Title,
		Summary:      raw.Summary,
		Depends:      raw.Depends,
		Provides:     raw.Provides,
		Uses:         raw.Uses,
		Schema:       raw.Schema,
		Resource:     raw.Resource,
		EnableAuth:   true,
		EnableEvents: true,
		Nav:          raw.Nav,
		Menus:        raw.Menus,
	}
	if raw.Events.Enabled != nil {
		if !*raw.Events.Enabled {
			return AppSpec{}, fmt.Errorf("events.enabled: false is not supported; all apps are event-sourced")
		}
		spec.EnableEvents = true
	}
	if raw.Extensions != nil {
		spec.EnableExtensions = *raw.Extensions
	}
	spec.EnableSPA = true
	if raw.SPA != nil {
		spec.EnableSPA = *raw.SPA
	}
	spec.EnableI18n = true
	if raw.I18n != nil {
		spec.EnableI18n = *raw.I18n
	}
	if raw.AutoInstall != nil {
		spec.AutoInstall = *raw.AutoInstall
	}

	seen := map[string]struct{}{}
	if source != "" {
		seen[filepath.Clean(source)] = struct{}{}
	}
	models, err := resolveModels(&raw.Models, appRoot, source, seen)
	if err != nil {
		return AppSpec{}, err
	}
	for _, m := range models {
		spec.Models = append(spec.Models, ModelSpec{
			Name:      m.Name,
			Stream:    m.Stream,
			Aggregate: m.Aggregate,
			Internal:  m.Internal,
			Fields:    m.Fields,
			Search:    m.Search,
		})
	}
	if appRoot != "" {
		pages, err := discoverPages(appRoot)
		if err != nil {
			return AppSpec{}, err
		}
		spec.Views = pages
	}
	for _, l := range raw.Locales {
		spec.Locales = append(spec.Locales, LocaleSpec{
			ID:   l.ID,
			Name: l.Name,
			Dir:  l.Dir,
		})
	}
	spec.Extends = raw.Extends
	spec.Exports = raw.Exports
	spec.ApplyDefaults()
	if err := spec.validate(appRoot != ""); err != nil {
		return AppSpec{}, err
	}
	return spec, nil
}

func resolveModels(node *yaml.Node, appRoot, source string, seen map[string]struct{}) ([]fileModel, error) {
	if node == nil || node.Kind == 0 || isNullNode(node) {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("models: expected a list")
	}
	var models []fileModel
	for i, item := range node.Content {
		loc := fmt.Sprintf("models[%d]", i)
		ms, err := resolveModelItem(item, loc, appRoot, source, seen)
		if err != nil {
			return nil, err
		}
		models = append(models, ms...)
	}
	return models, nil
}

func resolveModelItem(node *yaml.Node, loc, appRoot, source string, seen map[string]struct{}) ([]fileModel, error) {
	if node == nil || isNullNode(node) {
		return nil, fmt.Errorf("%s: empty entry", loc)
	}
	if node.Kind == yaml.ScalarNode {
		ref := strings.TrimSpace(node.Value)
		if ref == "" {
			return nil, fmt.Errorf("%s: empty include path", loc)
		}
		return loadModelInclude(ref, loc, appRoot, source, seen)
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: expected a model object or a spec path", loc)
	}
	var m fileModel
	if err := node.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", loc, err)
	}
	if strings.TrimSpace(m.Name) == "" {
		return nil, fmt.Errorf("%s: model name is required", loc)
	}
	return []fileModel{m}, nil
}

func loadModelInclude(ref, loc, appRoot, source string, seen map[string]struct{}) ([]fileModel, error) {
	if appRoot == "" || source == "" {
		return nil, fmt.Errorf("%s: spec includes require loading from a file (got %q)", loc, ref)
	}
	abs, err := resolveIncludePath(ref, source, appRoot)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", loc, err)
	}
	if _, ok := seen[abs]; ok {
		return nil, fmt.Errorf("%s: cyclic include of %s", loc, displayPath(appRoot, abs))
	}
	seen[abs] = struct{}{}

	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", loc, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: parse %s: %w", loc, displayPath(appRoot, abs), err)
	}
	root := docRoot(&doc)
	if root == nil {
		return nil, nil
	}
	models, err := decodeModelFragment(root, appRoot, abs, seen)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", loc, displayPath(appRoot, abs), err)
	}
	return models, nil
}

func decodeModelFragment(root *yaml.Node, appRoot, source string, seen map[string]struct{}) ([]fileModel, error) {
	if root == nil || isNullNode(root) {
		return nil, nil
	}
	switch root.Kind {
	case yaml.SequenceNode:
		return resolveModels(root, appRoot, source, seen)
	case yaml.MappingNode:
		if modelsNode := mappingKey(root, "models"); modelsNode != nil {
			if extra := mappingKeysExcept(root, "models"); len(extra) > 0 {
				return nil, fmt.Errorf("fragment with models: may not contain %s", strings.Join(extra, ", "))
			}
			return resolveModels(modelsNode, appRoot, source, seen)
		}
		return resolveModelItem(root, "models[0]", appRoot, source, seen)
	default:
		return nil, fmt.Errorf("expected a list or mapping")
	}
}

func mappingKey(root *yaml.Node, name string) *yaml.Node {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if strings.TrimSpace(root.Content[i].Value) == name {
			return root.Content[i+1]
		}
	}
	return nil
}

func mappingKeysExcept(root *yaml.Node, keep string) []string {
	var extra []string
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := strings.TrimSpace(root.Content[i].Value)
		if key != "" && key != keep {
			extra = append(extra, key)
		}
	}
	return extra
}

func resolveIncludePath(ref, fromFile, appRoot string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty include path")
	}
	if filepath.IsAbs(ref) {
		return "", fmt.Errorf("include path must be relative: %s", ref)
	}
	abs := filepath.Clean(filepath.Join(filepath.Dir(fromFile), ref))
	if err := withinApp(appRoot, abs, ref); err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if isSpecPath(ref) {
			return abs, nil
		}
		return "", fmt.Errorf("include %q: %w", ref, err)
	}
	if info.IsDir() {
		for _, name := range []string{"spec.yaml", "spec.yml", filepath.Base(abs) + ".yaml"} {
			candidate := filepath.Join(abs, name)
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("directory %q has no spec.yaml", ref)
	}
	if !isSpecPath(abs) {
		return "", fmt.Errorf("include path must be a .yaml file or a model directory: %s", ref)
	}
	return abs, nil
}

func withinApp(appRoot, abs, ref string) error {
	rel, err := filepath.Rel(appRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("include %q escapes app directory", ref)
	}
	return nil
}

func discoverPages(appRoot string) ([]ViewSpec, error) {
	viewsDir := filepath.Join(appRoot, "views")
	info, err := os.Stat(viewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	seen := map[string]string{}
	var pages []ViewSpec
	err = filepath.WalkDir(viewsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, PageSuffix) {
			return nil
		}
		viewName := strings.TrimSuffix(name, PageSuffix)
		if viewName == "" {
			return fmt.Errorf("invalid page file %s", displayPath(appRoot, path))
		}
		if prev, ok := seen[viewName]; ok {
			return fmt.Errorf("duplicate page %q (%s and %s)", viewName, prev, displayPath(appRoot, path))
		}
		seen[viewName] = displayPath(appRoot, path)
		pages = append(pages, ViewSpec{Name: viewName, Type: "page"})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })
	return pages, nil
}

func isSpecPath(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func isNullNode(n *yaml.Node) bool {
	if n == nil {
		return true
	}
	return n.Tag == "!!null" || (n.Kind == yaml.ScalarNode && (n.Value == "" || n.Value == "null"))
}

func docRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	if doc.Kind == yaml.MappingNode || doc.Kind == yaml.SequenceNode {
		return doc
	}
	return nil
}

func displayPath(appRoot, abs string) string {
	rel, err := filepath.Rel(appRoot, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}
