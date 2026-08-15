package productvariant

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	trim := inv.TrimFields("sku", "name", "upc", "ean", "barcode")
	engine.RegisterModelHooks("inventory", "product_variant", engine.Hooks{
		BeforeCreate: trim,
		BeforeUpdate: trim,
	})
}
