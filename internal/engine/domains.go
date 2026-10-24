package engine

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"kaizengo/internal/app"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
	"gopkg.in/yaml.v3"
)

// MaxDomainSize bounds one domain document.
const MaxDomainSize = 256 << 10

// DomainsKey is the host bag key for the domain manager.
const DomainsKey = "engine.domains"

// Revision sources.
const (
	SourceGraphQL  = "graphql"
	SourceCompiled = "compiled"
	SourceRestore  = "restore"
)

// Handler kinds for BindHandler.
const (
	KindQuery   = "query"
	KindCommand = "command"
)

// Domain kinds: compiled apps have a base layer from code, dynamic apps only
// have stored layers. The engine mounts both the same way.
const (
	DomainDynamic  = "dynamic"
	DomainCompiled = "compiled"
)

const bundleKind = "kaizengo.domains/v1"

var reservedDynamicNames = map[string]struct{}{
	"core": {}, "apps": {}, "platform": {}, "kaizengo": {}, "public": {},
}

// DomainInfo describes one app's domain: its layers and runtime state.
type DomainInfo struct {
	App           string
	Kind          string // DomainCompiled or DomainDynamic
	Title         string
	Summary       string
	Spec          string // effective document (base plus layer)
	Layer         string // stored overlay layer; "" when there is none
	Revision      int64  // newest revision of either layer
	BaseRevision  int64
	LayerRevision int64
	Source        string
	UpdatedAt     time.Time
	Loaded        bool
	Error         string
	Models        []string
	Queries       []string
	Commands      []string
}

// ApplyResult is returned by Apply, Revert, Delete, and Import (also for dry runs).
type ApplyResult struct {
	Domain  DomainInfo
	Changes []string
	DryRun  bool
}

// Domains resolves and mounts every app's domain from layered revisions in
// kaizengo.domain_revisions. A compiled app's base layer is its spec in code
// (recorded as a base revision whenever it changes); layers applied at runtime
// stack on top and are hot-mounted. Dynamic apps are the same with an empty base.
type Domains struct {
	host  *module.Host
	reg   *module.Registry
	store *app.DomainStore

	mu      sync.Mutex // serializes mounts
	dynamic map[string]*App

	emu    sync.Mutex
	errors map[string]string

	hmu      sync.RWMutex
	handlers map[string]QueryHandler
}

// NewDomains wires the domain manager. reg is the compiled-in registry.
func NewDomains(host *module.Host, reg *module.Registry, store *app.DomainStore) *Domains {
	if reg == nil {
		reg = module.Default
	}
	return &Domains{
		host:     host,
		reg:      reg,
		store:    store,
		dynamic:  map[string]*App{},
		errors:   map[string]string{},
		handlers: map[string]QueryHandler{},
	}
}

// DomainsFromHost returns the manager provided at platform build.
func DomainsFromHost(host *module.Host) (*Domains, error) {
	if host == nil {
		return nil, fmt.Errorf("host is nil")
	}
	raw, ok := host.Lookup(DomainsKey)
	if !ok {
		return nil, fmt.Errorf("%s not registered", DomainsKey)
	}
	d, ok := raw.(*Domains)
	if !ok || d == nil {
		return nil, fmt.Errorf("%s has unexpected type %T", DomainsKey, raw)
	}
	return d, nil
}

// BindHandler attaches an implementation for a query/command declared without
// shorthand and without a Go handler (used by external Go apps). kind is
// KindQuery or KindCommand. Lookup happens at call time.
func (d *Domains) BindHandler(appName, kind, name string, fn QueryHandler) {
	d.hmu.Lock()
	defer d.hmu.Unlock()
	d.handlers[handlerKey(appName, kind, name)] = fn
}

// UnbindHandler removes a binding made with BindHandler.
func (d *Domains) UnbindHandler(appName, kind, name string) {
	d.hmu.Lock()
	defer d.hmu.Unlock()
	delete(d.handlers, handlerKey(appName, kind, name))
}

