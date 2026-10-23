package appspec

import (
	"encoding/json"
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/acl"
)

// FilterPresetSpec declares a named list filter preset for KSearch (app.yaml models[].filters).
type FilterPresetSpec struct {
	ID       string   `yaml:"id"`
	Label    string   `yaml:"label"`
	LabelKey string   `yaml:"labelKey"`
	Domain   any      `yaml:"domain"`
	Q        string   `yaml:"q"`
	SearchIn []string `yaml:"searchIn"`
	GroupBy  []string `yaml:"groupBy"`
}

// EncodeFilterDomain JSON-encodes a YAML-loaded domain for GraphQL / list queries.
func EncodeFilterDomain(domain any) (string, error) {
	if domain == nil {
		return "", nil
	}
	switch v := domain.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", nil
		}
		if _, err := acl.ParseDomainExpr(s); err != nil {
			return "", err
		}
		return s, nil
	}
	b, err := json.Marshal(domain)
	if err != nil {
		return "", fmt.Errorf("domain: %w", err)
	}
	raw := string(b)
	if raw == "null" || raw == "[]" {
		return "", nil
	}
	if _, err := acl.ParseDomainExpr(raw); err != nil {
		return "", fmt.Errorf("domain: %w", err)
	}
	return raw, nil
}

func validateFilterPresets(model string, presets []FilterPresetSpec, fields map[string]struct{}) error {
	seen := map[string]struct{}{}
	for i, p := range presets {
		loc := fmt.Sprintf("model %q filters[%d]", model, i)
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
		if _, err := EncodeFilterDomain(p.Domain); err != nil {
			return fmt.Errorf("%s: %w", loc, err)
		}
		for j, f := range p.SearchIn {
			if _, ok := fields[f]; !ok {
				return fmt.Errorf("%s searchIn[%d]: unknown field %q", loc, j, f)
			}
		}
		for j, f := range p.GroupBy {
			if _, ok := fields[f]; !ok {
				return fmt.Errorf("%s groupBy[%d]: unknown field %q", loc, j, f)
			}
		}
	}
	return nil
}
