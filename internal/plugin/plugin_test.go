package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"google.golang.org/grpc"

	"kaizengo/internal/extension"
	"kaizengo/packages/sdk-go/appspec"
	sdkplugin "kaizengo/packages/sdk-go/plugin"
	pluginpb "kaizengo/packages/sdk-go/plugin/proto"
)

type fakePlugin struct {
	pluginpb.PluginClient
	fields map[string]any
	err    string
}

func (f *fakePlugin) Hook(_ context.Context, _ *pluginpb.HookRequest, _ ...grpc.CallOption) (*pluginpb.HookResponse, error) {
	raw, _ := json.Marshal(f.fields)
	return &pluginpb.HookResponse{FieldsJson: raw, Error: f.err}, nil
}

func TestHookReplacesFieldsOnlyBefore(t *testing.T) {
	rpc := &sdkplugin.Client{Plugin: &fakePlugin{fields: map[string]any{"title": "trimmed"}}}
	fn := hookFunc(rpc, 0)

	before := map[string]any{"title": "  trimmed  ", "drop": 1}
	ctx := extension.Context{Point: "model.a.b.beforeCreate", App: appspec.AppSpec{Name: "a"}, Fields: before}
	if err := fn(ctx); err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 || before["title"] != "trimmed" {
		t.Fatalf("before fields = %v", before)
	}

	after := map[string]any{"title": "kept"}
	ctx.Point, ctx.Fields = "model.a.b.afterCreate", after
	if err := fn(ctx); err != nil {
		t.Fatal(err)
	}
	if after["title"] != "kept" {
		t.Fatalf("after fields changed: %v", after)
	}
}

func TestHookError(t *testing.T) {
	rpc := &sdkplugin.Client{Plugin: &fakePlugin{err: "nope"}}
	err := hookFunc(rpc, 0)(extension.Context{Point: "model.a.b.beforeUpdate", Fields: map[string]any{}})
	if err == nil || err.Error() != "nope" {
		t.Fatalf("err = %v", err)
	}
}

func TestPathsFromEnv(t *testing.T) {
	t.Setenv(EnvPlugins, " ./a, ,./b ")
	got := PathsFromEnv()
	if len(got) != 2 || got[0] != "./a" || got[1] != "./b" {
		t.Fatalf("got %v", got)
	}
}