func handlerKey(appName, kind, name string) string {
	return appName + "/" + kind + "/" + name
}

func (d *Domains) dispatch(c HandlerCtx, args map[string]any, appName, kind, name string) (any, error) {
	d.hmu.RLock()
	fn := d.handlers[handlerKey(appName, kind, name)]
	d.hmu.RUnlock()
	if fn == nil {
		return nil, fmt.Errorf("%s %s.%s has no handler (bind one from an external Go app)", kind, appName, name)
	}
	return fn(c, args)
}

// IsDynamic reports whether name has no compiled base layer.
func (d *Domains) IsDynamic(name string) bool {
	_, compiled := d.reg.Get(name)
	return !compiled
}

func (d *Domains) setError(name, msg string) {
	d.emu.Lock()
	defer d.emu.Unlock()
	if msg == "" {
		delete(d.errors, name)
	} else {
		d.errors[name] = msg
	}
}

func (d *Domains) errorFor(name string) string {
	d.emu.Lock()
	defer d.emu.Unlock()
	return d.errors[name]
}

// resolveCompiled is called from a compiled app's Setup: it records base as a
// new base revision when the code changed and returns base with the stored
// layer applied. A layer that no longer fits the base is reported and skipped.
func (d *Domains) resolveCompiled(base appspec.AppSpec) appspec.AppSpec {
	if d.store == nil {
		return base
	}
	ctx := context.Background()
	name := base.Name
	heads, err := d.store.Head(ctx, name)
	if err != nil {
		d.host.Log.Warn("domain layers not read", "app", name, "err", err)
		return base
	}
	if doc, err := appspec.MarshalDomain(base); err == nil && (heads.Base == nil || heads.Base.Spec != string(doc)) {
		var prev *appspec.AppSpec
		if heads.Base != nil {
			if p, err := appspec.Parse([]byte(heads.Base.Spec)); err == nil {
				prev = &p
			}
		}
		changes, err := diffSpecs(prev, base)
		if err != nil {
			changes = []string{err.Error()}
		}
		rec, err := d.store.Append(ctx, app.DomainRevision{
			App: name, Layer: app.LayerBase, Spec: string(doc), Source: SourceCompiled, Changes: changes,
		})
		if err != nil {
			d.host.Log.Warn("domain base revision not stored", "app", name, "err", err)
		} else {
			d.host.Log.Info("domain base revision", "app", name, "revision", rec.Revision)
		}
	}
	if heads.Overlay == nil || heads.Overlay.Spec == "" {
		d.setError(name, "")
		return base
	}
	spec, err := d.effective(&base, heads.Overlay.Spec)
	if err != nil {
		msg := fmt.Sprintf("layer revision %d not applied: %v", heads.Overlay.Revision, err)
		d.setError(name, msg)
		d.host.Log.Warn("domain layer not applied", "app", name, "err", msg)
		return base
	}
	d.setError(name, "")
	return spec
}

// LoadAll mounts every stored app that has no compiled base. A broken
// definition is recorded (visible in platformDomains) and does not fail boot.
func (d *Domains) LoadAll(ctx context.Context) error {
	heads, err := d.store.Heads(ctx)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, name := range sortedKeys(heads) {
		h := heads[name]
		if _, compiled := d.reg.Get(name); compiled || h.Overlay == nil || h.Overlay.Spec == "" {
			continue
		}
		spec, err := d.parseDynamic(h.Overlay.Spec)
		if err == nil {
			err = d.mountNewLocked(spec)
		}
		if err != nil {
			d.setError(name, err.Error())
			d.host.Log.Warn("domain not loaded", "app", name, "err", err)
			continue
		}
		d.setError(name, "")
		d.host.Log.Info("domain loaded", "app", name, "revision", h.Revision(), "source", h.Overlay.Source)
	}
	return nil
}

