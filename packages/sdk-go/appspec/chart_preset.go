package appspec

import (
	"fmt"
	"strings"
)

// Allowed chart types for KChart (Apache ECharts).
var chartTypes = map[string]struct{}{
	"bar":       {},
	"line":      {},
	"area":      {},
	"pie":       {},
	"doughnut":  {},
	"scatter":   {},
	"radar":     {},
}

var chartMeasures = map[string]struct{}{
	"count": {},
	"sum":   {},
	"avg":   {},
}

// ChartPresetSpec declares a named chart config for KChart (app.yaml models[].charts).
type ChartPresetSpec struct {
	ID          string   `yaml:"id"`
	Label       string   `yaml:"label"`
	LabelKey    string   `yaml:"labelKey"`
	Type        string   `yaml:"type"`   // default chart type
	Types       []string `yaml:"types"`  // allowed types in the UI switcher (default: common set)
	XField      string   `yaml:"xField"` // category / dimension
	YField      string   `yaml:"yField"` // numeric measure field (optional for count)
	SeriesField string   `yaml:"seriesField"`
	Measure     string   `yaml:"measure"` // count|sum|avg (default count, or sum when yField set)
}

func validateChartPresets(model string, presets []ChartPresetSpec, fields map[string]struct{}) error {
	seen := map[string]struct{}{}
	for i, p := range presets {
		loc := fmt.Sprintf("model %q charts[%d]", model, i)
		if !nameRe.MatchString(p.ID) {
			return fmt.Errorf("%s: invalid id %q", loc, p.ID)
		}
		if _, ok := seen[p.ID]; ok {
			return fmt.Errorf("%s: duplicate id %q", loc, p.ID)
		}
		seen[p.ID] = struct{}{}
		if strings.TrimSpace(p.Label) == "" && strings.TrimSpace(p.LabelKey) == "" {
			return fmt.Errorf("%s: requires label or labelKey", loc)
		}
		chartType := strings.TrimSpace(p.Type)
		if chartType == "" {
			chartType = "bar"
		}
		if _, ok := chartTypes[chartType]; !ok {
			return fmt.Errorf("%s: unsupported type %q", loc, chartType)
		}
		for j, t := range p.Types {
			t = strings.TrimSpace(t)
			if _, ok := chartTypes[t]; !ok {
				return fmt.Errorf("%s types[%d]: unsupported type %q", loc, j, t)
			}
		}
		measure := strings.TrimSpace(p.Measure)
		if measure != "" {
			if _, ok := chartMeasures[measure]; !ok {
				return fmt.Errorf("%s: unsupported measure %q", loc, measure)
			}
		}
		xf := strings.TrimSpace(p.XField)
		if xf == "" {
			return fmt.Errorf("%s: requires xField", loc)
		}
		if _, ok := fields[xf]; !ok {
			return fmt.Errorf("%s: unknown xField %q", loc, xf)
		}
		if yf := strings.TrimSpace(p.YField); yf != "" {
			if _, ok := fields[yf]; !ok {
				return fmt.Errorf("%s: unknown yField %q", loc, yf)
			}
		}
		if sf := strings.TrimSpace(p.SeriesField); sf != "" {
			if _, ok := fields[sf]; !ok {
				return fmt.Errorf("%s: unknown seriesField %q", loc, sf)
			}
		}
	}
	return nil
}
