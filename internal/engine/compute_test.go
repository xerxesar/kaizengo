package engine

import (
	"context"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestApplyFieldComputes(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: topic
    fields:
      - { name: name, type: string }
  - name: item
    fields:
      - { name: title, type: string }
      - { name: topicId, type: many2one, relation: topic }
      - { name: label, type: string, compute: 'title + "!"' }
      - { name: topicName, type: string, related: topicId.name }
      - { name: shout, type: string, compute: hook, depends: [title] }
      - { name: preview, type: string, compute: lower(title), store: false }
`))
	if err != nil {
		t.Fatal(err)
	}
	item := spec.Models[1]
	fields := map[string]any{"title": "Hello", "topicId": "t1"}
	hooks := Hooks{Compute: map[string]func(HookContext) (any, error){
		"shout": func(hc HookContext) (any, error) {
			return "HELLO", nil
		},
	}}
	hop := func(_ context.Context, ref appspec.RelationRef, id string) (Record, appspec.ModelSpec, error) {
		if ref.Model != "topic" || id != "t1" {
			t.Fatalf("hop %s %s", ref.Model, id)
		}
		return Record{"name": "News"}, spec.Models[0], nil
	}
	if err := applyFieldComputes(context.Background(), spec, item, fields, nil, "org", hooks, true, hop); err != nil {
		t.Fatal(err)
	}
	if fields["label"] != "Hello!" || fields["topicName"] != "News" || fields["shout"] != "HELLO" {
		t.Fatalf("stored = %#v", fields)
	}
	if _, ok := fields["preview"]; ok {
		t.Fatal("non-stored compute must wait for read")
	}
	if err := applyFieldComputes(context.Background(), spec, item, fields, nil, "org", hooks, false, hop); err != nil {
		t.Fatal(err)
	}
	if fields["preview"] != "hello" {
		t.Fatalf("preview = %#v", fields["preview"])
	}
}