// Diff reports the changes applying doc would make without touching anything.
func (d *Domains) Diff(ctx context.Context, doc string) (string, []string, error) {
	res, err := d.Apply(ctx, doc, SourceGraphQL, true)
	if err != nil {
		return "", nil, err
	}
	return res.Domain.App, res.Changes, nil
}

// Apply takes a complete domain document for any app (as printed by
// platformDomain), stores what it adds on top of the app's base as a new layer
// revision, migrates tables (additive), and hot-mounts the result. With dryRun
// only validation and diff run.
func (d *Domains) Apply(ctx context.Context, doc, source string, dryRun bool) (*ApplyResult, error) {
	if source == "" {
		source = SourceGraphQL
	}
	full, err := parseDoc(doc)
	if err != nil {
		return nil, err
	}
	base, err := d.baseOf(full.Name)
	if err != nil {
		return nil, err
	}
	layer := doc
	if base != nil {
		l, err := appspec.Subtract(full, *base)
		if err != nil {
			return nil, err
		}
		layer = ""
		if !l.EmptyLayer() {
			b, err := appspec.MarshalDomain(l)
			if err != nil {
				return nil, err
			}
			layer = string(b)
		}
	}
	return d.applyLayer(ctx, full.Name, base, layer, source, dryRun)
}

// Revert stores the layer as it was at revision as a new revision and mounts
// it. The base layer always comes from the running code.
func (d *Domains) Revert(ctx context.Context, name string, revision int64, dryRun bool) (*ApplyResult, error) {
	rec, err := d.store.Get(ctx, name, revision)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("app %q has no revision %d", name, revision)
	}
	heads, err := d.store.HeadAt(ctx, name, revision)
	if err != nil {
		return nil, err
	}
	layer := ""
	if heads.Overlay != nil {
		layer = heads.Overlay.Spec
	}
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	return d.applyLayer(ctx, name, base, layer, fmt.Sprintf("revert:%d", revision), dryRun)
}

// Delete stores an empty layer: a compiled app is reset to its base, a dynamic
// app is unmounted. Tables are kept and the history can be reverted.
func (d *Domains) Delete(ctx context.Context, name string, dryRun bool) (*ApplyResult, error) {
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	return d.applyLayer(ctx, name, base, "", SourceGraphQL, dryRun)
}

// History returns every stored revision of name, newest first.
func (d *Domains) History(ctx context.Context, name string) ([]app.DomainRevision, error) {
	return d.store.History(ctx, name)
}

// Revision returns one stored revision, or nil.
func (d *Domains) Revision(ctx context.Context, name string, revision int64) (*app.DomainRevision, error) {
	return d.store.Get(ctx, name, revision)
}

type domainBundle struct {
	Kind       string        `yaml:"kind"`
	ExportedAt time.Time     `yaml:"exportedAt"`
	Domains    []bundleEntry `yaml:"domains"`
}

type bundleEntry struct {
	App      string `yaml:"app"`
	Revision int64  `yaml:"revision"`
	Layer    string `yaml:"layer"`
}

// Export returns a YAML bundle with the current layer of every app that has
// one (or only of names). Import restores it.
func (d *Domains) Export(ctx context.Context, names []string) (string, error) {
	heads, err := d.store.Heads(ctx)
	if err != nil {
		return "", err
	}
	want := map[string]bool{}
	for _, n := range names {
		if _, ok := heads[n]; !ok {
			if _, compiled := d.reg.Get(n); !compiled {
				return "", fmt.Errorf("app %q not found", n)
			}
		}
		want[n] = true
	}
	b := domainBundle{Kind: bundleKind, ExportedAt: time.Now().UTC().Truncate(time.Second), Domains: []bundleEntry{}}
	for _, name := range sortedKeys(heads) {
		h := heads[name]
		if (len(want) > 0 && !want[name]) || h.Overlay == nil || h.Overlay.Spec == "" {
			continue
		}
		b.Domains = append(b.Domains, bundleEntry{App: name, Revision: h.Overlay.Revision, Layer: h.Overlay.Spec})
	}
	out, err := yaml.Marshal(b)
	return string(out), err
}

