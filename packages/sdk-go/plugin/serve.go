package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pluginpb "kaizengo/packages/sdk-go/plugin/proto"
)

// Handshake must match between host and plugin; bump ProtocolVersion on
// incompatible proto changes.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "KAIZENGO_PLUGIN",
	MagicCookieValue: "kaizengo-external-app",
}

// PluginName is the go-plugin dispense name.
const PluginName = "app"

// GRPCPlugin wires App into go-plugin. The host uses it with a nil App.
type GRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	App *App
}

// Client is what the host gets from Dispense.
type Client struct {
	Plugin pluginpb.PluginClient
	Broker *goplugin.GRPCBroker
}

func (p *GRPCPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginpb.RegisterPluginServer(s, &server{app: p.App, broker: broker})
	return nil
}

func (p *GRPCPlugin) GRPCClient(_ context.Context, broker *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return &Client{Plugin: pluginpb.NewPluginClient(conn), Broker: broker}, nil
}

// Log writes to the host's log. go-plugin only keeps levels for JSON lines on
// stderr; plain stderr output is logged by the host at debug level.
var Log = hclog.New(&hclog.LoggerOptions{Output: os.Stderr, JSONFormat: true, Level: hclog.Info})

// Serve runs app as a plugin. It blocks until the host stops the process.
func Serve(app *App) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         goplugin.PluginSet{PluginName: &GRPCPlugin{App: app}},
		GRPCServer:      goplugin.DefaultGRPCServer,
		Logger:          Log,
	})
}

type server struct {
	pluginpb.UnimplementedPluginServer
	app    *App
	broker *goplugin.GRPCBroker

	mu   sync.Mutex
	host *HostClient
}

func (s *server) Describe(_ context.Context, req *pluginpb.DescribeRequest) (*pluginpb.DescribeResponse, error) {
	conn, err := s.broker.Dial(req.GetHostBrokerId())
	if err != nil {
		return nil, fmt.Errorf("dial host: %w", err)
	}
	s.mu.Lock()
	s.host = &HostClient{rpc: pluginpb.NewHostClient(conn)}
	s.mu.Unlock()

	out := &pluginpb.DescribeResponse{Name: s.app.Name, Version: s.app.Version, Domains: s.app.Domains}
	for _, h := range s.app.Hooks {
		out.Hooks = append(out.Hooks, &pluginpb.HookBinding{Point: h.Point, Priority: int32(h.Priority)})
	}
	for _, h := range s.app.Handlers {
		out.Handlers = append(out.Handlers, &pluginpb.HandlerBinding{App: h.App, Kind: string(h.Kind), Name: h.Name})
	}
	return out, nil
}

func (s *server) hostClient() *HostClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.host
}

func (s *server) Hook(ctx context.Context, req *pluginpb.HookRequest) (*pluginpb.HookResponse, error) {
	id := int(req.GetHookId())
	if id < 0 || id >= len(s.app.Hooks) {
		return nil, fmt.Errorf("unknown hook %d", id)
	}
	c := &HookCtx{
		Context:  ctx,
		Caller:   callerFrom(req.GetCaller()),
		Point:    req.GetPoint(),
		App:      req.GetApp(),
		Model:    req.GetModel(),
		RecordID: req.GetRecordId(),
		Host:     s.hostClient(),
	}
	if err := decodeMap(req.GetFieldsJson(), &c.Fields); err != nil {
		return nil, err
	}
	if err := decodeMap(req.GetRecordJson(), &c.Record); err != nil {
		return nil, err
	}
	if err := s.app.Hooks[id].Fn(c); err != nil {
		return &pluginpb.HookResponse{Error: err.Error()}, nil
	}
	fields, err := json.Marshal(c.Fields)
	if err != nil {
		return nil, err
	}
	return &pluginpb.HookResponse{FieldsJson: fields}, nil
}

func (s *server) Handle(ctx context.Context, req *pluginpb.HandleRequest) (*pluginpb.HandleResponse, error) {
	id := int(req.GetHandlerId())
	if id < 0 || id >= len(s.app.Handlers) {
		return nil, fmt.Errorf("unknown handler %d", id)
	}
	c := &HandlerCtx{Context: ctx, Caller: callerFrom(req.GetCaller()), Host: s.hostClient()}
	if err := decodeMap(req.GetArgsJson(), &c.Args); err != nil {
		return nil, err
	}
	result, err := s.app.Handlers[id].Fn(c)
	if err != nil {
		return &pluginpb.HandleResponse{Error: err.Error()}, nil
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &pluginpb.HandleResponse{ResultJson: raw}, nil
}

func callerFrom(c *pluginpb.Caller) Caller {
	return Caller{OrgID: c.GetOrgId(), UserID: c.GetUserId()}
}

func decodeMap(raw []byte, out *map[string]any) error {
	*out = map[string]any{}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
