package acl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DomainExpr is an Odoo-style domain tree.
// Supports polish operators "&" (and), "|" (or), "!" (not) and leaf [field, op, value].
// A flat list of leaves is an implicit AND (compatible with Domain / ParseDomain).
type DomainExpr struct {
	Kind     string // "and" | "or" | "not" | "leaf"
	Field    string
	Op       string
	Value    any
	Children []*DomainExpr
}

// ParseDomainExpr parses a JSON domain supporting polish & | ! and flat AND lists.
func ParseDomainExpr(raw string) (*DomainExpr, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "*" {
		return nil, nil
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil, fmt.Errorf("domain expr: %w", err)
	}
	return parseDomainNodes(arr)
}

func parseDomainNodes(arr []any) (*DomainExpr, error) {
	if len(arr) == 0 {
		return nil, nil
	}
	var nodes []*DomainExpr
	i := 0
	for i < len(arr) {
		n, next, err := parseDomainOne(arr, i)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
		i = next
	}
	if len(nodes) == 1 {
		return nodes[0], nil
	}
	return &DomainExpr{Kind: "and", Children: nodes}, nil
}

func parseDomainOne(arr []any, i int) (*DomainExpr, int, error) {
	if i >= len(arr) {
		return nil, i, fmt.Errorf("domain expr: unexpected end of domain")
	}
	item := arr[i]
	if s, ok := item.(string); ok {
		switch strings.TrimSpace(s) {
		case "|":
			left, j, err := parseDomainOne(arr, i+1)
			if err != nil {
				return nil, i, err
			}
			right, k, err := parseDomainOne(arr, j)
			if err != nil {
				return nil, i, err
			}
			return &DomainExpr{Kind: "or", Children: []*DomainExpr{left, right}}, k, nil
		case "&":
			left, j, err := parseDomainOne(arr, i+1)
			if err != nil {
				return nil, i, err
			}
			right, k, err := parseDomainOne(arr, j)
			if err != nil {
				return nil, i, err
			}
			return &DomainExpr{Kind: "and", Children: []*DomainExpr{left, right}}, k, nil
		case "!":
			child, j, err := parseDomainOne(arr, i+1)
			if err != nil {
				return nil, i, err
			}
			return &DomainExpr{Kind: "not", Children: []*DomainExpr{child}}, j, nil
		default:
			return nil, i, fmt.Errorf("domain expr: unknown operator %q", s)
		}
	}
	tuple, ok := item.([]any)
	if !ok || len(tuple) < 2 {
		return nil, i, fmt.Errorf("domain expr: condition must be [field, op, value?] or |/&/!")
	}
	field := strings.TrimSpace(fmt.Sprint(tuple[0]))
	op := strings.TrimSpace(strings.ToLower(fmt.Sprint(tuple[1])))
	var val any
	if len(tuple) >= 3 {
		val = tuple[2]
	}
	if field == "" || op == "" {
		return nil, i, fmt.Errorf("domain expr: missing field or op")
	}
	return &DomainExpr{Kind: "leaf", Field: field, Op: op, Value: val}, i + 1, nil
}

// Match evaluates the expression against a record.
func (e *DomainExpr) Match(record map[string]any, p PrincipalContext) bool {
	if e == nil {
		return true
	}
	switch e.Kind {
	case "and":
		for _, c := range e.Children {
			if !c.Match(record, p) {
				return false
			}
		}
		return true
	case "or":
		if len(e.Children) == 0 {
			return true
		}
		for _, c := range e.Children {
			if c.Match(record, p) {
				return true
			}
		}
		return false
	case "not":
		if len(e.Children) == 0 {
			return true
		}
		return !e.Children[0].Match(record, p)
	case "leaf":
		return Condition{Field: e.Field, Op: e.Op, Value: e.Value}.match(record, p)
	default:
		return false
	}
}

// FlattenAND returns leaf conditions if the expr is a pure AND of leaves (for ACL Domain).
func (e *DomainExpr) FlattenAND() (Domain, bool) {
	if e == nil {
		return nil, true
	}
	switch e.Kind {
	case "leaf":
		return Domain{{Field: e.Field, Op: e.Op, Value: e.Value}}, true
	case "and":
		var out Domain
		for _, c := range e.Children {
			part, ok := c.FlattenAND()
			if !ok {
				return nil, false
			}
			out = append(out, part...)
		}
		return out, true
	default:
		return nil, false
	}
}

// CompileDomainExpr builds a SQL fragment for the expression tree.
func CompileDomainExpr(e *DomainExpr, col ColumnFunc, startArg int, p PrincipalContext) (SQLFragment, int, error) {
	if e == nil {
		return SQLFragment{}, startArg, nil
	}
	switch e.Kind {
	case "and":
		var parts []string
		var args []any
		arg := startArg
		for _, c := range e.Children {
			frag, next, err := CompileDomainExpr(c, col, arg, p)
			if err != nil {
				return SQLFragment{}, startArg, err
			}
			arg = next
			if frag.Clause != "" {
				parts = append(parts, "("+frag.Clause+")")
				args = append(args, frag.Args...)
			}
		}
		if len(parts) == 0 {
			return SQLFragment{}, arg, nil
		}
		return SQLFragment{Clause: strings.Join(parts, " AND "), Args: args}, arg, nil
	case "or":
		var parts []string
		var args []any
		arg := startArg
		for _, c := range e.Children {
			frag, next, err := CompileDomainExpr(c, col, arg, p)
			if err != nil {
				return SQLFragment{}, startArg, err
			}
			arg = next
			if frag.Clause != "" {
				parts = append(parts, "("+frag.Clause+")")
				args = append(args, frag.Args...)
			}
		}
		if len(parts) == 0 {
			return SQLFragment{}, arg, nil
		}
		return SQLFragment{Clause: strings.Join(parts, " OR "), Args: args}, arg, nil
	case "not":
		if len(e.Children) == 0 {
			return SQLFragment{}, startArg, nil
		}
		frag, next, err := CompileDomainExpr(e.Children[0], col, startArg, p)
		if err != nil {
			return SQLFragment{}, startArg, err
		}
		if frag.Clause == "" {
			return SQLFragment{}, next, nil
		}
		return SQLFragment{Clause: "NOT (" + frag.Clause + ")", Args: frag.Args}, next, nil
	case "leaf":
		return compileDomain(Domain{{Field: e.Field, Op: e.Op, Value: e.Value}}, col, startArg, p)
	default:
		return SQLFragment{}, startArg, fmt.Errorf("domain expr: unknown kind %q", e.Kind)
	}
}