// Import restores a bundle from Export: each app's layer is applied as a new
// revision (source "restore"). Every entry is validated before any is applied.
func (d *Domains) Import(ctx context.Context, bundle string, dryRun bool) ([]ApplyResult, error) {
	var b domainBundle
	if err := yaml.Unmarshal([]byte(bundle), &b); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	if b.Kind != bundleKind {
		return nil, fmt.Errorf("bundle kind must be %q", bundleKind)
	}
	bases := make([]*appspec.AppSpec, len(b.Domains))
	out := make([]ApplyResult, 0, len(b.Domains))
	for i, e := range b.Domains {
		base, err := d.baseOf(e.App)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.App, err)
		}
		bases[i] = base
		res, err := d.applyLayer(ctx, e.App, base, e.Layer, SourceRestore, true)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.App, err)
		}
		out = append(out, *res)
	}
	if dryRun {
		return out, nil
	}
	out = out[:0]
	for i, e := range b.Domains {
		res, err := d.applyLayer(ctx, e.App, bases[i], e.Layer, SourceRestore, false)
		if err != nil {
			return out, fmt.Errorf("%s: %w", e.App, err)
		}
		out = append(out, *res)
	}
	return out, nil
}

// Exists reports whether name is a compiled app or has a stored layer.
func (d *Domains) Exists(ctx context.Context, name string) (bool, error) {
	if _, ok := d.reg.Get(name); ok {
		return true, nil
	}
	h, err := d.store.Head(ctx, name)
	return err == nil && h.Overlay != nil && h.Overlay.Spec != "", err
}

// List returns every compiled app with a spec, then every dynamic app.
func (d *Domains) List(ctx context.Context) ([]DomainInfo, error) {
	heads, err := d.store.Heads(ctx)
	if err != nil {
		return nil, err
	}
	var out []DomainInfo
	for _, name := range d.reg.Names() {
		raw, _ := d.reg.Get(name)
		if _, ok := raw.(*App); !ok {
			continue
		}
		out = append(out, d.describe(name, heads[name]))
	}
	for _, name := range sortedKeys(heads) {
		h := heads[name]
		if _, compiled := d.reg.Get(name); compiled || h.Overlay == nil || h.Overlay.Spec == "" {
			continue
		}
		out = append(out, d.describe(name, h))
	}
	return out, nil
}

// Get returns one app's domain, or nil when name is unknown.
func (d *Domains) Get(ctx context.Context, name string) (*DomainInfo, error) {
	ok, err := d.Exists(ctx, name)
	if err != nil || !ok {
		return nil, err
	}
	h, err := d.store.Head(ctx, name)
	if err != nil {
		return nil, err
	}
	info := d.describe(name, h)
	return &info, nil
}

func (d *Domains) describe(name string, heads app.DomainHeads) DomainInfo {
	layer := ""
	if heads.Overlay != nil {
		layer = heads.Overlay.Spec
	}
	base, err := d.baseOf(name)
	var eff *appspec.AppSpec
	switch {
	case err != nil:
	case base != nil:
		if a := d.compiledApp(name); a != nil {
			s := a.Spec()
			eff = &s
		} else if s, e := d.effective(base, layer); e == nil {
			eff = &s
		} else {
			eff, err = base, e
		}
	default:
		if s, e := d.parseDynamic(layer); e == nil {
			eff = &s
		} else {
			err = e
		}
	}
	info := d.info(name, base, heads, eff, layer)
	if err != nil {
		info.Error = err.Error()
	}
	return info
}

