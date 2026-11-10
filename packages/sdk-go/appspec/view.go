package appspec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// view names follow page names (GreetingList), not snake_case model names.
var viewNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// ViewColumnSpec is one list column or one field inside a form group.
// Order is ascending sequence within the collection or group.
type ViewColumnSpec struct {
	Field  string `yaml:"field"`
	Width  string `yaml:"width,omitempty"`
	Align  string `yaml:"align,omitempty"`
	Widget string `yaml:"widget,omitempty"`
	Span   int    `yaml:"span,omitempty"`
	Order  int    `yaml:"order,omitempty"`
	Hidden bool   `yaml:"hidden,omitempty"`
}

// ViewGroupSpec is a form section.
type ViewGroupSpec struct {
	Label    string           `yaml:"label,omitempty"`
	LabelKey string           `yaml:"labelKey,omitempty"`
	Columns  int              `yaml:"columns,omitempty"`
	Order    int              `yaml:"order,omitempty"`
	Fields   []ViewColumnSpec `yaml:"fields,omitempty"`
}

// IsPage reports a compiled React page (path set, or discovered Type=page).
// Layout fields may still be set; SpecScreen reads them at runtime.
func (v ViewSpec) IsPage() bool {
	return strings.TrimSpace(v.Path) != "" || v.Type == "page"
}

// HasLayout reports a spec-driven collection or form domain.
func (v ViewSpec) HasLayout() bool {
	return strings.TrimSpace(v.Kind) != ""
}

// forMarshal drops the discovered-page marker. Path and layout are both kept.
func (v ViewSpec) forMarshal() ViewSpec {
	out := v
	out.Type = ""
	return out
}

func validateViews(views []ViewSpec, models map[string]ModelSpec, appRoot string) error {
	seen := map[string]struct{}{}
	for _, v := range views {
		if !viewNameRe.MatchString(v.Name) {
			return fmt.Errorf("view %q has an invalid name", v.Name)
		}
		if _, ok := seen[v.Name]; ok {
			return fmt.Errorf("duplicate view %q", v.Name)
		}
		seen[v.Name] = struct{}{}

		if path := strings.TrimSpace(v.Path); path != "" && appRoot != "" {
			if _, err := resolveViewFile(appRoot, path); err != nil {
				return fmt.Errorf("view %q: %w", v.Name, err)
			}
		}
		if v.HasLayout() && len(v.Components) > 0 {
			return fmt.Errorf("view %q: set kind or components, not both", v.Name)
		}
		seenComponent := map[string]struct{}{}
		for _, c := range v.Components {
			if !nameRe.MatchString(c.ID) {
				return fmt.Errorf("view %q component %q has an invalid id", v.Name, c.ID)
			}
			if _, ok := seenComponent[c.ID]; ok {
				return fmt.Errorf("view %q: duplicate component %q", v.Name, c.ID)
			}
			seenComponent[c.ID] = struct{}{}
			if c.Span < 0 {
				return fmt.Errorf("view %q component %q has invalid span", v.Name, c.ID)
			}
			if c.Size != "" && !layoutSizeRe.MatchString(c.Size) {
				return fmt.Errorf("view %q component %q has invalid size %q", v.Name, c.ID, c.Size)
			}
			if err := validateDynamicView(c.Layout(v.Name), models); err != nil {
				return err
			}
		}
		if err := validateLayout(v.Name, v.ResolvedLayout(), seenComponent); err != nil {
			return err
		}
		if !v.HasLayout() {
			continue
		}
		if err := validateDynamicView(v, models); err != nil {
			return err
		}
	}
	return nil
}

