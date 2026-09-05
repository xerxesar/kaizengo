package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "kaizengo/apps"
	authsvc "kaizengo/apps/auth"
	"kaizengo/internal/auth"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	_ "kaizengo/internal/platform/drivers"
	"kaizengo/internal/platform/postgres"
	"kaizengo/internal/app"
	"kaizengo/internal/engine"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// HTTP router with request logging and panic recovery.
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Module host: shared registry for services, routes, and lifecycle.
	host := module.NewHost(r, slog.Default())

	// Postgres: connect, expose to modules, and inject into request context.
	db, err := postgres.Connect(context.Background(), config.PostgresDSN())
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	postgres.Attach(host, db)
	r.Use(postgres.Middleware(db))

	// Session auth: resolve the auth app's service and validate cookies per request.
	r.Use(auth.SessionMiddleware(func(sessionID string) (*auth.Principal, error) {
		raw, ok := host.Lookup(authsvc.Name)
		if !ok {
			return nil, auth.ErrUnauthenticated
		}
		svc, ok := raw.(*authsvc.Service)
		if !ok {
			return nil, auth.ErrUnauthenticated
		}
		return svc.ValidateSession(sessionID)
	}))

	// App engine: persistent store of installed apps + manager wired into the host.
	store, err := app.OpenInstalledStore(context.Background(), db.Pool())
	if err != nil {
		log.Fatalf("installed apps: %v", err)
	}
	mgr := engine.NewManager(host, module.Default, store)
	host.Provide(engine.ManagerKey, mgr)

	// Resolve KaizenGo_APPS, load those modules, then sync the installed-apps table.
	wanted, err := mgr.Wanted(context.Background(), module.ParseAppList(os.Getenv("KaizenGo_APPS")))
	if err != nil {
		log.Fatalf("resolve apps: %v", err)
	}
	if err := module.Load(host, module.Default, wanted); err != nil {
		log.Fatal(err)
	}
	if err := mgr.SyncLoaded(context.Background()); err != nil {
		log.Fatalf("sync installed apps: %v", err)
	}

	// Platform route listing loaded apps.
	r.Get("/apps", module.AppsHandler(host))

	// Serve HTTP and shut down cleanly on SIGINT/SIGTERM.
	addr := envOr("ADDR", ":8080")
	log.Printf("listening on %s (apps: %v)", addr, appNames(host))
	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = module.Shutdown(ctx, host)
		_ = srv.Shutdown(ctx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func appNames(host *module.Host) []string {
	names := make([]string, len(host.Loaded))
	for i, m := range host.Loaded {
		names[i] = m.Name
	}
	return names
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
