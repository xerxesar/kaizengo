package engine

import (
	"reflect"
	"strings"
	"testing"

	"kaizengo/internal/app"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
)

const crmDoc = `
name: crm
title: CRM
summary: Leads
models:
  - name: lead
    fields:
      - { name: title, type: string, required: true }
      - { name: amount, type: number }
queries:
  - { name: leads, list: lead }
commands:
  - { name: addLead, create: lead }
  - { name: score, returns: int }
security:
  entries:
    - { name: crm-member-leads, role: member, effect: allow, kind: query, resource: crm.query.crmLeads }
`

func testDomains() *Domains {
	host := module.NewHost(nil, nil)
	host.Loaded = []module.Manifest{{Name: "core"}}
	return NewDomains(host, module.NewRegistry(), nil)
}

func TestDynamicParseAcceptsInlineSecurity(t *testing.T) {
	spec, err := testDomains().parseDynamic(crmDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Security.Entries) != 1 || spec.Security.Entries[0].Resource != "crm.query.crmLeads" {
		t.Fatalf("security = %+v", spec.Security)
	}
	if !spec.EnableExtensions {
		t.Fatal("dynamic apps must enable extensions for plugin hooks")
	}
}

func TestDynamicParseAcceptsDeclarativeShell(t *testing.T) {
	doc := crmDoc + `
nav:
  label: CRM
  route: crm
menus:
  - id: leads
    label: Leads
    model: lead
locales:
  - id: en
    name: English
    dir: ltr
    messages:
      crm.menu.leads: Leads
keymap:
  bindings:
    - id: focus_leads
      action: nav.crm
      keys: alt+l
      label: Leads
`
	spec, err := testDomains().parseDynamic(doc)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Nav.Route != "crm" || len(spec.Menus) != 1 || spec.Menus[0].Model != "lead" {
		t.Fatalf("shell = nav %+v menus %+v", spec.Nav, spec.Menus)
	}
	if spec.Locales[0].Messages["crm.menu.leads"] != "Leads" || len(spec.Keymap.Bindings) != 1 {
		t.Fatalf("locales/keymap = %+v %+v", spec.Locales, spec.Keymap)
	}
}

func TestDynamicParseRejects(t *testing.T) {
	cases := map[string]string{
		"reserved": "name: core\ntitle: x\nsummary: y\n",
		"schema":   "name: crm\ntitle: x\nsummary: y\nschema: identity\n",
		"extends":  "name: crm\ntitle: x\nsummary: y\textends:\n  - { point: model.crm.lead.beforeCreate, handler: check }\n",
		"virtual":  "name: crm\ntitle: x\nsummary: y\nmodels:\n  - { name: lead, virtual: true, fields: [{ name: title, type: string }] }\n",
		"depends":  "name: crm\ntitle: x\nsummary: y\ndepends: [missing]\n",
		"include":  "name: crm\ntitle: x\nsummary: y\nmodels: [models/lead]\n",
	}
	d := testDomains()
	for name, doc := range cases {
		if _, err := d.parseDynamic(doc); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDiffSpecs(t *testing.T) {
	d := testDomains()
	old, err := d.parseDynamic(crmDoc)
	if err != nil {
		t.Fatal(err)
	}
	next, err := d.parseDynamic(strings.Replace(crmDoc,
		"      - { name: amount, type: number }",
		"      - { name: amount, type: number }\n      - { name: owner, type: string }", 1))
	if err != nil {
		t.Fatal(err)
	}
	changes, err := diffSpecs(&old, next)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(changes, []string{"add field lead.owner (string)"}) {
		t.Fatalf("changes = %v", changes)
	}

	retyped, err := d.parseDynamic(strings.Replace(crmDoc, "type: number", "type: int", 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := diffSpecs(&old, retyped); err == nil {
		t.Fatal("expected type change to be rejected")
	}

	removed, err := d.parseDynamic(strings.Replace(crmDoc,
		"      - { name: amount, type: number }\n", "", 1))
	if err != nil {
		t.Fatal(err)
	}
	soft, err := diffSpecs(&old, removed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(soft, []string{"remove field lead.amount (column kept)"}) {
		t.Fatalf("soft remove = %v", soft)
	}

	same, _ := diffSpecs(&old, old)
	if !reflect.DeepEqual(same, []string{"no changes"}) {
		t.Fatalf("same = %v", same)
	}
}

func TestDiffSpecsViewUpdate(t *testing.T) {
	base := appspec.AppSpec{Name: "hellospec", Title: "Hello", Views: []appspec.ViewSpec{
		{Name: "GreetingForm", Kind: "form", Model: "greeting", Groups: []appspec.ViewGroupSpec{
			{LabelKey: "g", Fields: []appspec.ViewColumnSpec{{Field: "message"}}},
		}},
	}}
	next := base
	next.Views = []appspec.ViewSpec{
		{Name: "GreetingForm", Kind: "form", Model: "greeting", Groups: []appspec.ViewGroupSpec{
			{LabelKey: "g", Fields: []appspec.ViewColumnSpec{{Field: "message"}, {Field: "okok"}}},
		}},
	}
	changes, err := diffSpecs(&base, next)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(changes, []string{"update view GreetingForm"}) {
		t.Fatalf("changes = %v", changes)
	}
}

func TestMigrationRequiredMentionsPrePost(t *testing.T) {
	err := migrationRequiredError([]MigrationHint{{Path: "models.x.fields.y", Hint: "new column", Stub: "-- UPDATE"}})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"--migrate-pre", "--migrate=", "--migrate-go", "--no-migrate"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
}

func TestPendingMigrateStash(t *testing.T) {
	d := testDomains()
	hints := []MigrationHint{{Path: "models.lead.fields.scratch", Hint: "new column", Stub: "-- UPDATE"}}
	d.setPendingMigrate("crm", "name: crm\n", hints)
	proposed, got, ok := d.PendingMigrate("crm")
	if !ok || proposed != "name: crm\n" || len(got) != 1 || got[0].Path != hints[0].Path {
		t.Fatalf("pending = %q %+v ok=%v", proposed, got, ok)
	}
	info := DomainInfo{App: "crm"}
	if _, _, ok := d.PendingMigrate("crm"); ok {
		info.MigratePending = true
	}
	if !info.MigratePending {
		t.Fatal("expected MigratePending")
	}
	d.clearPendingMigrate("crm")
	if _, _, ok := d.PendingMigrate("crm"); ok {
		t.Fatal("expected cleared")
	}
}

func TestDataMigrationNeeded(t *testing.T) {
	prev := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{
		{Name: "lead", Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString},
			{Name: "okok", Type: appspec.TypeString},
		}},
	}}
	next := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{
		{Name: "lead", Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString},
			{Name: "ok", Type: appspec.TypeString},
		}},
	}}
	hints := dataMigrationNeeded(&prev, next)
	if len(hints) < 2 {
		t.Fatalf("hints = %+v", hints)
	}
	paths := map[string]bool{}
	for _, h := range hints {
		paths[h.Path] = true
	}
	if !paths["models.lead.fields.okok"] || !paths["models.lead.fields.ok"] {
		t.Fatalf("paths = %v", paths)
	}
}

