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
	pattern  string
	priority int
	fn       func(Context) error
}

var (
	mu       sync.RWMutex
	handlers []handler
)

// Register adds a global extension handler for point pattern.
// Patterns use dot segments; * matches one segment (e.g. model.*.*.afterCreate).
func Register(pattern string, priority int, fn func(Context) error) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	handlers = append(handlers, handler{pattern: pattern, priority: priority, fn: fn})
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
	for _, h := range handlers {
		if !matchPattern(h.pattern, point) {
			continue
		}
		if !allowWildcard(h.pattern, ctx.App) {
			continue
		}
		out = append(out, h)
	}
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
