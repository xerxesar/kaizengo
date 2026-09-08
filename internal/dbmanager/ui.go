package dbmanager

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func serveManagerUI(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/manager.html")
	if err != nil {
		http.Error(w, "manager UI missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

// StaticFS exposes embedded assets for tests.
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return staticFS
	}
	return sub
}
