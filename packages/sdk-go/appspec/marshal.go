package appspec

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// MarshalDomain renders the domain part of spec (models, queries, commands,
// policies) as one self-contained app.yaml document: models and security are
// inline, so the output parses back with Parse. UI wiring (nav, menus, views,
// locales, keymap) and seeded users are left out.
func MarshalDomain(spec AppSpec) ([]byte, error) {
	doc := outSpec{
		Name:     spec.Name,
		Title:    spec.Title,
		Summary:  spec.Summary,
		Depends:  spec.Depends,
		Provides: spec.Provides,
		Uses:     spec.Uses,
	}
	if spec.Schema != "" && spec.Schema != spec.Name {
		doc.Schema = spec.Schema
	}
	if spec.Resource != "" && spec.Resource != spec.Name {
		doc.Resource = spec.Resource
	}
	if spec.EnableExtensions {
		t := true
		doc.Extensions = &t
	}
	for _, m := range spec.Models {
		om := outModel{Name: m.Name, Internal: m.Internal, Virtual: m.Virtual, Search: m.Search}
		for _, f := range m.Fields {
			om.Fields = append(om.Fields, outFieldOf(f))
		}
		doc.Models = append(doc.Models, om)
	}
	for _, q := range spec.Queries {
		doc.Queries = append(doc.Queries, outCQRS{Name: q.Name, List: q.List, Get: q.Get, Args: outFields(q.Args), Returns: q.Returns})
	}
	for _, c := range spec.Commands {
		doc.Commands = append(doc.Commands, outCQRS{
			Name: c.Name, Create: c.Create, Update: c.Update, Delete: c.Delete, Args: outFields(c.Args), Returns: c.Returns,
		})
	}
	sec := spec.Security
	if len(sec.Roles)+len(sec.Disable)+len(sec.Entries) > 0 {
		sd := &outSecurity{Roles: sec.Roles, Disable: sec.Disable}
		for _, e := range sec.Entries {
			oe := outEntry{
				Name: e.Name, Role: e.Role, Effect: e.Effect, Kind: e.Kind, Resource: e.Resource,
				Actions: e.Actions, Domain: e.Domain, Priority: e.Priority,
			}
			switch {
			case e.Fields.All:
				oe.Fields = "*"
			case len(e.Fields.Names) > 0:
				oe.Fields = e.Fields.Names
			}
			sd.Entries = append(sd.Entries, oe)
		}
		doc.Security = sd
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type outSpec struct {
	Name       string       `yaml:"name"`
	Title      string       `yaml:"title,omitempty"`
	Summary    string       `yaml:"summary,omitempty"`
	Depends    []string     `yaml:"depends,omitempty"`
	Provides   []string     `yaml:"provides,omitempty"`
	Uses       []string     `yaml:"uses,omitempty"`
	Schema     string       `yaml:"schema,omitempty"`
	Resource   string       `yaml:"resource,omitempty"`
	Extensions *bool        `yaml:"extensions,omitempty"`
	Models     []outModel   `yaml:"models,omitempty"`
	Queries    []outCQRS    `yaml:"queries,omitempty"`
	Commands   []outCQRS    `yaml:"commands,omitempty"`
	Security   *outSecurity `yaml:"security,omitempty"`
}

type outModel struct {
	Name     string      `yaml:"name"`
	Internal bool        `yaml:"internal,omitempty"`
	Virtual  bool        `yaml:"virtual,omitempty"`
	Fields   []outField  `yaml:"fields,omitempty"`
	Search   *SearchSpec `yaml:"search,omitempty"`
}

type outField struct {
	Name     string       `yaml:"name"`
	Type     string       `yaml:"type,omitempty"`
	Required bool         `yaml:"required,omitempty"`
	Indexed  bool         `yaml:"indexed,omitempty"`
	Readonly bool         `yaml:"readonly,omitempty"`
	Default  any          `yaml:"default,omitempty"`
	Values   []string     `yaml:"values,omitempty,flow"`
	Relation string       `yaml:"relation,omitempty"`
	Inverse  string       `yaml:"inverse,omitempty"`
	Validate *outValidate `yaml:"validate,omitempty"`
}

type outValidate struct {
	MinLength int    `yaml:"minLength,omitempty"`
	MaxLength int    `yaml:"maxLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty"`
	Min       *int   `yaml:"min,omitempty"`
	Max       *int   `yaml:"max,omitempty"`
}

type outCQRS struct {
	Name    string     `yaml:"name"`
	List    string     `yaml:"list,omitempty"`
	Get     string     `yaml:"get,omitempty"`
	Create  string     `yaml:"create,omitempty"`
	Update  string     `yaml:"update,omitempty"`
	Delete  string     `yaml:"delete,omitempty"`
	Args    []outField `yaml:"args,omitempty"`
	Returns string     `yaml:"returns,omitempty"`
}

type outSecurity struct {
	Roles   []SecurityRoleSpec `yaml:"roles,omitempty"`
	Disable []string           `yaml:"disable,omitempty"`
	Entries []outEntry         `yaml:"entries,omitempty"`
}

type outEntry struct {
	Name     string     `yaml:"name"`
	Role     string     `yaml:"role"`
	Effect   string     `yaml:"effect,omitempty"`
	Kind     string     `yaml:"kind,omitempty"`
	Resource string     `yaml:"resource"`
	Actions  []string   `yaml:"actions,omitempty,flow"`
	Fields   any        `yaml:"fields,omitempty,flow"`
	Domain   [][]string `yaml:"domain,omitempty,flow"`
	Priority int        `yaml:"priority,omitempty"`
}

func outFields(in []FieldSpec) []outField {
	var out []outField
	for _, f := range in {
		out = append(out, outFieldOf(f))
	}
	return out
}

func outFieldOf(f FieldSpec) outField {
	o := outField{
		Name: f.Name, Type: f.Type, Required: f.Required, Indexed: f.Indexed, Readonly: f.Readonly,
		Default: f.Default, Values: f.Values, Relation: f.Relation, Inverse: f.Inverse,
	}
	v := f.Validate
	if v.MinLength != 0 || v.MaxLength != 0 || v.Pattern != "" || v.Min != nil || v.Max != nil {
		o.Validate = &outValidate{MinLength: v.MinLength, MaxLength: v.MaxLength, Pattern: v.Pattern, Min: v.Min, Max: v.Max}
	}
	return o
}
