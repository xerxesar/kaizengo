package appspec

import (
	"fmt"
	"strconv"
	"strings"
)

// IndexSpec is a real Postgres index on a model table.
// A field may be written `name:desc` (btree only). `include` is a covering
// column list. `where` is a conjunction of `column = value` predicates.
type IndexSpec struct {
	Name    string   `yaml:"name,omitempty"`
	Fields  []string `yaml:"fields"`
	Orders  []string `yaml:"orders,omitempty"` // parallel to Fields: asc | desc
	Include []string `yaml:"include,omitempty"`
	Unique  bool     `yaml:"unique,omitempty"`
	Method  string   `yaml:"method,omitempty"` // btree | hash | gin
	Where   string   `yaml:"where,omitempty"`
}

// IndexPred is one comparison in an index WHERE clause.
type IndexPred struct {
	Column string
	Op     string // = or <>
	Value  string // raw literal: null, true, false, number, or 'text'
}

func (m *ModelSpec) fillIndexes() {
	for i := range m.Indexes {
		ix := &m.Indexes[i]
		if strings.TrimSpace(ix.Method) == "" {
			ix.Method = "btree"
		} else {
			ix.Method = strings.ToLower(strings.TrimSpace(ix.Method))
		}
		ix.Where = strings.TrimSpace(ix.Where)
		if len(ix.Orders) == 0 {
			fields, orders, err := splitIndexFields(ix.Fields)
			if err == nil {
				ix.Fields = fields
				if anyDesc(orders) {
					ix.Orders = orders
				}
			}
		}
		if strings.TrimSpace(ix.Name) == "" {
			ix.Name = defaultIndexName(m.Name, ix.Fields, ix.Unique, ix.Method)
		}
	}
}

func splitIndexFields(raw []string) (fields, orders []string, err error) {
	fields = make([]string, len(raw))
	orders = make([]string, len(raw))
	for i, item := range raw {
		field, order, splitErr := splitIndexField(item)
		if splitErr != nil {
			return nil, nil, splitErr
		}
		fields[i] = field
		orders[i] = order
	}
	return fields, orders, nil
}

func splitIndexField(raw string) (field, order string, err error) {
	field, order, ok := strings.Cut(strings.TrimSpace(raw), ":")
	if !ok {
		return strings.TrimSpace(raw), "asc", nil
	}
	order = strings.ToLower(strings.TrimSpace(order))
	if order != "asc" && order != "desc" {
		return "", "", fmt.Errorf("index field %q order must be asc or desc", raw)
	}
	return strings.TrimSpace(field), order, nil
}

func anyDesc(orders []string) bool {
	for _, order := range orders {
		if order == "desc" {
			return true
		}
	}
	return false
}

// OrderAt is asc or desc for the key column at i.
func (ix IndexSpec) OrderAt(i int) string {
	if i >= 0 && i < len(ix.Orders) && strings.EqualFold(ix.Orders[i], "desc") {
		return "desc"
	}
	return "asc"
}

