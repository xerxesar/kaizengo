package engine

import "testing"

func TestPluralize(t *testing.T) {
	cases := map[string]string{
		"greeting":  "greetings",
		"user":      "users",
		"role":      "roles",
		"acl_entry": "acl_entries",
		"org_unit":  "org_units",
		"user_role": "user_roles",
		"status":    "statuses",
		"box":       "boxes",
	}
	for in, want := range cases {
		if got := pluralize(in); got != want {
			t.Errorf("pluralize(%q) = %q, want %q", in, got, want)
		}
	}
}