func TestForceDropDDL(t *testing.T) {
	prev := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{
		{Name: "lead", Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString},
			{Name: "amount", Type: appspec.TypeNumber, Indexed: true},
			{Name: "tags", Type: appspec.TypeMany2Many, Relation: "category"},
		}, Indexes: []appspec.IndexSpec{{Name: "lead_title", Fields: []string{"title"}}}},
	}}
	next := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{
		{Name: "lead", Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString},
		}},
	}}
	joined := strings.Join(forceDropDDL("crm", prev, next), "\n")
	for _, want := range []string{
		`DROP COLUMN IF EXISTS "amount"`,
		`DROP INDEX IF EXISTS "crm"."idx_leads_amount"`,
		`DROP TABLE IF EXISTS "crm"."lead_tags_rel"`,
		`DROP INDEX IF EXISTS "crm"."lead_title"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestSpecFromTipSnapshotAndOverlay(t *testing.T) {
	d := testDomains()
	full, err := d.parseDynamic(crmDoc)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := appspec.MarshalDomain(full)
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.specFromTip(nil, &app.DomainRevision{Layer: app.LayerRevision, Spec: string(doc)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "crm" || len(got.Models) != 1 {
		t.Fatalf("snapshot tip = %+v", got)
	}

	base := appspec.AppSpec{Name: "crm", Title: "CRM", Summary: "Leads", Models: []appspec.ModelSpec{
		{Name: "lead", Fields: []appspec.FieldSpec{{Name: "title", Type: appspec.TypeString}}},
	}}
	overlay := `name: crm
models:
  - name: lead
    fields:
      - { name: amount, type: number }
`
	merged, err := d.specFromTip(&base, &app.DomainRevision{Layer: app.LayerOverlay, Spec: overlay})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Models[0].Fields) != 2 {
		t.Fatalf("overlay merge fields = %+v", merged.Models[0].Fields)
	}
}

func TestModelDDLIsAdditive(t *testing.T) {
	stmts := modelDDL("crm", appspec.ModelSpec{Name: "lead", Fields: []appspec.FieldSpec{
		{Name: "title", Type: appspec.TypeString},
		{Name: "amount", Type: appspec.TypeNumber, Indexed: true},
		{Name: "lines", Type: appspec.TypeOne2Many, Relation: "line", Inverse: "lead"},
	}}, true)
	joined := strings.Join(stmts, "\n")
	for _, want := range []string{
		`CREATE TABLE IF NOT EXISTS "crm"."leads"`,
		`ADD COLUMN IF NOT EXISTS "title" TEXT NOT NULL DEFAULT ''`,
		`ADD COLUMN IF NOT EXISTS "amount" DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS "idx_leads_amount" ON "crm"."leads" ("org_id", "amount")`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `"lines"`) || strings.Contains(joined, "DROP") {
		t.Fatalf("unexpected DDL:\n%s", joined)
	}
}

func TestRelationDDL(t *testing.T) {
	spec := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{
		{Name: "category", Fields: []appspec.FieldSpec{{Name: "name", Type: appspec.TypeString}}},
		{Name: "lead", Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString, Unique: true},
			{Name: "note", Type: appspec.TypeJSON},
			{Name: "owner", Type: appspec.TypeMany2One, Relation: "category", Required: true},
			{Name: "categoryId", Type: appspec.TypeMany2One, Relation: "category", OnDelete: "set null"},
			{Name: "tags", Type: appspec.TypeMany2Many, Relation: "category"},
			{Name: "lines", Type: appspec.TypeOne2Many, Relation: "category", Inverse: "leadId"},
		}, Indexes: []appspec.IndexSpec{
			{Name: "lead_note_gin", Fields: []string{"note"}, Method: "gin"},
		}},
	}}
	lead := spec.Models[1]
	joined := strings.Join(modelDDL("crm", lead, true), "\n")
	for _, want := range []string{
		`ADD COLUMN IF NOT EXISTS "owner" TEXT NOT NULL`,
		`ADD COLUMN IF NOT EXISTS "category_id" TEXT NULL`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `"tags"`) {
		t.Fatalf("many2many must not add a column:\n%s", joined)
	}
	rel := strings.Join(relationDDL("crm", spec, []newColumn{
		{Model: "lead", Field: "owner"},
		{Model: "lead", Field: "categoryId"},
	}), "\n")
	for _, want := range []string{
		`CREATE INDEX IF NOT EXISTS "idx_leads_owner_org"`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "uidx_leads_title" ON "crm"."leads" ("org_id", "title") WHERE deleted = false`,
		`USING gin (to_tsvector('simple', "note"))`,
		`CREATE TABLE IF NOT EXISTS "crm"."lead_tags_rel"`,
		`PRIMARY KEY (org_id, left_id, right_id)`,
		`REFERENCES "crm"."categories" ("id") ON DELETE RESTRICT`,
		`ON DELETE SET NULL`,
		`idx_categories_lead_id_org`,
	} {
		if !strings.Contains(rel, want) {
			t.Errorf("missing %q in:\n%s", want, rel)
		}
	}
}

