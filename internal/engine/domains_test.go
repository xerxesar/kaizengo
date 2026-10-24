package engine

import (
	"reflect"
	"strings"
	"testing"

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

func TestDynamicParseRejects(t *testing.T) {
	cases := map[string]string{
		"reserved": "name: core\ntitle: x\nsummary: y\n",
		"schema":   "name: crm\ntitle: x\nsummary: y\nschema: identity\n",
		"nav":      "name: crm\ntitle: x\nsummary: y\nnav: { route: crm, label: CRM }\n",
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

	same, _ := diffSpecs(&old, old)
	if !reflect.DeepEqual(same, []string{"no changes"}) {
		t.Fatalf("same = %v", same)
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
		`CREATE INDEX IF NOT EXISTS "idx_leads_amount"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `"lines"`) || strings.Contains(joined, "DROP") {
		t.Fatalf("unexpected DDL:\n%s", joined)
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
