package engine

import (
	"fmt"
	"regexp"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

func validateSpecFields(model appspec.ModelSpec, fields map[string]any) error {
	for _, f := range model.Fields {
		v, ok := fields[f.Name]
		if !ok || v == nil {
			continue
		}
		if err := validateFieldValue(f, v); err != nil {
			return err
		}
	}
	return nil
}

func validateFieldValue(f appspec.FieldSpec, v any) error {
	if f.IsEnum() && !f.IsRelation() {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return nil
		}
		for _, allowed := range f.Values {
			if allowed == s {
				return nil
			}
		}
		return fmt.Errorf("field %s must be one of %v", f.Name, f.Values)
	}
	vr := f.Validate
	switch f.CanonicalType() {
	case appspec.TypeInt:
		n, ok := asInt(v)
		if !ok {
			return fmt.Errorf("field %s must be int", f.Name)
		}
		if vr.Min != nil && n < *vr.Min {
			return fmt.Errorf("field %s must be >= %d", f.Name, *vr.Min)
		}
		if vr.Max != nil && n > *vr.Max {
			return fmt.Errorf("field %s must be <= %d", f.Name, *vr.Max)
		}
	case appspec.TypeNumber:
		n, ok := asFloat(v)
		if !ok {
			return fmt.Errorf("field %s must be number", f.Name)
		}
		if vr.Min != nil && n < float64(*vr.Min) {
			return fmt.Errorf("field %s must be >= %d", f.Name, *vr.Min)
		}
		if vr.Max != nil && n > float64(*vr.Max) {
			return fmt.Errorf("field %s must be <= %d", f.Name, *vr.Max)
		}
	case appspec.TypeBool:
		if _, err := coerceStoredValue(f, v); err != nil {
			return err
		}
	case appspec.TypeJSON:
		if _, err := asJSONString(v); err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
	case appspec.TypeMany2Many:
		if _, err := asStringSlice(v); err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
	case appspec.TypeOne2Many:
		return nil
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if vr.MinLength > 0 && len(s) < vr.MinLength {
			return fmt.Errorf("field %s must be at least %d characters", f.Name, vr.MinLength)
		}
		if vr.MaxLength > 0 && len(s) > vr.MaxLength {
			return fmt.Errorf("field %s must be at most %d characters", f.Name, vr.MaxLength)
		}
		if vr.Pattern != "" {
			re, err := regexp.Compile(vr.Pattern)
			if err != nil {
				return fmt.Errorf("field %s has invalid pattern: %w", f.Name, err)
			}
			if !re.MatchString(s) {
				return fmt.Errorf("field %s does not match required pattern", f.Name)
			}
		}
	}
	return nil
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
