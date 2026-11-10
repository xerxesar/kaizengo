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
	"kaizengo/internal/extension"
	"kaizengo/internal/module"
	"kaizengo/internal/platform/postgres"
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
	SourceRebase   = "rebase"
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

// pendingRebase holds incoming code that has not been applied as tip yet.
type pendingRebase struct {
	Incoming  string
	Conflicts []appspec.Conflict
}

// pendingMigrate holds a tip proposal that needs a data migration before apply.
type pendingMigrate struct {
	ProposedSpec string
	Hints        []MigrationHint
}

// DomainInfo describes one app's domain tip revision and runtime state.
type DomainInfo struct {
	App      string
	Kind     string // DomainCompiled or DomainDynamic
	Title    string
	Summary  string
	Spec     string // tip document (full effective domain)
	Revision int64  // tip revision number
	Source   string
	UpdatedAt time.Time
	Loaded   bool
	Error    string
	Models   []string
	Queries  []string
	Commands []string

	// RebaseStatus: "" | needs_rebase | conflict
	RebaseStatus string
	Incoming     string // pending compiled app.yaml (not tip until rebase)
	Conflicts    []appspec.Conflict

	// MigratePending is true when a migrate-required proposal is stashed for this app.
	MigratePending bool

	// Deprecated aliases kept for GraphQL clients during migration.
	Layer         string
	BaseRevision  int64
	LayerRevision int64
}

// ApplyResult is returned by Apply, Revert, Delete, and Import (also for dry runs).
type ApplyResult struct {
	Domain  DomainInfo
	Changes []string
	DryRun  bool

	// MigrateRequired is set when the tip needs a data migration and none was provided.
	// ProposedSpec is the tip document that was not applied; MigrateHints explain why.
	MigrateRequired bool
	MigrateHints    []MigrationHint
	ProposedSpec    string
}

// RebaseOptions controls three-way rebase onto pending/current code.
type RebaseOptions struct {
	Force       bool
	DryRun      bool
	Resolutions map[string]string // conflict path → "ours" | "theirs"
}

// Domains resolves and mounts every app's domain from snapshot revisions in
// kaizengo.domain_revisions. Each revision stores the full effective document
// (git-like commits). Compiled app.yaml seeds the first revision; runtime
// apply/set/remove append new tips.
type Domains struct {
	host  *module.Host
	reg   *module.Registry
	store *app.DomainStore

	mu      sync.Mutex // serializes mounts
	dynamic map[string]*App

	emu    sync.Mutex
	errors map[string]string

	pmu     sync.Mutex
	pending map[string]pendingRebase // compiled apps awaiting rebase

	migmu          sync.Mutex
	pendingMigrate map[string]pendingMigrate // tip proposals awaiting data migrate

	mmu        sync.Mutex
	migrations map[string]DomainMigrateFunc // "app/name" → Go migrate

	hmu      sync.RWMutex
	handlers map[string]QueryHandler
}

