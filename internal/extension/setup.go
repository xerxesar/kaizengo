package extension

import (
	"fmt"

	"kaizengo/packages/sdk-go/appspec"
)

// SetupAddon is called from addon App.Setup.
// Exports and extends are already applied in module.Load; this validates the addon spec.
func SetupAddon(spec appspec.AppSpec) error {
	return ValidateAddonSpec(spec)
}

// ValidateAddonSpec checks extends handlers and export component refs before startup wiring.
func ValidateAddonSpec(spec appspec.AppSpec) error {
	for _, e := range spec.Extends {
		if _, ok := LookupNamed(e.Handler); !ok {
			return fmt.Errorf("app %q extends handler %q is not registered", spec.Name, e.Handler)
		}
	}
	components := map[string]struct{}{}
	for _, c := range spec.Exports.Components {
		components[c.ID] = struct{}{}
	}
	for _, v := range spec.Exports.Views {
		if _, ok := components[v.Component]; !ok {
			return fmt.Errorf("app %q view extend references unknown component %q", spec.Name, v.Component)
		}
	}
	for _, m := range spec.Exports.Menus {
		if m.Component == "" {
			continue
		}
		if _, ok := components[m.Component]; !ok {
			return fmt.Errorf("app %q menu extend references unknown component %q", spec.Name, m.Component)
		}
	}
	return nil
}
