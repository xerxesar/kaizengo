package appspec_test

import (
	"strings"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestEvalCompute(t *testing.T) {
	fields := map[string]any{"message": "Hello, Ada", "mood": "happy"}
	got, err := appspec.EvalCompute(`message + " · " + mood`, fields)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello, Ada · happy" {
		t.Fatalf("label = %#v", got)
	}
	n, err := appspec.EvalCompute("len(message)", fields)
	if err != nil || n != float64(10) {
		t.Fatalf("len = %#v err %v", n, err)
	}
	low, err := appspec.EvalCompute("lower(message)", fields)
	if err != nil || low != "hello, ada" {
		t.Fatalf("lower = %#v err %v", low, err)
	}
	ids, err := appspec.ExprIdents(`len(message) + mood`)
	if err != nil || strings.Join(ids, ",") != "message,mood" {
		t.Fatalf("idents = %v err %v", ids, err)
	}
}

func TestComputedFieldsAndInverse(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: topic
    fields:
      - name: name
        type: string
      - name: items
        type: one2many
        relation: item
        inverse: topicId
  - name: item
    fields:
      - name: title
        type: string
      - name: topicId
        type: many2one
        relation: topic
        onDelete: set null
      - name: label
        type: string
        compute: title + "!"
      - name: topicName
        type: string
        related: topicId.name
      - name: preview
        type: string
        compute: lower(title)
        store: false
`))
	if err != nil {
		t.Fatal(err)
	}
	item := spec.Models[1]
	label := item.Fields[2]
	if !label.Computed() || !label.StoreValue() || label.Store == nil || !*label.Store {
		t.Fatalf("stored compute = %+v", label)
	}
	if got := strings.Join(label.Depends, ","); got != "title" {
		t.Fatalf("inferred depends = %q", got)
	}
	preview := item.Fields[4]
	if preview.Column() || preview.Stored() {
		t.Fatalf("preview should be read-time only: %+v", preview)
	}
	if _, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: topic
    fields:
      - name: name
        type: string
      - name: items
        type: one2many
        relation: item
        inverse: missing
  - name: item
    fields:
      - name: title
        type: string
`)); err == nil || !strings.Contains(err.Error(), "inverse") {
		t.Fatalf("expected inverse error, got %v", err)
	}
}

func TestIndexWhereDescInclude(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: task
    indexes:
      - name: task_open
        fields: [title:desc]
        include: [note]
        where: "title <> ''"
    fields:
      - name: title
        type: string
      - name: note
        type: text
`))
	if err != nil {
		t.Fatal(err)
	}
	ix := spec.Models[0].Indexes[0]
	if ix.Fields[0] != "title" || ix.OrderAt(0) != "desc" || ix.Include[0] != "note" {
		t.Fatalf("index = %+v", ix)
	}
	sql, err := ix.WhereSQL(func(col string) string { return col })
	if err != nil || sql != `"title" <> ''` {
		t.Fatalf("where sql = %q err %v", sql, err)
	}
	if _, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: task
    indexes:
      - fields: [title]
        method: gin
        where: "title = 'x'"
    fields:
      - name: title
        type: string
`)); err == nil || !strings.Contains(err.Error(), "gin") {
		t.Fatalf("expected gin where error, got %v", err)
	}
}