// NewDomains wires the domain manager. reg is the compiled-in registry.
func NewDomains(host *module.Host, reg *module.Registry, store *app.DomainStore) *Domains {
	if reg == nil {
		reg = module.Default
	}
	return &Domains{
		host:           host,
		reg:            reg,
		store:          store,
		dynamic:        map[string]*App{},
		errors:         map[string]string{},
		pending:        map[string]pendingRebase{},
		pendingMigrate: map[string]pendingMigrate{},
		handlers:       map[string]QueryHandler{},
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

// resolveCompiled is called from a compiled app's Setup: ensures a tip
// revision exists from code when needed, then returns the tip document.
// When the tip has runtime edits and code changed, tries a two-way merge.
// Clean merge → append tip (source=rebase). Conflicts → keep tip mounted,
// stash incoming code as pending (not tip), quarantine with rebase status.
func (d *Domains) resolveCompiled(base appspec.AppSpec) appspec.AppSpec {
	if d.store == nil {
		return base
	}
	ctx := context.Background()
	name := base.Name
	codeDoc, err := appspec.MarshalDomain(base)
	if err != nil {
		d.host.Log.Warn("domain marshal", "app", name, "err", err)
		return base
	}
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		d.host.Log.Warn("domain tip not read", "app", name, "err", err)
		return base
	}
	if tip == nil {
		rec, err := d.store.Append(ctx, app.DomainRevision{
			App: name, Layer: app.LayerRevision, Spec: string(codeDoc), Source: SourceCompiled,
			Changes: []string{"create app " + name + " from code"},
		})
		if err != nil {
			d.host.Log.Warn("domain revision not stored", "app", name, "err", err)
		} else {
			d.host.Log.Info("domain revision", "app", name, "revision", rec.Revision, "source", SourceCompiled)
		}
		d.clearPending(name)
		d.setError(name, "")
		return base
	}
	// Tip is pure code: fast-forward when app.yaml changed.
	if tip.Source == SourceCompiled || tip.Layer == app.LayerBase {
		if tip.Spec != string(codeDoc) {
			prev, _ := appspec.Parse([]byte(tip.Spec))
			changes, derr := diffSpecs(&prev, base)
			if derr != nil {
				changes = []string{derr.Error()}
			}
			rec, aerr := d.store.Append(ctx, app.DomainRevision{
				App: name, Layer: app.LayerRevision, Spec: string(codeDoc), Source: SourceCompiled, Changes: changes,
			})
			if aerr != nil {
				d.host.Log.Warn("domain revision not stored", "app", name, "err", aerr)
			} else {
				d.host.Log.Info("domain revision", "app", name, "revision", rec.Revision, "source", SourceCompiled)
			}
		}
		d.clearPending(name)
		d.setError(name, "")
		return base
	}
	tipSpec, err := d.specFromTip(&base, tip)
	if err != nil {
		msg := fmt.Sprintf("revision %d not applied: %v", tip.Revision, err)
		d.setError(name, msg)
		d.host.Log.Warn("domain revision not applied", "app", name, "err", msg)
		return base
	}
	if snapshotEqualsCode(tip.Spec, string(codeDoc)) {
		d.clearPending(name)
		d.setError(name, "")
		return tipSpec
	}
	// Runtime tip vs new code, relative to last compiled tip (three-way).
	// Do not append incoming until clean/forced.
	ancestor := d.ancestorCompiled(ctx, name)
	merged, merr := appspec.MergeThreeWay(ancestor, tipSpec, base, false)
	if merr != nil {
		d.setPending(name, string(codeDoc), nil)
		d.setError(name, fmt.Sprintf("rebase blocked: %v", merr))
		return tipSpec
	}
	if len(merged.Conflicts) == 0 {
		full, ferr := appspec.MarshalDomain(merged.Spec)
		if ferr != nil {
			d.setError(name, ferr.Error())
			return tipSpec
		}
		changes, _ := diffSpecs(&tipSpec, merged.Spec)
		rec, aerr := d.store.Append(ctx, app.DomainRevision{
			App: name, Layer: app.LayerRevision, Spec: string(full), Source: SourceRebase, Changes: changes,
		})
		if aerr != nil {
			d.host.Log.Warn("domain rebase not stored", "app", name, "err", aerr)
			d.setPending(name, string(codeDoc), nil)
			d.setError(name, aerr.Error())
			return tipSpec
		}
		d.clearPending(name)
		d.setError(name, "")
		d.host.Log.Info("domain rebased", "app", name, "revision", rec.Revision)
		return merged.Spec
	}
	d.setPending(name, string(codeDoc), merged.Conflicts)
	d.setError(name, fmt.Sprintf("conflict: %d path(s); tip kept — domain rebase %s [--force]", len(merged.Conflicts), name))
	d.host.Log.Warn("domain rebase conflicts", "app", name, "conflicts", len(merged.Conflicts))
	return tipSpec
}

func (d *Domains) setPending(name, incoming string, conflicts []appspec.Conflict) {
	d.pmu.Lock()
	defer d.pmu.Unlock()
	d.pending[name] = pendingRebase{Incoming: incoming, Conflicts: conflicts}
}

func (d *Domains) clearPending(name string) {
	d.pmu.Lock()
	defer d.pmu.Unlock()
	delete(d.pending, name)
}

func (d *Domains) pendingOf(name string) (pendingRebase, bool) {
	d.pmu.Lock()
	defer d.pmu.Unlock()
	p, ok := d.pending[name]
	return p, ok
}

func (d *Domains) setPendingMigrate(name, proposed string, hints []MigrationHint) {
	d.migmu.Lock()
	defer d.migmu.Unlock()
	d.pendingMigrate[name] = pendingMigrate{ProposedSpec: proposed, Hints: hints}
}

func (d *Domains) clearPendingMigrate(name string) {
	d.migmu.Lock()
	defer d.migmu.Unlock()
	delete(d.pendingMigrate, name)
}

// PendingMigrate returns a stashed migrate-required tip proposal for app.
func (d *Domains) PendingMigrate(name string) (proposed string, hints []MigrationHint, ok bool) {
	d.migmu.Lock()
	defer d.migmu.Unlock()
	p, ok := d.pendingMigrate[name]
	if !ok {
		return "", nil, false
	}
	return p.ProposedSpec, p.Hints, true
}

func snapshotEqualsCode(tip, code string) bool {
	a, err1 := appspec.Parse([]byte(tip))
	b, err2 := appspec.Parse([]byte(code))
	if err1 != nil || err2 != nil {
		return tip == code
	}
	return reflect.DeepEqual(a, b)
}

// ancestorCompiled returns the newest baseline tip for three-way rebase:
// source=compiled, source=rebase (prior clean/forced upgrade), or legacy base layer.
func (d *Domains) ancestorCompiled(ctx context.Context, name string) appspec.AppSpec {
	hist, err := d.store.History(ctx, name)
	if err != nil {
		return appspec.AppSpec{}
	}
	for _, r := range hist {
		switch {
		case r.Source == SourceCompiled, r.Source == SourceRebase, r.Layer == app.LayerBase:
		default:
			continue
		}
		spec, err := appspec.Parse([]byte(r.Spec))
		if err != nil {
			continue
		}
		return spec
	}
	return appspec.AppSpec{}
}

// Conflicts returns pending rebase conflicts for app (empty when none).
func (d *Domains) Conflicts(name string) []appspec.Conflict {
	p, ok := d.pendingOf(name)
	if !ok {
		return nil
	}
	return append([]appspec.Conflict(nil), p.Conflicts...)
}

// IncomingSpec returns the pending compiled document for app, or "".
func (d *Domains) IncomingSpec(name string) string {
	p, ok := d.pendingOf(name)
	if !ok {
		return ""
	}
	return p.Incoming
}

// Rebase merges tip with pending (or current) code. force takes code on every conflict.
// dryRun only reports conflicts/changes. On success appends a tip (source=rebase) and remounts.
func (d *Domains) Rebase(ctx context.Context, name string, force, dryRun bool) (*ApplyResult, error) {
	return d.RebaseWith(ctx, name, RebaseOptions{Force: force, DryRun: dryRun})
}

// RebaseWith is Rebase with per-path resolutions (path → "ours"|"theirs").
func (d *Domains) RebaseWith(ctx context.Context, name string, opts RebaseOptions) (*ApplyResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("app is required")
	}
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, fmt.Errorf("app %q is not compiled (nothing to rebase onto)", name)
	}
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		return nil, err
	}
	if tip == nil {
		return nil, fmt.Errorf("app %q has no tip", name)
	}
	ours, err := d.specFromTip(base, tip)
	if err != nil {
		return nil, err
	}
	incoming := d.IncomingSpec(name)
	var theirs appspec.AppSpec
	if incoming != "" {
		theirs, err = appspec.Parse([]byte(incoming))
		if err != nil {
			return nil, err
		}
	} else {
		theirs = *base
	}
	ancestor := d.ancestorCompiled(ctx, name)
	merged, err := appspec.MergeThreeWayOpts(ancestor, ours, theirs, appspec.MergeOpts{
		Force: opts.Force, Resolve: opts.Resolutions,
	})
	if err != nil {
		return nil, err
	}
	if len(merged.Conflicts) > 0 && !opts.Force {
		d.setPending(name, mustMarshal(theirs), merged.Conflicts)
		d.setError(name, fmt.Sprintf("conflict: %d path(s); tip kept — resolve in Settings → Domains or domain rebase %s --force", len(merged.Conflicts), name))
		info, _ := d.Get(ctx, name)
		if info == nil {
			info = &DomainInfo{App: name}
		}
		info.RebaseStatus = "conflict"
		info.Incoming = mustMarshal(theirs)
		info.Conflicts = merged.Conflicts
		return &ApplyResult{Domain: *info, Changes: conflictChanges(merged.Conflicts), DryRun: opts.DryRun}, nil
	}
	full, err := appspec.MarshalDomain(merged.Spec)
	if err != nil {
		return nil, err
	}
	changes, _ := diffSpecs(&ours, merged.Spec)
	if len(changes) == 0 {
		changes = []string{"rebase onto code"}
	}
	if opts.DryRun {
		info, _ := d.Get(ctx, name)
		if info == nil {
			info = &DomainInfo{App: name}
		}
		info.Spec = string(full)
		return &ApplyResult{Domain: *info, Changes: changes, DryRun: true}, nil
	}
	res, err := d.commitTip(ctx, name, base, string(full), &merged.Spec, &ours, SourceRebase, changes, ApplyOptions{})
	if err != nil {
		return nil, err
	}
	d.clearPending(name)
	d.setError(name, "")
	res.Domain.RebaseStatus = ""
	res.Domain.Incoming = ""
	res.Domain.Conflicts = nil
	return res, nil
}

