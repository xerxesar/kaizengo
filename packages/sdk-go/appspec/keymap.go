package appspec

import (
	"fmt"
	"strings"
)

// KeymapSpec is the merged declarative keyboard shortcut seed for an app.
type KeymapSpec struct {
	Disable  []string
	Bindings []KeymapBindingSpec
}

// Empty reports whether the spec has nothing to apply.
func (k KeymapSpec) Empty() bool {
	return len(k.Disable) == 0 && len(k.Bindings) == 0
}

// KeymapBindingSpec maps a hotkey to a UI action id.
//
// Actions are resolved on the client:
//   - shell.*     built-in shell commands
//   - nav.<app>   navigate to an app route
//   - element:<id> click [data-keymap-id="<id>"]
//   - custom:<name> registered handler via useKeymapAction
type KeymapBindingSpec struct {
	ID       string `yaml:"id"`
	Action   string `yaml:"action"`
	Keys     string `yaml:"keys"`
	Label    string `yaml:"label"`
	LabelKey string `yaml:"labelKey"`
	Scope    string `yaml:"scope"` // global | app | view (default global)
	Hint     *bool  `yaml:"hint"`  // show badge when Alt is held (default true)
	InForm   *bool  `yaml:"inForm"` // allow shortcut while focus is in form fields
}

type fileKeymap struct {
	Disable  []string
	Bindings []KeymapBindingSpec `yaml:"bindings"`
}

func (k KeymapSpec) validate(appName string) error {
	seenDisable := map[string]struct{}{}
	for i, id := range k.Disable {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("keymap.disable[%d]: empty id", i)
		}
		if _, ok := seenDisable[id]; ok {
			return fmt.Errorf("keymap.disable: duplicate %q", id)
		}
		seenDisable[id] = struct{}{}
	}

	seen := map[string]struct{}{}
	for i, b := range k.Bindings {
		loc := fmt.Sprintf("keymap.bindings[%d]", i)
		id := strings.TrimSpace(b.ID)
		if id == "" {
			return fmt.Errorf("%s: id is required", loc)
		}
		if !nameRe.MatchString(id) {
			return fmt.Errorf("%s: invalid id %q", loc, id)
		}
		key := appName + "." + id
		if _, ok := seen[key]; ok {
			return fmt.Errorf("keymap.bindings: duplicate id %q", id)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(b.Action) == "" {
			return fmt.Errorf("%s: action is required", loc)
		}
		if strings.TrimSpace(b.Keys) == "" {
			return fmt.Errorf("%s: keys is required", loc)
		}
		scope := strings.TrimSpace(b.Scope)
		if scope == "" {
			scope = "global"
		}
		switch scope {
		case "global", "app", "view":
		default:
			return fmt.Errorf("%s: scope must be global, app, or view", loc)
		}
	}
	return nil
}

func mergeKeymap(dst *KeymapSpec, src fileKeymap) {
	dst.Disable = append(dst.Disable, src.Disable...)
	dst.Bindings = append(dst.Bindings, src.Bindings...)
}
