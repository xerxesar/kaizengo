package appspec

import (
	"fmt"
	"reflect"
)

// Merge stacks a legacy overlay onto base. Runtime domain apply now stores full
// tip snapshots and no longer uses Merge for apply; Merge remains for reading
// pre-snapshot overlay revisions and for rebasing tooling.
// A layer may override title, summary, and nav, and add depends, models, fields,
// queries, commands, menus, locales, keymap bindings, filter and chart presets,
// indexes, views, roles, policy entries, and users. Anything it repeats from
// base must be identical. Go hook extends and component exports always come from base.
func Merge(base, layer AppSpec) (AppSpec, error) {
	if layer.Name != base.Name {
		return AppSpec{}, fmt.Errorf("layer is for app %q, not %q", layer.Name, base.Name)
	}
	out := base
	if layer.Title != "" {
		out.Title = layer.Title
	}
	if layer.Summary != "" {
		out.Summary = layer.Summary
	}
	out.Depends = union(base.Depends, layer.Depends)
	out.Provides = union(base.Provides, layer.Provides)
	out.Uses = union(base.Uses, layer.Uses)
	if !layer.Nav.IsZero() {
		out.Nav = layer.Nav
	}

	out.Models = append([]ModelSpec(nil), base.Models...)
	index := map[string]int{}
	for i, m := range out.Models {
		index[m.Name] = i
	}
	for _, lm := range layer.Models {
		i, ok := index[lm.Name]
		if !ok {
			index[lm.Name] = len(out.Models)
			out.Models = append(out.Models, lm)
			continue
		}
		m := out.Models[i]
		if (lm.Internal && !m.Internal) || (lm.Virtual && !m.Virtual) {
			return AppSpec{}, fmt.Errorf("model %s: internal/virtual flags come from the base layer", lm.Name)
		}
		if lm.Search != nil && !reflect.DeepEqual(lm.Search, m.Search) {
			if m.Search != nil {
				return AppSpec{}, fmt.Errorf("model %s: search is defined by the base layer", lm.Name)
			}
			m.Search = lm.Search
		}
		var err error
		if m.Filters, err = mergeNamed("filter", m.Filters, lm.Filters, func(f FilterPresetSpec) string { return f.ID }); err != nil {
			return AppSpec{}, err
		}
		if m.Charts, err = mergeNamed("chart", m.Charts, lm.Charts, func(c ChartPresetSpec) string { return c.ID }); err != nil {
			return AppSpec{}, err
		}
		if m.Indexes, err = mergeNamed("index", m.Indexes, lm.Indexes, func(ix IndexSpec) string { return ix.Name }); err != nil {
			return AppSpec{}, err
		}
		fields := append([]FieldSpec(nil), m.Fields...)
		have := map[string]FieldSpec{}
		for _, f := range fields {
			have[f.Name] = f
		}
		for _, f := range lm.Fields {
			if prev, ok := have[f.Name]; ok {
				if !reflect.DeepEqual(prev, f) {
					return AppSpec{}, fmt.Errorf("field %s.%s is defined by the base layer and cannot be changed", lm.Name, f.Name)
				}
				continue
			}
			have[f.Name] = f
			fields = append(fields, f)
		}
		m.Fields = fields
		out.Models[i] = m
	}

	var err error
	if out.Queries, err = mergeNamed("query", base.Queries, layer.Queries, func(q QuerySpec) string { return q.Name }); err != nil {
		return AppSpec{}, err
	}
	if out.Commands, err = mergeNamed("command", base.Commands, layer.Commands, func(c CommandSpec) string { return c.Name }); err != nil {
		return AppSpec{}, err
	}
	if out.Security.Roles, err = mergeNamed("role", base.Security.Roles, layer.Security.Roles, func(r SecurityRoleSpec) string { return r.Name }); err != nil {
		return AppSpec{}, err
	}
	if out.Security.Entries, err = mergeNamed("policy entry", base.Security.Entries, layer.Security.Entries, func(e SecurityEntrySpec) string { return e.Name }); err != nil {
		return AppSpec{}, err
	}
	if out.Security.Users, err = mergeNamed("user", base.Security.Users, layer.Security.Users, func(u SecurityUserSpec) string { return u.Email }); err != nil {
		return AppSpec{}, err
	}
	out.Security.Disable = union(base.Security.Disable, layer.Security.Disable)
	if out.Menus, err = mergeNamed("menu", base.Menus, layer.Menus, func(m MenuSpec) string { return m.ID }); err != nil {
		return AppSpec{}, err
	}
	if out.Locales, err = mergeNamed("locale", base.Locales, layer.Locales, func(l LocaleSpec) string { return l.ID }); err != nil {
		return AppSpec{}, err
	}
	if out.Keymap.Bindings, err = mergeNamed("keymap binding", base.Keymap.Bindings, layer.Keymap.Bindings, func(b KeymapBindingSpec) string { return b.ID }); err != nil {
		return AppSpec{}, err
	}
	if out.Views, err = mergeNamed("view", base.Views, layer.Views, func(v ViewSpec) string { return v.Name }); err != nil {
		return AppSpec{}, err
	}
	out.Keymap.Disable = union(base.Keymap.Disable, layer.Keymap.Disable)
	out.Exports.Components = base.Exports.Components
	out.Exports.Views = base.Exports.Views
	if out.Exports.Menus, err = mergeNamed("menu export", base.Exports.Menus, layer.Exports.Menus, func(m MenuExtendSpec) string { return m.App + "/" + m.ID }); err != nil {
		return AppSpec{}, err
	}
	return out, nil
}

