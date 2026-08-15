package stockmove

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func init() {
	engine.RegisterModelHooks("inventory", "stock_move", engine.Hooks{
		BeforeCreate: beforeCreate,
		BeforeUpdate: protectDone,
	})
}

func beforeCreate(hc engine.HookContext) error {
	if inv.AsNumber(hc.Fields["quantity"]) <= 0 {
		return i18n.Error("inventory.error.qty_positive")
	}
	return inv.AssignFEFOLot(hc)
}

func protectDone(hc engine.HookContext) error {
	if inv.IsInternal(hc.Context) {
		return nil
	}
	if inv.RecStr(hc.Record, "state") == "done" {
		return i18n.Error("inventory.error.move_done")
	}
	return nil
}
