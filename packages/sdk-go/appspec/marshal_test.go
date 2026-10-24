package appspec

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMarshalDomainRoundTrip(t *testing.T) {
	paths, err := filepath.Glob("../../../apps/*/app.yaml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no app specs found: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			spec, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := MarshalDomain(spec)
			if err != nil {
				t.Fatal(err)
			}
			back, err := Parse(doc)
			if err != nil {
				t.Fatalf("parse rendered spec: %v\n%s", err, doc)
			}
			if back.Name != spec.Name || back.EnableExtensions != spec.EnableExtensions {
				t.Fatalf("header mismatch: %+v vs %+v", back, spec)
			}
			if len(back.Models) != len(spec.Models) {
				t.Fatalf("models: got %d, want %d", len(back.Models), len(spec.Models))
			}
			for i, m := range spec.Models {
				if !reflect.DeepEqual(back.Models[i].Fields, m.Fields) {
					t.Fatalf("model %s fields differ:\n got %+v\nwant %+v", m.Name, back.Models[i].Fields, m.Fields)
				}
			}
			if !reflect.DeepEqual(back.Queries, spec.Queries) || !reflect.DeepEqual(back.Commands, spec.Commands) {
				t.Fatalf("queries/commands differ")
			}
			if !reflect.DeepEqual(back.Security.Entries, spec.Security.Entries) {
				t.Fatalf("security entries differ:\n got %+v\nwant %+v", back.Security.Entries, spec.Security.Entries)
			}
		})
	}
}