func conflictChanges(cs []appspec.Conflict) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, "conflict "+c.Path)
	}
	return out
}

func mustMarshal(spec appspec.AppSpec) string {
	b, err := appspec.MarshalDomain(spec)
	if err != nil {
		return ""
	}
	return string(b)
}

// LoadAll mounts every stored app that has no compiled module. A broken
// definition is recorded (visible in platformDomains) and does not fail boot.
func (d *Domains) LoadAll(ctx context.Context) error {
	tips, err := d.store.Tips(ctx)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	names := sortedKeys(tips)
	for _, name := range names {
		tip := tips[name]
		if _, compiled := d.reg.Get(name); compiled || strings.TrimSpace(tip.Spec) == "" {
			continue
		}
		spec, err := d.specFromTip(nil, &tip)
		if err == nil {
			err = d.mountNewLocked(spec)
		}
		if err != nil {
			d.setError(name, err.Error())
			d.host.Log.Warn("domain not loaded", "app", name, "err", err)
			continue
		}
		d.setError(name, "")
		d.host.Log.Info("domain loaded", "app", name, "revision", tip.Revision, "source", tip.Source)
	}
	return nil
}

// ApplyOptions controls soft vs force DDL and optional data migration on tip apply.
type ApplyOptions struct {
	Force       bool // DROP removed columns/indexes/junctions; allow type rewrites in changelog
	DryRun      bool
	MigratePre  string // SQL before additive DDL (e.g. RENAME COLUMN)
	MigratePost string // SQL after additive DDL (backfill); alias of legacy MigrateSQL
	MigrateGo   string // registered DomainMigrateFunc name (runs in post phase)
	NoMigrate   bool   // skip required data-migration prompt
}

