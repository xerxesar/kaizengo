package reorderrule

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	engine.RegisterModelHooks("inventory", "reorder_rule", engine.Hooks{
		BeforeCreate: compute,
		BeforeUpdate: compute,
	})
}

func fieldNum(hc engine.HookContext, key string) float64 {
	if v, ok := hc.Fields[key]; ok {
		return inv.AsNumber(v)
	}
	if hc.Record != nil {
		return inv.RecNum(hc.Record, key)
	}
	return 0
}

func fieldStr(hc engine.HookContext, key string) string {
	if v, ok := hc.Fields[key]; ok && inv.AsString(v) != "" {
		return inv.AsString(v)
	}
	if hc.Record != nil {
		return inv.RecStr(hc.Record, key)
	}
	return ""
}

func compute(hc engine.HookContext) error {
	method := fieldStr(hc, "method")
	lead := fieldNum(hc, "leadTimeDays")
	safety := fieldNum(hc, "safetyStock")
	demand := fieldNum(hc, "avgDailyDemand")
	minQty := fieldNum(hc, "minQty")
	maxQty := fieldNum(hc, "maxQty")

	rop := demand*lead + safety
	if method == "min_max" {
		rop = minQty
	}
	hc.Fields["reorderPoint"] = inv.RoundQty(rop, 0.0001)

	onHand := 0.0
	if inv.Registry != nil {
		onHand, _ = inv.OnHandAt(hc.Context, hc.OrgID, fieldStr(hc, "variantId"), fieldStr(hc, "locationId"))
	}
	order := maxQty - onHand
	if onHand >= rop && method != "min_max" {
		order = 0
	}
	if order < 0 {
		order = 0
	}
	hc.Fields["qtyToOrder"] = inv.RoundQty(order, 0.0001)
	return nil
}
