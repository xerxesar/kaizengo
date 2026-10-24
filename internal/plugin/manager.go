// Package plugin hosts external Go apps: binaries launched with
// hashicorp/go-plugin that contribute domain documents, model hooks, and
// handlers over gRPC (contract in packages/sdk-go/plugin/proto).
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"kaizengo/internal/engine"
	"kaizengo/internal/extension"
	"kaizengo/internal/module"
	sdkplugin "kaizengo/packages/sdk-go/plugin"
	pluginpb "kaizengo/packages/sdk-go/plugin/proto"
)

// EnvPlugins lists plugin binaries (comma-separated) started on each platform build.
const EnvPlugins = "KaizenGo_PLUGINS"

// ManagerKey is the host bag key for *Manager.
const ManagerKey = "platform.plugins"

// Info describes one plugin process for platformPlugins / the CLI.
type Info struct {
	Name     string
	Version  string
	Path     string
	Apps     []string
	Hooks    []string
	Handlers []string
	Error    string
}

// Manager starts plugin binaries for one platform (database) and binds what
// they describe into the engine. Stop undoes the bindings and kills processes.
type Manager struct {
	host *module.Host
	doms *engine.Domains

	mu    sync.Mutex
	procs []*proc
}

type proc struct {
	info     Info
	client   *goplugin.Client
	rpc      *sdkplugin.Client
	removers []func()
	handlers [][3]string
}

// NewManager wires a manager for host. doms applies plugin domains and late-binds handlers.
func NewManager(host *module.Host, doms *engine.Domains) *Manager {
	return &Manager{host: host, doms: doms}
}

// FromHost returns the manager provided on host, if any.
func FromHost(host *module.Host) *Manager {
	if host == nil {
		return nil
	}
	raw, ok := host.Lookup(ManagerKey)
	if !ok {
		return nil
	}
	m, _ := raw.(*Manager)
	return m
}

// PathsFromEnv parses KaizenGo_PLUGINS.
func PathsFromEnv() []string {
	var out []string
	for _, p := range strings.Split(os.Getenv(EnvPlugins), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Start launches every binary. A plugin that fails to start or describe is
// recorded with its error (see List) and does not block the others.
func (m *Manager) Start(ctx context.Context, paths []string) {
	for _, path := range paths {
		p, err := m.start(ctx, path)
		if err != nil {
			m.host.Log.Error("plugin failed", "path", path, "err", err)
			if p == nil {
				p = &proc{info: Info{Name: filepath.Base(path), Path: path}}
			}
			p.info.Error = err.Error()
			m.unbind(p)
		} else {
			m.host.Log.Info("plugin loaded", "name", p.info.Name, "version", p.info.Version,
				"apps", p.info.Apps, "hooks", len(p.info.Hooks), "handlers", len(p.info.Handlers))
		}
		m.mu.Lock()
		m.procs = append(m.procs, p)
		m.mu.Unlock()
	}
}

// Stop unbinds hooks and handlers and kills all plugin processes.
func (m *Manager) Stop() {
	m.mu.Lock()
	procs := m.procs
	m.procs = nil
	m.mu.Unlock()
	for _, p := range procs {
		m.unbind(p)
	}
}

// List returns loaded plugins in start order.
func (m *Manager) List() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Info, len(m.procs))
	for i, p := range m.procs {
		out[i] = p.info
	}
	return out
}