// Diff reports the changes applying doc would make without touching anything.
func (d *Domains) Diff(ctx context.Context, doc string) (string, []string, error) {
	app, changes, _, err := d.DiffWithMigrate(ctx, doc)
	return app, changes, err
}

// DiffWithMigrate is Diff plus structured data-migration hints.
func (d *Domains) DiffWithMigrate(ctx context.Context, doc string) (string, []string, []MigrationHint, error) {
	res, err := d.ApplyWith(ctx, doc, SourceGraphQL, ApplyOptions{DryRun: true, NoMigrate: true})
	if err != nil {
		return "", nil, nil, err
	}
	// Recompute hints against tip (NoMigrate skips the gate but we still want them for the UI).
	hints := res.MigrateHints
	if len(hints) == 0 && res.Domain.App != "" {
		prev, next, herr := d.specsForDoc(ctx, doc)
		if herr == nil {
			hints = dataMigrationNeeded(prev, next)
		}
	}
	return res.Domain.App, res.Changes, hints, nil
}

func (d *Domains) specsForDoc(ctx context.Context, doc string) (*appspec.AppSpec, appspec.AppSpec, error) {
	next, err := appspec.Parse([]byte(doc))
	if err != nil {
		return nil, appspec.AppSpec{}, err
	}
	base, _ := d.baseOf(next.Name)
	tip, err := d.store.Tip(ctx, next.Name)
	if err != nil {
		return nil, appspec.AppSpec{}, err
	}
	var prev *appspec.AppSpec
	if tip != nil {
		p, err := d.specFromTip(base, tip)
		if err != nil {
			return nil, appspec.AppSpec{}, err
		}
		prev = &p
	} else if base != nil {
		prev = base
	}
	return prev, next, nil
}

