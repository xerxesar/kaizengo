package appspec

import "testing"

func TestValidateChartPresets(t *testing.T) {
	spec := AppSpec{
		Name:    "appman",
		Title:   "App Manager",
		Summary: "x",
		Models: []ModelSpec{{
			Name: "app",
			Fields: []FieldSpec{
				{Name: "status", Type: "enum", Values: []string{"installed", "available"}},
			},
			Charts: []ChartPresetSpec{{
				ID:       "by_status",
				LabelKey: "appman.chart.by_status",
				Type:     "bar",
				Types:    []string{"bar", "pie", "doughnut"},
				XField:   "status",
				Measure:  "count",
			}},
		}},
	}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateChartPresetsRejectsBadType(t *testing.T) {
	spec := AppSpec{
		Name:    "appman",
		Title:   "App Manager",
		Summary: "x",
		Models: []ModelSpec{{
			Name: "app",
			Fields: []FieldSpec{
				{Name: "status", Type: "enum", Values: []string{"installed"}},
			},
			Charts: []ChartPresetSpec{{
				ID:     "bad",
				Label:  "Bad",
				Type:   "heatmap",
				XField: "status",
			}},
		}},
	}
	if err := spec.Validate(); err == nil {
		t.Fatal("expected error for unsupported chart type")
	}
}
