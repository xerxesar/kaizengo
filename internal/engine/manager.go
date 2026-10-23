package engine

import (
	"context"
	"fmt"
	"sync"

	"kaizengo/internal/module"
	"kaizengo/internal/app"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/internal/extension"
)

// ManagerKey is the host bag key for the app install manager.
const ManagerKey = "module.apps"

// AppInfo is one compiled-in app and its install state.
type AppInfo struct {
	Name             string
	Title            string
	Summary          string
	Version          string
	InstalledVersion string
	Depends          []string
	Installed        bool
	Loaded           bool
	AutoInstall      bool
	Upgrade          bool
}

// Manager installs and upgrades compiled-in apps (migrations + Setup/Mount).
type Manager struct {
	host  *module.Host
	reg   *module.Registry
	store *app.InstalledStore
	mu    sync.Mutex
}

// NewManager wires the install store to the process registry.
func NewManager(host *module.Host, reg *module.Registry, store *app.InstalledStore) *Manager {
	if reg == nil {
		reg = module.Default
	}
	return &Manager{host: host, reg: reg, store: store}
}

// ManagerFromHost returns the install manager provided at process start.
func ManagerFromHost(host *module.Host) (*Manager, error) {
	if host == nil {
		return nil, fmt.Errorf("host is nil")
	}
	raw, ok := host.Lookup(ManagerKey)
	if !ok {
		return nil, fmt.Errorf("%s not registered", ManagerKey)
	}
	mgr, ok := raw.(*Manager)
	if !ok || mgr == nil {
		return nil, fmt.Errorf("%s has unexpected type %T", ManagerKey, raw)
	}
	return mgr, nil
}

// Wanted returns the app names to load. A non-empty env list wins (dev override).
// Otherwise: rows in installed_apps plus autoInstall specs.
// Apps that are no longer on disk (apps/<name>/app.yaml) are dropped from the
// install table; only compiled-in apps that still exist are loaded.
func (m *Manager) Wanted(ctx context.Context, envList []string) ([]string, error) {
	present, err := appspec.ListNames()
	if err != nil {
		return nil, err
	}
	presentSet := toSet(present)
	known := func(name string) bool {
		if _, ok := presentSet[name]; !ok {
			return false
		}
		_, ok := m.reg.Get(name)
		return ok
	}

	if len(envList) > 0 {
		seen := map[string]struct{}{}
		var out []string
		for _, name := range envList {
			if !known(name) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("none of the requested apps exist under apps/")
		}
		return out, nil
	}

	installed, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	var installedNames []string
	for _, rec := range installed {
		installedNames = append(installedNames, rec.Name)
	}
	keep, drop := splitInstalled(installedNames, presentSet)
	for _, name := range drop {
		if err := m.store.Delete(ctx, name); err != nil {
			return nil, err
		}
		if m.host != nil && m.host.Log != nil {
			m.host.Log.Info("removed missing app from install list", "name", name)
		}
	}

	seen := map[string]struct{}{}
	var wanted []string
	add := func(name string) {
		if !known(name) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		wanted = append(wanted, name)
	}
	for _, name := range keep {
		add(name)
	}
	for _, name := range present {
		if specAutoInstall(name) {
			add(name)
		}
	}
	return wanted, nil
}

// SyncLoaded stamps auto-installed apps that are not yet in the table.
// Existing rows keep their version so Upgrade stays visible after a code bump.
func (m *Manager) SyncLoaded(ctx context.Context) error {
	for _, mf := range m.host.Loaded {
		rec, err := m.store.Get(ctx, mf.Name)
		if err != nil {
			return err
		}
		if rec == nil {
			if err := m.store.Upsert(ctx, mf.Name, mf.Version); err != nil {
				return err
			}
		}
	}
	return nil
}

