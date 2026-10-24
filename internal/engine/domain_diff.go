package engine

import (
	"fmt"
	"sort"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

// diffSpecs lists human-readable changes from old to next. Field type changes
// are rejected: DDL is additive, so a retyped column would no longer scan.
func diffSpecs(old *appspec.AppSpec, next appspec.AppSpec) ([]string, error) {
	if old == nil {
		changes := []string{"create app " + next.Name}
		for _, m := range next.Models {
			changes = append(changes, fmt.Sprintf("create model %s (%d fields)", m.Name, len(m.Fields)))
		}
		for _, q := range next.Queries {
			changes = append(changes, "add query "+q.Name)
		}
		for _, c := range next.Commands {
			changes = append(changes, "add command "+c.Name)
		}
		if n := len(next.Security.Entries); n > 0 {
			changes = append(changes, fmt.Sprintf("apply %d policy entries", n))
		}
		return changes, nil
	}
	var changes []string
	if old.Title != next.Title || old.Summary != next.Summary {
		changes = append(changes, "update title/summary")
	}
	oldModels := map[string]appspec.ModelSpec{}
	for _, m := range old.Models {
		oldModels[m.Name] = m
	}
	nextModels := map[string]struct{}{}
	for _, m := range next.Models {
		nextModels[m.Name] = struct{}{}
		prev, ok := oldModels[m.Name]
		if !ok {
			changes = append(changes, fmt.Sprintf("create model %s (%d fields)", m.Name, len(m.Fields)))
			continue
		}
		prevFields := map[string]appspec.FieldSpec{}
		for _, f := range prev.Fields {
			prevFields[f.Name] = f
		}
		nextFields := map[string]struct{}{}
		for _, f := range m.Fields {
			nextFields[f.Name] = struct{}{}
			pf, ok := prevFields[f.Name]
			if !ok {
				changes = append(changes, fmt.Sprintf("add field %s.%s (%s)", m.Name, f.Name, f.CanonicalType()))
				continue
			}
			if pf.CanonicalType() != f.CanonicalType() && pf.Stored() && f.Stored() {
				prevSQL, _ := columnType(pf)
				nextSQL, _ := columnType(f)
				if prevSQL != nextSQL {
					return nil, fmt.Errorf("field %s.%s: type change %s → %s is not supported (add a new field instead)",
						m.Name, f.Name, pf.CanonicalType(), f.CanonicalType())
				}
				changes = append(changes, fmt.Sprintf("change field %s.%s type %s → %s", m.Name, f.Name, pf.CanonicalType(), f.CanonicalType()))
			} else if fieldChanged(pf, f) {
				changes = append(changes, fmt.Sprintf("update field %s.%s", m.Name, f.Name))
			}
		}
		for _, f := range prev.Fields {
			if _, ok := nextFields[f.Name]; !ok {
				changes = append(changes, fmt.Sprintf("remove field %s.%s (column kept)", m.Name, f.Name))
			}
		}
		if prev.Internal != m.Internal || prev.Virtual != m.Virtual {
			changes = append(changes, fmt.Sprintf("update model %s flags", m.Name))
		}
	}
	for _, m := range old.Models {
		if _, ok := nextModels[m.Name]; !ok {
			changes = append(changes, fmt.Sprintf("remove model %s (table kept)", m.Name))
		}
	}
	changes = append(changes, diffNames("query", queryNames(old.Queries), queryNames(next.Queries))...)
	changes = append(changes, diffNames("command", commandNames(old.Commands), commandNames(next.Commands))...)
	if len(old.Security.Entries) != len(next.Security.Entries) || len(old.Security.Roles) != len(next.Security.Roles) {
		changes = append(changes, fmt.Sprintf("policies: %d → %d entries", len(old.Security.Entries), len(next.Security.Entries)))
	}
	if len(changes) == 0 {
		changes = []string{"no changes"}
	}
	return changes, nil
}

func fieldChanged(a, b appspec.FieldSpec) bool {
	return a.Required != b.Required || a.Readonly != b.Readonly || a.Indexed != b.Indexed ||
		a.Relation != b.Relation || strings.Join(a.Values, ",") != strings.Join(b.Values, ",") ||
		fmt.Sprint(a.Default) != fmt.Sprint(b.Default) || validateKey(a.Validate) != validateKey(b.Validate)
}

func validateKey(v appspec.ValidateSpec) string {
	bound := func(p *int) string {
		if p == nil {
			return "-"
		}
		return fmt.Sprint(*p)
	}
	return fmt.Sprintf("%d|%d|%s|%s|%s", v.MinLength, v.MaxLength, v.Pattern, bound(v.Min), bound(v.Max))
}

func queryNames(qs []appspec.QuerySpec) map[string]string {
	out := map[string]string{}
	for _, q := range qs {
		out[q.Name] = fmt.Sprintf("%s|%s|%s|%d", q.List, q.Get, q.Returns, len(q.Args))
	}
	return out
}

func commandNames(cs []appspec.CommandSpec) map[string]string {
	out := map[string]string{}
	for _, c := range cs {
		out[c.Name] = fmt.Sprintf("%s|%s|%s|%s|%d", c.Create, c.Update, c.Delete, c.Returns, len(c.Args))
	}
	return out
}

func diffNames(kind string, old, next map[string]string) []string {
	var out []string
	names := make([]string, 0, len(next))
	for n := range next {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		prev, ok := old[n]
		switch {
		case !ok:
			out = append(out, "add "+kind+" "+n)
		case prev != next[n]:
			out = append(out, "update "+kind+" "+n)
		}
	}
	var removed []string
	for n := range old {
		if _, ok := next[n]; !ok {
			removed = append(removed, n)
		}
	}
	sort.Strings(removed)
	for _, n := range removed {
		out = append(out, "remove "+kind+" "+n)
	}
	return out
}