func (d *Domains) info(name string, base *appspec.AppSpec, heads app.DomainHeads, eff *appspec.AppSpec, layer string) DomainInfo {
	info := DomainInfo{
		App: name, Kind: DomainDynamic, Layer: layer, Revision: heads.Revision(),
		Models: []string{}, Queries: []string{}, Commands: []string{},
	}
	if base != nil {
		info.Kind, info.Source = DomainCompiled, SourceCompiled
		info.Loaded = d.compiledApp(name) != nil
	} else {
		_, info.Loaded = AppFromHost(d.host, name)
	}
	if heads.Base != nil {
		info.BaseRevision, info.UpdatedAt = heads.Base.Revision, heads.Base.CreatedAt
	}
	if heads.Overlay != nil {
		info.LayerRevision = heads.Overlay.Revision
		if layer != "" || base == nil {
			info.Source = heads.Overlay.Source
		}
		if heads.Overlay.CreatedAt.After(info.UpdatedAt) {
			info.UpdatedAt = heads.Overlay.CreatedAt
		}
	}
	if eff != nil {
		info.Title, info.Summary = eff.Title, eff.Summary
		if base == nil {
			info.Spec = layer
		} else if b, err := appspec.MarshalDomain(*eff); err == nil {
			info.Spec = string(b)
		}
		for _, m := range eff.Models {
			info.Models = append(info.Models, m.Name)
		}
		for _, q := range eff.Queries {
			info.Queries = append(info.Queries, q.Name)
		}
		for _, c := range eff.Commands {
			info.Commands = append(info.Commands, c.Name)
		}
	}
	info.Error = d.errorFor(name)
	return info
}

