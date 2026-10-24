package shell

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	sdkgql "kaizengo/internal/gql"
	"kaizengo/internal/module"

	"github.com/graphql-go/graphql"
)

// SchemaExecutor runs documents against the host's live schema with the
// caller's context, so session and ACL checks apply exactly as over HTTP.
type SchemaExecutor struct {
	Host *module.Host
}

func (e SchemaExecutor) Exec(ctx context.Context, query string, vars map[string]any) (map[string]any, error) {
	schema, err := e.Host.GQL.Schema()
	if err != nil {
		return nil, err
	}
	res := graphql.Do(graphql.Params{
		Schema:         *schema,
		RequestString:  query,
		VariableValues: vars,
		Context:        ctx,
	})
	if len(res.Errors) > 0 {
		msgs := make([]string, len(res.Errors))
		for i, e := range res.Errors {
			msgs[i] = e.Message
		}
		return nil, errors.New(strings.Join(msgs, "; "))
	}
	data, _ := res.Data.(map[string]any)
	return data, nil
}

// Register adds platformShell (run one line) and platformShellCommands (completion/help).
// Each inner operation is ACL-checked, so the shell itself only needs a session.
func Register(host *module.Host) {
	ex := SchemaExecutor{Host: host}
	host.GQL.RegisterMutation("platformShell", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewObject(graphql.ObjectConfig{
			Name: "PlatformShellResult",
			Fields: graphql.Fields{
				"ok":     &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
				"output": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"data":   &graphql.Field{Type: graphql.String, Description: "JSON-encoded result data"},
			},
		})),
		Args: graphql.FieldConfigArgument{
			"line": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			line, _ := p.Args["line"].(string)
			res := Run(p.Context, ex, line)
			var data any
			if res.Data != nil {
				if b, err := json.Marshal(res.Data); err == nil {
					data = string(b)
				}
			}
			return map[string]any{"ok": res.OK, "output": res.Output, "data": data}, nil
		},
	})

	host.GQL.RegisterQuery("platformShellCommands", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.NewObject(graphql.ObjectConfig{
			Name: "PlatformShellCommand",
			Fields: graphql.Fields{
				"name":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"usage":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"summary": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			},
		})))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if _, err := sdkgql.RequirePrincipal(p); err != nil {
				return nil, err
			}
			cmds := Commands()
			out := make([]any, 0, len(cmds))
			for _, c := range cmds {
				out = append(out, map[string]any{"name": c.Name, "usage": c.Usage, "summary": c.Summary})
			}
			return out, nil
		},
	})
}
