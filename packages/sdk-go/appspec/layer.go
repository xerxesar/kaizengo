package appspec

import (
	"fmt"
	"reflect"
)

// Merge stacks layer onto base. A layer may override title and summary and add
// depends, models, fields, queries, commands, roles, and policy entries.
// Anything it repeats from base must be identical; base-only parts (nav, menus,
// views, locales, keymap, filters, charts) always come from base.
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
	out.Security.Disable = union(base.Security.Disable, layer.Security.Disable)
	return out, nil
}

// Subtract returns the layer that, merged onto base, yields full. It fails when
// full drops or changes something base defines.
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
		if len(lm.Fields) > 0 || lm.Search != nil {
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
	if layer.Security.Disable, err = extra("security.disable", full.Security.Disable, base.Security.Disable); err != nil {
		return AppSpec{}, err
	}
	return layer, nil
}

// EmptyLayer reports whether layer adds nothing.
func (s AppSpec) EmptyLayer() bool {
	return s.Title == "" && s.Summary == "" && len(s.Depends)+len(s.Provides)+len(s.Uses) == 0 &&
		len(s.Models)+len(s.Queries)+len(s.Commands) == 0 &&
		len(s.Security.Roles)+len(s.Security.Entries)+len(s.Security.Disable) == 0
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
