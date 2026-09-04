package app

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
)

// LoadAppSpec reads apps/<name>/app.yaml.
func LoadAppSpec(name string) (appspec.AppSpec, error) {
	return appspec.LoadApp(name)
}

// MustAppSpec loads apps/<name>/app.yaml or panics.
func MustAppSpec(name string) appspec.AppSpec {
	spec, err := LoadAppSpec(name)
	if err != nil {
		panic("app spec " + name + ": " + err.Error())
	}
	return spec
}

// ManifestFromSpec builds a module manifest from a parsed app spec.
func ManifestFromSpec(spec appspec.AppSpec, version string) module.Manifest {
	if version == "" {
		version = "0.1.0"
	}
	return module.Manifest{
		Name:        spec.Name,
		Version:     version,
		Summary:     spec.Summary,
		Depends:     spec.Depends,
		Installable: true,
	}
}

// RegisterNavFromSpec adds a shell menu entry from spec.nav.
func RegisterNavFromSpec(host *module.Host, appName string, spec appspec.AppSpec) {
	nav := spec.Nav
	route := nav.Route
	if route == "" {
		route = appName
	}
	titleKey := nav.LabelKey
	title := nav.Label
	if titleKey == "" {
		titleKey = "nav." + appName
	}
	if title == "" {
		title = spec.Title
	}
	host.RegisterNav(module.NavEntry{
		ID:       appName,
		TitleKey: titleKey,
		Title:    title,
		Route:    route,
		Order:    nav.Order,
	})
	RegisterShellNavResource(appName, spec)
}
