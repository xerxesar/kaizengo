package codegen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/codegen"
)

func TestGenerateAppTypes(t *testing.T) {
	dir := t.TempDir()
	spec, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo app
models:
  - name: item
    fields:
      - name: title
        type: string
        required: true
      - name: status
        type: enum
        values: [open, closed]
        default: open
      - name: locked
        type: bool
        readonly: true
        default: false
      - name: qty
        type: number
      - name: categoryId
        type: many2one
        relation: category
`))
	if err != nil {
		t.Fatal(err)
	}
	if err := codegen.GenerateAppTypes(spec, dir); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(dir, "__types__", "item.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(src)
	for _, want := range []string{
		"package types",
		"type ItemStatus string",
		"ItemStatusOpen",
		"type Item struct",
		"Title",
		"ItemStatus",
		"Locked",
		"Qty",
		"float64",
		"CategoryId",
		"readonly",
		"default=open",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("generated file missing %q:\n%s", want, out)
		}
	}

	ts, err := os.ReadFile(filepath.Join(dir, "__types__", "item.ts"))
	if err != nil {
		t.Fatal(err)
	}
	tsOut := string(ts)
	for _, want := range []string{
		"export type ItemStatus = \"open\" | \"closed\"",
		"export type Item = {",
		"title: string // required",
		"status: ItemStatus | null // default=open",
		"locked: boolean | null // readonly, default=false",
		"qty: number | null",
		"categoryId: string | null",
		"createdAt: string",
	} {
		if !strings.Contains(tsOut, want) {
			t.Fatalf("generated ts missing %q:\n%s", want, tsOut)
		}
	}
}
