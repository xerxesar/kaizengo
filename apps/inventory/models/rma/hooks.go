package rma

import (
	"fmt"

	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/engine"
)

func init() {
	engine.RegisterModelHooks("inventory", "rma", engine.Hooks{
		BeforeCreate: beforeCreate,
		BeforeUpdate: beforeUpdate,
	})
}

func beforeCreate(hc engine.HookContext) error {
	if inv.AsString(hc.Fields["name"]) == "" {
		n := 1
		if inv.Registry != nil {
			all, _ := inv.Registry.List(hc.Context, hc.OrgID, "rma")
			n = len(all) + 1
		}
		hc.Fields["name"] = fmt.Sprintf("RMA/%05d", n)
	}
	return nil
}

func beforeUpdate(hc engine.HookContext) error {
	prev, next := inv.NextState(hc, "state")
	if next == "received" && prev != "received" && prev != "dispositioned" && prev != "closed" {
		return inv.PostRMA(hc)
	}
	return nil
}
