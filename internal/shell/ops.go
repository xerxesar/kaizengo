package shell

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type operation struct {
	App          string   `json:"app"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Field        string   `json:"field"`
	Operation    string   `json:"operation"`
	Args         []opArg  `json:"args"`
	Returns      string   `json:"returns"`
	ReturnFields []string `json:"returnFields"`
	Dynamic      bool     `json:"dynamic"`
}

type opArg struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

const catalogQuery = `query($app:String){ platformCatalog(app:$app) {
	app kind name field operation returns returnFields dynamic args { name type required }
} }`

func (s *session) catalog(app string) ([]operation, error) {
	var vars map[string]any
	if app != "" {
		vars = map[string]any{"app": app}
	}
	data, err := s.ex.Exec(s.ctx, catalogQuery, vars)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(data["platformCatalog"])
	if err != nil {
		return nil, err
	}
	var ops []operation
	if err := json.Unmarshal(b, &ops); err != nil {
		return nil, err
	}
	return ops, nil
}

// resolve finds an operation by app.name, app.field, or bare GraphQL field.
func (s *session) resolve(target string) (operation, error) {
	app, name, dotted := strings.Cut(target, ".")
	if !dotted {
		app, name = "", target
	}
	ops, err := s.catalog(app)
	if err != nil {
		return operation{}, err
	}
	for _, op := range ops {
		if op.Name == name || op.Field == name {
			return op, nil
		}
	}
	return operation{}, fmt.Errorf("unknown operation %q (try ls %s)", target, app)
}

func argSummary(op operation) string {
	parts := make([]string, 0, len(op.Args))
	for _, a := range op.Args {
		if a.Required {
			parts = append(parts, a.Name+"*")
		} else {
			parts = append(parts, a.Name)
		}
	}
	return strings.Join(parts, " ")
}

type invocation struct {
	values map[string]any
	fields []string
	all    bool
	json   bool
}

func parseInvocation(op operation, tokens []string) (*invocation, error) {
	inv := &invocation{values: map[string]any{}}
	types := map[string]string{}
	for _, a := range op.Args {
		types[a.Name] = a.Type
	}
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "--all":
			inv.all = true
		case tok == "--json":
			inv.json = true
		case tok == "--fields" || strings.HasPrefix(tok, "--fields="):
			val := strings.TrimPrefix(tok, "--fields=")
			if tok == "--fields" {
				if i+1 >= len(tokens) {
					return nil, fmt.Errorf("--fields needs a value")
				}
				i++
				val = tokens[i]
			}
			inv.fields = splitList(val)
		case strings.HasPrefix(tok, "--"):
			return nil, fmt.Errorf("unknown flag %s", tok)
		default:
			key, raw, ok := strings.Cut(tok, "=")
			if !ok {
				return nil, fmt.Errorf("expected key=value, got %q", tok)
			}
			typ, known := types[key]
			if !known {
				return nil, fmt.Errorf("unknown arg %q for %s.%s (args: %s)", key, op.App, op.Name, argSummary(op))
			}
			v, err := coerce(typ, raw)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			inv.values[key] = v
		}
	}
	for _, a := range op.Args {
		if _, ok := inv.values[a.Name]; a.Required && !ok {
			return nil, fmt.Errorf("missing required arg %s (%s)", a.Name, a.Type)
		}
	}
	return inv, nil
}

func (inv *invocation) document(op operation) (string, map[string]any) {
	var decls, call []string
	for _, a := range op.Args {
		if _, ok := inv.values[a.Name]; !ok {
			continue
		}
		decls = append(decls, fmt.Sprintf("$%s: %s", a.Name, a.Type))
		call = append(call, fmt.Sprintf("%s: $%s", a.Name, a.Name))
	}
	head := op.Operation
	if len(decls) > 0 {
		head += "(" + strings.Join(decls, ", ") + ")"
	}
	field := op.Field
	if len(call) > 0 {
		field += "(" + strings.Join(call, ", ") + ")"
	}
	sel := ""
	if fields := inv.selection(op); len(fields) > 0 {
		sel = " { " + strings.Join(fields, " ") + " }"
	}
	return fmt.Sprintf("%s { %s%s }", head, field, sel), inv.values
}

func (inv *invocation) selection(op operation) []string {
	if len(op.ReturnFields) == 0 {
		return nil
	}
	if len(inv.fields) > 0 {
		return inv.fields
	}
	return op.ReturnFields
}

var auditFields = map[string]struct{}{
	"orgId": {}, "authorId": {}, "deleted": {}, "createdAt": {}, "updatedAt": {},
}

func (inv *invocation) columns(op operation) []string {
	sel := inv.selection(op)
	if inv.all || len(inv.fields) > 0 {
		return sel
	}
	out := make([]string, 0, len(sel))
	for _, f := range sel {
		if _, audit := auditFields[f]; !audit {
			out = append(out, f)
		}
	}
	return out
}

// coerce converts a raw CLI value to the GraphQL argument type (e.g. "Int!", "[String!]").
func coerce(gqlType, raw string) (any, error) {
	t := strings.TrimSuffix(gqlType, "!")
	if strings.HasPrefix(t, "[") {
		inner := strings.TrimSuffix(strings.TrimPrefix(t, "["), "]")
		var items []string
		if strings.HasPrefix(strings.TrimSpace(raw), "[") {
			var decoded []any
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				return nil, fmt.Errorf("invalid JSON list: %w", err)
			}
			for _, d := range decoded {
				items = append(items, fmt.Sprint(d))
			}
		} else {
			items = splitList(raw)
		}
		out := make([]any, 0, len(items))
		for _, item := range items {
			v, err := coerce(inner, item)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	switch t {
	case "Int":
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("expected Int, got %q", raw)
		}
		return n, nil
	case "Float":
		f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return nil, fmt.Errorf("expected Float, got %q", raw)
		}
		return f, nil
	case "Boolean":
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("expected Boolean, got %q", raw)
		}
		return b, nil
	case "String", "ID":
		return raw, nil
	default:
		// Input objects (e.g. saveListViewPreset input=…) are passed as JSON.
		if strings.HasPrefix(strings.TrimSpace(raw), "{") {
			var obj map[string]any
			if err := json.Unmarshal([]byte(raw), &obj); err != nil {
				return nil, fmt.Errorf("%s: invalid JSON object: %w", t, err)
			}
			return obj, nil
		}
		return raw, nil
	}
}

func splitList(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
