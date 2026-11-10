package appspec

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Conflict is one path where both sides changed away from the ancestor differently
// (or, with no ancestor, where ours and theirs simply differ).
type Conflict struct {
	Path     string `json:"path"`
	Ancestor string `json:"ancestor,omitempty"`
	Ours     string `json:"ours"`
	Theirs   string `json:"theirs"`
}

// MergeResult is the outcome of MergeTwoWay / MergeThreeWay.
type MergeResult struct {
	Spec      AppSpec
	Conflicts []Conflict
}

// TwoWayResult is an alias kept for callers.
type TwoWayResult = MergeResult

// MergeOpts controls three-way rebase conflict handling.
type MergeOpts struct {
	Force bool
	// Resolve maps a conflict path to "ours" or "theirs" (skips recording that conflict).
	Resolve map[string]string
}

// MergeTwoWay merges tip (ours) with incoming code (theirs) with no ancestor.
// Prefer MergeThreeWay when a last-compiled snapshot is available.
func MergeTwoWay(ours, theirs AppSpec, force bool) (MergeResult, error) {
	return MergeThreeWay(AppSpec{}, ours, theirs, force)
}

// MergeThreeWay rebases ours onto theirs relative to ancestor (last compiled tip).
// force takes code (theirs) on every conflict. Prefer MergeThreeWayOpts for per-path resolve.
func MergeThreeWay(ancestor, ours, theirs AppSpec, force bool) (MergeResult, error) {
	return MergeThreeWayOpts(ancestor, ours, theirs, MergeOpts{Force: force})
}

// MergeThreeWayOpts is MergeThreeWay with per-path resolutions.
//
// Per path:
//   - only ours changed vs ancestor → keep ours
//   - only theirs changed → take theirs
//   - both changed the same way → keep
//   - both changed differently → conflict (force / Resolve → take side)
//   - only in ours / only in theirs (not in ancestor) → keep / take
//
// Extends and Exports.Components/Views always come from theirs (code).
func MergeThreeWayOpts(ancestor, ours, theirs AppSpec, opts MergeOpts) (MergeResult, error) {
	if err := sameApp(ancestor, ours, theirs); err != nil {
		return MergeResult{}, err
	}
	anc := hasAncestorDoc(ancestor)
	out := AppSpec{}
	out.Name = firstNonEmpty(theirs.Name, ours.Name, ancestor.Name)
	out.Schema = firstNonEmpty(theirs.Schema, ours.Schema, ancestor.Schema, out.Name)
	out.Resource = firstNonEmpty(theirs.Resource, ours.Resource, ancestor.Resource, out.Name)
	out.EnableExtensions = theirs.EnableExtensions || ours.EnableExtensions
	out.EnableAuth = theirs.EnableAuth || ours.EnableAuth
	out.EnableI18n = theirs.EnableI18n || ours.EnableI18n
	out.EnableSPA = theirs.EnableSPA
	out.AutoInstall = theirs.AutoInstall
	out.Extends = append([]ExtendSpec(nil), theirs.Extends...)
	out.Exports.Components = append([]ComponentExportSpec(nil), theirs.Exports.Components...)
	out.Exports.Views = append([]ViewExtendSpec(nil), theirs.Exports.Views...)

	var conflicts []Conflict
	out.Title, conflicts = pick3("title", ancestor.Title, ours.Title, theirs.Title, opts, anc, conflicts)
	out.Summary, conflicts = pick3("summary", ancestor.Summary, ours.Summary, theirs.Summary, opts, anc, conflicts)
	out.Depends = union(union(ancestor.Depends, ours.Depends), theirs.Depends)
	out.Provides = union(union(ancestor.Provides, ours.Provides), theirs.Provides)
	out.Uses = union(union(ancestor.Uses, ours.Uses), theirs.Uses)

	out.Nav, conflicts = pickNav3(ancestor.Nav, ours.Nav, theirs.Nav, opts, anc, conflicts)
	out.Models, conflicts = mergeModels3(ancestor.Models, ours.Models, theirs.Models, opts, anc, conflicts)
	out.Views, conflicts = mergeKeyed3("views", declaredViews(ancestor.Views), declaredViews(ours.Views), declaredViews(theirs.Views), opts, anc, conflicts, func(v ViewSpec) string { return v.Name })
	out.Menus, conflicts = mergeKeyed3("menus", ancestor.Menus, ours.Menus, theirs.Menus, opts, anc, conflicts, func(m MenuSpec) string { return m.ID })
	out.Queries, conflicts = mergeKeyed3("queries", ancestor.Queries, ours.Queries, theirs.Queries, opts, anc, conflicts, func(q QuerySpec) string { return q.Name })
	out.Commands, conflicts = mergeKeyed3("commands", ancestor.Commands, ours.Commands, theirs.Commands, opts, anc, conflicts, func(c CommandSpec) string { return c.Name })
	out.Locales, conflicts = mergeKeyed3("locales", ancestor.Locales, ours.Locales, theirs.Locales, opts, anc, conflicts, func(l LocaleSpec) string { return l.ID })
	out.Exports.Menus, conflicts = mergeKeyed3("exports.menus", ancestor.Exports.Menus, ours.Exports.Menus, theirs.Exports.Menus, opts, anc, conflicts, func(m MenuExtendSpec) string { return m.App + "/" + m.ID })
	out.Security, conflicts = mergeSecurity3(ancestor.Security, ours.Security, theirs.Security, opts, anc, conflicts)
	out.Keymap, conflicts = mergeKeymap3(ancestor.Keymap, ours.Keymap, theirs.Keymap, opts, anc, conflicts)

	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Path < conflicts[j].Path })
	return MergeResult{Spec: out, Conflicts: conflicts}, nil
}

