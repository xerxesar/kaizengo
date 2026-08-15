package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

func storedFields(model appspec.ModelSpec) []appspec.FieldSpec {
	out := make([]appspec.FieldSpec, 0, len(model.Fields))
	for _, f := range model.Fields {
		if f.Stored() {
			out = append(out, f)
		}
	}
	return out
}

func coerceStoredValue(f appspec.FieldSpec, v any) (any, error) {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		n, ok := asInt(v)
		if !ok {
			return nil, fmt.Errorf("field %s must be int", f.Name)
		}
		return n, nil
	case appspec.TypeNumber:
		n, ok := asFloat(v)
		if !ok {
			return nil, fmt.Errorf("field %s must be number", f.Name)
		}
		return n, nil
	case appspec.TypeBool:
		b, ok := v.(bool)
		if !ok {
			s := strings.TrimSpace(strings.ToLower(fmt.Sprint(v)))
			if s == "true" || s == "1" {
				return true, nil
			}
			if s == "false" || s == "0" || s == "" {
				return false, nil
			}
			return nil, fmt.Errorf("field %s must be bool", f.Name)
		}
		return b, nil
	case appspec.TypeMany2Many:
		ids, err := asStringSlice(v)
		if err != nil {
			return nil, fmt.Errorf("field %s must be a list of ids: %w", f.Name, err)
		}
		return ids, nil
	case appspec.TypeJSON:
		return asJSONString(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v)), nil
	}
}

func isEmptyValue(f appspec.FieldSpec, v any) bool {
	switch f.CanonicalType() {
	case appspec.TypeBool, appspec.TypeInt, appspec.TypeNumber:
		return false
	case appspec.TypeMany2Many:
		ids, _ := v.([]string)
		return len(ids) == 0
	default:
		return strings.TrimSpace(fmt.Sprint(v)) == ""
	}
}

func projectValue(f appspec.FieldSpec, v any) any {
	if f.CanonicalType() == appspec.TypeMany2Many {
		ids, _ := asStringSlice(v)
		b, _ := json.Marshal(ids)
		return string(b)
	}
	return v
}

func scanPtr(f appspec.FieldSpec) any {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		var v int
		return &v
	case appspec.TypeNumber:
		var v float64
		return &v
	case appspec.TypeBool:
		var v bool
		return &v
	default:
		var v string
		return &v
	}
}

func derefScan(f appspec.FieldSpec, p any) any {
	switch v := p.(type) {
	case *int:
		return *v
	case *float64:
		return *v
	case *bool:
		return *v
	case *string:
		if f.CanonicalType() == appspec.TypeMany2Many {
			ids, _ := asStringSlice(*v)
			return ids
		}
		return *v
	default:
		return nil
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func asStringSlice(v any) ([]string, error) {
	switch t := v.(type) {
	case nil:
		return []string{}, nil
	case []string:
		return t, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return []string{}, nil
		}
		if strings.HasPrefix(s, "[") {
			var ids []string
			if err := json.Unmarshal([]byte(s), &ids); err != nil {
				return nil, err
			}
			return ids, nil
		}
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

func asJSONString(v any) (string, error) {
	switch t := v.(type) {
	case nil:
		return "", nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return "", nil
		}
		if json.Valid([]byte(s)) {
			return s, nil
		}
		return "", fmt.Errorf("invalid json")
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
