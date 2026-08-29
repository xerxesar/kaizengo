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
	mustWrite("views/Items.page.tsx", "export default function Items() { return null }\n")
	mustWrite("views/Extra.tsx", "export default function Extra() { return null }\n")

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

func TestLoadFileSecurityMerge(t *testing.T) {
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
security:
  - security/roles.yaml
  - security/entries.yaml
`)
	mustWrite("security/roles.yaml", `
roles:
  - name: member
    label: Member
disable:
  - old-grant
`)
	mustWrite("security/entries.yaml", `
entries:
  - name: demo-member-read
    role: member
    effect: allow
    resource: demo.item
    actions: [read]
    fields: [title]
  - name: demo-member-all-fields
    role: member
    resource: demo.item
    actions: [delete]
    fields: "*"
    domain: [[authorId, "=", "$user.id"]]
users:
  - email: demo@example.com
    name: Demo User
    password: secret
    roles: [member]
`)

	spec, err := appspec.LoadFile(filepath.Join(root, "app.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	sec := spec.Security
	if len(sec.Roles) != 1 || sec.Roles[0].Name != "member" {
		t.Fatalf("roles = %+v", sec.Roles)
	}
	if len(sec.Disable) != 1 || sec.Disable[0] != "old-grant" {
		t.Fatalf("disable = %+v", sec.Disable)
	}
	if len(sec.Entries) != 2 {
		t.Fatalf("entries = %+v", sec.Entries)
	}
	if !sec.Entries[1].Fields.All {
		t.Fatalf("expected fields *, got %+v", sec.Entries[1].Fields)
	}
	if len(sec.Entries[1].Domain) != 1 || sec.Entries[1].Domain[0][0] != "authorId" {
		t.Fatalf("domain = %+v", sec.Entries[1].Domain)
	}
	if len(sec.Users) != 1 || sec.Users[0].Email != "demo@example.com" {
		t.Fatalf("users = %+v", sec.Users)
	}
}

func TestLoadFileSecurityRejectsDuplicateEntry(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.yaml")
	if err := os.WriteFile(path, []byte(`
name: demo
title: Demo
summary: Demo
security:
  - security.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "security.yaml"), []byte(`
entries:
  - name: same
    role: member
    resource: demo.item
    actions: [read]
  - name: same
    role: member
    resource: demo.item
    actions: [update]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := appspec.LoadFile(path)
	if err == nil {
		t.Fatal("expected duplicate entry error")
	}
}

func TestLoadFileKeymapMerge(t *testing.T) {
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
keymap:
  - keymap/base.yaml
  - keymap/extra.yaml
`)
	mustWrite("keymap/base.yaml", `
disable:
  - old_save
bindings:
  - id: save
    action: element:save
    keys: mod+s
    labelKey: keymap.save
    scope: view
`)
	mustWrite("keymap/extra.yaml", `
bindings:
  - id: refresh
    action: element:refresh
    keys: mod+r
    scope: app
`)

	spec, err := appspec.LoadFile(filepath.Join(root, "app.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	km := spec.Keymap
	if len(km.Disable) != 1 || km.Disable[0] != "old_save" {
		t.Fatalf("disable = %+v", km.Disable)
	}
	if len(km.Bindings) != 2 {
		t.Fatalf("bindings = %+v", km.Bindings)
	}
	if km.Bindings[0].Action != "element:save" || km.Bindings[1].Keys != "mod+r" {
		t.Fatalf("bindings = %+v", km.Bindings)
	}
}

