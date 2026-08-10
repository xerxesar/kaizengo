package module

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
)

// Host is the shared runtime bag for loaded apps (DB, clients, router, …).
type Host struct {
	Router *chi.Mux
	Log    *slog.Logger
	Loaded []Manifest

	services map[string]any
	nav      []NavEntry
	GQL      *GraphQLRegistry
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

// Provide registers a named service for other apps.
func (h *Host) Provide(name string, svc any) {
	h.services[name] = svc
}

// Lookup returns a service registered with Provide.
func (h *Host) Lookup(name string) (any, bool) {
	svc, ok := h.services[name]
	return svc, ok
}
