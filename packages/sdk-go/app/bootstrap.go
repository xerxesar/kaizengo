package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/i18n"
)

func Env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func MustMkdirForFile(path string) {
	dir := filepath.Dir(path)
	if dir == "." || dir == "/" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
}

func LoadLocales(appName string) error {
	return i18n.LoadLocaleDir(filepath.Join("apps", appName, "locale"))
}

func MustLoadLocales(appName string) {
	if err := LoadLocales(appName); err != nil {
		panic(err)
	}
}

func LoadLocalesFS(fsys fs.FS, root string) error {
	return i18n.LoadLocaleFS(fsys, root)
}

func RegisterNav(host *module.Host, appName, route string) {
	if route == "" {
		route = strings.ReplaceAll(appName, "_", "-")
	}
	host.RegisterNav(module.NavEntry{
		ID:       appName,
		TitleKey: "nav." + appName,
		Route:    route,
	})
}