// Subtract returns the overlay that, merged onto base, yields full. Used for
// changelog / rebase helpers — not for storing tip revisions (tips are full
// snapshots). It fails when full drops or changes something base defines.
func Subtract(full, base AppSpec) (AppSpec, error) {
	if full.Name != base.Name {
		return AppSpec{}, fmt.Errorf("document is for app %q, not %q", full.Name, base.Name)
	}
	layer := AppSpec{Name: base.Name}
	if full.Title != base.Title {
		layer.Title = full.Title
	}
	if full.Summary != base.Summary {
		layer.Summary = full.Summary
	}
	var err error
	if layer.Depends, err = extra("depends", full.Depends, base.Depends); err != nil {
		return AppSpec{}, err
	}
	if layer.Provides, err = extra("provides", full.Provides, base.Provides); err != nil {
		return AppSpec{}, err
	}
	if layer.Uses, err = extra("uses", full.Uses, base.Uses); err != nil {
		return AppSpec{}, err
	}
	if !reflect.DeepEqual(full.Nav, base.Nav) {
		layer.Nav = full.Nav
	}

	fullModels := map[string]ModelSpec{}
	for _, m := range full.Models {
		fullModels[m.Name] = m
	}
	baseModels := map[string]struct{}{}
	for _, bm := range base.Models {
		baseModels[bm.Name] = struct{}{}
		fm, ok := fullModels[bm.Name]
		if !ok {
			return AppSpec{}, fmt.Errorf("model %s is defined by the base layer and cannot be removed", bm.Name)
		}
		if fm.Internal != bm.Internal || fm.Virtual != bm.Virtual {
			return AppSpec{}, fmt.Errorf("model %s: internal/virtual flags come from the base layer", bm.Name)
		}
		lm := ModelSpec{Name: bm.Name}
		if !reflect.DeepEqual(fm.Search, bm.Search) {
			if bm.Search != nil {
				return AppSpec{}, fmt.Errorf("model %s: search is defined by the base layer", bm.Name)
			}
			lm.Search = fm.Search
		}
		if lm.Filters, err = subtractNamed("filter", fm.Filters, bm.Filters, func(f FilterPresetSpec) string { return f.ID }); err != nil {
			return AppSpec{}, err
		}
		if lm.Charts, err = subtractNamed("chart", fm.Charts, bm.Charts, func(c ChartPresetSpec) string { return c.ID }); err != nil {
			return AppSpec{}, err
		}
		if lm.Indexes, err = subtractNamed("index", fm.Indexes, bm.Indexes, func(ix IndexSpec) string { return ix.Name }); err != nil {
			return AppSpec{}, err
		}
		have := map[string]FieldSpec{}
		for _, f := range fm.Fields {
			have[f.Name] = f
		}
		inBase := map[string]struct{}{}
		for _, bf := range bm.Fields {
			inBase[bf.Name] = struct{}{}
			f, ok := have[bf.Name]
			if !ok {
				return AppSpec{}, fmt.Errorf("field %s.%s is defined by the base layer and cannot be removed", bm.Name, bf.Name)
			}
			if !reflect.DeepEqual(f, bf) {
				return AppSpec{}, fmt.Errorf("field %s.%s is defined by the base layer and cannot be changed", bm.Name, bf.Name)
			}
		}
		for _, f := range fm.Fields {
			if _, ok := inBase[f.Name]; !ok {
				lm.Fields = append(lm.Fields, f)
			}
		}
		if len(lm.Fields) > 0 || lm.Search != nil || len(lm.Filters) > 0 || len(lm.Charts) > 0 || len(lm.Indexes) > 0 {
			layer.Models = append(layer.Models, lm)
		}
	}
	for _, m := range full.Models {
		if _, ok := baseModels[m.Name]; !ok {
			layer.Models = append(layer.Models, m)
		}
	}

	if layer.Queries, err = subtractNamed("query", full.Queries, base.Queries, func(q QuerySpec) string { return q.Name }); err != nil {
		return AppSpec{}, err
	}
	if layer.Commands, err = subtractNamed("command", full.Commands, base.Commands, func(c CommandSpec) string { return c.Name }); err != nil {
		return AppSpec{}, err
	}
	if layer.Security.Roles, err = subtractNamed("role", full.Security.Roles, base.Security.Roles, func(r SecurityRoleSpec) string { return r.Name }); err != nil {
		return AppSpec{}, err
	}
	if layer.Security.Entries, err = subtractNamed("policy entry", full.Security.Entries, base.Security.Entries, func(e SecurityEntrySpec) string { return e.Name }); err != nil {
		return AppSpec{}, err
	}
	if layer.Security.Users, err = subtractNamed("user", full.Security.Users, base.Security.Users, func(u SecurityUserSpec) string { return u.Email }); err != nil {
		return AppSpec{}, err
	}
	if layer.Security.Disable, err = extra("security.disable", full.Security.Disable, base.Security.Disable); err != nil {
		return AppSpec{}, err
	}
	if layer.Menus, err = subtractNamed("menu", full.Menus, base.Menus, func(m MenuSpec) string { return m.ID }); err != nil {
		return AppSpec{}, err
	}
	if layer.Locales, err = subtractNamed("locale", full.Locales, base.Locales, func(l LocaleSpec) string { return l.ID }); err != nil {
		return AppSpec{}, err
	}
	if layer.Keymap.Bindings, err = subtractNamed("keymap binding", full.Keymap.Bindings, base.Keymap.Bindings, func(b KeymapBindingSpec) string { return b.ID }); err != nil {
		return AppSpec{}, err
	}
	if layer.Keymap.Disable, err = extra("keymap.disable", full.Keymap.Disable, base.Keymap.Disable); err != nil {
		return AppSpec{}, err
	}
	if layer.Views, err = subtractNamed("view", declaredViews(full.Views), declaredViews(base.Views), func(v ViewSpec) string { return v.Name }); err != nil {
		return AppSpec{}, err
	}
	if layer.Exports.Menus, err = subtractNamed("menu export", full.Exports.Menus, base.Exports.Menus, func(m MenuExtendSpec) string { return m.App + "/" + m.ID }); err != nil {
		return AppSpec{}, err
	}
	return layer, nil
}

