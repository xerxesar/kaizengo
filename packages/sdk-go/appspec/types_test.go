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
