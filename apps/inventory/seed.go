package inventory

import (
	"context"
	"fmt"

	inv "kaizengo/apps/inventory/internal"
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

const seedAuthor = "00000000-0000-0000-0000-000000000001"

func setup(host *module.Host, events *engine.EventsSetup) error {
	inv.Registry = events.Models
	return seed(host, events)
}

func seed(host *module.Host, events *engine.EventsSetup) error {
	ctx := context.Background()
	existing, err := events.Models.ListAll(ctx, "product")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	idModels, err := engine.ModelsFromHost(host, "identity")
	if err != nil {
		return nil
	}
	orgs, err := idModels.ListAll(ctx, "organization")
	if err != nil || len(orgs) == 0 {
		return err
	}
	return seedDemo(ctx, events.Models, fmt.Sprint(orgs[0]["id"]))
}

type seeder struct {
	ctx   context.Context
	m     *engine.ModelRegistry
	orgID string
}

func (s seeder) create(model string, fields map[string]any) (string, error) {
	rec, err := s.m.Create(s.ctx, s.orgID, seedAuthor, model, fields)
	if err != nil {
		return "", fmt.Errorf("seed %s: %w", model, err)
	}
	return fmt.Sprint(rec["id"]), nil
}

func seedDemo(ctx context.Context, m *engine.ModelRegistry, orgID string) error {
	s := seeder{ctx: ctx, m: m, orgID: orgID}

	unitGroup, err := s.create("uom_group", map[string]any{"name": "Unit"})
	if err != nil {
		return err
	}
	weightGroup, err := s.create("uom_group", map[string]any{"name": "Weight"})
	if err != nil {
		return err
	}
	each, err := s.create("uom", map[string]any{"name": "Each", "symbol": "ea", "groupId": unitGroup, "ratio": 1.0, "uomType": "reference"})
	if err != nil {
		return err
	}
	if _, err := s.create("uom", map[string]any{"name": "Dozen", "symbol": "doz", "groupId": unitGroup, "ratio": 12.0, "uomType": "bigger"}); err != nil {
		return err
	}
	if _, err := s.create("uom", map[string]any{"name": "Case", "symbol": "cs", "groupId": unitGroup, "ratio": 24.0, "uomType": "bigger"}); err != nil {
		return err
	}
	kg, err := s.create("uom", map[string]any{"name": "Kilogram", "symbol": "kg", "groupId": weightGroup, "ratio": 1.0, "uomType": "reference"})
	if err != nil {
		return err
	}
	if _, err := s.create("uom", map[string]any{"name": "Gram", "symbol": "g", "groupId": weightGroup, "ratio": 0.001, "uomType": "smaller"}); err != nil {
		return err
	}

	color, err := s.create("attribute", map[string]any{"name": "Finish", "code": "finish", "attributeType": "select"})
	if err != nil {
		return err
	}
	if _, err := s.create("attribute_value", map[string]any{"attributeId": color, "name": "Zinc", "code": "zinc"}); err != nil {
		return err
	}
	if _, err := s.create("attribute_value", map[string]any{"attributeId": color, "name": "Stainless", "code": "ss"}); err != nil {
		return err
	}

	wh, err := s.create("location", map[string]any{"name": "Main Warehouse", "code": "WH", "locationType": "warehouse", "usage": "internal", "barcode": "LOC-WH"})
	if err != nil {
		return err
	}
	recv, err := s.create("location", map[string]any{"name": "Receiving", "code": "WH-IN", "locationType": "zone", "parentId": wh, "usage": "internal"})
	if err != nil {
		return err
	}
	storage, err := s.create("location", map[string]any{"name": "Storage", "code": "WH-ST", "locationType": "zone", "parentId": wh, "usage": "internal"})
	if err != nil {
		return err
	}
	aisle, err := s.create("location", map[string]any{"name": "Aisle A", "code": "WH-ST-A", "locationType": "aisle", "parentId": storage, "usage": "internal"})
	if err != nil {
		return err
	}
	rack, err := s.create("location", map[string]any{"name": "Rack A1", "code": "WH-ST-A1", "locationType": "rack", "parentId": aisle, "usage": "internal"})
	if err != nil {
		return err
	}
	bin, err := s.create("location", map[string]any{"name": "Bin A1-01", "code": "WH-ST-A1-01", "locationType": "bin", "parentId": rack, "usage": "internal", "barcode": "BIN-A101"})
	if err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Packing", "code": "WH-PK", "locationType": "zone", "parentId": wh, "usage": "internal"}); err != nil {
		return err
	}
	ship, err := s.create("location", map[string]any{"name": "Shipping", "code": "WH-OUT", "locationType": "zone", "parentId": wh, "usage": "internal"})
	if err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "In Transit", "code": "TRANSIT", "locationType": "transit", "usage": "virtual"}); err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Suppliers", "code": "SUPP", "locationType": "supplier", "usage": "virtual"}); err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Customers", "code": "CUST", "locationType": "customer", "usage": "virtual"}); err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Quarantine", "code": "QA", "locationType": "quarantine", "usage": "internal"}); err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Scrap", "code": "SCRAP", "locationType": "scrap", "usage": "virtual"}); err != nil {
		return err
	}
	if _, err := s.create("location", map[string]any{"name": "Inventory Adjustment", "code": "INVADJ", "locationType": "inventory", "usage": "virtual"}); err != nil {
		return err
	}

	bolt, err := s.create("product", map[string]any{
		"name": "Industrial Bolt M8", "code": "BOLT-M8", "category": "Fasteners",
		"tracking": "none", "costingMethod": "moving_average", "dispatchStrategy": "fifo",
		"status": "active", "description": "M8 hex bolt, zinc or stainless finish.",
	})
	if err != nil {
		return err
	}
	boltSKU, err := s.create("product_variant", map[string]any{
		"productId": bolt, "sku": "BOLT-M8-ZINC", "name": "Bolt M8 Zinc",
		"upc": "012345678905", "buyUomId": each, "stockUomId": each, "sellUomId": each,
		"standardCost": 0.18, "listPrice": 0.35, "averageCost": 0.18, "status": "active",
		"attributes": `{"finish":"zinc"}`,
	})
	if err != nil {
		return err
	}

	chem, err := s.create("product", map[string]any{
		"name": "Reagent A", "code": "CHEM-A", "category": "Chemicals",
		"tracking": "lot", "costingMethod": "fifo", "dispatchStrategy": "fefo",
		"shelfLifeDays": 365, "status": "active",
		"description": "Lot-tracked reagent with FEFO dispatch and FIFO costing.",
	})
	if err != nil {
		return err
	}
	chemSKU, err := s.create("product_variant", map[string]any{
		"productId": chem, "sku": "CHEM-A-1L", "name": "Reagent A 1L",
		"ean": "4006381333931", "buyUomId": kg, "stockUomId": kg, "sellUomId": kg,
		"standardCost": 12.5, "listPrice": 19.0, "averageCost": 12.5, "status": "active",
	})
	if err != nil {
		return err
	}

	sensor, err := s.create("product", map[string]any{
		"name": "Calibrated Sensor X1", "code": "SENSOR-X1", "category": "Instruments",
		"tracking": "serial", "costingMethod": "standard", "dispatchStrategy": "fifo",
		"status": "active", "description": "Serialized instrument with standard costing.",
	})
	if err != nil {
		return err
	}
	sensorSKU, err := s.create("product_variant", map[string]any{
		"productId": sensor, "sku": "SENSOR-X1", "name": "Sensor X1",
		"barcode": "SEN-X1", "buyUomId": each, "stockUomId": each, "sellUomId": each,
		"standardCost": 240, "listPrice": 399, "averageCost": 240, "status": "active",
	})
	if err != nil {
		return err
	}

	if _, err := s.create("stock_lot", map[string]any{
		"variantId": chemSKU, "name": "LOT-2026-001", "expiryDate": "2027-03-01", "manufacturedDate": "2026-03-01",
	}); err != nil {
		return err
	}
	if _, err := s.create("stock_serial", map[string]any{
		"variantId": sensorSKU, "serial": "SN-X1-00041", "locationId": bin, "status": "available",
	}); err != nil {
		return err
	}

	if _, err := s.create("reorder_rule", map[string]any{
		"variantId": boltSKU, "locationId": bin, "method": "safety_stock",
		"minQty": 100, "maxQty": 800, "safetyStock": 80, "leadTimeDays": 14, "avgDailyDemand": 12,
	}); err != nil {
		return err
	}

	if _, err := s.create("iot_device", map[string]any{
		"name": "Dock door RFID", "deviceType": "rfid_reader", "identifier": "rfid://wh/dock-1",
		"locationId": recv, "status": "offline", "config": `{"protocol":"mqtt"}`,
	}); err != nil {
		return err
	}
	if _, err := s.create("iot_device", map[string]any{
		"name": "Handheld scanner", "deviceType": "barcode_scanner", "identifier": "SCAN-1042",
		"locationId": ship, "status": "online",
	}); err != nil {
		return err
	}
	if _, err := s.create("integration", map[string]any{
		"name": "ERP stock webhook", "kind": "webhook", "endpoint": "https://erp.example.local/hooks/stock",
		"events": `["inventory.StockLedgerCreated"]`, "active": true,
	}); err != nil {
		return err
	}

	return nil
}
