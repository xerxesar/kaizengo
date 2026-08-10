package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	_ "kaizengo/apps"
	_ "kaizengo/internal/platform/drivers"
	"kaizengo/internal/module"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	host := module.NewHost(r, slog.Default())
	wanted := module.ParseAppList(os.Getenv("KaizenGo_APPS"))
	if err := module.Load(host, module.Default, wanted); err != nil {
		log.Fatal(err)
	}

	r.Get("/apps", module.AppsHandler(host))

	addr := envOr("ADDR", ":8080")
	log.Printf("listening on %s (apps: %v)", addr, appNames(host))
	if err := http.ListenAndServe(addr, r); err != nil {
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
