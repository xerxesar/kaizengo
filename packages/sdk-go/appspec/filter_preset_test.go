package appspec

import "testing"

func TestEncodeFilterDomainFlatAndNested(t *testing.T) {
	flat := [][]any{{"status", "=", "installed"}}
	raw, err := EncodeFilterDomain(flat)
	if err != nil {
		t.Fatal(err)
	}
	if raw != `[["status","=","installed"]]` {
		t.Fatalf("flat domain = %q", raw)
	}

	nested := []any{
		"&",
		[]any{"|", []any{"status", "=", "installed"}, []any{"status", "=", "available"}},
		[]any{"version", "=", "1.0.0"},
	}
	raw, err = EncodeFilterDomain(nested)
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" {
		t.Fatal("expected nested domain")
	}
}

func TestValidateFilterPresets(t *testing.T) {
	spec := AppSpec{
		Name:  "appman",
		Title: "App Manager",
		Summary: "x",
		Models: []ModelSpec{{
			Name: "app",
			Fields: []FieldSpec{
				{Name: "status", Type: "enum", Values: []string{"installed", "available"}},
				{Name: "version", Type: "string"},
			},
			Filters: []FilterPresetSpec{{
				ID:       "installed",
				LabelKey: "appman.filter.installed",
				Domain:   [][]any{{"status", "=", "installed"}},
			}},
		}},
	}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}
}
