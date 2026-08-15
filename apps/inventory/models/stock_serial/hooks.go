package stockserial

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	trim := inv.TrimFields("serial")
	engine.RegisterModelHooks("inventory", "stock_serial", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