// applyLayer validates layer on base, diffs it against the current state,
// mounts it, and stores it as a new layer revision (unless unchanged).
func (d *Domains) applyLayer(ctx context.Context, name string, base *appspec.AppSpec, layer, source string, dryRun bool) (*ApplyResult, error) {
	heads, err := d.store.Head(ctx, name)
	if err != nil {
		return nil, err
	}
	current := ""
	if heads.Overlay != nil {
		current = heads.Overlay.Spec
	}
	if base == nil && layer == "" && current == "" {
		return nil, fmt.Errorf("app %q not found", name)
	}
	var prev *appspec.AppSpec
	if base != nil || current != "" {
		if p, err := d.effective(base, current); err == nil {
			prev = &p
		} else {
			prev = base
		}
	}
	var next *appspec.AppSpec
	if base != nil || layer != "" {
		n, err := d.effective(base, layer)
		if err != nil {
			return nil, err
		}
		next = &n
	}
	var changes []string
	if next == nil {
		changes = []string{"delete app " + name + " (tables kept)"}
	} else if changes, err = diffSpecs(prev, *next); err != nil {
		return nil, err
	}
	if layer == current {
		changes = []string{"no changes"}
	}
	res := &ApplyResult{Changes: changes, DryRun: dryRun}
	if dryRun {
		res.Domain = d.info(name, base, heads, next, layer)
		return res, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.mountLocked(name, base, next); err != nil {
		return nil, err
	}
	if layer != current {
		rec, err := d.store.Append(ctx, app.DomainRevision{
			App: name, Layer: app.LayerOverlay, Spec: layer, Source: source, Changes: changes,
		})
		if err != nil {
			return nil, err
		}
		heads.Overlay = &rec
	}
	d.setError(name, "")
	res.Domain = d.info(name, base, heads, next, layer)
	return res, nil
}

// baseOf returns the code layer of a compiled app (nil for dynamic apps).
func (d *Domains) baseOf(name string) (*appspec.AppSpec, error) {
	raw, ok := d.reg.Get(name)
	if !ok {
		return nil, nil
	}
	if a := d.compiledApp(name); a != nil {
		b := a.BaseSpec()
		return &b, nil
	}
	a, ok := raw.(*App)
	if !ok {
		return nil, fmt.Errorf("app %q is compiled in without a spec", name)
	}
	spec, err := a.loadSpec()
	if err != nil {
		return nil, err
	}
	return &spec, nil
}

// compiledApp returns the loaded engine App of a compiled app.
func (d *Domains) compiledApp(name string) *App {
	a, ok := AppFromHost(d.host, name)
	if !ok || a.opts.managed {
		return nil
	}
	return a
}

// effective merges layer onto base (base nil: layer is a complete dynamic
// definition) and validates the result.
func (d *Domains) effective(base *appspec.AppSpec, layer string) (appspec.AppSpec, error) {
	if base == nil {
		return d.parseDynamic(layer)
	}
	if strings.TrimSpace(layer) == "" {
		return *base, nil
	}
	l, err := appspec.ParseLayer([]byte(layer))
	if err != nil {
		return appspec.AppSpec{}, err
	}
	spec, err := appspec.Merge(*base, l)
	if err != nil {
		return appspec.AppSpec{}, err
	}
	if err := spec.Validate(); err != nil {
		return appspec.AppSpec{}, err
	}
	for _, dep := range l.Depends {
		if !d.loaded(dep) {
			return appspec.AppSpec{}, fmt.Errorf("depends: app %q is not loaded", dep)
		}
	}
	return spec, nil
}

func parseDoc(doc string) (appspec.AppSpec, error) {
	if strings.TrimSpace(doc) == "" {
		return appspec.AppSpec{}, fmt.Errorf("domain document is empty")
	}
	if len(doc) > MaxDomainSize {
		return appspec.AppSpec{}, fmt.Errorf("domain document is %d bytes; the limit is %d", len(doc), MaxDomainSize)
	}
	return appspec.Parse([]byte(doc))
}

// parseDynamic validates the definition of an app without code: a
// self-contained app.yaml without file includes, React pages, menus, locales,
// or cross-app extends/exports.
func (d *Domains) parseDynamic(doc string) (appspec.AppSpec, error) {
	spec, err := parseDoc(doc)
	if err != nil {
		return appspec.AppSpec{}, err
	}
	if _, ok := reservedDynamicNames[spec.Name]; ok {
		return appspec.AppSpec{}, fmt.Errorf("app name %q is reserved", spec.Name)
	}
	if spec.Schema != spec.Name {
		return appspec.AppSpec{}, fmt.Errorf("schema: dynamic apps always use schema %q", spec.Name)
	}
	switch {
	case len(spec.Menus) > 0 || hasNav(spec.Nav):
		return appspec.AppSpec{}, fmt.Errorf("nav/menus: dynamic apps have no React pages; use the CLI or platformCatalog")
	case len(spec.Locales) > 0:
		return appspec.AppSpec{}, fmt.Errorf("locales: dynamic apps cannot ship .po catalogs")
	case len(spec.Extends) > 0 || len(spec.Exports.Components)+len(spec.Exports.Views)+len(spec.Exports.Menus) > 0:
		return appspec.AppSpec{}, fmt.Errorf("extends/exports: not supported for dynamic apps (bind hooks from an external Go app instead)")
	}
	for _, dep := range spec.Depends {
		if !d.loaded(dep) {
			return appspec.AppSpec{}, fmt.Errorf("depends: app %q is not loaded", dep)
		}
	}
	// Dynamic apps always run with extensions on so external Go apps can bind wildcard hooks.
	spec.EnableExtensions = true
	spec.EnableSPA = false
	return spec, nil
}

func (d *Domains) loaded(name string) bool {
	for _, mf := range d.host.Loaded {
		if mf.Name == name {
			return true
		}
	}
	return false
}

func (d *Domains) mountLocked(name string, base, next *appspec.AppSpec) error {
	if base != nil {
		a := d.compiledApp(name)
		if a == nil || reflect.DeepEqual(a.Spec(), *next) {
			// Not installed: Setup resolves the stored layer when it is.
			return nil
		}
		return d.swapLocked(a, *next)
	}
	if next == nil {
		d.unmountLocked(name)
		return nil
	}
	if a, ok := d.dynamic[name]; ok {
		if reflect.DeepEqual(a.Spec(), *next) {
			return nil
		}
		return d.swapLocked(a, *next)
	}
	return d.mountNewLocked(*next)
}

type appState struct {
	spec      appspec.AppSpec
	queries   map[string]*graphql.Field
	mutations map[string]*graphql.Field
	qnames    []string
	mnames    []string
	models    map[string]*modelService
}

func (d *Domains) saveState(a *App) appState {
	st := appState{
		spec: a.spec, qnames: a.ownedQueries, mnames: a.ownedMutations,
		queries: map[string]*graphql.Field{}, mutations: map[string]*graphql.Field{},
	}
	for _, n := range a.ownedQueries {
		if f, ok := d.host.GQL.QueryField(n); ok {
			st.queries[n] = f
		}
	}
	for _, n := range a.ownedMutations {
		if f, ok := d.host.GQL.MutationField(n); ok {
			st.mutations[n] = f
		}
	}
	if a.models != nil {
		a.models.mu.RLock()
		st.models = a.models.byName
		a.models.mu.RUnlock()
	}
	return st
}

func (d *Domains) restoreState(a *App, st appState) {
	a.spec, a.ownedQueries, a.ownedMutations = st.spec, st.qnames, st.mnames
	for n, f := range st.queries {
		d.host.GQL.RegisterQuery(n, f)
	}
	for n, f := range st.mutations {
		d.host.GQL.RegisterMutation(n, f)
	}
	if a.models != nil {
		a.models.swap(st.models)
	}
}

// swapLocked hot-remounts a set-up app with next. On failure the previous
// surface is restored so a bad apply never leaves the app half-mounted.
func (d *Domains) swapLocked(a *App, next appspec.AppSpec) (err error) {
	prev := d.saveState(a)
	d.removeFieldsLocked(prev.qnames, prev.mnames)
	snap := d.snapshotFields()
	q0, m0 := d.host.GQL.FieldNames()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mount %s: %v", next.Name, r)
		}
		if err == nil {
			return
		}
		d.restoreFields(snap)
		q1, m1 := d.host.GQL.FieldNames()
		d.removeFieldsLocked(added(q0, q1), added(m0, m1))
		d.restoreState(a, prev)
		d.host.GQL.Invalidate()
	}()
	if err := a.remount(d.host, next); err != nil {
		return err
	}
	if clash := d.replacedFields(snap); len(clash) > 0 {
		return fmt.Errorf("GraphQL fields already defined by another app: %s", strings.Join(clash, ", "))
	}
	sec := next.Security
	sec.Users = nil
	sec.Disable = append(append([]string(nil), sec.Disable...), droppedEntries(prev.spec.Security, next.Security)...)
	if err := ApplySecurity(d.host, sec); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	d.host.GQL.Invalidate()
	return nil
}

