package acl

import "strings"

// MatchResource reports whether a policy resource pattern covers want.
// Patterns: "*", "inventory.*", exact "inventory.product".
func MatchResource(pattern, want string) bool {
	pattern = strings.TrimSpace(pattern)
	want = strings.TrimSpace(want)
	if pattern == "" || want == "" {
		return false
	}
	if pattern == ResAll {
		return true
	}
	if strings.EqualFold(pattern, want) {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-1] // keep trailing "."
		return strings.HasPrefix(strings.ToLower(want), strings.ToLower(prefix))
	}
	return false
}

// MatchAction reports whether a policy action list covers want.
func MatchAction(actions []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	if len(actions) == 0 {
		return true // empty = all
	}
	for _, a := range actions {
		a = strings.TrimSpace(a)
		if a == ActAll || strings.EqualFold(a, want) {
			return true
		}
	}
	return false
}

// CoversField reports whether an entry's fields list covers field.
func CoversField(fields []string, field string) bool {
	if len(fields) == 0 {
		return true
	}
	field = strings.TrimSpace(field)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == FieldsAll || strings.EqualFold(f, field) {
			return true
		}
	}
	return false
}
