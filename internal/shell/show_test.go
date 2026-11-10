package shell

import (
	"strings"
	"testing"
)

func TestExtractNamedYAML(t *testing.T) {
	doc := `
name: demo
views:
  - name: List
    kind: collection
    model: lead
  - name: Form
    kind: form
    model: lead
menus:
  - id: leads
    label: Leads
security:
  entries:
    - name: demo-member
      role: member
      effect: allow
      kind: query
      resource: demo.query.x
`
	got, err := extractNamedYAML(doc, "views", "name", "Form")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "name: Form") || !strings.Contains(got, "kind: form") {
		t.Fatalf("view = %q", got)
	}
	menu, err := extractNamedYAML(doc, "menus", "id", "leads")
	if err != nil || !strings.Contains(menu, "id: leads") {
		t.Fatalf("menu = %q err=%v", menu, err)
	}
	pol, err := extractPolicyYAML(doc, "demo-member")
	if err != nil || !strings.Contains(pol, "name: demo-member") {
		t.Fatalf("policy = %q err=%v", pol, err)
	}
	if _, err := extractNamedYAML(doc, "views", "name", "Nope"); err == nil {
		t.Fatal("expected missing view error")
	}
}

func TestTedSoftPayload(t *testing.T) {
	out, data, err := tedSoft("domain set-view demo List", "kind: collection\n")
	if err == nil || data == nil {
		t.Fatalf("want soft error + data, got err=%v data=%v", err, data)
	}
	m, _ := data.(map[string]any)
	if m["ted"] != true || m["command"] != "domain set-view demo List" {
		t.Fatalf("payload = %#v", m)
	}
	if !strings.Contains(out, "open terminal editor") || !strings.Contains(out, "kind: collection") {
		t.Fatalf("output = %q", out)
	}
}