// EmptyLayer reports whether layer adds nothing.
func (s AppSpec) EmptyLayer() bool {
	return s.Title == "" && s.Summary == "" && s.Nav.IsZero() &&
		len(s.Depends)+len(s.Provides)+len(s.Uses) == 0 &&
		len(s.Models)+len(s.Queries)+len(s.Commands)+len(s.Menus)+len(s.Locales)+len(declaredViews(s.Views)) == 0 &&
		len(s.Security.Roles)+len(s.Security.Entries)+len(s.Security.Disable)+len(s.Security.Users) == 0 &&
		s.Keymap.Empty() && len(s.Exports.Menus) == 0
}

// IsZero reports whether nav carries no shell entry.
func (n NavSpec) IsZero() bool {
	return n.Label == "" && n.LabelKey == "" && n.Route == "" && n.Order == 0
}

func union(base, add []string) []string {
	if len(add) == 0 {
		return base
	}
	out := append([]string(nil), base...)
	seen := map[string]struct{}{}
	for _, v := range base {
		seen[v] = struct{}{}
	}
	for _, v := range add {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func extra(what string, full, base []string) ([]string, error) {
	have := map[string]struct{}{}
	for _, v := range full {
		have[v] = struct{}{}
	}
	inBase := map[string]struct{}{}
	for _, v := range base {
		inBase[v] = struct{}{}
		if _, ok := have[v]; !ok {
			return nil, fmt.Errorf("%s: %q comes from the base layer and cannot be removed", what, v)
		}
	}
	var out []string
	for _, v := range full {
		if _, ok := inBase[v]; !ok {
			out = append(out, v)
		}
	}
	return out, nil
}

func mergeNamed[T any](kind string, base, layer []T, name func(T) string) ([]T, error) {
	if len(layer) == 0 {
		return base, nil
	}
	out := append([]T(nil), base...)
	have := map[string]T{}
	for _, v := range base {
		have[name(v)] = v
	}
	for _, v := range layer {
		if prev, ok := have[name(v)]; ok {
			if !reflect.DeepEqual(prev, v) {
				return nil, fmt.Errorf("%s %s is defined by the base layer and cannot be changed", kind, name(v))
			}
			continue
		}
		have[name(v)] = v
		out = append(out, v)
	}
	return out, nil
}

func subtractNamed[T any](kind string, full, base []T, name func(T) string) ([]T, error) {
	have := map[string]T{}
	for _, v := range full {
		have[name(v)] = v
	}
	inBase := map[string]struct{}{}
	for _, b := range base {
		inBase[name(b)] = struct{}{}
		v, ok := have[name(b)]
		if !ok {
			return nil, fmt.Errorf("%s %s is defined by the base layer and cannot be removed", kind, name(b))
		}
		if !reflect.DeepEqual(v, b) {
			return nil, fmt.Errorf("%s %s is defined by the base layer and cannot be changed", kind, name(b))
		}
	}
	var out []T
	for _, v := range full {
		if _, ok := inBase[name(v)]; !ok {
			out = append(out, v)
		}
	}
	return out, nil
}
