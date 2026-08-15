package location

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	trim := inv.TrimFields("name", "code", "barcode")
	engine.RegisterModelHooks("inventory", "location", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