// Apply stores a full tip snapshot for the app named in doc, migrates tables
// (additive unless ApplyWith Force), and hot-mounts. dryRun only validates and diffs.
func (d *Domains) Apply(ctx context.Context, doc, source string, dryRun bool) (*ApplyResult, error) {
	return d.ApplyWith(ctx, doc, source, ApplyOptions{DryRun: dryRun})
}

// ApplyWith is Apply with soft/force control.
func (d *Domains) ApplyWith(ctx context.Context, doc, source string, opts ApplyOptions) (*ApplyResult, error) {
	if source == "" {
		source = SourceGraphQL
	}
	full, err := parseDoc(doc)
	if err != nil {
		return nil, err
	}
	name := full.Name
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	var next appspec.AppSpec
	if base == nil {
		next, err = d.parseDynamic(doc)
	} else {
		next = full
		if err = next.Validate(); err != nil {
			return nil, err
		}
		for _, dep := range next.Depends {
			if !d.loaded(dep) {
				return nil, fmt.Errorf("depends: app %q is not loaded", dep)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		return nil, err
	}
	var prev *appspec.AppSpec
	if tip != nil && tip.Spec != "" {
		if p, perr := d.specFromTip(base, tip); perr == nil {
			prev = &p
		}
	} else if base != nil {
		prev = base
	}
	changes, err := diffSpecs(prev, next)
	if err != nil {
		if !opts.Force {
			return nil, err
		}
		changes = []string{err.Error() + " (--force)"}
	}
	fullDoc, err := appspec.MarshalDomain(next)
	if err != nil {
		return nil, err
	}
	return d.commitTip(ctx, name, base, string(fullDoc), &next, prev, source, changes, opts)
}

// Revert appends a new tip equal to the document at revision (append-only history).
func (d *Domains) Revert(ctx context.Context, name string, revision int64, dryRun bool) (*ApplyResult, error) {
	rec, err := d.store.Get(ctx, name, revision)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("app %q has no revision %d", name, revision)
	}
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	next, err := d.specFromTip(base, rec)
	if err != nil {
		return nil, err
	}
	fullDoc, err := appspec.MarshalDomain(next)
	if err != nil {
		return nil, err
	}
	tip, _ := d.store.Tip(ctx, name)
	var prev *appspec.AppSpec
	if tip != nil {
		if p, perr := d.specFromTip(base, tip); perr == nil {
			prev = &p
		}
	}
	changes := []string{fmt.Sprintf("restore tip from revision %d", revision)}
	return d.commitTip(ctx, name, base, string(fullDoc), &next, prev, fmt.Sprintf("revert:%d", revision), changes, ApplyOptions{DryRun: dryRun})
}

// Delete soft-clears the tip: compiled apps remount code; dynamic apps unmount.
// Tables and revision history are kept.
func (d *Domains) Delete(ctx context.Context, name string, dryRun bool) (*ApplyResult, error) {
	base, err := d.baseOf(name)
	if err != nil {
		return nil, err
	}
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		return nil, err
	}
	if tip == nil && base == nil {
		return nil, fmt.Errorf("app %q not found", name)
	}
	var prev *appspec.AppSpec
	if tip != nil {
		if p, perr := d.specFromTip(base, tip); perr == nil {
			prev = &p
		}
	}
	if base != nil {
		fullDoc, merr := appspec.MarshalDomain(*base)
		if merr != nil {
			return nil, merr
		}
		return d.commitTip(ctx, name, base, string(fullDoc), base, prev, SourceGraphQL, []string{"clear tip"}, ApplyOptions{DryRun: dryRun})
	}
	return d.commitTip(ctx, name, nil, "", nil, prev, SourceGraphQL, []string{"delete app " + name + " (tables kept)"}, ApplyOptions{DryRun: dryRun})
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
	Spec     string `yaml:"spec"`
	Layer    string `yaml:"layer,omitempty"` // legacy export key
}

