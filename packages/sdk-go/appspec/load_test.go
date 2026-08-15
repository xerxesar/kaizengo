package appspec_test

import (
	"os"
	"path/filepath"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestListNamesAt(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("appman/app.yaml", "name: appman\n")
	mustWrite("inventory/app.yaml", "name: inventory\n")
	mustWrite("notes/readme.txt", "not an app")
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	names, err := appspec.ListNamesAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "appman" || names[1] != "inventory" {
		t.Fatalf("ListNamesAt = %v, want [appman inventory]", names)
	}
}

func TestLoadFileModelDirectoriesAndPages(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("app.yaml", `
name: demo
title: Demo
summary: Demo app
models:
  - models/item
  - name: tag
    fields:
      - name: label
        type: string
menus:
  - id: items
    labelKey: demo.menu.items
    view: Items
`)
	mustWrite("models/item/spec.yaml", `
name: item
fields:
  - name: title
    type: string
    required: true
  - name: tagId
    type: many2one
    relation: tag
`)
	mustWrite("views/Items.page.svelte", "<h1>Items</h1>\n")
	mustWrite("views/Extra.svelte", "<p>not a page</p>\n")

	spec, err := appspec.LoadFile(filepath.Join(root, "app.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Models) != 2 || spec.Models[0].Name != "item" || spec.Models[1].Name != "tag" {
		t.Fatalf("models = %+v", spec.Models)
	}
	if len(spec.Views) != 1 || spec.Views[0].Name != "Items" {
		t.Fatalf("pages = %+v", spec.Views)
	}
}

func TestParseRejectsViewsKey(t *testing.T) {
	_, err := appspec.Parse([]byte(`
name: demo
title: Demo
summary: Demo
views:
  - name: Items
    type: page
`))
	if err == nil {
		t.Fatal("expected views: error")
	}
}

func TestLoadFileRejectsUnknownPage(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.yaml")
	if err := os.WriteFile(path, []byte(`
name: demo
title: Demo
summary: Demo
menus:
  - id: items
    labelKey: demo.menu.items
    view: Missing
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := appspec.LoadFile(path)
	if err == nil {
		t.Fatal("expected missing page error")
	}
}
