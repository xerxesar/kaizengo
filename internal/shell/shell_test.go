package shell

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestAddFieldToDoc(t *testing.T) {
	doc := "name: crm # sales\nmodels:\n  - name: lead\n    fields:\n      - name: title\n        type: string\n  - name: deal\n"
	got, err := addFieldToDoc(doc, "lead", newField{Name: "stage", Type: "enum", Values: []string{"new", "won"}, Required: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# sales", "- name: stage\n        type: enum\n        required: true\n        values: [new, won]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	got, err = addFieldToDoc(doc, "deal", newField{Name: "amount", Type: "float"})
	if err != nil || !strings.Contains(got, "  - name: deal\n    fields:\n      - name: amount\n        type: float") {
		t.Fatalf("err %v, doc:\n%s", err, got)
	}
	if _, err := addFieldToDoc(doc, "lead", newField{Name: "title", Type: "string"}); err == nil {
		t.Fatal("expected duplicate field error")
	}
	if _, err := addFieldToDoc(doc, "nope", newField{Name: "x", Type: "string"}); err == nil {
		t.Fatal("expected missing model error")
	}
	got, err = addFieldToDoc(doc, "lead", newField{Name: "owner", Type: "many2one", Relation: "crm.user", Unique: true, OnDelete: "set null"})
	if err != nil || !strings.Contains(got, "unique: true") || !strings.Contains(got, "onDelete: set null") {
		t.Fatalf("err %v, doc:\n%s", err, got)
	}
}

func TestSetViewAndRemoveField(t *testing.T) {
	doc := `name: hellospec
models:
  - name: greeting
    fields:
      - { name: message, type: string }
      - { name: okok, type: string }
views:
  - name: GreetingForm
    kind: form
    model: greeting
    groups:
      - fields:
          - field: message
`
	got, err := setViewInDoc(doc, "GreetingForm", "kind: form\nmodel: greeting\ngroups:\n  - fields:\n      - field: message\n      - field: okok\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "field: okok") {
		t.Fatalf("set-view missing okok:\n%s", got)
	}
	soft, err := removeFieldFromDoc(got, "greeting", "okok")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(soft, "name: okok") {
		t.Fatalf("field still present:\n%s", soft)
	}
	if _, err := removeFieldFromDoc(doc, "greeting", "missing"); err == nil {
		t.Fatal("expected missing field error")
	}
}

func TestSetFieldRenameInDoc(t *testing.T) {
	doc := `name: hellospec
models:
  - name: greeting
    fields:
      - name: message
        type: string
      - name: okok
        type: string
        required: true
        indexed: true
    search:
      fields: [message, okok]
views:
  - name: GreetingForm
    kind: form
    model: greeting
    groups:
      - fields:
          - field: message
          - field: okok
  - name: GreetingList
    kind: collection
    model: greeting
    groupBy: [okok]
    columns:
      - field: okok
`
	got, err := setFieldInDoc(doc, "greeting", "okok", newField{Name: "ok", Type: "string"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name: ok",
		"required: true",
		"indexed: true",
		"field: ok",
		"groupBy: [ok]",
		"fields: [message, ok]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "okok") {
		t.Fatalf("okok still present:\n%s", got)
	}
	if _, err := setFieldInDoc(doc, "greeting", "okok", newField{Name: "message", Type: "string"}); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestAddIndexAndScreenToDoc(t *testing.T) {
	doc := "name: crm\nmodels:\n  - name: lead\n    fields:\n      - name: title\n        type: string\n"
	got, err := addIndexToDoc(doc, "lead", "lead_title", "gin", true, []string{"title", "stage"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"indexes:", "name: lead_title", "fields: [title, stage]", "unique: true", "method: gin"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	board, err := addViewToDoc(doc, "LeadBoard", "collection", "lead", "", "columns:\n  - field: title\n    width: 2fr\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"views:", "name: LeadBoard", "kind: collection", "model: lead", "field: title", "width: 2fr"} {
		if !strings.Contains(board, want) {
			t.Fatalf("missing %q in:\n%s", want, board)
		}
	}
	page, err := addViewToDoc(doc, "LeadPage", "", "", "views/Lead.page.tsx", "")
	if err != nil || !strings.Contains(page, "path: views/Lead.page.tsx") {
		t.Fatalf("err %v, doc:\n%s", err, page)
	}
	if _, err := addViewToDoc(board, "LeadBoard", "form", "lead", "", ""); err == nil {
		t.Fatal("expected duplicate view error")
	}
}

func TestSplit(t *testing.T) {
	got, err := split(`run crm.addLead title="Big deal" note='it''s' tags=a,b`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "crm.addLead", "title=Big deal", "note=its", "tags=a,b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSplitHeredoc(t *testing.T) {
	got, err := split("domain apply --dry-run <<EOF\nname: crm\ntitle: CRM\nEOF")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"domain", "apply", "--dry-run", "name: crm\ntitle: CRM\n"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := split("domain apply <<EOF\nname: crm"); err == nil {
		t.Fatal("expected missing terminator error")
	}
}

func TestHeredocPending(t *testing.T) {
	cases := map[string]bool{
		"query crm.leads":                  false,
		"domain apply <<EOF":               true,
		"domain apply <<EOF\nname: x":      true,
		"domain apply <<EOF\nname: x\nEOF": false,
	}
	for in, want := range cases {
		if got := HeredocPending(in); got != want {
			t.Errorf("HeredocPending(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSplitBackslashContinuation(t *testing.T) {
	got, err := split("domain add-field hellospec.greeting ok string --migrate=\"UPDATE greetings \\\nSET ok = okok || '$' WHERE ok = '';\"")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"domain", "add-field", "hellospec.greeting", "ok", "string", "--migrate=UPDATE greetings SET ok = okok || '$' WHERE ok = '';"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := split("foo \\"); err == nil {
		t.Fatal("expected trailing backslash error")
	}
}

func TestSplitOpenQuoteMultiline(t *testing.T) {
	got, err := split("domain add-field x.y ok string --migrate=\"\nUPDATE t SET a = 1;\n\"")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"domain", "add-field", "x.y", "ok", "string", "--migrate=\nUPDATE t SET a = 1;\n"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := split("domain apply --migrate=\"oops"); err == nil {
		t.Fatal("expected unterminated quote")
	}
}

func TestSplitMigrateEqualsWithHeredoc(t *testing.T) {
	got, err := split("domain apply --migrate=\"ALTER TABLE t RENAME COLUMN a TO b;\" <<EOF\nname: hellospec\nEOF")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"domain", "apply", "--migrate=ALTER TABLE t RENAME COLUMN a TO b;", "name: hellospec\n"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestInputPending(t *testing.T) {
	cases := map[string]bool{
		"query crm.leads":                                           false,
		"domain apply <<EOF":                                        true,
		"domain apply <<EOF\nname: x\nEOF":                          false,
		"domain add-field x.y ok string --migrate=\"":               true,
		"domain add-field x.y ok string --migrate=\"\nUPDATE t\n\"": false,
		"foo \\":      true,
		"foo \\\nbar": false,
		`--migrate="UPDATE greetings SET ok = okok || '$'"`: false,
	}
	for in, want := range cases {
		if got := InputPending(in); got != want {
			t.Errorf("InputPending(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestQuoteRoundTrip(t *testing.T) {
	args := []string{"run", "crm.addLead", `title=Big "deal"`, `note=a\b`, "x<<y", ""}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = Quote(a)
	}
	got, err := split(strings.Join(parts, " "))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, args) {
		t.Fatalf("got %q, want %q", got, args)
	}
}

var addLead = operation{
	App: "crm", Kind: "command", Name: "addLead", Field: "crmAddLead", Operation: "mutation",
	Args: []opArg{
		{Name: "amount", Type: "Float"},
		{Name: "tags", Type: "[String!]"},
		{Name: "title", Type: "String!", Required: true},
	},
	ReturnFields: []string{"id", "title", "createdAt"},
}

func TestInvocationDocument(t *testing.T) {
	inv, err := parseInvocation(addLead, []string{"title=123", "amount=9.5", "tags=a,b"})
	if err != nil {
		t.Fatal(err)
	}
	doc, vars := inv.document(addLead)
	wantDoc := "mutation($amount: Float, $tags: [String!], $title: String!) { crmAddLead(amount: $amount, tags: $tags, title: $title) { id title createdAt } }"
	if doc != wantDoc {
		t.Fatalf("doc = %s", doc)
	}
	if vars["title"] != "123" || vars["amount"] != 9.5 || !reflect.DeepEqual(vars["tags"], []any{"a", "b"}) {
		t.Fatalf("vars = %#v", vars)
	}
	if cols := inv.columns(addLead); !reflect.DeepEqual(cols, []string{"id", "title"}) {
		t.Fatalf("columns = %v", cols)
	}
}

func TestCoerceInputObject(t *testing.T) {
	v, err := coerce("ListViewPresetInput!", `{"model":"crm.lead","label":"Won"}`)
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := v.(map[string]any); !ok || m["label"] != "Won" {
		t.Fatalf("got %#v", v)
	}
	if v, _ := coerce("String", `{"kept":"as string"}`); v != `{"kept":"as string"}` {
		t.Fatalf("String coerced to %#v", v)
	}
}

func TestInvocationErrors(t *testing.T) {
	for _, args := range [][]string{{}, {"title=x", "nope=1"}, {"title=x", "amount=abc"}, {"title"}} {
		if _, err := parseInvocation(addLead, args); err == nil {
			t.Errorf("%v: expected error", args)
		}
	}
}

type fakeExec struct{ queries []string }

func (f *fakeExec) Exec(_ context.Context, q string, _ map[string]any) (map[string]any, error) {
	f.queries = append(f.queries, q)
	switch {
	case strings.Contains(q, "platformCatalog"):
		return map[string]any{"platformCatalog": []any{map[string]any{
			"app": "crm", "kind": "command", "name": "addLead", "field": "crmAddLead", "operation": "mutation",
			"returns": "CrmLeadRecord!", "returnFields": []any{"id", "title"}, "dynamic": true,
			"args": []any{map[string]any{"name": "title", "type": "String!", "required": true}},
		}}}, nil
	case strings.Contains(q, "crmAddLead"):
		return map[string]any{"crmAddLead": map[string]any{"id": "1", "title": "Big deal"}}, nil
	}
	return nil, nil
}

func TestRunInvokesThroughExecutor(t *testing.T) {
	ex := &fakeExec{}
	res := Run(context.Background(), ex, `run crm.addLead title="Big deal"`)
	if !res.OK {
		t.Fatalf("output: %s", res.Output)
	}
	if !strings.Contains(res.Output, "Big deal") {
		t.Fatalf("output: %s", res.Output)
	}
	if res := Run(context.Background(), ex, "query crm.addLead"); res.OK {
		t.Fatal("expected kind mismatch error")
	}
	if res := Run(context.Background(), ex, "bogus"); res.OK {
		t.Fatal("expected unknown command error")
	}
}
