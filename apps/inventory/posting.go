package inventory

import (
	"context"
	"fmt"
	"math"
	"time"

	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func mergedRecord(hc engine.HookContext) engine.Record {
	rec := engine.Record{}
	for k, v := range hc.Record {
		rec[k] = v
	}
	for k, v := range hc.Fields {
		rec[k] = v
	}
	rec["id"] = hc.RecordID
	return rec
}

func cloneFields(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func convertToStock(ctx context.Context, orgID string, variant engine.Record, qty float64, fromUomID string) (float64, string, error) {
	stockUomID := recStr(variant, "stockUomId")
	if fromUomID == "" {
		fromUomID = stockUomID
	}
	if fromUomID == "" || fromUomID == stockUomID {
		if stockUomID == "" {
			stockUomID = fromUomID
		}
		return qty, stockUomID, nil
	}
	from, err := models.Get(ctx, orgID, "uom", fromUomID)
	if err != nil {
		return 0, "", fmt.Errorf("from uom: %w", err)
	}
	to, err := models.Get(ctx, orgID, "uom", stockUomID)
	if err != nil {
		return 0, "", fmt.Errorf("stock uom: %w", err)
	}
	if recStr(from, "groupId") != recStr(to, "groupId") {
		return 0, "", i18n.Error("inventory.error.uom_convert")
	}
	converted, err := ConvertQty(qty, recNum(from, "ratio"), recNum(to, "ratio"), recNum(to, "rounding"))
	return converted, stockUomID, err
}

func loadLayers(ctx context.Context, orgID, variantID string) ([]CostLayer, error) {
	rows, err := models.ListBy(ctx, orgID, "cost_layer", "variantId", variantID)
	if err != nil {
		return nil, err
	}
	out := make([]CostLayer, 0, len(rows))
	for _, row := range rows {
		rem := recNum(row, "quantity")
		if rem <= 0 {
			continue
		}
		out = append(out, CostLayer{
			ID:         recStr(row, "id"),
			Remaining:  rem,
			UnitCost:   recNum(row, "unitCost"),
			ReceivedAt: recStr(row, "receivedAt"),
		})
	}
	return out, nil
}

func saveLayers(ctx context.Context, orgID string, layers []CostLayer) error {
	ctx = withInternal(ctx)
	for _, layer := range layers {
		if _, err := models.Update(ctx, orgID, "cost_layer", layer.ID, map[string]any{
			"quantity": layer.Remaining,
		}); err != nil {
			return err
		}
	}
	return nil
}

func resolveAndApplyCost(ctx context.Context, orgID, userID string, product, variant, move engine.Record, qty float64, incoming bool) (float64, error) {
	method := recStr(product, "costingMethod")
	stated := recNum(move, "unitCost")
	std := recNum(variant, "standardCost")
	avg := recNum(variant, "averageCost")
	if avg == 0 {
		avg = std
	}
	variantID := recStr(variant, "id")

	if incoming {
		cost := stated
		if cost <= 0 {
			cost = std
		}
		if cost <= 0 {
			cost = avg
		}
		cost = RoundMoney(cost)
		if method == "fifo" || method == "lifo" || method == "moving_average" {
			if _, err := models.Create(withInternal(ctx), orgID, userID, "cost_layer", map[string]any{
				"variantId":    variantID,
				"lotId":        recStr(move, "lotId"),
				"quantity":     qty,
				"originalQty":  qty,
				"unitCost":     cost,
				"receivedAt":   time.Now().UTC().Format(time.RFC3339),
				"sourceMoveId": recStr(move, "id"),
			}); err != nil {
				return 0, err
			}
		}
		if method == "moving_average" {
			onHand, onHandCost := onHandAt(ctx, orgID, variantID, "")
			newAvg := movingAverage(onHand, onHandCost, qty, cost)
			if _, err := models.Update(withInternal(ctx), orgID, "product_variant", variantID, map[string]any{
				"averageCost": newAvg,
			}); err != nil {
				return 0, err
			}
		}
		return cost, nil
	}

	switch method {
	case "standard":
		if std > 0 {
			return RoundMoney(std), nil
		}
		return RoundMoney(avg), nil
	case "fifo", "lifo":
		layers, err := loadLayers(ctx, orgID, variantID)
		if err != nil {
			return 0, err
		}
		unit, remaining, err := ConsumeLayers(layers, qty, method == "lifo")
		if err != nil {
			return 0, err
		}
		if err := saveLayers(ctx, orgID, remaining); err != nil {
			return 0, err
		}
		if unit <= 0 {
			if stated > 0 {
				return RoundMoney(stated), nil
			}
			return RoundMoney(avg), nil
		}
		return unit, nil
	default:
		if avg > 0 {
			return RoundMoney(avg), nil
		}
		if stated > 0 {
			return RoundMoney(stated), nil
		}
		return RoundMoney(std), nil
	}
}

func validateTracking(product, move engine.Record) error {
	switch recStr(product, "tracking") {
	case "serial":
		if recStr(move, "serialId") == "" {
			return i18n.Error("inventory.error.serial_required")
		}
	case "lot":
		if recStr(move, "lotId") == "" {
			return i18n.Error("inventory.error.lot_required")
		}
	case "both":
		if recStr(move, "serialId") == "" || recStr(move, "lotId") == "" {
			return i18n.Error("inventory.error.lot_and_serial_required")
		}
	}
	return nil
}

func postMove(ctx context.Context, orgID, userID string, picking, move engine.Record) error {
	variantID := recStr(move, "variantId")
	variant, err := models.Get(ctx, orgID, "product_variant", variantID)
	if err != nil {
		return fmt.Errorf("variant: %w", err)
	}
	product, err := models.Get(ctx, orgID, "product", recStr(variant, "productId"))
	if err != nil {
		return fmt.Errorf("product: %w", err)
	}
	if err := validateTracking(product, move); err != nil {
		return err
	}

	fromID := recStr(move, "fromLocationId")
	if fromID == "" {
		fromID = recStr(picking, "sourceLocationId")
	}
	toID := recStr(move, "toLocationId")
	if toID == "" {
		toID = recStr(picking, "destLocationId")
	}
	if fromID == "" || toID == "" {
		return i18n.Error("inventory.error.move_locations")
	}

	qty := recNum(move, "quantity")
	if qty <= 0 {
		return i18n.Error("inventory.error.move_qty")
	}
	stockQty, stockUom, err := convertToStock(ctx, orgID, variant, qty, recStr(move, "uomId"))
	if err != nil {
		return err
	}

	pickingType := recStr(picking, "pickingType")
	incoming := pickingType == "incoming" || pickingType == "return"
	unitCost, err := resolveAndApplyCost(ctx, orgID, userID, product, variant, move, stockQty, incoming)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	value := RoundMoney(stockQty * unitCost)
	memo := recStr(picking, "name")
	if origin := recStr(picking, "origin"); origin != "" {
		memo += " / " + origin
	}

	base := map[string]any{
		"pickingId":  recStr(picking, "id"),
		"moveId":     recStr(move, "id"),
		"variantId":  variantID,
		"lotId":      recStr(move, "lotId"),
		"serialId":   recStr(move, "serialId"),
		"quantity":   stockQty,
		"uomId":      stockUom,
		"unitCost":   unitCost,
		"value":      value,
		"occurredAt": now,
		"memo":       memo,
	}
	credit := cloneFields(base)
	credit["locationId"] = fromID
	credit["side"] = "credit"
	debit := cloneFields(base)
	debit["locationId"] = toID
	debit["side"] = "debit"

	if _, err := models.Create(withInternal(ctx), orgID, userID, "stock_ledger", credit); err != nil {
		return err
	}
	if _, err := models.Create(withInternal(ctx), orgID, userID, "stock_ledger", debit); err != nil {
		return err
	}

	_, err = models.Update(withInternal(ctx), orgID, "stock_move", recStr(move, "id"), map[string]any{
		"state":          "done",
		"doneQty":        stockQty,
		"unitCost":       unitCost,
		"fromLocationId": fromID,
		"toLocationId":   toID,
	})
	return err
}

func postPicking(hc engine.HookContext) error {
	if models == nil {
		return fmt.Errorf("inventory models not initialized")
	}
	existing, err := models.ListBy(hc.Context, hc.OrgID, "stock_ledger", "pickingId", hc.RecordID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	moves, err := models.ListBy(hc.Context, hc.OrgID, "stock_move", "pickingId", hc.RecordID)
	if err != nil {
		return err
	}
	if len(moves) == 0 {
		return i18n.Error("inventory.error.picking_no_moves")
	}
	picking := mergedRecord(hc)
	for _, move := range moves {
		if recStr(move, "state") == "cancelled" {
			continue
		}
		if err := postMove(hc.Context, hc.OrgID, hc.UserID, picking, move); err != nil {
			return err
		}
	}
	hc.Fields["doneDate"] = time.Now().UTC().Format(time.RFC3339)
	return nil
}

func reservePicking(hc engine.HookContext) error {
	if models == nil {
		return nil
	}
	picking := mergedRecord(hc)
	moves, err := models.ListBy(hc.Context, hc.OrgID, "stock_move", "pickingId", hc.RecordID)
	if err != nil {
		return err
	}
	ctx := withInternal(hc.Context)
	for _, move := range moves {
		if recStr(move, "state") == "cancelled" {
			continue
		}
		fromID := recStr(move, "fromLocationId")
		if fromID == "" {
			fromID = recStr(picking, "sourceLocationId")
		}
		variant, err := models.Get(hc.Context, hc.OrgID, "product_variant", recStr(move, "variantId"))
		if err != nil {
			return err
		}
		need, _, err := convertToStock(hc.Context, hc.OrgID, variant, recNum(move, "quantity"), recStr(move, "uomId"))
		if err != nil {
			return err
		}
		q := findQuant(hc.Context, hc.OrgID, recStr(move, "variantId"), fromID, recStr(move, "lotId"))
		if q == nil {
			return i18n.Error("inventory.error.no_qty_reserve")
		}
		avail := recNum(q, "quantity") - recNum(q, "reservedQty")
		if avail+defaultRounding < need {
			return i18n.Error("inventory.error.insufficient_reserve")
		}
		if _, err := models.Update(ctx, hc.OrgID, "stock_quant", recStr(q, "id"), map[string]any{
			"reservedQty": recNum(q, "reservedQty") + need,
		}); err != nil {
			return err
		}
	}
	return nil
}

func unreservePicking(hc engine.HookContext) error {
	prev, _ := nextState(hc, "state")
	if prev != "assigned" || models == nil {
		return nil
	}
	picking := mergedRecord(hc)
	moves, err := models.ListBy(hc.Context, hc.OrgID, "stock_move", "pickingId", hc.RecordID)
	if err != nil {
		return err
	}
	ctx := withInternal(hc.Context)
	for _, move := range moves {
		fromID := recStr(move, "fromLocationId")
		if fromID == "" {
			fromID = recStr(picking, "sourceLocationId")
		}
		variant, err := models.Get(hc.Context, hc.OrgID, "product_variant", recStr(move, "variantId"))
		if err != nil {
			continue
		}
		need, _, err := convertToStock(hc.Context, hc.OrgID, variant, recNum(move, "quantity"), recStr(move, "uomId"))
		if err != nil {
			continue
		}
		q := findQuant(hc.Context, hc.OrgID, recStr(move, "variantId"), fromID, recStr(move, "lotId"))
		if q == nil {
			continue
		}
		next := recNum(q, "reservedQty") - need
		if next < 0 {
			next = 0
		}
		_, _ = models.Update(ctx, hc.OrgID, "stock_quant", recStr(q, "id"), map[string]any{
			"reservedQty": next,
		})
	}
	return nil
}

func postLedgerPair(ctx context.Context, orgID, userID, variantID, fromID, toID, lotID, serialID, uomID, memo string, qty, unitCost float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	value := RoundMoney(qty * unitCost)
	base := map[string]any{
		"variantId":  variantID,
		"lotId":      lotID,
		"serialId":   serialID,
		"quantity":   qty,
		"uomId":      uomID,
		"unitCost":   unitCost,
		"value":      value,
		"occurredAt": now,
		"memo":       memo,
	}
	credit := cloneFields(base)
	credit["locationId"] = fromID
	credit["side"] = "credit"
	debit := cloneFields(base)
	debit["locationId"] = toID
	debit["side"] = "debit"
	if _, err := models.Create(withInternal(ctx), orgID, userID, "stock_ledger", credit); err != nil {
		return err
	}
	_, err := models.Create(withInternal(ctx), orgID, userID, "stock_ledger", debit)
	return err
}

func postAdjustment(hc engine.HookContext) error {
	if models == nil {
		return fmt.Errorf("inventory models not initialized")
	}
	rec := mergedRecord(hc)
	variantID := recStr(rec, "variantId")
	memo := "adjustment:" + hc.RecordID
	if ledgerPosted(hc.Context, hc.OrgID, variantID, memo) {
		return nil
	}
	counted := recNum(rec, "countedQty")
	system := recNum(rec, "systemQty")
	delta := RoundQty(counted-system, defaultRounding)
	if almostEqual(delta, 0) {
		return nil
	}
	inv, err := findLocationByType(hc.Context, hc.OrgID, "inventory")
	if err != nil {
		return err
	}
	locID := recStr(rec, "locationId")
	fromID, toID := locID, recStr(inv, "id")
	incoming := false
	qty := math.Abs(delta)
	if delta > 0 {
		fromID, toID = recStr(inv, "id"), locID
		incoming = true
	}
	variant, err := models.Get(hc.Context, hc.OrgID, "product_variant", variantID)
	if err != nil {
		return err
	}
	product, err := models.Get(hc.Context, hc.OrgID, "product", recStr(variant, "productId"))
	if err != nil {
		return err
	}
	move := engine.Record{"id": hc.RecordID, "lotId": recStr(rec, "lotId"), "unitCost": recNum(variant, "averageCost")}
	unitCost, err := resolveAndApplyCost(hc.Context, hc.OrgID, hc.UserID, product, variant, move, qty, incoming)
	if err != nil {
		return err
	}
	return postLedgerPair(hc.Context, hc.OrgID, hc.UserID, variantID, fromID, toID, recStr(rec, "lotId"), "", recStr(variant, "stockUomId"), memo, qty, unitCost)
}

func postRMA(hc engine.HookContext) error {
	if models == nil {
		return fmt.Errorf("inventory models not initialized")
	}
	rec := mergedRecord(hc)
	variantID := recStr(rec, "variantId")
	memo := "rma:" + hc.RecordID
	if ledgerPosted(hc.Context, hc.OrgID, variantID, memo) {
		return nil
	}
	customer, err := findLocationByType(hc.Context, hc.OrgID, "customer")
	if err != nil {
		return err
	}
	destID := recStr(rec, "locationId")
	switch recStr(rec, "disposition") {
	case "quarantine", "repair":
		if loc, err := findLocationByType(hc.Context, hc.OrgID, "quarantine"); err == nil {
			destID = recStr(loc, "id")
		}
	case "scrap":
		if loc, err := findLocationByType(hc.Context, hc.OrgID, "scrap"); err == nil {
			destID = recStr(loc, "id")
		}
	case "credit":
		return nil
	}
	if destID == "" {
		if loc, err := findLocationByType(hc.Context, hc.OrgID, "warehouse"); err == nil {
			destID = recStr(loc, "id")
		}
	}
	if destID == "" {
		return i18n.Error("inventory.error.rma_location")
	}
	variant, err := models.Get(hc.Context, hc.OrgID, "product_variant", variantID)
	if err != nil {
		return err
	}
	product, err := models.Get(hc.Context, hc.OrgID, "product", recStr(variant, "productId"))
	if err != nil {
		return err
	}
	qty := recNum(rec, "quantity")
	if qty <= 0 {
		return i18n.Error("inventory.error.rma_qty")
	}
	move := engine.Record{"id": hc.RecordID, "unitCost": recNum(variant, "averageCost")}
	unitCost, err := resolveAndApplyCost(hc.Context, hc.OrgID, hc.UserID, product, variant, move, qty, true)
	if err != nil {
		return err
	}
	return postLedgerPair(hc.Context, hc.OrgID, hc.UserID, variantID, recStr(customer, "id"), destID, "", "", recStr(variant, "stockUomId"), memo, qty, unitCost)
}

func assignFEFOLot(hc engine.HookContext) error {
	if models == nil {
		return nil
	}
	if asString(hc.Fields["lotId"]) != "" {
		return nil
	}
	variantID := asString(hc.Fields["variantId"])
	if variantID == "" {
		return nil
	}
	variant, err := models.Get(hc.Context, hc.OrgID, "product_variant", variantID)
	if err != nil {
		return nil
	}
	product, err := models.Get(hc.Context, hc.OrgID, "product", recStr(variant, "productId"))
	if err != nil {
		return nil
	}
	tracking := recStr(product, "tracking")
	if tracking != "lot" && tracking != "both" {
		return nil
	}
	strategy := recStr(product, "dispatchStrategy")
	lots, err := models.ListBy(hc.Context, hc.OrgID, "stock_lot", "variantId", variantID)
	if err != nil || len(lots) == 0 {
		return nil
	}
	best := engine.Record(nil)
	for _, lot := range lots {
		if recNum(lot, "quantity") <= 0 {
			continue
		}
		if best == nil {
			best = lot
			continue
		}
		a, b := recStr(lot, "expiryDate"), recStr(best, "expiryDate")
		switch strategy {
		case "fefo":
			if a != "" && (b == "" || a < b) {
				best = lot
			}
		default:
			if recStr(lot, "createdAt") < recStr(best, "createdAt") {
				best = lot
			}
		}
	}
	if best != nil {
		hc.Fields["lotId"] = recStr(best, "id")
	}
	return nil
}
