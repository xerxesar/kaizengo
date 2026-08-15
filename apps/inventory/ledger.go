package inventory

import (
	"context"
	"fmt"
	"log/slog"

	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/i18n"
)

func findQuant(ctx context.Context, orgID, variantID, locationID, lotID string) engine.Record {
	if models == nil || variantID == "" || locationID == "" {
		return nil
	}
	rows, err := models.ListBy(ctx, orgID, "stock_quant", "variantId", variantID)
	if err != nil {
		return nil
	}
	for _, row := range rows {
		if recStr(row, "locationId") == locationID && recStr(row, "lotId") == lotID {
			return row
		}
	}
	return nil
}

func onHandAt(ctx context.Context, orgID, variantID, locationID string) (qty, unitCost float64) {
	if models == nil || variantID == "" {
		return 0, 0
	}
	rows, err := models.ListBy(ctx, orgID, "stock_quant", "variantId", variantID)
	if err != nil {
		return 0, 0
	}
	var value float64
	for _, row := range rows {
		if locationID != "" && recStr(row, "locationId") != locationID {
			continue
		}
		q := recNum(row, "quantity")
		qty += q
		value += recNum(row, "value")
	}
	if qty != 0 {
		unitCost = RoundMoney(value / qty)
	}
	return RoundQty(qty, defaultRounding), unitCost
}

func rebuildQuant(ctx context.Context, orgID, userID, variantID, locationID, lotID string) error {
	if models == nil || variantID == "" || locationID == "" {
		return nil
	}
	ctx = withInternal(ctx)
	lines, err := models.ListBy(ctx, orgID, "stock_ledger", "variantId", variantID)
	if err != nil {
		return err
	}
	var qty, value float64
	uomID := ""
	for _, line := range lines {
		if recStr(line, "locationId") != locationID || recStr(line, "lotId") != lotID {
			continue
		}
		q := recNum(line, "quantity")
		v := recNum(line, "value")
		if recStr(line, "side") == "credit" {
			q, v = -q, -v
		}
		qty += q
		value += v
		if uomID == "" {
			uomID = recStr(line, "uomId")
		}
	}
	qty = RoundQty(qty, defaultRounding)
	value = RoundMoney(value)
	unitCost := 0.0
	if qty != 0 {
		unitCost = RoundMoney(value / qty)
	}

	fields := map[string]any{
		"variantId":  variantID,
		"locationId": locationID,
		"lotId":      lotID,
		"quantity":   qty,
		"unitCost":   unitCost,
		"value":      value,
		"uomId":      uomID,
	}
	existing := findQuant(ctx, orgID, variantID, locationID, lotID)
	if existing != nil {
		_, err = models.Update(ctx, orgID, "stock_quant", recStr(existing, "id"), fields)
		return err
	}
	if almostEqual(qty, 0) {
		return nil
	}
	_, err = models.Create(ctx, orgID, userID, "stock_quant", fields)
	return err
}

func rebuildLot(ctx context.Context, orgID, lotID string) error {
	if models == nil || lotID == "" {
		return nil
	}
	ctx = withInternal(ctx)
	lines, err := models.ListBy(ctx, orgID, "stock_ledger", "lotId", lotID)
	if err != nil {
		return err
	}
	var qty float64
	for _, line := range lines {
		q := recNum(line, "quantity")
		if recStr(line, "side") == "credit" {
			q = -q
		}
		qty += q
	}
	_, err = models.Update(ctx, orgID, "stock_lot", lotID, map[string]any{
		"quantity": RoundQty(qty, defaultRounding),
	})
	return err
}

func applySerial(ctx context.Context, orgID, serialID, locationID, locType string) error {
	if models == nil || serialID == "" {
		return nil
	}
	status := "available"
	switch locType {
	case "customer":
		status = "sold"
	case "scrap":
		status = "scrapped"
	case "transit":
		status = "in_transit"
	case "quarantine":
		status = "quarantine"
	}
	_, err := models.Update(withInternal(ctx), orgID, "stock_serial", serialID, map[string]any{
		"locationId": locationID,
		"status":     status,
	})
	return err
}

func afterCreateLedger(hc engine.HookContext) error {
	rec := hc.Record
	if rec == nil {
		rec = engine.Record(hc.Fields)
	}
	variantID := recStr(rec, "variantId")
	locationID := recStr(rec, "locationId")
	lotID := recStr(rec, "lotId")
	if err := rebuildQuant(hc.Context, hc.OrgID, hc.UserID, variantID, locationID, lotID); err != nil {
		slog.Error("inventory: rebuild quant", "err", err, "variant", variantID, "location", locationID)
	}
	if err := rebuildLot(hc.Context, hc.OrgID, lotID); err != nil {
		slog.Error("inventory: rebuild lot", "err", err, "lot", lotID)
	}
	if recStr(rec, "side") == "debit" {
		locType := ""
		if loc, err := models.Get(hc.Context, hc.OrgID, "location", locationID); err == nil {
			locType = recStr(loc, "locationType")
		}
		if err := applySerial(hc.Context, hc.OrgID, recStr(rec, "serialId"), locationID, locType); err != nil {
			slog.Error("inventory: apply serial", "err", err)
		}
	}
	return nil
}

func ledgerPosted(ctx context.Context, orgID, variantID, memo string) bool {
	if models == nil || memo == "" {
		return false
	}
	lines, err := models.ListBy(ctx, orgID, "stock_ledger", "variantId", variantID)
	if err != nil {
		return false
	}
	for _, line := range lines {
		if recStr(line, "memo") == memo {
			return true
		}
	}
	return false
}

func findLocationByType(ctx context.Context, orgID, locType string) (engine.Record, error) {
	if models == nil {
		return nil, fmt.Errorf("inventory models not initialized")
	}
	rows, err := models.ListBy(ctx, orgID, "location", "locationType", locType)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, i18n.Errorf("inventory.error.no_location", locType)
	}
	return rows[0], nil
}
