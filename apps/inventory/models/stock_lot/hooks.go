package stocklot

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	trim := inv.TrimFields("name")
	engine.RegisterModelHooks("inventory", "stock_lot", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