// mountNewLocked sets up an app without code for the first time.
func (d *Domains) mountNewLocked(spec appspec.AppSpec) (err error) {
	a := New(Options{AppName: spec.Name, Spec: &spec, AutoSchema: true, managed: true})
	snap := d.snapshotFields()
	q0, m0 := d.host.GQL.FieldNames()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mount %s: %v", spec.Name, r)
		}
		if err == nil {
			return
		}
		d.restoreFields(snap)
		q1, m1 := d.host.GQL.FieldNames()
		d.removeFieldsLocked(added(q0, q1), added(m0, m1))
		d.host.Unprovide(spec.Name)
		d.host.Unprovide(AppKey(spec.Name))
		d.host.Unprovide(ModelsKey(spec.Name))
		d.host.GQL.Invalidate()
	}()
	if err := a.Setup(d.host); err != nil {
		return err
	}
	if err := a.Mount(d.host); err != nil {
		return err
	}
	if clash := d.replacedFields(snap); len(clash) > 0 {
		return fmt.Errorf("GraphQL fields already defined by another app: %s", strings.Join(clash, ", "))
	}
	if err := ApplySecurity(d.host, spec.Security); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	d.dynamic[spec.Name] = a
	d.host.Loaded = append(withoutManifest(d.host.Loaded, spec.Name), a.Manifest())
	d.host.GQL.Invalidate()
	return nil
}

