package engine

import (
	"sort"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"
)

// appendDeclaredViews adds spec layouts as list/form views.
// A view whose name matches the generated view replaces that view; any other
// name is added beside it. A compiled page stays mounted by the SPA registry.
// Each components[] entry is published as "{page}.{id}" for SpecScreen.
func appendDeclaredViews(spec appspec.AppSpec, out []views.View) []views.View {
	for _, sc := range spec.Views {
		if len(sc.Components) > 0 {
			out = publishScreen(out, sc)
		}
		if sc.HasLayout() {
			out = publishLayout(spec, out, sc)
		}
		for _, c := range sc.Components {
			out = publishLayout(spec, out, c.Layout(sc.Name))
		}
	}
	return out
}

func publishScreen(out []views.View, sc appspec.ViewSpec) []views.View {
	ids := make([]string, len(sc.Components))
	for i, c := range sc.Components {
		ids[i] = c.ID
	}
	tree := layoutNode(sc.ResolvedLayout())
	next := views.View{Name: sc.Name, Kind: views.ScreenView, Layout: &tree, Components: ids}
	for i := range out {
		if out[i].Name == sc.Name {
			out[i] = next
			return out
		}
	}
	return append(out, next)
}

func layoutNode(n appspec.ViewLayoutSpec) views.LayoutNode {
	out := views.LayoutNode{Direction: n.Direction, Component: n.Component, Size: n.Size, Span: n.Span}
	for _, child := range n.Items {
		out.Items = append(out.Items, layoutNode(child))
	}
	return out
}

func publishLayout(spec appspec.AppSpec, out []views.View, sc appspec.ViewSpec) []views.View {
	m, ok := modelByName(spec, sc.Model)
	if !ok {
		return out
	}
	var base views.View
	if sc.Kind == "form" {
		base = buildFormView(spec, m)
	} else {
		base = buildListView(spec, m)
	}
	next := overlayView(base, sc, m)
	for i := range out {
		if out[i].Name == sc.Name && out[i].Kind == next.Kind {
			out[i] = next
			return out
		}
	}
	return append(out, next)
}

func overlayView(v views.View, sc appspec.ViewSpec, m appspec.ModelSpec) views.View {
	v.Name = sc.Name
	v.DefaultView = sc.DefaultView
	v.Views = append([]string{}, sc.Views...)
	v.GroupBy = append([]string{}, sc.GroupBy...)
	if v.Kind == views.ListView && len(sc.Columns) > 0 {
		v.Columns = columnsFromView(m, sc.Columns)
	}
	if v.Kind == views.FormView && (len(sc.Groups) > 0 || len(sc.Columns) > 0) {
		v.Fields, v.Groups = fieldsFromView(m, sc)
	}
	if len(sc.Filters) > 0 {
		v.FilterPresets = filterPresetsFromList(sc.Filters)
	}
	return v
}

func columnsFromView(m appspec.ModelSpec, cols []appspec.ViewColumnSpec) []views.Column {
	sorted := append([]appspec.ViewColumnSpec(nil), cols...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })
	out := make([]views.Column, 0, len(sorted))
	for _, c := range sorted {
		label := pascal(c.Field)
		if f, ok := fieldByName(m, c.Field); ok {
			label = pascal(f.Name)
		}
		out = append(out, views.Column{
			Key: c.Field, Label: label, Width: c.Width, Align: c.Align,
			Hidden: c.Hidden, Widget: c.Widget, Span: c.Span, Order: c.Order,
		})
	}
	return out
}

func fieldsFromView(m appspec.ModelSpec, sc appspec.ViewSpec) ([]views.Field, []views.ViewGroup) {
	groups := append([]appspec.ViewGroupSpec(nil), sc.Groups...)
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Order < groups[j].Order })
	var viewGroups []views.ViewGroup
	var fields []views.Field
	if len(groups) == 0 {
		fields = append(fields, viewFields(m, "", sc.Columns)...)
		return fields, nil
	}
	for _, g := range groups {
		id := g.LabelKey
		if id == "" {
			id = g.Label
		}
		if id == "" {
			id = "group"
		}
		viewGroups = append(viewGroups, views.ViewGroup{
			ID: id, Label: g.Label, LabelKey: g.LabelKey, Columns: g.Columns, Order: g.Order,
		})
		fields = append(fields, viewFields(m, id, g.Fields)...)
	}
	return fields, viewGroups
}

func viewFields(m appspec.ModelSpec, group string, cols []appspec.ViewColumnSpec) []views.Field {
	sorted := append([]appspec.ViewColumnSpec(nil), cols...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })
	out := make([]views.Field, 0, len(sorted))
	for _, c := range sorted {
		f, ok := fieldByName(m, c.Field)
		item := views.Field{
			Key: c.Field, Label: pascal(c.Field), Span: c.Span, Widget: c.Widget,
			Group: group, Order: c.Order, Width: c.Width, Align: c.Align,
		}
		if ok {
			item.Label = pascal(f.Name)
			item.Type = f.CanonicalType()
			item.Required = f.Required
			item.Relation = f.Relation
			item.Inverse = f.Inverse
			item.Values = append([]string{}, f.Values...)
		}
		out = append(out, item)
	}
	return out
}
