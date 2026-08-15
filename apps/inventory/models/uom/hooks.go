package uom

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func init() {
	engine.RegisterModelHooks("inventory", "uom", engine.Hooks{
		BeforeCreate: before,
		BeforeUpdate: before,
	})
}

func before(hc engine.HookContext) error {
	if err := inv.TrimFields("name", "symbol")(hc); err != nil {
		return err
	}
	if v, ok := hc.Fields["ratio"]; ok && inv.AsNumber(v) <= 0 {
		return i18n.Error("inventory.error.uom_ratio")
	}
	return nil
}