// Export returns a YAML bundle with the tip snapshot of every app that has
// one (or only of names). Import restores it.
func (d *Domains) Export(ctx context.Context, names []string) (string, error) {
	tips, err := d.store.Tips(ctx)
	if err != nil {
		return "", err
	}
	want := map[string]bool{}
	for _, n := range names {
		if _, ok := tips[n]; !ok {
			if _, compiled := d.reg.Get(n); !compiled {
				return "", fmt.Errorf("app %q not found", n)
			}
		}
		want[n] = true
	}
	b := domainBundle{Kind: bundleKind, ExportedAt: time.Now().UTC().Truncate(time.Second), Domains: []bundleEntry{}}
	for _, name := range sortedKeys(tips) {
		tip := tips[name]
		if (len(want) > 0 && !want[name]) || strings.TrimSpace(tip.Spec) == "" {
			continue
		}
		// Skip pure compiled tips that match code (nothing runtime to export).
		if tip.Source == SourceCompiled {
			continue
		}
		b.Domains = append(b.Domains, bundleEntry{App: name, Revision: tip.Revision, Spec: tip.Spec})
	}
	out, err := yaml.Marshal(b)
	return string(out), err
}

// Import restores a bundle from Export: each app's tip is applied as a new revision.
func (d *Domains) Import(ctx context.Context, bundle string, dryRun bool) ([]ApplyResult, error) {
	var b domainBundle
	if err := yaml.Unmarshal([]byte(bundle), &b); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	if b.Kind != bundleKind {
		return nil, fmt.Errorf("bundle kind must be %q", bundleKind)
	}
	out := make([]ApplyResult, 0, len(b.Domains))
	for _, e := range b.Domains {
		doc := e.Spec
		if doc == "" {
			doc = e.Layer
		}
		res, err := d.Apply(ctx, doc, SourceRestore, true)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.App, err)
		}
		out = append(out, *res)
	}
	if dryRun {
		return out, nil
	}
	out = out[:0]
	for _, e := range b.Domains {
		doc := e.Spec
		if doc == "" {
			doc = e.Layer
		}
		res, err := d.Apply(ctx, doc, SourceRestore, false)
		if err != nil {
			return out, fmt.Errorf("%s: %w", e.App, err)
		}
		out = append(out, *res)
	}
	return out, nil
}

// Exists reports whether name is a compiled app or has a stored tip.
func (d *Domains) Exists(ctx context.Context, name string) (bool, error) {
	if _, ok := d.reg.Get(name); ok {
		return true, nil
	}
	tip, err := d.store.Tip(ctx, name)
	return err == nil && tip != nil && tip.Spec != "", err
}

// List returns every compiled app with a spec, then every dynamic app.
func (d *Domains) List(ctx context.Context) ([]DomainInfo, error) {
	tips, err := d.store.Tips(ctx)
	if err != nil {
		return nil, err
	}
	var out []DomainInfo
	for _, name := range d.reg.Names() {
		raw, _ := d.reg.Get(name)
		if _, ok := raw.(*App); !ok {
			continue
		}
		out = append(out, d.describe(name, tips[name]))
	}
	for _, name := range sortedKeys(tips) {
		tip := tips[name]
		if _, compiled := d.reg.Get(name); compiled || tip.Spec == "" {
			continue
		}
		out = append(out, d.describe(name, tip))
	}
	return out, nil
}

// Get returns one app's domain, or nil when name is unknown.
func (d *Domains) Get(ctx context.Context, name string) (*DomainInfo, error) {
	ok, err := d.Exists(ctx, name)
	if err != nil || !ok {
		return nil, err
	}
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		return nil, err
	}
	var rec app.DomainRevision
	if tip != nil {
		rec = *tip
	}
	info := d.describe(name, rec)
	return &info, nil
}

func (d *Domains) describe(name string, tip app.DomainRevision) DomainInfo {
	base, err := d.baseOf(name)
	var eff *appspec.AppSpec
	switch {
	case err != nil:
	case tip.Spec != "":
		if s, e := d.specFromTip(base, &tip); e == nil {
			eff = &s
		} else {
			err = e
		}
	case base != nil:
		if a := d.compiledApp(name); a != nil {
			s := a.Spec()
			eff = &s
		} else {
			eff = base
		}
	}
	info := d.info(name, base, tip, eff)
	if err != nil {
		info.Error = err.Error()
	}
	return info
}

