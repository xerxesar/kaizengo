// Command notes is an example external Go app. Build with `make plugins` and
// run the server with KaizenGo_PLUGINS=./bin/plugins/notes.
package main

import (
	"errors"
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/plugin"
)

const domain = `
name: notes
title: Notes
summary: Notes managed by an external Go app
models:
  - name: note
    fields:
      - { name: title, type: string, required: true }
      - { name: body, type: text }
      - { name: pinned, type: bool }
queries:
  - { name: notes, list: note }
  - { name: note, get: note }
  - name: noteCount
    returns: int
commands:
  - { name: addNote, create: note }
  - name: pinNote
    args:
      - { name: id, type: string, required: true }
    returns: note
`

func main() {
	plugin.Serve(&plugin.App{
		Name:    "notes-plugin",
		Version: "0.1.0",
		Domains: []string{domain},
		Hooks: []plugin.Hook{
			{
				Point: "model.notes.note.beforeCreate",
				Fn: func(c *plugin.HookCtx) error {
					title := strings.TrimSpace(fmt.Sprint(c.Fields["title"]))
					existing, err := c.Host.ListRecords(c, "notes", "note")
					if err != nil {
						return err
					}
					for _, n := range existing {
						if strings.EqualFold(fmt.Sprint(n["title"]), title) {
							return fmt.Errorf("notes: a note titled %q already exists", title)
						}
					}
					c.Fields["title"] = title
					return nil
				},
			},
			{
				Point:    "model.*.*.afterCreate",
				Priority: 100,
				Fn: func(c *plugin.HookCtx) error {
					plugin.Log.Info("record created", "app", c.App, "model", c.Model, "id", c.RecordID, "user", c.UserID)
					return nil
				},
			},
		},
		Handlers: []plugin.Handler{
			{
				App: "notes", Kind: plugin.Query, Name: "noteCount",
				Fn: func(c *plugin.HandlerCtx) (any, error) {
					rows, err := c.Host.ListRecords(c, "notes", "note")
					return len(rows), err
				},
			},
			{
				App: "notes", Kind: plugin.Command, Name: "pinNote",
				Fn: func(c *plugin.HandlerCtx) (any, error) {
					id := fmt.Sprint(c.Args["id"])
					rec, err := c.Host.GetRecord(c, "notes", "note", id)
					if err != nil {
						return nil, err
					}
					d, err := c.Host.Authorize(c, "notes.note", "update", rec, "pinned")
					if err != nil {
						return nil, err
					}
					if !d.Allowed {
						return nil, errors.New("notes: not allowed to pin this note")
					}
					return c.Host.UpdateRecord(c, "notes", "note", id, map[string]any{"pinned": true})
				},
			},
		},
	})
}
