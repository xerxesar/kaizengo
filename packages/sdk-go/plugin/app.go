// Package plugin is the SDK for external Go apps: separate binaries the
// KaizenGo host launches with hashicorp/go-plugin and talks to over gRPC.
//
// An external app declares domain documents, hooks, and handlers, then calls
// Serve from main. Hooks and handlers reach platform services (ACL, principal,
// records, dynamic domains) through HookCtx.Host / HandlerCtx.Host.
package plugin

import "context"

// Kind is the CQRS kind of a handler.
type Kind string

const (
	Query   Kind = "query"
	Command Kind = "command"
)

// App is an external app's manifest and implementation.
type App struct {
	Name    string
	Version string
	// Domains are app.yaml documents applied through the dynamic-domain path
	// (source "plugin:<Name>") when the host starts the plugin.
	Domains  []string
	Hooks    []Hook
	Handlers []Handler
}

// Hook runs on a model lifecycle point, e.g. model.notes.note.beforeCreate or
// model.*.*.afterUpdate. Before-hooks may change Fields or return an error to
// reject the write; after-hook errors are logged by the host.
type Hook struct {
	Point    string
	Priority int
	Fn       func(*HookCtx) error
}

// Handler implements a query or command declared without list/get/create/
// update/delete shorthand in one of the app's domains.
type Handler struct {
	App  string
	Kind Kind
	Name string
	Fn   func(*HandlerCtx) (any, error)
}

// Caller is the user a hook or handler runs for.
type Caller struct {
	OrgID  string
	UserID string
}

// CallerInfo returns c; it lets HookCtx and HandlerCtx satisfy Call.
func (c Caller) CallerInfo() Caller { return c }

// Call is a request context that carries its caller (HookCtx, HandlerCtx).
type Call interface {
	context.Context
	CallerInfo() Caller
}

// HookCtx is passed to Hook.Fn.
type HookCtx struct {
	context.Context
	Caller
	Point    string
	App      string
	Model    string
	RecordID string
	// Fields is the write payload; mutate it in before-hooks.
	Fields map[string]any
	// Record is the stored record for update/delete phases.
	Record map[string]any
	Host   *HostClient
}

// HandlerCtx is passed to Handler.Fn.
type HandlerCtx struct {
	context.Context
	Caller
	Args map[string]any
	Host *HostClient
}
