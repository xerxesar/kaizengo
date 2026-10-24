// Package shell implements the KaizenGo CLI grammar. Every command compiles to
// GraphQL operations run through an Executor, so the web terminal, command
// palette, and native kaizen client share one implementation and the same ACL.
package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Executor runs a GraphQL document as the calling user.
type Executor interface {
	Exec(ctx context.Context, query string, vars map[string]any) (map[string]any, error)
}

// Result is the outcome of one command line.
type Result struct {
	OK     bool
	Output string
	Data   any
}

type command struct {
	usage   string
	summary string
	run     func(s *session, args []string) (string, any, error)
}

type session struct {
	ctx context.Context
	ex  Executor
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"help":     {"help [command]", "Show commands or one command's usage", cmdHelp},
		"whoami":   {"whoami", "Current user, org, and roles", cmdWhoami},
		"apps":     {"apps", "Apps with callable operations", cmdApps},
		"ls":       {"ls [app]", "List queries and commands", cmdLs},
		"describe": {"describe <app.op>", "Arguments and return fields of an operation", cmdDescribe},
		"query":    {"query <app.op> [key=value …] [--fields a,b] [--all] [--json]", "Run a query", cmdInvoke("query", "query")},
		"run":      {"run <app.op> [key=value …] [--fields a,b] [--all] [--json]", "Run a command", cmdInvoke("command", "run")},
		"domain":   {domainUsage, "Show, apply, revert, back up, and restore app domains", cmdDomain},
		"plugins":  {"plugins", "External Go apps loaded over gRPC", cmdPlugins},
		"gql":      {"gql <document> [variables-json]", "Run raw GraphQL", cmdGQL},
	}
}

