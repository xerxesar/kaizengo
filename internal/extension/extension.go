package extension

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"kaizengo/packages/sdk-go/appspec"
)

// Context is passed to global extension handlers.
type Context struct {
	Context  context.Context
	Point    string
	App      appspec.AppSpec
	Model    appspec.ModelSpec
	OrgID    string
	UserID   string
	RecordID string
	Fields   map[string]any
	Record   map[string]any
}

type handler struct {
	id       uint64
	pattern  string
	priority int
	fn       func(Context) error
}

var (
	mu             sync.RWMutex
	baseHandlers   []handler // from init() Register calls — never cleared
	loadHandlers   []handler // from ApplyExtends — cleared on each module.Load
	scopedHandlers []handler // from RegisterScoped — removed by their owner
	nextScopedID   uint64
)

// Register adds a global extension handler for point pattern.
// Patterns use dot segments; * matches one segment (e.g. model.*.*.afterCreate).
// Handlers registered here survive platform rebuilds (database switches).
func Register(pattern string, priority int, fn func(Context) error) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	baseHandlers = append(baseHandlers, handler{pattern: pattern, priority: priority, fn: fn})
}

// RegisterScoped adds a handler like Register and returns a func that removes it.
// External Go apps use it so their hooks disappear when the plugin process stops.
func RegisterScoped(pattern string, priority int, fn func(Context) error) (remove func()) {
	if fn == nil {
		return func() {}
	}
	mu.Lock()
	nextScopedID++
	id := nextScopedID
	scopedHandlers = append(scopedHandlers, handler{id: id, pattern: pattern, priority: priority, fn: fn})
	mu.Unlock()
	return func() {
		mu.Lock()
		defer mu.Unlock()
		for i, h := range scopedHandlers {
			if h.id == id {
				scopedHandlers = append(scopedHandlers[:i:i], scopedHandlers[i+1:]...)
				return
			}
		}
	}
}

// registerLoad adds a handler that is cleared by ResetLoadState (yaml extends).
func registerLoad(pattern string, priority int, fn func(Context) error) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	loadHandlers = append(loadHandlers, handler{pattern: pattern, priority: priority, fn: fn})
}

// Run executes matching handlers in priority order.
// Before* phases stop on first error; After* phases log errors and continue.
func Run(point string, ctx Context, stopOnError bool) error {
	for _, h := range matching(point, ctx) {
		if err := h.fn(ctx); err != nil {
			if stopOnError {
				return err
			}
			slog.Warn("extension after-hook failed", "point", point, "pattern", h.pattern, "err", err)
		}
	}
	return nil
}

func matching(point string, ctx Context) []handler {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]handler, 0)
	appendMatching := func(list []handler) {
		for _, h := range list {
			if !matchPattern(h.pattern, point) {
				continue
			}
			if !allowWildcard(h.pattern, ctx.App) {
				continue
			}
			out = append(out, h)
		}
	}
	appendMatching(baseHandlers)
	appendMatching(loadHandlers)
	appendMatching(scopedHandlers)
	sortHandlers(out)
	return out
}

func allowWildcard(pattern string, app appspec.AppSpec) bool {
	if !strings.Contains(pattern, ".*") && !strings.Contains(pattern, "*.") {
		return true
	}
	return app.EnableExtensions
}

func matchPattern(pattern, point string) bool {
	p := strings.Split(pattern, ".")
	t := strings.Split(point, ".")
	if len(p) != len(t) {
		return false
	}
	for i := range p {
		if p[i] == "*" {
			continue
		}
		if p[i] != t[i] {
			return false
		}
	}
	return true
}

func sortHandlers(list []handler) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].priority < list[i].priority {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

// ModelPoint builds a model lifecycle extension point name.
func ModelPoint(app, model, phase string) string {
	return "model." + app + "." + model + "." + phase
}

func clearLoadHandlers() {
	mu.Lock()
	loadHandlers = nil
	mu.Unlock()
}