func TestIndexRefinements(t *testing.T) {
	spec := appspec.AppSpec{Name: "crm", Models: []appspec.ModelSpec{{
		Name: "lead",
		Fields: []appspec.FieldSpec{
			{Name: "title", Type: appspec.TypeString},
			{Name: "note", Type: appspec.TypeText},
		},
		Indexes: []appspec.IndexSpec{{
			Name: "lead_title", Fields: []string{"title"}, Orders: []string{"desc"},
			Include: []string{"note"}, Where: "title <> ''",
		}},
	}}}
	rel := strings.Join(relationDDL("crm", spec, nil), "\n")
	for _, want := range []string{
		`"title" DESC`,
		`INCLUDE ("note")`,
		`WHERE "title" <> ''`,
	} {
		if !strings.Contains(rel, want) {
			t.Fatalf("missing %q in:\n%s", want, rel)
		}
	}
}

func TestSchemaDDLOnlyCoversLayerAdditions(t *testing.T) {
	base := appspec.AppSpec{Name: "hr", Models: []appspec.ModelSpec{
		{Name: "employee", Fields: []appspec.FieldSpec{{Name: "name", Type: appspec.TypeString}}},
	}}
	eff := base
	eff.Models = []appspec.ModelSpec{
		{Name: "employee", Fields: []appspec.FieldSpec{{Name: "name", Type: appspec.TypeString}, {Name: "badge", Type: appspec.TypeInt}}},
		{Name: "team", Fields: []appspec.FieldSpec{{Name: "label", Type: appspec.TypeString}}},
	}
	ddl := schemaDDL(base, eff, false)
	if len(ddl) != 2 || ddl[0].create || len(ddl[0].model.Fields) != 1 || ddl[0].model.Fields[0].Name != "badge" || !ddl[1].create {
		t.Fatalf("ddl = %+v", ddl)
	}
	joined := strings.Join(modelDDL("hr", ddl[0].model, ddl[0].create), "\n")
	if strings.Contains(joined, "CREATE TABLE") || strings.Contains(joined, `"name"`) || !strings.Contains(joined, `"badge" BIGINT`) {
		t.Fatalf("unexpected DDL:\n%s", joined)
	}
	if got := schemaDDL(base, base, false); len(got) != 0 {
		t.Fatalf("base alone must not touch migrated tables: %+v", got)
	}
	if got := schemaDDL(base, base, true); len(got) != 1 || !got[0].create {
		t.Fatalf("autoSchema ensures every table: %+v", got)
	}
}