// Commands returns command names with usage, for client-side completion and help.
func Commands() []CommandInfo {
	out := make([]CommandInfo, 0, len(commands))
	for name, c := range commands {
		out = append(out, CommandInfo{Name: name, Usage: c.usage, Summary: c.summary})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// CommandInfo describes one shell command.
type CommandInfo struct {
	Name    string
	Usage   string
	Summary string
}

// Run parses and executes one command line (optionally with a heredoc).
func Run(ctx context.Context, ex Executor, input string) Result {
	tokens, err := split(strings.TrimSpace(input))
	if err != nil {
		return fail(err)
	}
	if len(tokens) == 0 {
		return Result{OK: true}
	}
	c, ok := commands[tokens[0]]
	if !ok {
		return fail(fmt.Errorf("unknown command %q (try help)", tokens[0]))
	}
	out, data, err := c.run(&session{ctx: ctx, ex: ex}, tokens[1:])
	if err != nil {
		return fail(err)
	}
	return Result{OK: true, Output: out, Data: data}
}

func fail(err error) Result {
	return Result{OK: false, Output: "error: " + err.Error()}
}

func cmdHelp(_ *session, args []string) (string, any, error) {
	if len(args) > 0 {
		c, ok := commands[args[0]]
		if !ok {
			return "", nil, fmt.Errorf("unknown command %q", args[0])
		}
		return c.usage + "\n  " + c.summary, nil, nil
	}
	var b strings.Builder
	rows := [][]string{}
	for _, c := range Commands() {
		rows = append(rows, []string{c.Name, c.Summary})
	}
	b.WriteString(table([]string{"command", "summary"}, rows))
	b.WriteString("\nValues: key=value (quote spaces: title=\"Big deal\"); lists: tags=a,b or tags='[\"a\",\"b\"]'.")
	return b.String(), nil, nil
}

func cmdWhoami(s *session, _ []string) (string, any, error) {
	data, err := s.ex.Exec(s.ctx, `{ me { email name orgId roles } }`, nil)
	if err != nil {
		return "", nil, err
	}
	return formatValue(data["me"], nil), data["me"], nil
}

func cmdApps(s *session, _ []string) (string, any, error) {
	ops, err := s.catalog("")
	if err != nil {
		return "", nil, err
	}
	type counts struct {
		queries, commands int
		dynamic           bool
	}
	byApp := map[string]*counts{}
	var names []string
	for _, op := range ops {
		c, ok := byApp[op.App]
		if !ok {
			c = &counts{}
			byApp[op.App] = c
			names = append(names, op.App)
		}
		if op.Kind == "query" {
			c.queries++
		} else {
			c.commands++
		}
		c.dynamic = c.dynamic || op.Dynamic
	}
	rows := [][]string{}
	for _, n := range names {
		c := byApp[n]
		src := "compiled"
		switch {
		case c.dynamic:
			src = "dynamic"
		case n == "graphql":
			src = "root fields"
		}
		rows = append(rows, []string{n, strconv.Itoa(c.queries), strconv.Itoa(c.commands), src})
	}
	return table([]string{"app", "queries", "commands", "source"}, rows), names, nil
}

func cmdLs(s *session, args []string) (string, any, error) {
	app := ""
	if len(args) > 0 {
		app = args[0]
	}
	ops, err := s.catalog(app)
	if err != nil {
		return "", nil, err
	}
	rows := [][]string{}
	for _, op := range ops {
		rows = append(rows, []string{op.App + "." + op.Name, op.Kind, argSummary(op)})
	}
	if len(rows) == 0 {
		return "no operations", ops, nil
	}
	return table([]string{"operation", "kind", "args"}, rows), ops, nil
}

func cmdDescribe(s *session, args []string) (string, any, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: describe <app.op>")
	}
	op, err := s.resolve(args[0])
	if err != nil {
		return "", nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s.%s  (%s, GraphQL %s %s)\n", op.App, op.Name, op.Kind, op.Operation, op.Field)
	rows := [][]string{}
	for _, a := range op.Args {
		req := ""
		if a.Required {
			req = "required"
		}
		rows = append(rows, []string{a.Name, a.Type, req})
	}
	if len(rows) > 0 {
		b.WriteString(table([]string{"arg", "type", ""}, rows))
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "returns %s", op.Returns)
	if len(op.ReturnFields) > 0 {
		fmt.Fprintf(&b, " { %s }", strings.Join(op.ReturnFields, " "))
	}
	return b.String(), op, nil
}

func cmdInvoke(kind, name string) func(*session, []string) (string, any, error) {
	return func(s *session, args []string) (string, any, error) {
		if len(args) == 0 {
			return "", nil, fmt.Errorf("usage: %s", commands[name].usage)
		}
		op, err := s.resolve(args[0])
		if err != nil {
			return "", nil, err
		}
		if op.Kind != kind {
			alt := "run"
			if op.Kind == "query" {
				alt = "query"
			}
			return "", nil, fmt.Errorf("%s.%s is a %s; use %s", op.App, op.Name, op.Kind, alt)
		}
		inv, err := parseInvocation(op, args[1:])
		if err != nil {
			return "", nil, err
		}
		doc, vars := inv.document(op)
		data, err := s.ex.Exec(s.ctx, doc, vars)
		if err != nil {
			return "", nil, err
		}
		value := data[op.Field]
		if inv.json {
			return prettyJSON(value), value, nil
		}
		return formatValue(value, inv.columns(op)), value, nil
	}
}

func cmdPlugins(s *session, _ []string) (string, any, error) {
	data, err := s.ex.Exec(s.ctx, `{ platformPlugins { name version path apps hooks handlers error } }`, nil)
	if err != nil {
		return "", nil, err
	}
	list, _ := data["platformPlugins"].([]any)
	if len(list) == 0 {
		return "no plugins loaded (set KaizenGo_PLUGINS)", list, nil
	}
	return formatValue(list, []string{"name", "version", "apps", "hooks", "handlers", "error"}), list, nil
}

func cmdGQL(s *session, args []string) (string, any, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: gql <document> [variables-json]")
	}
	var vars map[string]any
	if len(args) > 1 {
		if err := json.Unmarshal([]byte(args[1]), &vars); err != nil {
			return "", nil, fmt.Errorf("variables: %w", err)
		}
	}
	data, err := s.ex.Exec(s.ctx, args[0], vars)
	if err != nil {
		return "", nil, err
	}
	return prettyJSON(data), data, nil
}

func changeList(title string, raw any) string {
	var b strings.Builder
	b.WriteString(title)
	list, _ := raw.([]any)
	for _, c := range list {
		fmt.Fprintf(&b, "\n  • %v", c)
	}
	return b.String()
}