func (d *Domains) unmountLocked(name string) {
	a, ok := d.dynamic[name]
	if !ok {
		return
	}
	d.removeFieldsLocked(a.ownedQueries, a.ownedMutations)
	if dropped := droppedEntries(a.spec.Security, appspec.SecuritySpec{}); len(dropped) > 0 {
		if err := ApplySecurity(d.host, appspec.SecuritySpec{Disable: dropped}); err != nil {
			d.host.Log.Warn("domain policies not disabled", "app", name, "err", err)
		}
	}
	delete(d.dynamic, name)
	d.host.Unprovide(name)
	d.host.Unprovide(AppKey(name))
	d.host.Unprovide(ModelsKey(name))
	d.host.Loaded = withoutManifest(d.host.Loaded, name)
	d.host.GQL.Invalidate()
}

// droppedEntries names policy entries in prev that next no longer declares.
func droppedEntries(prev, next appspec.SecuritySpec) []string {
	keep := map[string]struct{}{}
	for _, e := range next.Entries {
		keep[e.Name] = struct{}{}
	}
	var out []string
	for _, e := range prev.Entries {
		if _, ok := keep[e.Name]; !ok {
			out = append(out, e.Name)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// fieldSnapshot records root field definitions so a mount that replaces a
// field owned by another app can be detected and undone.
type fieldSnapshot struct {
	queries   map[string]*graphql.Field
	mutations map[string]*graphql.Field
}

func (d *Domains) snapshotFields() fieldSnapshot {
	qs, ms := d.host.GQL.FieldNames()
	s := fieldSnapshot{queries: make(map[string]*graphql.Field, len(qs)), mutations: make(map[string]*graphql.Field, len(ms))}
	for _, n := range qs {
		s.queries[n], _ = d.host.GQL.QueryField(n)
	}
	for _, n := range ms {
		s.mutations[n], _ = d.host.GQL.MutationField(n)
	}
	return s
}

func (d *Domains) replacedFields(s fieldSnapshot) []string {
	var out []string
	for n, f := range s.queries {
		if cur, ok := d.host.GQL.QueryField(n); ok && cur != f {
			out = append(out, n)
		}
	}
	for n, f := range s.mutations {
		if cur, ok := d.host.GQL.MutationField(n); ok && cur != f {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func (d *Domains) restoreFields(s fieldSnapshot) {
	for n, f := range s.queries {
		if cur, ok := d.host.GQL.QueryField(n); !ok || cur != f {
			d.host.GQL.RegisterQuery(n, f)
		}
	}
	for n, f := range s.mutations {
		if cur, ok := d.host.GQL.MutationField(n); !ok || cur != f {
			d.host.GQL.RegisterMutation(n, f)
		}
	}
}

func (d *Domains) removeFieldsLocked(queries, mutations []string) {
	for _, q := range queries {
		d.host.GQL.RemoveQuery(q)
	}
	for _, m := range mutations {
		d.host.GQL.RemoveMutation(m)
	}
}

func added(before, after []string) []string {
	seen := make(map[string]struct{}, len(before))
	for _, n := range before {
		seen[n] = struct{}{}
	}
	var out []string
	for _, n := range after {
		if _, ok := seen[n]; !ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func withoutManifest(list []module.Manifest, name string) []module.Manifest {
	out := make([]module.Manifest, 0, len(list))
	for _, mf := range list {
		if mf.Name != name {
			out = append(out, mf)
		}
	}
	return out
}
