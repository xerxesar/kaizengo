package acl

import (
	"fmt"
	"strings"
)

// SQLFragment is a parameterized WHERE clause piece (without leading WHERE).
type SQLFragment struct {
	Clause string
	Args   []any
}

// ColumnFunc maps a record field name (camelCase) to a SQL column identifier (already quoted or raw).
type ColumnFunc func(field string) (column string, ok bool)

// CompileListFilter builds a SQL AND-able fragment for ListFilter.
// startArg is the next $N index (1-based). Returns empty fragment if Unrestricted.
func CompileListFilter(f ListFilter, col ColumnFunc, startArg int, p PrincipalContext) (SQLFragment, error) {
	if f.DenyAll {
		return SQLFragment{Clause: "false"}, nil
	}
	if f.Unrestricted {
		return SQLFragment{}, nil
	}

	var parts []string
	var args []any
	arg := startArg

	if len(f.Allow) > 0 {
		var allowParts []string
		for _, d := range f.Allow {
			frag, next, err := compileDomain(d, col, arg, p)
			if err != nil {
				return SQLFragment{}, err
			}
			arg = next
			if frag.Clause != "" {
				allowParts = append(allowParts, "("+frag.Clause+")")
				args = append(args, frag.Args...)
			}
		}
		if len(allowParts) > 0 {
			parts = append(parts, "("+strings.Join(allowParts, " OR ")+")")
		} else if !f.Unrestricted {
			// allows were non-empty domains but none compiled — deny
			return SQLFragment{Clause: "false"}, nil
		}
	} else if !f.Unrestricted {
		// only denies, no unrestricted allow
		// still need an allow — BuildListFilter should have set DenyAll
	}

	for _, d := range f.Deny {
		frag, next, err := compileDomain(d, col, arg, p)
		if err != nil {
			return SQLFragment{}, err
		}
		arg = next
		if frag.Clause != "" {
			parts = append(parts, "NOT ("+frag.Clause+")")
			args = append(args, frag.Args...)
		}
	}

	if len(parts) == 0 {
		if f.Unrestricted {
			return SQLFragment{}, nil
		}
		// allow domains empty but unrestrictedAllow was false with only denies handled
		return SQLFragment{}, nil
	}
	return SQLFragment{Clause: strings.Join(parts, " AND "), Args: args}, nil
}

func compileDomain(d Domain, col ColumnFunc, startArg int, p PrincipalContext) (SQLFragment, int, error) {
	if len(d) == 0 {
		return SQLFragment{}, startArg, nil
	}
	var parts []string
	var args []any
	arg := startArg
	for _, c := range d {
		field := c.Field
		if strings.HasPrefix(field, "$") {
			return SQLFragment{}, startArg, fmt.Errorf("acl sql: domain field cannot be %q", field)
		}
		column, ok := col(field)
		if !ok {
			return SQLFragment{}, startArg, fmt.Errorf("acl sql: unknown field %q", field)
		}
		right := resolveLiteral(c.Value, p)
		op := strings.ToLower(c.Op)
		switch op {
		case "=", "==", "eq":
			parts = append(parts, fmt.Sprintf("%s = $%d", column, arg))
			args = append(args, right)
			arg++
		case "!=", "<>", "ne":
			parts = append(parts, fmt.Sprintf("%s IS DISTINCT FROM $%d", column, arg))
			args = append(args, right)
			arg++
		case "in":
			list, err := toAnySlice(right)
			if err != nil {
				return SQLFragment{}, startArg, err
			}
			if len(list) == 0 {
				parts = append(parts, "false")
				continue
			}
			ph := make([]string, len(list))
			for i, item := range list {
				ph[i] = fmt.Sprintf("$%d", arg)
				args = append(args, item)
				arg++
			}
			parts = append(parts, fmt.Sprintf("%s IN (%s)", column, strings.Join(ph, ",")))
		case "not in", "notin":
			list, err := toAnySlice(right)
			if err != nil {
				return SQLFragment{}, startArg, err
			}
			if len(list) == 0 {
				parts = append(parts, "true")
				continue
			}
			ph := make([]string, len(list))
			for i, item := range list {
				ph[i] = fmt.Sprintf("$%d", arg)
				args = append(args, item)
				arg++
			}
			parts = append(parts, fmt.Sprintf("%s NOT IN (%s)", column, strings.Join(ph, ",")))
		case ">", "gt":
			parts = append(parts, fmt.Sprintf("%s > $%d", column, arg))
			args = append(args, right)
			arg++
		case "<", "lt":
			parts = append(parts, fmt.Sprintf("%s < $%d", column, arg))
			args = append(args, right)
			arg++
		case ">=", "gte":
			parts = append(parts, fmt.Sprintf("%s >= $%d", column, arg))
			args = append(args, right)
			arg++
		case "<=", "lte":
			parts = append(parts, fmt.Sprintf("%s <= $%d", column, arg))
			args = append(args, right)
			arg++
		case "like", "ilike":
			parts = append(parts, fmt.Sprintf("%s ILIKE $%d", column, arg))
			args = append(args, fmt.Sprint(right))
			arg++
		case "is set", "isset", "set":
			parts = append(parts, fmt.Sprintf("(%s IS NOT NULL AND %s::text <> '')", column, column))
		case "is not set", "isnotset", "not set", "unset":
			parts = append(parts, fmt.Sprintf("(%s IS NULL OR %s::text = '')", column, column))
		default:
			return SQLFragment{}, startArg, fmt.Errorf("acl sql: unsupported op %q", c.Op)
		}
	}
	return SQLFragment{Clause: strings.Join(parts, " AND "), Args: args}, arg, nil
}

func toAnySlice(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, nil
	default:
		return []any{v}, nil
	}
}
