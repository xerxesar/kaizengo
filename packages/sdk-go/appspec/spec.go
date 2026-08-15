package appspec

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var fieldNameRe = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]*$`)

// AppSpec is the declarative module contract for Odoo-like app scaffolding.
type AppSpec struct {
	Name             string
	Title            string
	Summary          string
	Depends          []string
	Provides         []string
	Uses             []string
	EnableExtensions bool
	Schema           string // Postgres schema (default: Name)
	Resource         string // permissions resource (default: Name)
	EnableAuth       bool
	EnableI18n       bool
	EnableSPA        bool
	EnableEvents     bool
	AutoInstall      bool // install on first boot; always loaded with the shell
	Nav              NavSpec
	Menus            []MenuSpec
	Models           []ModelSpec
	Views            []ViewSpec
	Locales          []LocaleSpec
	Extends          []ExtendSpec
	Exports          ExportSpec
}

type ModelSpec struct {
	Name      string
	Stream    string
	Aggregate string
	Fields    []FieldSpec
	Search    *SearchSpec
}

// SearchSpec declares a search collection for a model.
type SearchSpec struct {
	Collection string   `yaml:"collection"`
	Fields     []string `yaml:"fields"`
}

type FieldSpec struct {
	Name     string       `yaml:"name"`
	Type     string       `yaml:"type"` // see CanonicalFieldType
	Required bool         `yaml:"required"`
	Indexed  bool         `yaml:"indexed"`
	Readonly bool         `yaml:"readonly"`
	Default  any          `yaml:"default"`
	Values   []string     `yaml:"values"`   // required when type is enum
	Relation string       `yaml:"relation"` // app.model or model (relation types)
	Inverse  string       `yaml:"inverse"`  // one2many: field name on the related model
	Validate ValidateSpec `yaml:"validate"`
}

// IsEnum reports whether the field is an enumeration.
func (f FieldSpec) IsEnum() bool {
	return strings.EqualFold(f.Type, "enum") || len(f.Values) > 0
}

// EnumTypeName returns the Go type name for an enum field (e.g. UserStatus).
func (f FieldSpec) EnumTypeName(model string) string {
	return pascal(model) + pascal(f.Name)
}

// ValidateSpec declares declarative field constraints (spec hooks).
type ValidateSpec struct {
	MinLength int    `yaml:"minLength"`
	MaxLength int    `yaml:"maxLength"`
	Pattern   string `yaml:"pattern"`
	Min       *int   `yaml:"min"`
	Max       *int   `yaml:"max"`
}

type ViewSpec struct {
	Name string
	Type string // page — list/form/kanban metadata is auto-generated from models
}

// NavSpec configures the shell apps dropdown entry for this module.
type NavSpec struct {
	Label    string `yaml:"label"`
	LabelKey string `yaml:"labelKey"`
	Route    string `yaml:"route"`
	Order    int    `yaml:"order"`
}

// MenuSpec declares in-app navigation (Odoo-style menuitems).
type MenuSpec struct {
	ID       string     `yaml:"id"`
	Label    string     `yaml:"label"`
	LabelKey string     `yaml:"labelKey"`
	View     string     `yaml:"view"`
	Route    string     `yaml:"route"`
	Order    int        `yaml:"order"`
	Children []MenuSpec `yaml:"children"`
}

type LocaleSpec struct {
	ID   string
	Name string
	Dir  string // ltr|rtl
}

func (s AppSpec) Validate() error {
	if !nameRe.MatchString(s.Name) {
		return fmt.Errorf("invalid app name %q (must match %s)", s.Name, nameRe.String())
	}
	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(s.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	models := map[string]struct{}{}
	for _, m := range s.Models {
		if !nameRe.MatchString(m.Name) {
			return fmt.Errorf("invalid model name %q", m.Name)
		}
		if _, ok := models[m.Name]; ok {
			return fmt.Errorf("duplicate model name %q", m.Name)
		}
		models[m.Name] = struct{}{}
		if strings.TrimSpace(m.Stream) == "" {
			return fmt.Errorf("model %q stream is required", m.Name)
		}
		for _, f := range m.Fields {
			if !fieldNameRe.MatchString(f.Name) {
				return fmt.Errorf("model %q has invalid field %q", m.Name, f.Name)
			}
			canon := f.CanonicalType()
			if _, ok := fieldTypeAliases[canon]; !ok && canon != TypeEnum {
				return fmt.Errorf("model %q field %q has unsupported type %q", m.Name, f.Name, f.Type)
			}
			if canon == TypeEnum && len(f.Values) == 0 {
				return fmt.Errorf("model %q field %q (enum) requires values:", m.Name, f.Name)
			}
			if f.IsRelation() {
				if strings.TrimSpace(f.Relation) == "" {
					return fmt.Errorf("model %q field %q (%s) requires relation:", m.Name, f.Name, canon)
				}
				if _, ok := ParseRelation(f.Relation); !ok {
					return fmt.Errorf("model %q field %q has invalid relation %q (use model or app.model)", m.Name, f.Name, f.Relation)
				}
				if canon == TypeOne2Many && strings.TrimSpace(f.Inverse) == "" {
					return fmt.Errorf("model %q field %q (one2many) requires inverse:", m.Name, f.Name)
				}
			}
			if len(f.Values) > 0 {
				seen := map[string]struct{}{}
				for _, v := range f.Values {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("model %q field %q has empty enum value", m.Name, f.Name)
					}
					if _, ok := seen[v]; ok {
						return fmt.Errorf("model %q field %q has duplicate enum value %q", m.Name, f.Name, v)
					}
					seen[v] = struct{}{}
				}
				if f.Default != nil {
					def := fmt.Sprint(f.Default)
					if _, ok := seen[def]; !ok {
						return fmt.Errorf("model %q field %q default %q is not in values", m.Name, f.Name, def)
					}
				}
			}
			if f.Readonly && f.Required && f.Default == nil {
				return fmt.Errorf("model %q field %q is readonly+required but has no default", m.Name, f.Name)
			}
		}
	}
	for _, v := range s.Views {
		switch v.Type {
		case "", "page":
			// Svelte page views — menus mount apps/{app}/views/{Name}.svelte.
		default:
			return fmt.Errorf("view %q has unsupported type %q (only page views belong in app.yaml; list/form views are generated from models)", v.Name, v.Type)
		}
	}
	for _, l := range s.Locales {
		if l.ID == "" {
			return fmt.Errorf("locale id is required")
		}
		if l.Dir != "" && l.Dir != "ltr" && l.Dir != "rtl" {
			return fmt.Errorf("locale %q has invalid dir %q", l.ID, l.Dir)
		}
	}
	if err := validateMenus(s.Menus, viewNames(s.Views), map[string]struct{}{}); err != nil {
		return err
	}
	if err := validateCapabilityNames("provides", s.Provides); err != nil {
		return err
	}
	if err := validateCapabilityNames("uses", s.Uses); err != nil {
		return err
	}
	if err := validateExtends(s.Extends); err != nil {
		return err
	}
	if err := validateExports(s.Exports); err != nil {
		return err
	}
	return nil
}

func viewNames(views []ViewSpec) map[string]struct{} {
	out := make(map[string]struct{}, len(views))
	for _, v := range views {
		out[v.Name] = struct{}{}
	}
	return out
}

func validateMenus(items []MenuSpec, views map[string]struct{}, seen map[string]struct{}) error {
	for _, m := range items {
		if !nameRe.MatchString(m.ID) {
			return fmt.Errorf("menu %q has invalid id (must match %s)", m.ID, nameRe.String())
		}
		if _, ok := seen[m.ID]; ok {
			return fmt.Errorf("duplicate menu id %q", m.ID)
		}
		seen[m.ID] = struct{}{}
		if strings.TrimSpace(m.Label) == "" && strings.TrimSpace(m.LabelKey) == "" {
			return fmt.Errorf("menu %q requires label or labelKey", m.ID)
		}
		if m.View != "" {
			if _, ok := views[m.View]; !ok {
				return fmt.Errorf("menu %q references unknown view %q", m.ID, m.View)
			}
		}
		if m.View == "" && len(m.Children) == 0 {
			return fmt.Errorf("menu %q must set view and/or children", m.ID)
		}
		if err := validateMenus(m.Children, views, seen); err != nil {
			return err
		}
	}
	return nil
}

// ApplyDefaults fills schema, resource, streams, and aggregates when omitted.
func (s *AppSpec) ApplyDefaults() {
	if s.Schema == "" {
		s.Schema = s.Name
	}
	if s.Resource == "" {
		s.Resource = s.Name
	}
	if s.Nav.LabelKey == "" && s.Nav.Label == "" {
		s.Nav.LabelKey = "nav." + s.Name
	}
	if s.Nav.Label == "" && s.Nav.LabelKey == "nav."+s.Name {
		s.Nav.Label = s.Title
	}
	if s.Nav.Route == "" {
		s.Nav.Route = s.Name
	}
	if len(s.Menus) == 0 {
		s.Menus = defaultMenusFromViews(s.Views)
	}
	for i := range s.Models {
		m := &s.Models[i]
		if m.Stream == "" {
			m.Stream = s.Name + "." + m.Name
		}
		if m.Aggregate == "" {
			m.Aggregate = pascal(m.Name) + "Aggregate"
		}
		for j := range m.Fields {
			f := &m.Fields[j]
			if f.Type == "" {
				if len(f.Values) > 0 {
					f.Type = TypeEnum
				} else if strings.TrimSpace(f.Relation) != "" {
					f.Type = TypeMany2One
				} else {
					f.Type = TypeString
				}
			} else {
				f.Type = CanonicalFieldType(f.Type)
			}
			if f.Type == TypeEnum && f.Default == nil && len(f.Values) > 0 && !f.Required {
				f.Default = f.Values[0]
			}
		}
	}
}

func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func defaultMenusFromViews(views []ViewSpec) []MenuSpec {
	out := make([]MenuSpec, 0, len(views))
	for i, v := range views {
		out = append(out, MenuSpec{
			ID:    menuIDFromView(v.Name),
			Label: humanViewLabel(v.Name),
			View:  v.Name,
			Order: i * 10,
		})
	}
	return out
}

func menuIDFromView(name string) string {
	var b strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func humanViewLabel(name string) string {
	var b strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' && i > 0 {
			b.WriteByte(' ')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		} else {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
