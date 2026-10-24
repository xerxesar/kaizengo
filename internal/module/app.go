package module

import (
	"context"
	"log/slog"
	"sync"

	"github.com/go-chi/chi/v5"
)

// Host is the shared runtime bag for loaded apps (DB, clients, router, …).
type Host struct {
	Router *chi.Mux
	Log    *slog.Logger
	Loaded []Manifest

	servicesMu sync.RWMutex
	services   map[string]any
	nav        []NavEntry
	GQL        *GraphQLRegistry
	startup    []func(context.Context) error
	shutdown   []func(context.Context) error
}

func NewHost(router *chi.Mux, log *slog.Logger) *Host {
	if log == nil {
		log = slog.Default()
	}
	return &Host{
		Router:   router,
		Log:      log,
		services: make(map[string]any),
		GQL:      newGraphQLRegistry(),
	}
}

// OnStart registers a lifecycle hook run after all app Setup phases.
func (h *Host) OnStart(fn func(context.Context) error) {
	h.startup = append(h.startup, fn)
}

// OnStop registers a lifecycle hook for graceful shutdown.
func (h *Host) OnStop(fn func(context.Context) error) {
	h.shutdown = append(h.shutdown, fn)
}

// Provide registers a named service for other apps.
func (h *Host) Provide(name string, svc any) {
	h.servicesMu.Lock()
	defer h.servicesMu.Unlock()
	h.services[name] = svc
}

// Unprovide removes a named service (dynamic apps on delete).
func (h *Host) Unprovide(name string) {
	h.servicesMu.Lock()
	defer h.servicesMu.Unlock()
	delete(h.services, name)
}

// Lookup returns a service registered with Provide.
func (h *Host) Lookup(name string) (any, bool) {
	h.servicesMu.RLock()
	defer h.servicesMu.RUnlock()
	svc, ok := h.services[name]
	return svc, ok
}
