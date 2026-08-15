package inv

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"kaizengo/packages/sdk-go/engine"
)

func withInternal(ctx context.Context) context.Context {
	return engine.WithInternal(ctx)
}

func recStr(rec engine.Record, key string) string {
	return asString(rec[key])
}

func recNum(rec engine.Record, key string) float64 {
	return asNumber(rec[key])
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func asNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func asBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	default:
		s := strings.ToLower(asString(v))
		return s == "true" || s == "1"
	}
}

func nextState(hc engine.HookContext, field string) (prev, next string) {
	prev = recStr(hc.Record, field)
	if v, ok := hc.Fields[field]; ok && asString(v) != "" {
		return prev, asString(v)
	}
	return prev, prev
}