func (d *Domains) info(name string, base *appspec.AppSpec, tip app.DomainRevision, eff *appspec.AppSpec) DomainInfo {
	info := DomainInfo{
		App: name, Kind: DomainDynamic,
		Models: []string{}, Queries: []string{}, Commands: []string{},
	}
	if tip.Revision != 0 {
		info.Revision = tip.Revision
		info.LayerRevision = tip.Revision
		info.Source = tip.Source
		info.UpdatedAt = tip.CreatedAt
		info.Layer = tip.Spec
		info.Spec = tip.Spec
	}
	if base != nil {
		info.Kind = DomainCompiled
		if info.Source == "" {
			info.Source = SourceCompiled
		}
		info.Loaded = d.compiledApp(name) != nil
		if tip.Layer == app.LayerBase || tip.Source == SourceCompiled {
			info.BaseRevision = tip.Revision
		}
	} else {
		_, info.Loaded = AppFromHost(d.host, name)
	}
	if tip.Layer == app.LayerOverlay && tip.Spec != "" {
		// Legacy: Spec in DomainInfo is effective document.
		if eff != nil {
			if b, err := appspec.MarshalDomain(*eff); err == nil {
				info.Spec = string(b)
			}
		}
	} else if tip.Spec != "" {
		info.Spec = tip.Spec
	} else if eff != nil {
		if b, err := appspec.MarshalDomain(*eff); err == nil {
			info.Spec = string(b)
		}
	}
	if eff != nil {
		info.Title, info.Summary = eff.Title, eff.Summary
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
	if p, ok := d.pendingOf(name); ok {
		info.Incoming = p.Incoming
		info.Conflicts = append([]appspec.Conflict(nil), p.Conflicts...)
		if len(p.Conflicts) > 0 {
			info.RebaseStatus = "conflict"
		} else {
			info.RebaseStatus = "needs_rebase"
		}
	}
	if _, _, ok := d.PendingMigrate(name); ok {
		info.MigratePending = true
	}
	return info
}

// commitTip stores the tip snapshot (unless unchanged), mounts, runs optional
// data migration, and optionally force-drops.
func (d *Domains) commitTip(ctx context.Context, name string, base *appspec.AppSpec, doc string, next, prev *appspec.AppSpec, source string, changes []string, opts ApplyOptions) (*ApplyResult, error) {
	tip, err := d.store.Tip(ctx, name)
	if err != nil {
		return nil, err
	}
	current := ""
	if tip != nil {
		current = tip.Spec
	}
	if base == nil && doc == "" && current == "" {
		return nil, fmt.Errorf("app %q not found", name)
	}
	unchanged := doc == current && next != nil && prev != nil
	if unchanged {
		changes = []string{"no changes"}
	}
	var hints []MigrationHint
	if !unchanged && next != nil && prev != nil {
		hints = dataMigrationNeeded(prev, *next)
	}
	needsMigrate := len(hints) > 0
	hasPre := strings.TrimSpace(opts.MigratePre) != ""
	hasPost := strings.TrimSpace(opts.MigratePost) != "" || strings.TrimSpace(opts.MigrateGo) != ""
	hasMigrate := hasPre || hasPost
	if needsMigrate && !opts.NoMigrate && !hasMigrate {
		d.setPendingMigrate(name, doc, hints)
		info := d.info(name, base, tipOr(tip, doc, source), next)
		if next != nil && doc != "" {
			info.Spec = doc
			info.Layer = doc
		}
		info.MigratePending = true
		changes = append([]string{"migration required — open Settings → Domains"}, changes...)
		return &ApplyResult{
			Domain: info, Changes: changes, DryRun: true,
			MigrateRequired: true, MigrateHints: hints, ProposedSpec: doc,
		}, nil
	}
	if hasPre {
		changes = append(changes, "migrate-pre")
	}
	if strings.TrimSpace(opts.MigrateGo) != "" {
		changes = append(changes, "migrate-go "+strings.TrimSpace(opts.MigrateGo))
	} else if strings.TrimSpace(opts.MigratePost) != "" {
		changes = append(changes, "migrate-post")
	}
	res := &ApplyResult{Changes: changes, DryRun: opts.DryRun}
	if opts.DryRun {
		res.Domain = d.info(name, base, tipOr(tip, doc, source), next)
		if next != nil && doc != "" {
			res.Domain.Spec = doc
			res.Domain.Layer = doc
		}
		return res, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	schema := schemaOf(base, next)
	var p appspec.AppSpec
	if prev != nil {
		p = *prev
	}
	// Pre migrate (e.g. RENAME) → additive DDL → post migrate / Go → force drops.
	if hasPre && next != nil {
		if err := d.runDomainMigratePre(ctx, name, schema, p, *next, opts); err != nil {
			return nil, err
		}
	}
	if err := d.mountLocked(name, base, next); err != nil {
		return nil, err
	}
	if hasPost && next != nil {
		if err := d.runDomainMigratePost(ctx, name, schema, p, *next, opts); err != nil {
			return nil, err
		}
	}
	if opts.Force && prev != nil && next != nil {
		if err := d.forceDrop(ctx, schemaOf(base, next), *prev, *next); err != nil {
			return nil, fmt.Errorf("force drop: %w", err)
		}
	}
	if doc != current {
		rec, err := d.store.Append(ctx, app.DomainRevision{
			App: name, Layer: app.LayerRevision, Spec: doc, Source: source, Changes: changes,
		})
		if err != nil {
			return nil, err
		}
		tip = &rec
	}
	d.clearPendingMigrate(name)
	d.setError(name, "")
	res.Domain = d.info(name, base, tipOr(tip, doc, source), next)
	return res, nil
}

func tipOr(tip *app.DomainRevision, doc, source string) app.DomainRevision {
	if tip != nil {
		return *tip
	}
	return app.DomainRevision{Spec: doc, Source: source, Layer: app.LayerRevision}
}

func schemaOf(base, next *appspec.AppSpec) string {
	if next != nil && next.Schema != "" {
		return next.Schema
	}
	if next != nil {
		return next.Name
	}
	if base != nil && base.Schema != "" {
		return base.Schema
	}
	if base != nil {
		return base.Name
	}
	return ""
}

// specFromTip builds the effective AppSpec from a stored tip (full snapshot or legacy overlay).
func (d *Domains) specFromTip(base *appspec.AppSpec, tip *app.DomainRevision) (appspec.AppSpec, error) {
	if tip == nil || strings.TrimSpace(tip.Spec) == "" {
		if base != nil {
			return *base, nil
		}
		return appspec.AppSpec{}, fmt.Errorf("empty tip")
	}
	if tip.Layer == app.LayerOverlay {
		return d.effective(base, tip.Spec)
	}
	if base == nil {
		return d.parseDynamic(tip.Spec)
	}
	spec, err := appspec.Parse([]byte(tip.Spec))
	if err != nil {
		return appspec.AppSpec{}, err
	}
	if err := spec.Validate(); err != nil {
		// Legacy mislabeled overlay stored as revision: try merge.
		if merged, merr := d.effective(base, tip.Spec); merr == nil {
			return merged, nil
		}
		return appspec.AppSpec{}, err
	}
	for _, dep := range spec.Depends {
		if !d.loaded(dep) {
			return appspec.AppSpec{}, fmt.Errorf("depends: app %q is not loaded", dep)
		}
	}
	return spec, nil
}

func (d *Domains) forceDrop(ctx context.Context, schema string, prev, next appspec.AppSpec) error {
	db, err := postgres.FromHost(d.host)
	if err != nil || db == nil || db.Pool() == nil {
		return nil // no DB in unit tests
	}
	if schema == "" {
		schema = next.Name
		if schema == "" {
			schema = prev.Name
		}
	}
	return forceDropRemoved(ctx, db.Pool(), schema, prev, next)
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
// self-contained app.yaml. File includes, Go hook extends, and compiled
// React component/view exports are rejected. Nav, menus, locales, keymap,
// filters, charts, and security are inline.
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
	case len(spec.Extends) > 0:
		return appspec.AppSpec{}, fmt.Errorf("extends: Go hook bindings are not part of a domain document (bind them from an external Go app)")
	case len(spec.Exports.Components)+len(spec.Exports.Views) > 0:
		return appspec.AppSpec{}, fmt.Errorf("exports.components/views: compiled React modules are not part of a domain document")
	case hasVirtual(spec):
		return appspec.AppSpec{}, fmt.Errorf("virtual models need a Go RegisterModel implementation")
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
	a.syncChrome(d.host)
	if clash := d.replacedFields(snap); len(clash) > 0 {
		return fmt.Errorf("GraphQL fields already defined by another app: %s", strings.Join(clash, ", "))
	}
	sec := next.Security
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
	d.host.RemoveNav(name)
	extension.ReplaceMenuContributions(name, nil)
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
