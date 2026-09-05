package adjustment

import (
	"fmt"

	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/engine"
)

func init() {
	engine.RegisterModelHooks("inventory", "adjustment", engine.Hooks{
		BeforeCreate: beforeCreate,
		BeforeUpdate: beforeUpdate,
	})
}

func beforeCreate(hc engine.HookContext) error {
	if inv.AsString(hc.Fields["name"]) == "" {
		n := 1
		if inv.Registry != nil {
			all, _ := inv.Registry.List(hc.Context, hc.OrgID, "adjustment")
			n = len(all) + 1
		}
		hc.Fields["name"] = fmt.Sprintf("ADJ/%05d", n)
	}
	vid := inv.AsString(hc.Fields["variantId"])
	loc := inv.AsString(hc.Fields["locationId"])
	lot := inv.AsString(hc.Fields["lotId"])
	if inv.Registry != nil && vid != "" && loc != "" {
		if q := inv.FindQuant(hc.Context, hc.OrgID, vid, loc, lot); q != nil {
			hc.Fields["systemQty"] = inv.RecNum(q, "quantity")
		}
	}
	return nil
}

func beforeUpdate(hc engine.HookContext) error {
	prev, next := inv.NextState(hc, "state")
	if next == "done" && prev != "done" {
		return inv.PostAdjustment(hc)
	}
	return nil
}
