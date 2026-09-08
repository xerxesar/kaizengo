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
	"kaizengo/internal/app"
	"kaizengo/internal/auth"
	"kaizengo/internal/dbmanager"
	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/config"
	_ "kaizengo/internal/platform/drivers"
	"kaizengo/internal/platform/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	store, err := dbmanager.OpenStore(config.DatabasesConfigPath())
	if err != nil {
		log.Fatalf("db catalog: %v", err)
	}

	root := chi.NewRouter()
	root.Use(middleware.Logger)
	root.Use(middleware.Recoverer)

	swap := dbmanager.NewSwapHandler(dbmanager.BootstrapHandler())
	rt := dbmanager.NewRuntime(store, swap, buildPlatform)
	root.Use(rt.CookieMiddleware)

	rt.Mount(root)
	// Unmatched paths (/, /app, /auth, …) go to the active platform or bootstrap.
	// Do not use Handle("/*") — it can shadow /web/database on some chi trees.
	root.NotFound(swap.ServeHTTP)
	root.MethodNotAllowed(swap.ServeHTTP)

	// Selection is client-driven; start in bootstrap until a client activates a DB.
	rt.EnterBootstrap()

	addr := envOr("ADDR", ":8080")
	log.Printf("listening on %s (database manager — client selects DB)", addr)

	srv := &http.Server{Addr: addr, Handler: root}
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if host := rt.CurrentHost(); host != nil {
			_ = module.Shutdown(shutdownCtx, host)
		}
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func buildPlatform(ctx context.Context, dsn string) (*dbmanager.Platform, error) {
	r := chi.NewRouter()
	host := module.NewHost(r, slog.Default())

	db, err := postgres.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	postgres.Attach(host, db)
	r.Use(postgres.Middleware(db))

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

	store, err := app.OpenInstalledStore(ctx, db.Pool())
	if err != nil {
		db.Close()
		return nil, err
	}
	mgr := engine.NewManager(host, module.Default, store)
	host.Provide(engine.ManagerKey, mgr)

	wanted, err := mgr.Wanted(ctx, module.ParseAppList(os.Getenv("KaizenGo_APPS")))
	if err != nil {
		db.Close()
		return nil, err
	}
	if err := module.Load(host, module.Default, wanted); err != nil {
		_ = module.Shutdown(ctx, host)
		return nil, err
	}
	if err := mgr.SyncLoaded(ctx); err != nil {
		_ = module.Shutdown(ctx, host)
		return nil, err
	}

	r.Get("/apps", module.AppsHandler(host))

	host.Log.Info("platform ready", "apps", appNames(host))
	return &dbmanager.Platform{Handler: r, Host: host}, nil
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
