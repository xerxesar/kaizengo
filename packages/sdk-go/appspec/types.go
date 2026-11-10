package appspec

import (
	"strings"
)

// Canonical scalar and relation field types.
const (
	TypeString    = "string"
	TypeText      = "text"
	TypeInt       = "int"
	TypeNumber    = "number"
	TypeBool      = "bool"
	TypeEnum      = "enum"
	TypeDate      = "date"
	TypeDatetime  = "datetime"
	TypeJSON      = "json"
	TypeMany2One  = "many2one"
	TypeOne2Many  = "one2many"
	TypeMany2Many = "many2many"
)

var fieldTypeAliases = map[string]string{
	"string":    TypeString,
	"char":      TypeString,
	"varchar":   TypeString,
	"text":      TypeText,
	"html":      TypeText,
	"int":       TypeInt,
	"integer":   TypeInt,
	"number":    TypeNumber,
	"float":     TypeNumber,
	"float64":   TypeNumber,
	"double":    TypeNumber,
	"decimal":   TypeNumber,
	"bool":      TypeBool,
	"boolean":   TypeBool,
	"enum":      TypeEnum,
	"date":      TypeDate,
	"datetime":  TypeDatetime,
	"timestamp": TypeDatetime,
	"json":      TypeJSON,
	"jsonb":     TypeJSON,
	"many2one":  TypeMany2One,
	"fk":        TypeMany2One,
	"relation":  TypeMany2One,
	"one2many":  TypeOne2Many,
	"many2many": TypeMany2Many,
}

// CanonicalFieldType maps aliases (float, integer, fk, …) to a canonical type.
// Unknown names are returned trimmed and lowercased so Validate can reject them.
func CanonicalFieldType(t string) string {
	key := strings.ToLower(strings.TrimSpace(t))
	if key == "" {
		return TypeString
	}
	if canon, ok := fieldTypeAliases[key]; ok {
		return canon
	}
	return key
}

// CanonicalType returns the field's canonical type (aliases resolved).
func (f FieldSpec) CanonicalType() string {
	if f.IsEnum() && CanonicalFieldType(f.Type) != TypeMany2One {
		t := CanonicalFieldType(f.Type)
		if t == TypeEnum || t == TypeString || t == "" {
			return TypeEnum
		}
	}
	return CanonicalFieldType(f.Type)
}

// Computed reports a formula field (`compute:`) or a related path (`related:`).
func (f FieldSpec) Computed() bool {
	return strings.TrimSpace(f.Compute) != "" || strings.TrimSpace(f.Related) != ""
}

// HookCompute reports compute: hook — the value comes from Hooks.Compute[name].
func (f FieldSpec) HookCompute() bool {
	return strings.EqualFold(strings.TrimSpace(f.Compute), "hook")
}

// StoreValue reports whether a computed field is written to a column.
// Non-computed fields are stored according to their type. Omitted `store` on a
// computed field means stored.
func (f FieldSpec) StoreValue() bool {
	if !f.Computed() {
		return true
	}
	if f.Store == nil {
		return true
	}
	return *f.Store
}

// Stored reports whether the field is persisted (a column, a stored compute, or a many2many junction).
// one2many is a virtual inverse. store: false computes are filled on read.
func (f FieldSpec) Stored() bool {
	if f.CanonicalType() == TypeOne2Many {
		return false
	}
	if f.Computed() && !f.StoreValue() {
		return false
	}
	return true
}

// Column reports whether the field has a column on the model table.
// many2many lives in a junction table; one2many and non-stored computes have no column.
func (f FieldSpec) Column() bool {
	if f.Computed() && !f.StoreValue() {
		return false
	}
	switch f.CanonicalType() {
	case TypeOne2Many, TypeMany2Many:
		return false
	default:
		return true
	}
}

// IsRelation reports many2one / one2many / many2many.
func (f FieldSpec) IsRelation() bool {
	switch f.CanonicalType() {
	case TypeMany2One, TypeOne2Many, TypeMany2Many:
		return true
	default:
		return false
	}
}

// RelationRef is a parsed `relation:` value (`model` or `app.model`).
type RelationRef struct {
	App   string
	Model string
}

// ParseRelation splits `app.model` or `model`. Empty app means the declaring app.
func ParseRelation(rel string) (RelationRef, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return RelationRef{}, false
	}
	app, model, ok := strings.Cut(rel, ".")
	if !ok {
		if !nameRe.MatchString(rel) {
			return RelationRef{}, false
		}
		return RelationRef{Model: rel}, true
	}
	if !nameRe.MatchString(app) || !nameRe.MatchString(model) {
		return RelationRef{}, false
	}
	return RelationRef{App: app, Model: model}, true
}

// ResolveRelation returns the target app (defaulting to fromApp) and model.
func (f FieldSpec) ResolveRelation(fromApp string) (RelationRef, bool) {
	ref, ok := ParseRelation(f.Relation)
	if !ok {
		return RelationRef{}, false
	}
	if ref.App == "" {
		ref.App = fromApp
	}
	return ref, true
}
