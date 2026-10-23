package engine

import (
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func camel(s string) string {
	p := pascal(s)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func typeName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return pascal(spec.Name) + pascal(model.Name) + "Record"
}

func listName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return camel(spec.Name) + pascal(model.Name) + "s"
}

func getName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return camel(spec.Name) + pascal(model.Name)
}

func createName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return "create" + pascal(spec.Name) + pascal(model.Name)
}

func updateName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return "update" + pascal(spec.Name) + pascal(model.Name)
}

func deleteName(spec appspec.AppSpec, model appspec.ModelSpec) string {
	return "delete" + pascal(spec.Name) + pascal(model.Name)
}

func modelTable(model appspec.ModelSpec) string {
	return pluralize(model.Name)
}

// pluralize turns a model name into a SQL table name (greeting→greetings, acl_entry→acl_entries).
func pluralize(s string) string {
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		prev := s[len(s)-2]
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func colName(field string) string {
	switch field {
	case "orgId":
		return "org_id"
	case "authorId":
		return "author_id"
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	default:
		var b strings.Builder
		for i, r := range field {
			if r >= 'A' && r <= 'Z' {
				if i > 0 {
					b.WriteByte('_')
				}
				b.WriteRune(r - 'A' + 'a')
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}
