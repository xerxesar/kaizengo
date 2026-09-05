package product

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/engine"
)

func init() {
	trim := inv.TrimFields("name", "code")
	engine.RegisterModelHooks("inventory", "product", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