func (m *Manager) start(ctx context.Context, path string) (*proc, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  sdkplugin.Handshake,
		Plugins:          goplugin.PluginSet{sdkplugin.PluginName: &sdkplugin.GRPCPlugin{}},
		Cmd:              exec.Command(abs),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		StartTimeout:     15 * time.Second,
		Logger: hclog.New(&hclog.LoggerOptions{
			Name: "plugin", Level: hclog.Info, Output: os.Stderr,
		}),
	})
	p := &proc{info: Info{Name: filepath.Base(abs), Path: abs}, client: client}

	conn, err := client.Client()
	if err != nil {
		return p, fmt.Errorf("start: %w", err)
	}
	raw, err := conn.Dispense(sdkplugin.PluginName)
	if err != nil {
		return p, fmt.Errorf("dispense: %w", err)
	}
	rpc, ok := raw.(*sdkplugin.Client)
	if !ok {
		return p, fmt.Errorf("unexpected plugin client %T", raw)
	}
	p.rpc = rpc

	brokerID := rpc.Broker.NextId()
	go rpc.Broker.AcceptAndServe(brokerID, func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		pluginpb.RegisterHostServer(s, &hostServer{host: m.host, doms: m.doms, source: "plugin:" + p.info.Name})
		return s
	})

	desc, err := rpc.Plugin.Describe(ctx, &pluginpb.DescribeRequest{HostBrokerId: brokerID})
	if err != nil {
		return p, fmt.Errorf("describe: %w", err)
	}
	if desc.GetName() != "" {
		p.info.Name = desc.GetName()
	}
	p.info.Version = desc.GetVersion()

	for i, doc := range desc.GetDomains() {
		res, err := m.doms.Apply(ctx, doc, "plugin:"+p.info.Name, false)
		if err != nil {
			return p, fmt.Errorf("domain %d: %w", i, err)
		}
		p.info.Apps = append(p.info.Apps, res.Domain.App)
	}
	for i, h := range desc.GetHandlers() {
		kind := h.GetKind()
		if kind != engine.KindQuery && kind != engine.KindCommand {
			return p, fmt.Errorf("handler %s.%s: kind must be query or command, got %q", h.GetApp(), h.GetName(), kind)
		}
		m.doms.BindHandler(h.GetApp(), kind, h.GetName(), handlerFunc(rpc, int32(i)))
		p.handlers = append(p.handlers, [3]string{h.GetApp(), kind, h.GetName()})
		p.info.Handlers = append(p.info.Handlers, fmt.Sprintf("%s %s.%s", kind, h.GetApp(), h.GetName()))
	}
	for i, h := range desc.GetHooks() {
		remove := extension.RegisterScoped(h.GetPoint(), int(h.GetPriority()), hookFunc(rpc, int32(i)))
		p.removers = append(p.removers, remove)
		p.info.Hooks = append(p.info.Hooks, h.GetPoint())
	}
	return p, nil
}

func (m *Manager) unbind(p *proc) {
	for _, remove := range p.removers {
		remove()
	}
	p.removers = nil
	for _, h := range p.handlers {
		m.doms.UnbindHandler(h[0], h[1], h[2])
	}
	p.handlers = nil
	if p.client != nil {
		p.client.Kill()
	}
}

func hookFunc(rpc *sdkplugin.Client, id int32) func(extension.Context) error {
	return func(ec extension.Context) error {
		ctx := ec.Context
		if ctx == nil {
			ctx = context.Background()
		}
		fields, err := json.Marshal(ec.Fields)
		if err != nil {
			return err
		}
		record, err := json.Marshal(ec.Record)
		if err != nil {
			return err
		}
		res, err := rpc.Plugin.Hook(ctx, &pluginpb.HookRequest{
			HookId:     id,
			Point:      ec.Point,
			Caller:     &pluginpb.Caller{OrgId: ec.OrgID, UserId: ec.UserID},
			App:        ec.App.Name,
			Model:      ec.Model.Name,
			RecordId:   ec.RecordID,
			FieldsJson: fields,
			RecordJson: record,
		})
		if err != nil {
			return fmt.Errorf("plugin hook %s: %w", ec.Point, err)
		}
		if msg := res.GetError(); msg != "" {
			return errors.New(msg)
		}
		if !isBeforePoint(ec.Point) || ec.Fields == nil || len(res.GetFieldsJson()) == 0 {
			return nil
		}
		next := map[string]any{}
		if err := json.Unmarshal(res.GetFieldsJson(), &next); err != nil {
			return err
		}
		for k := range ec.Fields {
			delete(ec.Fields, k)
		}
		for k, v := range next {
			ec.Fields[k] = v
		}
		return nil
	}
}

func isBeforePoint(point string) bool {
	i := strings.LastIndex(point, ".")
	return strings.HasPrefix(point[i+1:], "before")
}

func handlerFunc(rpc *sdkplugin.Client, id int32) engine.QueryHandler {
	return func(hc engine.HandlerCtx, args map[string]any) (any, error) {
		ctx := hc.Context
		if ctx == nil {
			ctx = context.Background()
		}
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		res, err := rpc.Plugin.Handle(ctx, &pluginpb.HandleRequest{
			HandlerId: id,
			Caller:    &pluginpb.Caller{OrgId: hc.OrgID, UserId: hc.UserID},
			ArgsJson:  raw,
		})
		if err != nil {
			return nil, fmt.Errorf("plugin handler: %w", err)
		}
		if msg := res.GetError(); msg != "" {
			return nil, errors.New(msg)
		}
		var out any
		if len(res.GetResultJson()) > 0 {
			if err := json.Unmarshal(res.GetResultJson(), &out); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
}
