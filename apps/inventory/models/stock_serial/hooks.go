package stockserial

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/engine"
)

func init() {
	trim := inv.TrimFields("serial")
	engine.RegisterModelHooks("inventory", "stock_serial", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
