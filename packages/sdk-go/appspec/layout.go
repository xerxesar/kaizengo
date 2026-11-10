package appspec

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

var layoutSizeRe = regexp.MustCompile(`^(?:auto|\d+(?:\.\d+)?(?:fr|%|px|rem|em|vw|vh))$`)

func (l *ViewLayoutSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var dir string
		if err := node.Decode(&dir); err != nil {
			return err
		}
		l.Direction = dir
		return nil
	}
	var plain struct {
		Direction string           `yaml:"direction,omitempty"`
		Component string           `yaml:"component,omitempty"`
		Size      string           `yaml:"size,omitempty"`
		Span      int              `yaml:"span,omitempty"`
		Items     []ViewLayoutSpec `yaml:"items,omitempty"`
	}
	if err := node.Decode(&plain); err != nil {
		return err
	}
	*l = ViewLayoutSpec{
		Direction: plain.Direction, Component: plain.Component,
		Size: plain.Size, Span: plain.Span, Items: plain.Items,
	}
	return nil
}

func (l ViewLayoutSpec) IsZero() bool {
	return l.Direction == "" && l.Component == "" && l.Size == "" && l.Span == 0 && len(l.Items) == 0
}

// ResolvedLayout fills a direction shorthand with one item per component.
func (v ViewSpec) ResolvedLayout() ViewLayoutSpec {
	layout := v.Layout
	if len(layout.Items) > 0 || layout.Component != "" {
		return layout
	}
	if layout.Direction == "" {
		layout.Direction = "column"
	}
	for _, c := range v.Components {
		layout.Items = append(layout.Items, ViewLayoutSpec{
			Component: c.ID, Span: c.Span, Size: c.Size,
		})
	}
	return layout
}

func validateLayout(view string, layout ViewLayoutSpec, ids map[string]struct{}) error {
	if layout.IsZero() {
		return nil
	}
	return walkLayout(view, layout, ids, map[string]struct{}{})
}

func walkLayout(view string, node ViewLayoutSpec, ids, used map[string]struct{}) error {
	if node.Size != "" && !layoutSizeRe.MatchString(node.Size) {
		return fmt.Errorf("view %q layout size %q is invalid", view, node.Size)
	}
	if node.Span < 0 {
		return fmt.Errorf("view %q layout span must be >= 0", view)
	}
	switch node.Direction {
	case "", "column", "row":
	default:
		return fmt.Errorf("view %q layout direction %q is invalid", view, node.Direction)
	}
	if node.Component != "" && len(node.Items) > 0 {
		return fmt.Errorf("view %q layout node %q cannot nest items", view, node.Component)
	}
	if node.Component != "" {
		if _, ok := ids[node.Component]; !ok {
			return fmt.Errorf("view %q layout references unknown component %q", view, node.Component)
		}
		if _, ok := used[node.Component]; ok {
			return fmt.Errorf("view %q layout repeats component %q", view, node.Component)
		}
		used[node.Component] = struct{}{}
		return nil
	}
	if len(node.Items) == 0 && node.Direction == "" && node.Size == "" && node.Span == 0 {
		return fmt.Errorf("view %q layout node is empty", view)
	}
	for _, child := range node.Items {
		if err := walkLayout(view, child, ids, used); err != nil {
			return err
		}
	}
	return nil
}