// takeTheirs reports whether a conflict at path should take code (theirs).
// stillConflict is true when the path remains unresolved.
func takeTheirs(path string, opts MergeOpts) (theirs bool, stillConflict bool) {
	switch strings.ToLower(strings.TrimSpace(opts.Resolve[path])) {
	case "theirs", "code":
		return true, false
	case "ours", "tip":
		return false, false
	}
	if opts.Force {
		return true, false
	}
	return false, true
}

func sameApp(vals ...AppSpec) error {
	var name string
	for _, v := range vals {
		if v.Name == "" {
			continue
		}
		if name == "" {
			name = v.Name
			continue
		}
		if v.Name != name {
			return fmt.Errorf("merge: app %q vs %q", name, v.Name)
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func hasAncestorDoc(a AppSpec) bool {
	return a.Name != "" || a.Title != "" || a.Summary != "" ||
		len(a.Models) > 0 || len(a.Views) > 0 || len(a.Menus) > 0 ||
		len(a.Queries) > 0 || len(a.Commands) > 0
}

func pick3(path, ancestor, ours, theirs string, opts MergeOpts, anc bool, conflicts []Conflict) (string, []Conflict) {
	if !anc {
		switch {
		case ours == theirs:
			return ours, conflicts
		case ours == "":
			return theirs, conflicts
		case theirs == "":
			return ours, conflicts
		default:
			if take, still := takeTheirs(path, opts); !still {
				if take {
					return theirs, conflicts
				}
				return ours, conflicts
			}
			return ours, append(conflicts, Conflict{Path: path, Ours: ours, Theirs: theirs})
		}
	}
	switch {
	case ours == theirs:
		return ours, conflicts
	case ancestor == ours:
		return theirs, conflicts
	case ancestor == theirs:
		return ours, conflicts
	default:
		if take, still := takeTheirs(path, opts); !still {
			if take {
				return theirs, conflicts
			}
			return ours, conflicts
		}
		return ours, append(conflicts, Conflict{Path: path, Ancestor: ancestor, Ours: ours, Theirs: theirs})
	}
}

func pickNav3(ancestor, ours, theirs NavSpec, opts MergeOpts, anc bool, conflicts []Conflict) (NavSpec, []Conflict) {
	path := "nav"
	if !anc {
		switch {
		case reflect.DeepEqual(ours, theirs):
			return ours, conflicts
		case ours.IsZero():
			return theirs, conflicts
		case theirs.IsZero():
			return ours, conflicts
		default:
			if take, still := takeTheirs(path, opts); !still {
				if take {
					return theirs, conflicts
				}
				return ours, conflicts
			}
			return ours, append(conflicts, Conflict{Path: path, Ours: fmt.Sprint(ours), Theirs: fmt.Sprint(theirs)})
		}
	}
	switch {
	case reflect.DeepEqual(ours, theirs):
		return ours, conflicts
	case reflect.DeepEqual(ancestor, ours):
		return theirs, conflicts
	case reflect.DeepEqual(ancestor, theirs):
		return ours, conflicts
	case ours.IsZero():
		return theirs, conflicts
	case theirs.IsZero():
		return ours, conflicts
	default:
		if take, still := takeTheirs(path, opts); !still {
			if take {
				return theirs, conflicts
			}
			return ours, conflicts
		}
		return ours, append(conflicts, Conflict{Path: path, Ancestor: fmt.Sprint(ancestor), Ours: fmt.Sprint(ours), Theirs: fmt.Sprint(theirs)})
	}
}

func mergeModels3(ancestor, ours, theirs []ModelSpec, opts MergeOpts, anc bool, conflicts []Conflict) ([]ModelSpec, []Conflict) {
	type trip struct{ a, o, t *ModelSpec }
	by := map[string]trip{}
	var order []string
	seen := map[string]struct{}{}
	add := func(items []ModelSpec, side int) {
		for i := range items {
			n := items[i].Name
			p := by[n]
			item := items[i]
			switch side {
			case 0:
				p.a = &item
			case 1:
				p.o = &item
			default:
				p.t = &item
			}
			by[n] = p
			if _, ok := seen[n]; !ok {
				seen[n] = struct{}{}
				order = append(order, n)
			}
		}
	}
	add(ancestor, 0)
	add(ours, 1)
	add(theirs, 2)
	out := make([]ModelSpec, 0, len(order))
	for _, n := range order {
		p := by[n]
		switch {
		case p.o == nil && p.t == nil:
			continue // dropped both sides
		case p.o == nil && p.t != nil:
			path := "models." + n
			if p.a != nil && !reflect.DeepEqual(*p.a, *p.t) {
				if take, still := takeTheirs(path, opts); still {
					conflicts = append(conflicts, Conflict{
						Path: path, Ancestor: fmt.Sprint(*p.a), Ours: "(removed)", Theirs: fmt.Sprint(*p.t),
					})
					continue // keep removed until resolved
				} else if !take {
					continue
				}
			}
			out = append(out, *p.t)
		case p.o != nil && p.t == nil:
			path := "models." + n
			if p.a != nil && !reflect.DeepEqual(*p.a, *p.o) {
				if take, still := takeTheirs(path, opts); still {
					conflicts = append(conflicts, Conflict{
						Path: path, Ancestor: fmt.Sprint(*p.a), Ours: fmt.Sprint(*p.o), Theirs: "(removed)",
					})
					out = append(out, *p.o)
					continue
				} else if take {
					continue // take theirs = removed
				}
			}
			if p.a != nil && reflect.DeepEqual(*p.a, *p.o) {
				continue // theirs deleted unchanged model
			}
			out = append(out, *p.o) // ours-only add, or keep ours
		case reflect.DeepEqual(*p.o, *p.t):
			out = append(out, *p.o)
		case p.a != nil && reflect.DeepEqual(*p.a, *p.o):
			out = append(out, *p.t)
		case p.a != nil && reflect.DeepEqual(*p.a, *p.t):
			out = append(out, *p.o)
		default:
			a := ModelSpec{Name: n}
			if p.a != nil {
				a = *p.a
			}
			m, c := mergeOneModel3(a, *p.o, *p.t, opts, anc)
			conflicts = append(conflicts, c...)
			out = append(out, m)
		}
	}
	return out, conflicts
}

func mergeOneModel3(ancestor, ours, theirs ModelSpec, opts MergeOpts, anc bool) (ModelSpec, []Conflict) {
	prefix := "models." + ours.Name
	out := ModelSpec{Name: ours.Name}
	var conflicts []Conflict

	out.Internal, conflicts = pickBool3(prefix+".internal", ancestor.Internal, ours.Internal, theirs.Internal, opts, anc, conflicts)
	out.Virtual, conflicts = pickBool3(prefix+".virtual", ancestor.Virtual, ours.Virtual, theirs.Virtual, opts, anc, conflicts)

	switch {
	case reflect.DeepEqual(ours.Search, theirs.Search):
		out.Search = ours.Search
	case reflect.DeepEqual(ancestor.Search, ours.Search):
		out.Search = theirs.Search
	case reflect.DeepEqual(ancestor.Search, theirs.Search):
		out.Search = ours.Search
	default:
		sp := prefix + ".search"
		if take, still := takeTheirs(sp, opts); !still {
			if take {
				out.Search = theirs.Search
			} else {
				out.Search = ours.Search
			}
		} else {
			conflicts = append(conflicts, Conflict{Path: sp, Ancestor: fmt.Sprint(ancestor.Search), Ours: fmt.Sprint(ours.Search), Theirs: fmt.Sprint(theirs.Search)})
			out.Search = ours.Search
		}
	}

	out.Fields, conflicts = mergeKeyed3(prefix+".fields", ancestor.Fields, ours.Fields, theirs.Fields, opts, anc, conflicts, func(f FieldSpec) string { return f.Name })
	out.Indexes, conflicts = mergeKeyed3(prefix+".indexes", ancestor.Indexes, ours.Indexes, theirs.Indexes, opts, anc, conflicts, indexMergeKey)
	out.Filters, conflicts = mergeKeyed3(prefix+".filters", ancestor.Filters, ours.Filters, theirs.Filters, opts, anc, conflicts, func(f FilterPresetSpec) string { return f.ID })
	out.Charts, conflicts = mergeKeyed3(prefix+".charts", ancestor.Charts, ours.Charts, theirs.Charts, opts, anc, conflicts, func(c ChartPresetSpec) string { return c.ID })
	return out, conflicts
}

func pickBool3(path string, ancestor, ours, theirs bool, opts MergeOpts, anc bool, conflicts []Conflict) (bool, []Conflict) {
	if !anc {
		if ours == theirs {
			return ours, conflicts
		}
		if take, still := takeTheirs(path, opts); !still {
			if take {
				return theirs, conflicts
			}
			return ours, conflicts
		}
		return ours, append(conflicts, Conflict{Path: path, Ours: fmt.Sprint(ours), Theirs: fmt.Sprint(theirs)})
	}
	switch {
	case ours == theirs:
		return ours, conflicts
	case ancestor == ours:
		return theirs, conflicts
	case ancestor == theirs:
		return ours, conflicts
	default:
		if take, still := takeTheirs(path, opts); !still {
			if take {
				return theirs, conflicts
			}
			return ours, conflicts
		}
		return ours, append(conflicts, Conflict{
			Path: path, Ancestor: fmt.Sprint(ancestor), Ours: fmt.Sprint(ours), Theirs: fmt.Sprint(theirs),
		})
	}
}

func indexMergeKey(ix IndexSpec) string {
	if ix.Name != "" {
		return ix.Name
	}
	return fmt.Sprintf("%v|%v|%s|%v|%s", ix.Fields, ix.Unique, ix.Method, ix.Include, ix.Where)
}

func mergeKeyed3[T any](prefix string, ancestor, ours, theirs []T, opts MergeOpts, anc bool, conflicts []Conflict, key func(T) string) ([]T, []Conflict) {
	type trip struct{ a, o, t *T }
	by := map[string]trip{}
	var order []string
	seen := map[string]struct{}{}
	add := func(items []T, side int) {
		for i := range items {
			k := key(items[i])
			p := by[k]
			item := items[i]
			switch side {
			case 0:
				p.a = &item
			case 1:
				p.o = &item
			default:
				p.t = &item
			}
			by[k] = p
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				order = append(order, k)
			}
		}
	}
	add(ancestor, 0)
	add(ours, 1)
	add(theirs, 2)
	out := make([]T, 0, len(order))
	for _, k := range order {
		p := by[k]
		path := prefix + "." + k
		switch {
		case p.o == nil && p.t == nil:
			continue
		case p.o == nil && p.t != nil:
			if p.a != nil && !reflect.DeepEqual(*p.a, *p.t) {
				if take, still := takeTheirs(path, opts); still {
					conflicts = append(conflicts, Conflict{Path: path, Ancestor: fmt.Sprint(*p.a), Ours: "(removed)", Theirs: fmt.Sprint(*p.t)})
					continue
				} else if !take {
					continue
				}
			}
			out = append(out, *p.t)
		case p.o != nil && p.t == nil:
			if p.a != nil && !reflect.DeepEqual(*p.a, *p.o) {
				if take, still := takeTheirs(path, opts); still {
					conflicts = append(conflicts, Conflict{Path: path, Ancestor: fmt.Sprint(*p.a), Ours: fmt.Sprint(*p.o), Theirs: "(removed)"})
					out = append(out, *p.o)
					continue
				} else if take {
					continue
				}
			}
			if p.a != nil && reflect.DeepEqual(*p.a, *p.o) {
				continue
			}
			out = append(out, *p.o)
		case reflect.DeepEqual(*p.o, *p.t):
			out = append(out, *p.o)
		case p.a != nil && reflect.DeepEqual(*p.a, *p.o):
			out = append(out, *p.t)
		case p.a != nil && reflect.DeepEqual(*p.a, *p.t):
			out = append(out, *p.o)
		default:
			if take, still := takeTheirs(path, opts); !still {
				if take {
					out = append(out, *p.t)
				} else {
					out = append(out, *p.o)
				}
				continue
			}
			anc := ""
			if p.a != nil {
				anc = fmt.Sprint(*p.a)
			}
			conflicts = append(conflicts, Conflict{Path: path, Ancestor: anc, Ours: fmt.Sprint(*p.o), Theirs: fmt.Sprint(*p.t)})
			out = append(out, *p.o)
		}
	}
	return out, conflicts
}

func mergeSecurity3(ancestor, ours, theirs SecuritySpec, opts MergeOpts, anc bool, conflicts []Conflict) (SecuritySpec, []Conflict) {
	var out SecuritySpec
	out.Roles, conflicts = mergeKeyed3("security.roles", ancestor.Roles, ours.Roles, theirs.Roles, opts, anc, conflicts, func(r SecurityRoleSpec) string { return r.Name })
	out.Entries, conflicts = mergeKeyed3("security.entries", ancestor.Entries, ours.Entries, theirs.Entries, opts, anc, conflicts, func(e SecurityEntrySpec) string { return e.Name })
	out.Users, conflicts = mergeKeyed3("security.users", ancestor.Users, ours.Users, theirs.Users, opts, anc, conflicts, func(u SecurityUserSpec) string { return u.Email })
	out.Disable = union(union(ancestor.Disable, ours.Disable), theirs.Disable)
	return out, conflicts
}

func mergeKeymap3(ancestor, ours, theirs KeymapSpec, opts MergeOpts, anc bool, conflicts []Conflict) (KeymapSpec, []Conflict) {
	out := theirs
	out.Bindings, conflicts = mergeKeyed3("keymap.bindings", ancestor.Bindings, ours.Bindings, theirs.Bindings, opts, anc, conflicts, func(b KeymapBindingSpec) string { return b.ID })
	out.Disable = union(union(ancestor.Disable, ours.Disable), theirs.Disable)
	return out, conflicts
}
