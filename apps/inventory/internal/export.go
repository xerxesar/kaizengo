package inv

import (
	"context"
	"strings"

	"kaizengo/internal/engine"
)

// Registry is the event-sourced model store, set during app Setup.
var Registry *engine.ModelRegistry

func TrimFields(keys ...string) func(engine.HookContext) error {
	return func(hc engine.HookContext) error {
		for _, key := range keys {
			if v, ok := hc.Fields[key].(string); ok {
				hc.Fields[key] = strings.TrimSpace(v)
			}
		}
		return nil
	}
}

func IsInternal(ctx context.Context) bool { return engine.IsInternal(ctx) }

func RecStr(rec engine.Record, key string) string { return recStr(rec, key) }

func RecNum(rec engine.Record, key string) float64 { return recNum(rec, key) }

func AsString(v any) string { return asString(v) }

func AsNumber(v any) float64 { return asNumber(v) }

func NextState(hc engine.HookContext, field string) (prev, next string) {
	return nextState(hc, field)
}

func FindQuant(ctx context.Context, orgID, variantID, locationID, lotID string) engine.Record {
	return findQuant(ctx, orgID, variantID, locationID, lotID)
}

func OnHandAt(ctx context.Context, orgID, variantID, locationID string) (qty, unitCost float64) {
	return onHandAt(ctx, orgID, variantID, locationID)
}

func AfterCreateLedger(hc engine.HookContext) error { return afterCreateLedger(hc) }

func PostPicking(hc engine.HookContext) error { return postPicking(hc) }

func ReservePicking(hc engine.HookContext) error { return reservePicking(hc) }

func UnreservePicking(hc engine.HookContext) error { return unreservePicking(hc) }

func PostAdjustment(hc engine.HookContext) error { return postAdjustment(hc) }

func PostRMA(hc engine.HookContext) error { return postRMA(hc) }

func AssignFEFOLot(hc engine.HookContext) error { return assignFEFOLot(hc) }
