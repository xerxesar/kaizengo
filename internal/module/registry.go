package module

import (
	"fmt"
	"sort"
)

// Manifest describes a loadable app (Odoo-style metadata).
type Manifest struct {
	Name        string
	Version     string
	Summary     string
	Depends     []string
	Installable bool
}

// App is a loadable application package.
type App interface {
	Manifest() Manifest
	Setup(host *Host) error
	Mount(host *Host) error
}

// Registry holds all compiled-in apps.
type Registry struct {
	byName map[string]App
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]App)}
}

func (r *Registry) Register(a App) {
	name := a.Manifest().Name
	if name == "" {
		panic("module: empty manifest name")
	}
	if _, ok := r.byName[name]; ok {
		panic(fmt.Sprintf("module: duplicate registration: %s", name))
	}
	r.byName[name] = a
}

func (r *Registry) Get(name string) (App, bool) {
	a, ok := r.byName[name]
	return a, ok
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for n := range r.byName {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Resolve returns apps in dependency order.
// If wanted is empty, all installable apps are selected (plus their deps).
func (r *Registry) Resolve(wanted []string) ([]App, error) {
	if len(wanted) == 0 {
		for _, name := range r.Names() {
			a := r.byName[name]
			if a.Manifest().Installable {
				wanted = append(wanted, name)
			}
		}
	}

	selected := make(map[string]App)
	var visit func(string) error
	visit = func(name string) error {
		if _, ok := selected[name]; ok {
			return nil
		}
		a, ok := r.byName[name]
		if !ok {
			return fmt.Errorf("app %q not found", name)
		}
		for _, dep := range a.Manifest().Depends {
			if err := visit(dep); err != nil {
				return fmt.Errorf("%s depends on %s: %w", name, dep, err)
			}
		}
		selected[name] = a
		return nil
	}

	for _, name := range wanted {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return topoSort(selected)
}

func topoSort(selected map[string]App) ([]App, error) {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(selected))
	order := make([]App, 0, len(selected))

	var dfs func(string) error
	dfs = func(name string) error {
		switch color[name] {
		case gray:
			return fmt.Errorf("dependency cycle involving %q", name)
		case black:
			return nil
		}
		color[name] = gray
		a := selected[name]
		for _, dep := range a.Manifest().Depends {
			if _, ok := selected[dep]; !ok {
				continue
			}
			if err := dfs(dep); err != nil {
				return err
			}
		}
		color[name] = black
		order = append(order, a)
		return nil
	}

	names := make([]string, 0, len(selected))
	for n := range selected {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := dfs(n); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// Default is the process-wide app registry. Apps register in init().
var Default = NewRegistry()

func Register(a App) { Default.Register(a) }
