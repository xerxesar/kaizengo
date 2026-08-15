package module

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
)

// Health responds with a plain-text ok for load balancers and probes.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

// SPA serves a Vite-built SPA from dir, with index.html fallback for client routes.
func SPA(dir string) http.Handler {
	root := http.Dir(dir)
	fileServer := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" || upath == "." {
			serveIndex(w, r, dir)
			return
		}

		f, err := root.Open(upath)
		if err != nil {
			if os.IsNotExist(err) {
				serveIndex(w, r, dir)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			if _, err := root.Open(path.Join(upath, "index.html")); err != nil {
				serveIndex(w, r, dir)
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, dir string) {
	http.ServeFile(w, r, path.Join(dir, "index.html"))
}
