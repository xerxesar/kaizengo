package appspec

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSubtractRenderedBaseIsEmpty(t *testing.T) {
	paths, _ := filepath.Glob("../../../apps/*/app.yaml")
	for _, path := range paths {
		base, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := MarshalDomain(base)
		if err != nil {
			t.Fatal(err)
		}
		full, err := Parse(doc)
		if err != nil {
			t.Fatal(err)
		}
		layer, err := Subtract(full, base)
		if err != nil {
			t.Fatalf("%s: %v", base.Name, err)
		}
		if !layer.EmptyLayer() {
			t.Fatalf("%s: rendered base leaves a layer: %+v", base.Name, layer)
		}
	}
}

func TestLayerRoundTrip(t *testing.T) {
	base, err := LoadFile("../../../apps/hellospec/app.yaml")
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := MarshalDomain(base)
	edited := strings.Replace(string(doc), "      - name: internalNote\n        type: string\n",
		"      - name: internalNote\n        type: string\n      - name: tag\n        type: string\n        indexed: true\n", 1)
	edited = strings.Replace(edited, "queries:\n",
		"  - name: tag_note\n    fields:\n      - name: label\n        type: string\nqueries:\n  - name: noteTags\n    list: tag_note\n", 1)
	if edited == string(doc) || !strings.Contains(edited, "name: tag\n") {
		t.Fatalf("test edit did not apply:\n%s", doc)
	}
	full, err := Parse([]byte(edited))
	if err != nil {
		t.Fatalf("%v\n%s", err, edited)
	}
	layer, err := Subtract(full, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(layer.Models) != 2 || len(layer.Models[0].Fields) != 1 || layer.Models[0].Fields[0].Name != "tag" || len(layer.Queries) != 1 {
		t.Fatalf("layer = %+v", layer)
	}
	layerDoc, err := MarshalDomain(layer)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseLayer(layerDoc)
	if err != nil {
		t.Fatalf("%v\n%s", err, layerDoc)
	}
	merged, err := Merge(base, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if err := merged.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := len(merged.Models[0].Fields); got != len(base.Models[0].Fields)+1 {
		t.Fatalf("merged greeting has %d fields", got)
	}
	if len(merged.Menus) != len(base.Menus) || merged.Nav != base.Nav {
		t.Fatal("base-only parts must survive the merge")
	}

	removed, _ := Parse([]byte(strings.Replace(string(doc), "      - name: internalNote\n        type: string\n", "", 1)))
	if _, err := Subtract(removed, base); err == nil || !strings.Contains(err.Error(), "internalNote") {
		t.Fatalf("expected removal error, got %v", err)
	}
	changed := strings.Replace(string(doc), "values: [happy, neutral, formal]", "values: [happy, neutral]", 1)
	if full, err := Parse([]byte(changed)); err == nil {
		if _, err := Subtract(full, base); err == nil {
			t.Fatal("expected change error")
		}
	}
	if _, err := Merge(base, AppSpec{Name: "hellospec", Models: []ModelSpec{{Name: "greeting", Fields: []FieldSpec{{Name: "message", Type: "int"}}}}}); err == nil {
		t.Fatal("expected merge conflict")
	}
}