func defaultIndexName(model string, fields []string, unique bool, method string) string {
	prefix := "idx_"
	if unique {
		prefix = "uidx_"
	}
	name := prefix + model + "_" + strings.Join(fields, "_")
	if method != "" && method != "btree" {
		name += "_" + method
	}
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

func validateIndexes(model string, indexes []IndexSpec, fields map[string]FieldSpec) error {
	seen := map[string]struct{}{}
	for _, ix := range indexes {
		name := ix.Name
		if name == "" {
			name = defaultIndexName(model, ix.Fields, ix.Unique, ix.Method)
		}
		if !nameRe.MatchString(name) {
			return fmt.Errorf("model %q index %q must match %s", model, name, nameRe.String())
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("model %q has duplicate index %q", model, name)
		}
		seen[name] = struct{}{}
		if len(ix.Fields) == 0 {
			return fmt.Errorf("model %q index %q needs fields", model, name)
		}
		method := ix.Method
		if method == "" {
			method = "btree"
		}
		switch method {
		case "btree", "hash", "gin":
		default:
			return fmt.Errorf("model %q index %q has unsupported method %q", model, name, method)
		}
		if ix.Unique && method != "btree" {
			return fmt.Errorf("model %q index %q: unique indexes are btree", model, name)
		}
		if (method == "hash" || method == "gin") && len(ix.Fields) != 1 {
			return fmt.Errorf("model %q index %q: %s indexes take one field", model, name, method)
		}
		if len(ix.Orders) > 0 && len(ix.Orders) != len(ix.Fields) {
			return fmt.Errorf("model %q index %q: orders must match fields", model, name)
		}
		desc := false
		seenField := map[string]struct{}{}
		for i, field := range ix.Fields {
			if strings.Contains(field, ":") {
				return fmt.Errorf("model %q index %q field %q order must be asc or desc", model, name, field)
			}
			if _, ok := seenField[field]; ok {
				return fmt.Errorf("model %q index %q lists field %q twice", model, name, field)
			}
			seenField[field] = struct{}{}
			f, ok := fields[field]
			if !ok {
				return fmt.Errorf("model %q index %q references unknown field %q", model, name, field)
			}
			if !f.Column() {
				return fmt.Errorf("model %q index %q: field %q is not a column", model, name, field)
			}
			order := ix.OrderAt(i)
			if order == "desc" {
				desc = true
			}
			if method == "gin" {
				switch f.CanonicalType() {
				case TypeString, TypeText, TypeJSON, TypeEnum:
				default:
					return fmt.Errorf("model %q index %q: gin is for text or json fields", model, name)
				}
			}
		}
		if method != "btree" && (desc || len(ix.Include) > 0 || ix.Where != "") {
			return fmt.Errorf("model %q index %q: %s indexes are a single column with no include, where, or desc", model, name, method)
		}
		for _, field := range ix.Include {
			if _, ok := seenField[field]; ok {
				return fmt.Errorf("model %q index %q include %q is already a key", model, name, field)
			}
			f, ok := fields[field]
			if !ok || !f.Column() {
				return fmt.Errorf("model %q index %q include %q is not a column", model, name, field)
			}
			seenField[field] = struct{}{}
		}
		preds, err := ParseIndexWhere(ix.Where)
		if err != nil {
			return fmt.Errorf("model %q index %q: %w", model, name, err)
		}
		for _, pred := range preds {
			if pred.Column == "id" || pred.Column == "org_id" || pred.Column == "deleted" {
				continue
			}
			f, ok := fields[pred.Column]
			if !ok {
				// Column names in WHERE are SQL names (topic_id) or field names (topicId).
				f, ok = fields[fieldFromColumn(pred.Column, fields)]
			}
			if !ok || !f.Column() {
				return fmt.Errorf("model %q index %q where column %q is not a column", model, name, pred.Column)
			}
		}
	}
	return nil
}

func fieldFromColumn(col string, fields map[string]FieldSpec) string {
	if _, ok := fields[col]; ok {
		return col
	}
	for name := range fields {
		if snakeColumn(name) == col {
			return name
		}
	}
	return ""
}

func snakeColumn(field string) string {
	var b strings.Builder
	for i, r := range field {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ParseIndexWhere parses `col = value AND col <> value`.
// Values are null, true, false, a number, or a single-quoted string.
func ParseIndexWhere(s string) ([]IndexPred, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, " AND ")
	if len(parts) == 1 {
		parts = strings.Split(s, " and ")
	}
	out := make([]IndexPred, 0, len(parts))
	for _, part := range parts {
		pred, err := parseIndexPred(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		out = append(out, pred)
	}
	return out, nil
}

func parseIndexPred(s string) (IndexPred, error) {
	var col, op, value string
	switch {
	case strings.Contains(s, "<>"):
		col, value, _ = strings.Cut(s, "<>")
		op = "<>"
	case strings.Contains(s, "="):
		col, value, _ = strings.Cut(s, "=")
		op = "="
	default:
		return IndexPred{}, fmt.Errorf("where %q must be column = value", s)
	}
	col = strings.TrimSpace(col)
	value = strings.TrimSpace(value)
	if !nameRe.MatchString(col) && !fieldNameRe.MatchString(col) {
		return IndexPred{}, fmt.Errorf("where column %q is invalid", col)
	}
	switch {
	case value == "null" || value == "true" || value == "false":
	case len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'':
		if strings.Contains(value[1:len(value)-1], "'") {
			return IndexPred{}, fmt.Errorf("where string %q cannot contain quotes", value)
		}
	default:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return IndexPred{}, fmt.Errorf("where value %q is invalid", value)
		}
	}
	return IndexPred{Column: col, Op: op, Value: value}, nil
}

// WhereSQL returns a parenthesized predicate with quoted column names.
func (ix IndexSpec) WhereSQL(column func(string) string) (string, error) {
	preds, err := ParseIndexWhere(ix.Where)
	if err != nil || len(preds) == 0 {
		return "", err
	}
	parts := make([]string, len(preds))
	for i, pred := range preds {
		col := pred.Column
		if column != nil {
			if mapped := column(pred.Column); mapped != "" {
				col = mapped
			}
		}
		lit := pred.Value
		switch lit {
		case "null":
			lit = "NULL"
		case "true":
			lit = "TRUE"
		case "false":
			lit = "FALSE"
		}
		parts[i] = quoteSQLIdent(col) + " " + pred.Op + " " + lit
	}
	return strings.Join(parts, " AND "), nil
}

func quoteSQLIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func validateFieldStorage(model string, f FieldSpec) error {
	if f.Unique && !f.Column() {
		return fmt.Errorf("model %q field %q: unique applies to stored columns", model, f.Name)
	}
	if f.Indexed && !f.Column() {
		return fmt.Errorf("model %q field %q: indexed applies to stored columns", model, f.Name)
	}
	if strings.TrimSpace(f.OnDelete) == "" {
		return nil
	}
	if f.CanonicalType() != TypeMany2One {
		return fmt.Errorf("model %q field %q: onDelete is only for many2one", model, f.Name)
	}
	switch canonicalOnDelete(f.OnDelete) {
	case "restrict", "set null", "cascade":
	default:
		return fmt.Errorf("model %q field %q: onDelete must be restrict, set null, or cascade", model, f.Name)
	}
	if canonicalOnDelete(f.OnDelete) == "set null" && f.Required {
		return fmt.Errorf("model %q field %q: onDelete set null requires a nullable field", model, f.Name)
	}
	return nil
}

// canonicalOnDelete maps aliases to restrict | set null | cascade.
func canonicalOnDelete(v string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(v), " ", "")) {
	case "", "restrict":
		return "restrict"
	case "setnull":
		return "set null"
	case "cascade":
		return "cascade"
	default:
		return strings.TrimSpace(v)
	}
}

// OnDeleteAction is the SQL ON DELETE action for a many2one (default RESTRICT).
func (f FieldSpec) OnDeleteAction() string {
	switch canonicalOnDelete(f.OnDelete) {
	case "set null":
		return "SET NULL"
	case "cascade":
		return "CASCADE"
	default:
		return "RESTRICT"
	}
}