// Apps lists every compiled-in app and its install state.
func (m *Manager) Apps(ctx context.Context) ([]AppInfo, error) {
	installed, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	byName := map[string]app.InstalledApp{}
	for _, rec := range installed {
		byName[rec.Name] = rec
	}
	present, err := appspec.ListNames()
	if err != nil {
		return nil, err
	}
	presentSet := toSet(present)
	var out []AppInfo
	for _, name := range m.reg.Names() {
		if _, ok := presentSet[name]; !ok {
			continue
		}
		a, ok := m.reg.Get(name)
		if !ok {
			continue
		}
		mf := a.Manifest()
		title, summary := mf.Name, mf.Summary
		auto := false
		depends := mf.Depends
		if spec, err := appspec.LoadApp(name); err == nil {
			if spec.Title != "" {
				title = spec.Title
			}
			if spec.Summary != "" {
				summary = spec.Summary
			}
			auto = spec.AutoInstall
			if len(spec.Depends) > 0 {
				depends = spec.Depends
			}
		}
		rec, ok := byName[name]
		info := AppInfo{
			Name:        name,
			Title:       title,
			Summary:     summary,
			Version:     mf.Version,
			Depends:     depends,
			Installed:   ok,
			Loaded:      m.loaded(name),
			AutoInstall: auto,
		}
		if ok {
			info.InstalledVersion = rec.Version
			info.Upgrade = module.VersionLess(rec.Version, mf.Version)
		}
		out = append(out, info)
	}
	return out, nil
}

// Install runs migrations, Setup, and Mount for name and its missing depends.
func (m *Manager) Install(ctx context.Context, name string) (*AppInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.installLocked(ctx, name)
}

// Upgrade applies pending migrations and stamps the code version.
func (m *Manager) Upgrade(ctx context.Context, name string) (*AppInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, err := m.store.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("app %q is not installed", name)
	}
	if !m.loaded(name) {
		return m.installLocked(ctx, name)
	}
	if err := app.Migrate(m.host, name); err != nil {
		return nil, fmt.Errorf("migrate %s: %w", name, err)
	}
	mf := mustManifest(m.reg, name)
	if err := m.store.Upsert(ctx, name, mf.Version); err != nil {
		return nil, err
	}
	m.host.Log.Info("upgrade app", "name", name, "version", mf.Version)
	return m.info(ctx, name)
}

func (m *Manager) installLocked(ctx context.Context, name string) (*AppInfo, error) {
	if _, ok := m.reg.Get(name); !ok {
		return nil, fmt.Errorf("app %q is not compiled in", name)
	}
	if m.loaded(name) {
		mf := mustManifest(m.reg, name)
		if err := m.store.Upsert(ctx, name, mf.Version); err != nil {
			return nil, err
		}
		return m.info(ctx, name)
	}
	apps, err := m.reg.Resolve([]string{name})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, a := range apps {
		names = append(names, a.Manifest().Name)
	}
	for _, mf := range m.host.Loaded {
		names = append(names, mf.Name)
	}
	if err := appspec.ValidateLoadedCapabilities(unique(names)); err != nil {
		return nil, err
	}
	for _, a := range apps {
		mf := a.Manifest()
		if m.loaded(mf.Name) {
			continue
		}
		spec, err := appspec.LoadApp(mf.Name)
		if err == nil {
			extension.ApplyExports(spec)
			if err := extension.ApplyExtends(spec); err != nil {
				return nil, err
			}
		}
		m.host.Log.Info("install app", "name", mf.Name, "version", mf.Version)
		if err := a.Setup(m.host); err != nil {
			return nil, fmt.Errorf("setup %s: %w", mf.Name, err)
		}
		if err := a.Mount(m.host); err != nil {
			return nil, fmt.Errorf("mount %s: %w", mf.Name, err)
		}
		m.host.Loaded = append(m.host.Loaded, mf)
		if err := m.store.Upsert(ctx, mf.Name, mf.Version); err != nil {
			return nil, err
		}
	}
	m.host.GQL.Invalidate()
	return m.info(ctx, name)
}

func (m *Manager) loaded(name string) bool {
	for _, mf := range m.host.Loaded {
		if mf.Name == name {
			return true
		}
	}
	return false
}

func (m *Manager) info(ctx context.Context, name string) (*AppInfo, error) {
	all, err := m.Apps(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].Name == name {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("unknown app %q", name)
}

func specAutoInstall(name string) bool {
	spec, err := appspec.LoadApp(name)
	return err == nil && spec.AutoInstall
}

func mustManifest(reg *module.Registry, name string) module.Manifest {
	a, ok := reg.Get(name)
	if !ok {
		return module.Manifest{Name: name, Version: "0.0.0"}
	}
	return a.Manifest()
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func toSet(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}
	return out
}

// splitInstalled keeps install rows whose apps still exist on disk.
func splitInstalled(installed []string, present map[string]struct{}) (keep, drop []string) {
	for _, name := range installed {
		if _, ok := present[name]; ok {
			keep = append(keep, name)
			continue
		}
		drop = append(drop, name)
	}
	return keep, drop
}
