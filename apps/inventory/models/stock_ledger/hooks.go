package stockledger

import (
	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func init() {
	engine.RegisterModelHooks("inventory", "stock_ledger", engine.Hooks{
		AfterCreate:  inv.AfterCreateLedger,
		BeforeUpdate: rejectMutation,
		BeforeDelete: rejectMutation,
	})
}

func rejectMutation(engine.HookContext) error {
	return i18n.Error("inventory.error.ledger_immutable")
}
