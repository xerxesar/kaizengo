package module

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Load sets up and mounts apps from the registry in dependency order.
func Load(host *Host, reg *Registry, wanted []string) error {
	if reg == nil {
		reg = Default
	}
	apps, err := reg.Resolve(wanted)
	if err != nil {
		return err
	}

	host.Loaded = make([]Manifest, 0, len(apps))
	for _, a := range apps {
		mf := a.Manifest()
		host.Log.Info("setup app", "name", mf.Name, "version", mf.Version)
		if err := a.Setup(host); err != nil {
			return fmt.Errorf("setup %s: %w", mf.Name, err)
		}
		host.Loaded = append(host.Loaded, mf)
	}
	for _, a := range apps {
		mf := a.Manifest()
		host.Log.Info("mount app", "name", mf.Name)
		if err := a.Mount(host); err != nil {
			return fmt.Errorf("mount %s: %w", mf.Name, err)
		}
	}
	return nil
}

// ParseAppList splits KaizenGo_APPS / CLI list: "core,status".
func ParseAppList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// AppsHandler lists loaded apps.
func AppsHandler(host *Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, m := range host.Loaded {
			deps := strings.Join(m.Depends, ", ")
			if deps == "" {
				deps = "-"
			}
			fmt.Fprintf(w, "%s %s  depends=[%s]  %s\n", m.Name, m.Version, deps, m.Summary)
		}
	}
}

// NavHandler returns the core SPA apps menu catalog.
func NavHandler(host *Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(host.NavEntries())
	}
}
