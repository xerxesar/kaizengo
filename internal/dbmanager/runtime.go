package dbmanager

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"kaizengo/internal/module"
)

const ClientDBCookie = "kg_db"

// Platform is a fully built application stack for one active database.
type Platform struct {
	Handler http.Handler
	Host    *module.Host
}

// PlatformBuilder builds (or rebuilds) the full platform for a DSN.
type PlatformBuilder func(ctx context.Context, dsn string) (*Platform, error)

// SwapHandler atomically swaps the active platform HTTP handler.
type SwapHandler struct {
	mu sync.RWMutex
	h  http.Handler
}

func NewSwapHandler(h http.Handler) *SwapHandler {
	if h == nil {
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "platform not ready", http.StatusServiceUnavailable)
		})
	}
	return &SwapHandler{h: h}
}

func (s *SwapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	h := s.h
	s.mu.RUnlock()
	h.ServeHTTP(w, r)
}

func (s *SwapHandler) Set(h http.Handler) {
	s.mu.Lock()
	s.h = h
	s.mu.Unlock()
}

// BootstrapHandler redirects browser traffic to the database manager.
func BootstrapHandler() http.Handler {
	redir := http.RedirectHandler("/web/database/manager", http.StatusFound)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redir.ServeHTTP(w, r)
	})
}

// Runtime owns the catalog, swap handler, and platform rebuild lifecycle.
// The active database is client-driven (cookie /use), not persisted in the catalog.
type Runtime struct {
	Store    *Store
	Swap     *SwapHandler
	Build    PlatformBuilder
	Admin    func() *Admin
	sessions *sessionStore

	mu     sync.Mutex
	plat   *Platform
	active string
}

// NewRuntime wires catalog + swap + platform builder.
func NewRuntime(store *Store, swap *SwapHandler, build PlatformBuilder) *Runtime {
	rt := &Runtime{
		Store:    store,
		Swap:     swap,
		Build:    build,
		sessions: newSessionStore(),
	}
	rt.Admin = func() *Admin {
		dsn, err := store.AdminDSN()
		if err != nil {
			return NewAdmin("")
		}
		return NewAdmin(dsn)
	}
	return rt
}

// ActiveName returns the database currently serving the platform (empty if bootstrap).
func (rt *Runtime) ActiveName() string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.active
}

// CurrentHost returns the active platform host, if any.
func (rt *Runtime) CurrentHost() *module.Host {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.plat == nil {
		return nil
	}
	return rt.plat.Host
}

// EnterBootstrap shuts down the platform and serves only the DB manager.
func (rt *Runtime) EnterBootstrap() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	old := rt.plat
	rt.plat = nil
	rt.active = ""
	rt.Swap.Set(BootstrapHandler())
	if old != nil && old.Host != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = module.Shutdown(shutdownCtx, old.Host)
	}
}

// UseDatabase activates a catalog DB for this process without writing catalog selection.
func (rt *Runtime) UseDatabase(ctx context.Context, name string) error {
	if err := ValidateDBName(name); err != nil {
		return err
	}
	if !rt.Store.HasDatabase(name) {
		return errNotRegistered(name)
	}
	rt.mu.Lock()
	if rt.active == name && rt.plat != nil {
		rt.mu.Unlock()
		return nil
	}
	rt.mu.Unlock()

	dsn, err := rt.Store.DatabaseDSN(name)
	if err != nil {
		return err
	}
	return rt.rebuild(ctx, dsn, name, true)
}

// EnsureFromRequest activates the DB named in the kg_db cookie when present.
func (rt *Runtime) EnsureFromRequest(ctx context.Context, r *http.Request) error {
	name := ClientDBFromRequest(r)
	if name == "" {
		return nil
	}
	if !rt.Store.HasDatabase(name) {
		return nil
	}
	return rt.UseDatabase(ctx, name)
}

// ClientDBFromRequest reads the client DB cookie.
func ClientDBFromRequest(r *http.Request) string {
	c, err := r.Cookie(ClientDBCookie)
	if err != nil || c == nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

// SetClientDBCookie writes the client DB preference cookie.
func SetClientDBCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     ClientDBCookie,
		Value:    name,
		Path:     "/",
		HttpOnly: false, // SPA also mirrors this in localStorage
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 60 * 60,
	})
}

// ClearClientDBCookie clears the client DB preference cookie.
func ClearClientDBCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     ClientDBCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func errNotRegistered(name string) error {
	return &apiError{status: http.StatusBadRequest, msg: "database \"" + name + "\" not registered"}
}

func (rt *Runtime) rebuild(ctx context.Context, dsn, name string, keepOldOnFail bool) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	old := rt.plat
	oldActive := rt.active
	plat, err := rt.Build(ctx, dsn)
	if err != nil {
		if !keepOldOnFail || old == nil {
			rt.plat = nil
			rt.active = ""
			rt.Swap.Set(BootstrapHandler())
		}
		return err
	}
	rt.Swap.Set(plat.Handler)
	rt.plat = plat
	rt.active = name

	if old != nil && old.Host != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = module.Shutdown(shutdownCtx, old.Host)
		_ = oldActive
	}
	return nil
}

// CookieMiddleware activates the client-selected DB before non-manager requests.
func (rt *Runtime) CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/web/database") {
			_ = rt.EnsureFromRequest(r.Context(), r)
		}
		next.ServeHTTP(w, r)
	})
}
