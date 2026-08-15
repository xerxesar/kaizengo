package appspec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

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
	Models      []fileModel  `yaml:"models"`
	Views       []fileView   `yaml:"views"`
	Locales     []fileLocale `yaml:"locales"`
	Extends     []ExtendSpec `yaml:"extends"`
	Exports     ExportSpec   `yaml:"exports"`
}

type fileModel struct {
	Name      string      `yaml:"name"`
	Stream    string      `yaml:"stream"`
	Aggregate string      `yaml:"aggregate"`
	Fields    []FieldSpec `yaml:"fields"`
	Search    *SearchSpec `yaml:"search"`
}

type fileView struct {
	Name  string `yaml:"name"`
	Model string `yaml:"model"`
	Type  string `yaml:"type"`
}

type fileLocale struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Dir  string `yaml:"dir"`
}

// LoadFile reads and validates an app.yaml spec from disk.
func LoadFile(path string) (AppSpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return AppSpec{}, err
	}
	return Parse(b)
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
func Parse(b []byte) (AppSpec, error) {
	var raw fileSpec
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return AppSpec{}, fmt.Errorf("parse app spec: %w", err)
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
	for _, m := range raw.Models {
		spec.Models = append(spec.Models, ModelSpec{
			Name:      m.Name,
			Stream:    m.Stream,
			Aggregate: m.Aggregate,
			Fields:    m.Fields,
			Search:    m.Search,
		})
	}
	for _, v := range raw.Views {
		viewType := v.Type
		if viewType == "" {
			viewType = "page"
		}
		spec.Views = append(spec.Views, ViewSpec{
			Type: viewType,
			Name: v.Name,
		})
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
	if err := spec.Validate(); err != nil {
		return AppSpec{}, err
	}
	return spec, nil
}
