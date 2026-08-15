package inventory

import (
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

var models *engine.ModelRegistry

func trimFields(keys ...string) func(engine.HookContext) error {
	return func(hc engine.HookContext) error {
		for _, key := range keys {
			if v, ok := hc.Fields[key].(string); ok {
				hc.Fields[key] = strings.TrimSpace(v)
			}
		}
		return nil
	}
}

func rejectLedgerMutation(engine.HookContext) error {
	return i18n.Error("inventory.error.ledger_immutable")
}

func protectQuantCreate(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	return i18n.Error("inventory.error.quant_create")
}

func protectLedgerCreate(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	return i18n.Error("inventory.error.ledger_create")
}

func protectQuantUpdate(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	for _, key := range []string{"quantity", "reservedQty", "unitCost", "value"} {
		if _, ok := hc.Fields[key]; ok {
			return i18n.Error("inventory.error.quant_update")
		}
	}
	return nil
}

func protectQuantDelete(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	return i18n.Error("inventory.error.quant_delete")
}

func protectCostLayer(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	return i18n.Error("inventory.error.cost_layer")
}

func protectDoneMove(hc engine.HookContext) error {
	if isInternal(hc.Context) {
		return nil
	}
	if recStr(hc.Record, "state") == "done" {
		return i18n.Error("inventory.error.move_done")
	}
	return nil
}

func beforeUom(hc engine.HookContext) error {
	if err := trimFields("name", "symbol")(hc); err != nil {
		return err
	}
	if v, ok := hc.Fields["ratio"]; ok && asNumber(v) <= 0 {
		return i18n.Error("inventory.error.uom_ratio")
	}
	return nil
}

func beforeCreatePicking(hc engine.HookContext) error {
	if asString(hc.Fields["name"]) != "" {
		return nil
	}
	prefix := map[string]string{
		"incoming": "IN",
		"outgoing": "OUT",
		"internal": "INT",
		"return":   "RET",
	}[asString(hc.Fields["pickingType"])]
	if prefix == "" {
		prefix = "WH"
	}
	n := 1
	if models != nil {
		all, _ := models.List(hc.Context, hc.OrgID, "picking")
		n = len(all) + 1
	}
	hc.Fields["name"] = fmt.Sprintf("%s/%05d", prefix, n)
	return nil
}

func beforeUpdatePicking(hc engine.HookContext) error {
	prev, next := nextState(hc, "state")
	if next == prev {
		return nil
	}
	switch next {
	case "done":
		if prev == "cancelled" {
			return i18n.Error("inventory.error.picking_cancelled_complete")
		}
		return postPicking(hc)
	case "cancelled":
		if prev == "done" {
			return i18n.Error("inventory.error.picking_done_cancel")
		}
		return unreservePicking(hc)
	case "assigned":
		return reservePicking(hc)
	}
	return nil
}

func beforeDeletePicking(hc engine.HookContext) error {
	if recStr(hc.Record, "state") == "done" {
		return i18n.Error("inventory.error.picking_done_delete")
	}
	return nil
}

func beforeCreateMove(hc engine.HookContext) error {
	if asNumber(hc.Fields["quantity"]) <= 0 {
		return i18n.Error("inventory.error.qty_positive")
	}
	return assignFEFOLot(hc)
}

func beforeCreateAdjustment(hc engine.HookContext) error {
	if asString(hc.Fields["name"]) == "" {
		n := 1
		if models != nil {
			all, _ := models.List(hc.Context, hc.OrgID, "adjustment")
			n = len(all) + 1
		}
		hc.Fields["name"] = fmt.Sprintf("ADJ/%05d", n)
	}
	vid := asString(hc.Fields["variantId"])
	loc := asString(hc.Fields["locationId"])
	lot := asString(hc.Fields["lotId"])
	if models != nil && vid != "" && loc != "" {
		if q := findQuant(hc.Context, hc.OrgID, vid, loc, lot); q != nil {
			hc.Fields["systemQty"] = recNum(q, "quantity")
		}
	}
	return nil
}

func beforeUpdateAdjustment(hc engine.HookContext) error {
	prev, next := nextState(hc, "state")
	if next == "done" && prev != "done" {
		return postAdjustment(hc)
	}
	return nil
}

func beforeCreateRMA(hc engine.HookContext) error {
	if asString(hc.Fields["name"]) == "" {
		n := 1
		if models != nil {
			all, _ := models.List(hc.Context, hc.OrgID, "rma")
			n = len(all) + 1
		}
		hc.Fields["name"] = fmt.Sprintf("RMA/%05d", n)
	}
	return nil
}

func beforeUpdateRMA(hc engine.HookContext) error {
	prev, next := nextState(hc, "state")
	if next == "received" && prev != "received" && prev != "dispositioned" && prev != "closed" {
		return postRMA(hc)
	}
	return nil
}

func fieldNum(hc engine.HookContext, key string) float64 {
	if v, ok := hc.Fields[key]; ok {
		return asNumber(v)
	}
	if hc.Record != nil {
		return recNum(hc.Record, key)
	}
	return 0
}

func fieldStr(hc engine.HookContext, key string) string {
	if v, ok := hc.Fields[key]; ok && asString(v) != "" {
		return asString(v)
	}
	if hc.Record != nil {
		return recStr(hc.Record, key)
	}
	return ""
}

func computeReorder(hc engine.HookContext) error {
	method := fieldStr(hc, "method")
	lead := fieldNum(hc, "leadTimeDays")
	safety := fieldNum(hc, "safetyStock")
	demand := fieldNum(hc, "avgDailyDemand")
	minQty := fieldNum(hc, "minQty")
	maxQty := fieldNum(hc, "maxQty")

	rop := demand*lead + safety
	if method == "min_max" {
		rop = minQty
	}
	hc.Fields["reorderPoint"] = RoundQty(rop, 0.0001)

	onHand := 0.0
	if models != nil {
		onHand, _ = onHandAt(hc.Context, hc.OrgID, fieldStr(hc, "variantId"), fieldStr(hc, "locationId"))
	}
	order := maxQty - onHand
	if onHand >= rop && method != "min_max" {
		order = 0
	}
	if order < 0 {
		order = 0
	}
	hc.Fields["qtyToOrder"] = RoundQty(order, 0.0001)
	return nil
}

func registerHooks(app *engine.App) *engine.App {
	return app.
		Hooks("uom", engine.Hooks{BeforeCreate: beforeUom, BeforeUpdate: beforeUom}).
		Hooks("product", engine.Hooks{BeforeCreate: trimFields("name", "code"), BeforeUpdate: trimFields("name", "code")}).
		Hooks("product_variant", engine.Hooks{BeforeCreate: trimFields("sku", "name", "upc", "ean", "barcode"), BeforeUpdate: trimFields("sku", "name", "upc", "ean", "barcode")}).
		Hooks("location", engine.Hooks{BeforeCreate: trimFields("name", "code", "barcode"), BeforeUpdate: trimFields("name", "code", "barcode")}).
		Hooks("stock_lot", engine.Hooks{BeforeCreate: trimFields("name"), BeforeUpdate: trimFields("name")}).
		Hooks("stock_serial", engine.Hooks{BeforeCreate: trimFields("serial"), BeforeUpdate: trimFields("serial")}).
		Hooks("stock_quant", engine.Hooks{BeforeCreate: protectQuantCreate, BeforeUpdate: protectQuantUpdate, BeforeDelete: protectQuantDelete}).
		Hooks("stock_ledger", engine.Hooks{
			BeforeCreate: protectLedgerCreate,
			AfterCreate:  afterCreateLedger,
			BeforeUpdate: rejectLedgerMutation,
			BeforeDelete: rejectLedgerMutation,
		}).
		Hooks("cost_layer", engine.Hooks{BeforeCreate: protectCostLayer, BeforeUpdate: protectCostLayer, BeforeDelete: protectCostLayer}).
		Hooks("picking", engine.Hooks{
			BeforeCreate: beforeCreatePicking,
			BeforeUpdate: beforeUpdatePicking,
			BeforeDelete: beforeDeletePicking,
		}).
		Hooks("stock_move", engine.Hooks{BeforeCreate: beforeCreateMove, BeforeUpdate: protectDoneMove}).
		Hooks("adjustment", engine.Hooks{BeforeCreate: beforeCreateAdjustment, BeforeUpdate: beforeUpdateAdjustment}).
		Hooks("rma", engine.Hooks{BeforeCreate: beforeCreateRMA, BeforeUpdate: beforeUpdateRMA}).
		Hooks("reorder_rule", engine.Hooks{BeforeCreate: computeReorder, BeforeUpdate: computeReorder})
}
