package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"kaizengo/internal/module"
	"kaizengo/internal/platform/i18n"
	"kaizengo/packages/sdk-go/appspec"
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

// RegisterSpecLocales records locale metadata and inline messages from a domain document.
// Compiled apps still load .po catalogs from disk; this covers catalogs shipped in the spec.
func RegisterSpecLocales(spec appspec.AppSpec) {
	for _, l := range spec.Locales {
		dir := i18n.LTR
		if l.Dir == "rtl" {
			dir = i18n.RTL
		}
		if len(l.Messages) > 0 {
			i18n.Register(l.ID, l.Messages)
		}
		if i18n.Has(l.ID) {
			continue
		}
		name := l.Name
		if name == "" {
			name = l.ID
		}
		i18n.RegisterLocale(i18n.LocaleInfo{ID: l.ID, Name: name, Dir: dir})
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
	RegisterShellNavResource(appName, appspec.AppSpec{Name: appName, Title: appName, Nav: appspec.NavSpec{Route: route}})
}