func validateDynamicView(v ViewSpec, models map[string]ModelSpec) error {
	switch v.Kind {
	case "collection", "table", "kanban", "chart", "form":
	default:
		return fmt.Errorf("view %q has unsupported kind %q", v.Name, v.Kind)
	}
	model, ok := models[v.Model]
	if !ok {
		return fmt.Errorf("view %q references unknown model %q", v.Name, v.Model)
	}
	fields := map[string]struct{}{}
	for _, f := range model.Fields {
		fields[f.Name] = struct{}{}
	}
	for _, audit := range []string{"id", "updatedAt", "createdAt", "authorId"} {
		fields[audit] = struct{}{}
	}
	if err := validateViewColumns(v.Name, v.Columns, fields); err != nil {
		return err
	}
	if err := validateFilterPresets(v.Name, v.Filters, fields); err != nil {
		return fmt.Errorf("view %q: %w", v.Name, err)
	}
	for _, g := range v.Groups {
		if strings.TrimSpace(g.Label) == "" && strings.TrimSpace(g.LabelKey) == "" {
			return fmt.Errorf("view %q group requires label or labelKey", v.Name)
		}
		if g.Columns < 0 {
			return fmt.Errorf("view %q group has invalid columns", v.Name)
		}
		if err := validateViewColumns(v.Name, g.Fields, fields); err != nil {
			return err
		}
	}
	allowed := map[string]struct{}{"table": {}, "kanban": {}, "chart": {}, "pivot": {}}
	for _, mode := range v.Views {
		if _, ok := allowed[mode]; !ok {
			return fmt.Errorf("view %q has unsupported views entry %q", v.Name, mode)
		}
	}
	if v.DefaultView != "" {
		if _, ok := allowed[v.DefaultView]; !ok {
			return fmt.Errorf("view %q has unsupported defaultView %q", v.Name, v.DefaultView)
		}
		if len(v.Views) > 0 {
			found := false
			for _, mode := range v.Views {
				if mode == v.DefaultView {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("view %q defaultView %q is not in views", v.Name, v.DefaultView)
			}
		}
	}
	for _, g := range v.GroupBy {
		if _, ok := fields[g]; !ok {
			return fmt.Errorf("view %q groupBy references unknown field %q", v.Name, g)
		}
	}
	return nil
}

func validateViewColumns(view string, cols []ViewColumnSpec, fields map[string]struct{}) error {
	for _, c := range cols {
		if _, ok := fields[c.Field]; !ok {
			return fmt.Errorf("view %q references unknown field %q", view, c.Field)
		}
		if c.Span < 0 {
			return fmt.Errorf("view %q field %q has invalid span", view, c.Field)
		}
		switch c.Align {
		case "", "left", "center", "right":
		default:
			return fmt.Errorf("view %q field %q has invalid align %q", view, c.Field, c.Align)
		}
	}
	return nil
}

func viewNames(views []ViewSpec) map[string]struct{} {
	out := make(map[string]struct{}, len(views))
	for _, v := range views {
		out[v.Name] = struct{}{}
	}
	return out
}

func pageViewNames(views []ViewSpec) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range views {
		if v.IsPage() {
			out[v.Name] = struct{}{}
		}
	}
	return out
}

func declaredViews(views []ViewSpec) []ViewSpec {
	var out []ViewSpec
	for _, v := range views {
		if strings.TrimSpace(v.Path) != "" || strings.TrimSpace(v.Kind) != "" || len(v.Components) > 0 {
			out = append(out, v.forMarshal())
		}
	}
	return out
}

// resolveViewFile resolves a views path relative to the app root.
// Accepts "views/Name.page.tsx", "Name.page.tsx", or "Name".
func resolveViewFile(appRoot, ref string) (string, error) {
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	if ref == "" {
		return "", fmt.Errorf("path is empty")
	}
	candidates := []string{ref}
	if !strings.Contains(ref, "/") && !strings.HasSuffix(ref, PageSuffix) {
		candidates = append(candidates, "views/"+ref+PageSuffix, ref+PageSuffix)
	} else if !strings.HasPrefix(ref, "views/") && strings.HasSuffix(ref, PageSuffix) {
		candidates = append(candidates, "views/"+ref)
	}
	var last string
	for _, c := range candidates {
		abs := c
		if !filepath.IsAbs(c) {
			abs = filepath.Join(appRoot, filepath.FromSlash(c))
		}
		last = displayPath(appRoot, abs)
		info, err := os.Stat(abs)
		if err == nil && !info.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf("path %q not found (tried %s)", ref, last)
}

func mergeViews(appRoot string, declared, discovered []ViewSpec) ([]ViewSpec, error) {
	byName := map[string]ViewSpec{}
	var order []string
	for _, v := range declared {
		if path := strings.TrimSpace(v.Path); path != "" {
			abs, err := resolveViewFile(appRoot, path)
			if err != nil {
				return nil, fmt.Errorf("view %q: %w", v.Name, err)
			}
			rel, err := filepath.Rel(appRoot, abs)
			if err != nil {
				rel = path
			}
			v.Path = filepath.ToSlash(rel)
			v.Type = "page"
		}
		if _, ok := byName[v.Name]; ok {
			return nil, fmt.Errorf("duplicate view %q", v.Name)
		}
		byName[v.Name] = v
		order = append(order, v.Name)
	}
	for _, p := range discovered {
		if _, ok := byName[p.Name]; ok {
			continue
		}
		byName[p.Name] = p
		order = append(order, p.Name)
	}
	out := make([]ViewSpec, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out, nil
}
