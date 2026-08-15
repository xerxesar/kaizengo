package picking

import (
	"fmt"

	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func init() {
	engine.RegisterModelHooks("inventory", "picking", engine.Hooks{
		BeforeCreate: beforeCreate,
		BeforeUpdate: beforeUpdate,
		BeforeDelete: beforeDelete,
	})
}

func beforeCreate(hc engine.HookContext) error {
	if inv.AsString(hc.Fields["name"]) != "" {
		return nil
	}
	prefix := map[string]string{
		"incoming": "IN",
		"outgoing": "OUT",
		"internal": "INT",
		"return":   "RET",
	}[inv.AsString(hc.Fields["pickingType"])]
	if prefix == "" {
		prefix = "WH"
	}
	n := 1
	if inv.Registry != nil {
		all, _ := inv.Registry.List(hc.Context, hc.OrgID, "picking")
		n = len(all) + 1
	}
	hc.Fields["name"] = fmt.Sprintf("%s/%05d", prefix, n)
	return nil
}

func beforeUpdate(hc engine.HookContext) error {
	prev, next := inv.NextState(hc, "state")
	if next == prev {
		return nil
	}
	switch next {
	case "done":
		if prev == "cancelled" {
			return i18n.Error("inventory.error.picking_cancelled_complete")
		}
		return inv.PostPicking(hc)
	case "cancelled":
		if prev == "done" {
			return i18n.Error("inventory.error.picking_done_cancel")
		}
		return inv.UnreservePicking(hc)
	case "assigned":
		return inv.ReservePicking(hc)
	}
	return nil
}

func beforeDelete(hc engine.HookContext) error {
	if inv.RecStr(hc.Record, "state") == "done" {
		return i18n.Error("inventory.error.picking_done_delete")
	}
	return nil
}
