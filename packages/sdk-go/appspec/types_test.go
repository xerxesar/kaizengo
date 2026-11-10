package appspec_test

import (
	"strings"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestCanonicalFieldType(t *testing.T) {
	cases := map[string]string{
		"":          appspec.TypeString,
		"string":    appspec.TypeString,
		"integer":   appspec.TypeInt,
		"number":    appspec.TypeNumber,
		"float":     appspec.TypeNumber,
		"decimal":   appspec.TypeNumber,
		"boolean":   appspec.TypeBool,
		"timestamp": appspec.TypeDatetime,
		"fk":        appspec.TypeMany2One,
		"relation":  appspec.TypeMany2One,
		"html":      appspec.TypeText,
	}
	for in, want := range cases {
		if got := appspec.CanonicalFieldType(in); got != want {
			t.Errorf("CanonicalFieldType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseAcceptsNumberAndRelations(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: inventory
title: Inventory
summary: Stock
models:
  - name: category
    fields:
      - name: name
        type: string
        required: true
  - name: item
    fields:
      - name: quantity
        type: number
      - name: categoryId
        type: many2one
        relation: category
      - name: tagIds
        type: many2many
        relation: inventory.tag
      - name: moves
        type: one2many
        relation: stock_move
        inverse: itemId
`))
	if err != nil {
		t.Fatal(err)
	}
	item := spec.Models[1]
	if item.Fields[0].Type != appspec.TypeNumber {
		t.Fatalf("quantity type = %q", item.Fields[0].Type)
	}
	if !item.Fields[1].IsRelation() || item.Fields[1].Type != appspec.TypeMany2One {
		t.Fatalf("categoryId = %+v", item.Fields[1])
	}
	if item.Fields[3].Stored() {
		t.Fatal("one2many should not be stored")
	}
	if !item.Fields[2].Stored() || item.Fields[2].Column() {
		t.Fatal("many2many is stored in GraphQL and has no column")
	}
}

func TestParseInternalModel(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: layer
    internal: true
    fields:
      - name: qty
        type: number
`))
	if err != nil {
		t.Fatal(err)
	}
	if !spec.Models[0].Internal {
		t.Fatal("expected internal model")
	}
}

func TestParseCQRS(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: task
    fields:
      - name: title
        type: string
        required: true
queries:
  - name: tasks
    list: task
  - name: task
    get: task
commands:
  - name: postTask
    create: task
  - name: reviseTask
    update: task
  - name: discardTask
    delete: task
`))
	if err != nil {
		t.Fatal(err)
	}
	if !spec.HasCQRS() || len(spec.Queries) != 2 || len(spec.Commands) != 3 {
		t.Fatalf("cqrs parse: %+v", spec)
	}
}

func TestParseCQRSRejectsUnknownModel(t *testing.T) {
	_, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: task
    fields:
      - name: title
        type: string
queries:
  - name: items
    list: missing
`))
	if err == nil || !strings.Contains(err.Error(), "unknown model") {
		t.Fatalf("expected unknown model error, got %v", err)
	}
}

func TestParseRejectsRelationWithoutTarget(t *testing.T) {
	_, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: item
    fields:
      - name: categoryId
        type: many2one
`))
	if err == nil || !strings.Contains(err.Error(), "relation") {
		t.Fatalf("expected relation error, got %v", err)
	}
}

func TestParseViewAndIndex(t *testing.T) {
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
models:
  - name: task
    indexes:
      - fields: [title, note]
        unique: true
    fields:
      - name: title
        type: string
        unique: true
      - name: note
        type: text
views:
  - name: TaskBoard
    kind: collection
    model: task
    defaultView: table
    views: [table, kanban]
    groupBy: [title]
    columns:
      - field: note
        order: 2
      - field: title
        width: 2fr
        order: 1
  - name: TaskForm
    kind: form
    model: task
    groups:
      - label: Details
        columns: 2
        fields:
          - field: title
            span: 2
            widget: textarea
menus:
  - id: board
    label: Board
    view: TaskBoard
`))
	if err != nil {
		t.Fatal(err)
	}
	if !spec.Models[0].Fields[0].Unique || len(spec.Models[0].Indexes) != 1 || !spec.Models[0].Indexes[0].Unique {
		t.Fatalf("index: %+v", spec.Models[0])
	}
	if spec.Models[0].Indexes[0].Method != "btree" || spec.Models[0].Indexes[0].Name == "" {
		t.Fatalf("index defaults: %+v", spec.Models[0].Indexes[0])
	}
	if len(spec.Views) != 2 || spec.Views[0].Columns[0].Order != 2 || spec.Menus[0].View != "TaskBoard" {
		t.Fatalf("views: %+v menus: %+v", spec.Views, spec.Menus)
	}
}
