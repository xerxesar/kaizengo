package acl

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Domain is a flat AND list of conditions: [["field","=","value"], ...].
type Domain []Condition

// Condition is [field, op, value].
type Condition struct {
	Field string
	Op    string
	Value any
}

// ParseDomain parses a JSON domain string. Empty / "[]" / "*" → empty domain (all records).
func ParseDomain(raw string) (Domain, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "*" {
		return nil, nil
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil, fmt.Errorf("acl domain: %w", err)
	}
	out := make(Domain, 0, len(arr))
	for i, item := range arr {
		tuple, ok := item.([]any)
		if !ok || len(tuple) < 2 {
			return nil, fmt.Errorf("acl domain: condition %d must be [field, op, value?]", i)
		}
		field := strings.TrimSpace(fmt.Sprint(tuple[0]))
		op := strings.TrimSpace(strings.ToLower(fmt.Sprint(tuple[1])))
		var val any
		if len(tuple) >= 3 {
			val = tuple[2]
		}
		if field == "" || op == "" {
			return nil, fmt.Errorf("acl domain: condition %d missing field or op", i)
		}
		out = append(out, Condition{Field: field, Op: op, Value: val})
	}
	return out, nil
}

// ParseStringList parses a JSON string array or "*" / empty meaning all.
// Returns (nil, true) for all-fields/all-actions sentinel.
func ParseStringList(raw string) (items []string, all bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" || raw == `["*"]` {
		return nil, true, nil
	}
	var single string
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		if single == "*" || single == "" {
			return nil, true, nil
		}
		return []string{single}, false, nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil, false, fmt.Errorf("acl list: %w", err)
	}
	for _, s := range arr {
		s = strings.TrimSpace(s)
		if s == "*" {
			return nil, true, nil
		}
		if s != "" {
			items = append(items, s)
		}
	}
	return items, false, nil
}

// Match evaluates the domain against a record with principal substitutions.
// Empty domain always matches.
func (d Domain) Match(record map[string]any, p PrincipalContext) bool {
	if len(d) == 0 {
		return true
	}
	if record == nil {
		return false
	}
	for _, c := range d {
		if !c.match(record, p) {
			return false
		}
	}
	return true
}

func (c Condition) match(record map[string]any, p PrincipalContext) bool {
	var left any
	if strings.HasPrefix(c.Field, "$") {
		left = resolveLiteral(c.Field, p)
	} else {
		left = record[c.Field]
	}
	right := resolveLiteral(c.Value, p)

	switch c.Op {
	case "=", "==", "eq":
		return cmpEqual(left, right)
	case "!=", "<>", "ne":
		return !cmpEqual(left, right)
	case "in":
		return inList(left, right)
	case "not in", "notin":
		return !inList(left, right)
	case ">", "gt":
		return cmpNum(left, right) > 0
	case "<", "lt":
		return cmpNum(left, right) < 0
	case ">=", "gte":
		return cmpNum(left, right) >= 0
	case "<=", "lte":
		return cmpNum(left, right) <= 0
	case "like", "ilike":
		return strings.Contains(
			strings.ToLower(fmt.Sprint(left)),
			strings.ToLower(strings.Trim(fmt.Sprint(right), "%")),
		)
	case "is set", "isset", "set":
		return left != nil && strings.TrimSpace(fmt.Sprint(left)) != ""
	case "is not set", "isnotset", "not set", "unset":
		return left == nil || strings.TrimSpace(fmt.Sprint(left)) == ""
	default:
		return false
	}
}

func resolveLiteral(v any, p PrincipalContext) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	switch s {
	case "$user.id":
		return p.UserID
	case "$user.orgId", "$user.org_id":
		return p.OrgID
	case "$user.roles":
		return p.Roles
	default:
		return v
	}
}

func cmpEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// numeric equality
	if af, aok := asFloat(a); aok {
		if bf, bok := asFloat(b); bok {
			return af == bf
		}
	}
	return strings.EqualFold(fmt.Sprint(a), fmt.Sprint(b))
}

func inList(needle, haystack any) bool {
	switch t := haystack.(type) {
	case []any:
		for _, item := range t {
			if cmpEqual(needle, item) {
				return true
			}
		}
	case []string:
		for _, item := range t {
			if cmpEqual(needle, item) {
				return true
			}
		}
	default:
		return cmpEqual(needle, haystack)
	}
	return false
}

func cmpNum(a, b any) int {
	af, aok := asFloat(a)
	bf, bok := asFloat(b)
	if !aok || !bok {
		return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	}
	switch {
	case af < bf:
		return -1
	case af > bf:
		return 1
	default:
		return 0
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
