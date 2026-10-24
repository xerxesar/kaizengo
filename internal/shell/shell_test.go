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
