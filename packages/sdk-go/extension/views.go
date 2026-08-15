package extension

import (
	"strings"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
)

// ViewSlot declares a component injection into an app view.
type ViewSlot struct {
	SourceApp string
	TargetApp string
	ViewMatch string
	Slot      string
	Component string
	Module    string
	ID        string
	LabelKey  string
	Label     string
}

type viewSlotEntry struct {
	slot ViewSlot
}

var (
	viewMu     sync.RWMutex
	viewSlots  []viewSlotEntry
	components = map[string]string{}
)

// RegisterComponentExport records a component id exported by an app.
func RegisterComponentExport(sourceApp, id, module string) {
	if id == "" || module == "" {
		return
	}
	viewMu.Lock()
	defer viewMu.Unlock()
	components[id] = module
}

// RegisterViewSlot adds a view slot injection from an addon app.
func RegisterViewSlot(sourceApp string, spec appspec.ViewExtendSpec) {
	if spec.App == "" || spec.Match == "" || spec.Slot == "" || spec.Component == "" {
		return
	}
	id := strings.TrimSpace(spec.ID)
	if id == "" {
		id = defaultSlotID(spec.Component)
	}
	viewMu.Lock()
	defer viewMu.Unlock()
	viewSlots = append(viewSlots, viewSlotEntry{
		slot: ViewSlot{
			SourceApp: sourceApp,
			TargetApp: spec.App,
			ViewMatch: spec.Match,
			Slot:      spec.Slot,
			Component: spec.Component,
			ID:        id,
			LabelKey:  strings.TrimSpace(spec.LabelKey),
			Label:     strings.TrimSpace(spec.Label),
		},
	})
}

func defaultSlotID(component string) string {
	component = strings.TrimSpace(component)
	if i := strings.LastIndex(component, "."); i >= 0 && i+1 < len(component) {
		return strings.ToLower(component[i+1:])
	}
	return strings.ToLower(component)
}

// ApplyExports registers component exports, view slots, and menu contributions from spec.
func ApplyExports(spec appspec.AppSpec) {
	for _, c := range spec.Exports.Components {
		RegisterComponentExport(spec.Name, c.ID, c.Module)
	}
	for _, v := range spec.Exports.Views {
		RegisterViewSlot(spec.Name, v)
	}
	for _, m := range spec.Exports.Menus {
		RegisterMenuContribution(spec.Name, m)
	}
}

// ViewSlotsFor returns matching slots for a target app view.
func ViewSlotsFor(app, view string) []ViewSlot {
	viewMu.RLock()
	defer viewMu.RUnlock()
	out := make([]ViewSlot, 0)
	for _, e := range viewSlots {
		if e.slot.TargetApp != app {
			continue
		}
		if !matchViewPattern(e.slot.ViewMatch, view) {
			continue
		}
		slot := e.slot
		slot.Module = components[slot.Component]
		out = append(out, slot)
	}
	return out
}

func matchViewPattern(pattern, view string) bool {
	if pattern == view {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		v := strings.ToLower(view)
		s := strings.ToLower(suffix)
		return strings.HasSuffix(v, s)
	}
	return false
}
