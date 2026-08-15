package extension

import (
	"fmt"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
)

var (
	namedMu sync.RWMutex
	named   = map[string]func(Context) error{}
)

// RegisterNamed registers a handler referenced from app.yaml extends entries.
func RegisterNamed(name string, fn func(Context) error) {
	if name == "" || fn == nil {
		return
	}
	namedMu.Lock()
	defer namedMu.Unlock()
	named[name] = fn
}

// LookupNamed returns a registered named handler.
func LookupNamed(name string) (func(Context) error, bool) {
	namedMu.RLock()
	defer namedMu.RUnlock()
	fn, ok := named[name]
	return fn, ok
}

// ApplyExtends wires yaml extends entries to the global extension registry.
func ApplyExtends(spec appspec.AppSpec) error {
	for _, e := range spec.Extends {
		fn, ok := LookupNamed(e.Handler)
		if !ok {
			return fmt.Errorf("app %q extends handler %q is not registered", spec.Name, e.Handler)
		}
		priority := e.Priority
		if priority == 0 {
			priority = 100
		}
		Register(e.Point, priority, fn)
	}
	return nil
}
