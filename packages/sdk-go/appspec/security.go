package appspec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecuritySpec is the merged declarative ACL / demo-user seed for an app.
type SecuritySpec struct {
	Roles   []SecurityRoleSpec
	Disable []string
	Entries []SecurityEntrySpec
	Users   []SecurityUserSpec
}

// Empty reports whether the spec has nothing to apply.
func (s SecuritySpec) Empty() bool {
	return len(s.Roles) == 0 && len(s.Disable) == 0 && len(s.Entries) == 0 && len(s.Users) == 0
}

// SecurityRoleSpec ensures a role exists (by name).
type SecurityRoleSpec struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label"`
}

// SecurityEntrySpec is one acl_entry (native YAML; engine marshals to store JSON).
type SecurityEntrySpec struct {
	Name     string       `yaml:"name"`
	Role     string       `yaml:"role"`
	Effect   string       `yaml:"effect"` // allow|deny
	Resource string       `yaml:"resource"`
	Actions  []string     `yaml:"actions"`
	Fields   FieldsSpec   `yaml:"fields"`
	Domain   [][]string   `yaml:"domain"`
	Priority int          `yaml:"priority"`
}

// SecurityUserSpec seeds a demo identity user, password, and role assignments.
type SecurityUserSpec struct {
	Email    string   `yaml:"email"`
	Name     string   `yaml:"name"`
	Password string   `yaml:"password"`
	Roles    []string `yaml:"roles"`
}

// FieldsSpec is either "*" (all fields) or a list of field names.
type FieldsSpec struct {
	All   bool
	Names []string
}

// UnmarshalYAML accepts fields: "*" or fields: [a, b].
func (f *FieldsSpec) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*f = FieldsSpec{}
		return nil
	}
	if value.Kind == yaml.ScalarNode {
		s := strings.TrimSpace(value.Value)
		if s == "" || s == "null" {
			*f = FieldsSpec{}
			return nil
		}
		if s == "*" {
			*f = FieldsSpec{All: true}
			return nil
		}
		return fmt.Errorf("fields: expected \"*\" or a list, got %q", s)
	}
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("fields: expected \"*\" or a list")
	}
	var names []string
	if err := value.Decode(&names); err != nil {
		return fmt.Errorf("fields: %w", err)
	}
	*f = FieldsSpec{Names: names}
	return nil
}

type fileSecurity struct {
	Roles   []SecurityRoleSpec  `yaml:"roles"`
	Disable []string            `yaml:"disable"`
	Entries []SecurityEntrySpec `yaml:"entries"`
	Users   []SecurityUserSpec  `yaml:"users"`
}

func (s SecuritySpec) validate() error {
	seenRoles := map[string]struct{}{}
	for i, r := range s.Roles {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			return fmt.Errorf("security.roles[%d]: name is required", i)
		}
		if !nameRe.MatchString(name) {
			return fmt.Errorf("security.roles[%d]: invalid name %q", i, name)
		}
		if _, ok := seenRoles[name]; ok {
			return fmt.Errorf("security.roles: duplicate role %q", name)
		}
		seenRoles[name] = struct{}{}
	}

	seenDisable := map[string]struct{}{}
	for i, name := range s.Disable {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("security.disable[%d]: empty name", i)
		}
		if _, ok := seenDisable[name]; ok {
			return fmt.Errorf("security.disable: duplicate %q", name)
		}
		seenDisable[name] = struct{}{}
	}

	seenEntries := map[string]struct{}{}
	for i, e := range s.Entries {
		loc := fmt.Sprintf("security.entries[%d]", i)
		if strings.TrimSpace(e.Name) == "" {
			return fmt.Errorf("%s: name is required", loc)
		}
		if strings.TrimSpace(e.Role) == "" {
			return fmt.Errorf("%s: role is required", loc)
		}
		if strings.TrimSpace(e.Resource) == "" {
			return fmt.Errorf("%s: resource is required", loc)
		}
		effect := strings.ToLower(strings.TrimSpace(e.Effect))
		if effect == "" {
			effect = "allow"
		}
		if effect != "allow" && effect != "deny" {
			return fmt.Errorf("%s: effect must be allow or deny", loc)
		}
		key := e.Name + "\x00" + e.Role
		if _, ok := seenEntries[key]; ok {
			return fmt.Errorf("security.entries: duplicate name %q for role %q", e.Name, e.Role)
		}
		seenEntries[key] = struct{}{}
		for j, triple := range e.Domain {
			if len(triple) != 3 {
				return fmt.Errorf("%s.domain[%d]: expected [field, op, value]", loc, j)
			}
		}
	}

	seenUsers := map[string]struct{}{}
	for i, u := range s.Users {
		loc := fmt.Sprintf("security.users[%d]", i)
		email := strings.TrimSpace(u.Email)
		if email == "" {
			return fmt.Errorf("%s: email is required", loc)
		}
		if _, ok := seenUsers[email]; ok {
			return fmt.Errorf("security.users: duplicate email %q", email)
		}
		seenUsers[email] = struct{}{}
	}
	return nil
}

func mergeSecurity(dst *SecuritySpec, src fileSecurity) {
	dst.Roles = append(dst.Roles, src.Roles...)
	dst.Disable = append(dst.Disable, src.Disable...)
	dst.Entries = append(dst.Entries, src.Entries...)
	dst.Users = append(dst.Users, src.Users...)
}
