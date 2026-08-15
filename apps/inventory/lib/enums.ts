export const ENUMS: Record<string, string[]> = {
  'uom.uomType': ['reference', 'bigger', 'smaller'],
  'attribute.attributeType': ['select', 'text', 'number'],
  'product.tracking': ['none', 'lot', 'serial', 'both'],
  'product.costingMethod': ['fifo', 'lifo', 'moving_average', 'standard'],
  'product.dispatchStrategy': ['fifo', 'fefo', 'lifo'],
  'product.status': ['draft', 'active', 'discontinued'],
  'product_variant.status': ['active', 'inactive'],
  'location.locationType': [
    'warehouse',
    'zone',
    'aisle',
    'rack',
    'shelf',
    'bin',
    'supplier',
    'customer',
    'transit',
    'scrap',
    'inventory',
    'quarantine',
    'production',
  ],
  'location.usage': ['internal', 'virtual'],
  'stock_serial.status': ['available', 'reserved', 'in_transit', 'sold', 'scrapped', 'quarantine'],
  'stock_ledger.side': ['debit', 'credit'],
  'picking.pickingType': ['incoming', 'outgoing', 'internal', 'return'],
  'picking.state': ['draft', 'confirmed', 'assigned', 'done', 'cancelled'],
  'picking.qcStatus': ['pending', 'passed', 'failed', 'skipped'],
  'stock_move.state': ['draft', 'done', 'cancelled'],
  'adjustment.reason': ['cycle_count', 'physical', 'damage', 'found', 'data_fix'],
  'adjustment.state': ['draft', 'confirmed', 'done'],
  'rma.state': ['draft', 'received', 'dispositioned', 'closed'],
  'rma.disposition': ['restock', 'repair', 'quarantine', 'scrap', 'credit'],
  'reorder_rule.method': ['min_max', 'safety_stock', 'forecast'],
  'iot_device.deviceType': ['barcode_scanner', 'rfid_reader', 'mobile', 'mqtt_sensor', 'bluetooth_sensor'],
  'iot_device.status': ['online', 'offline', 'disabled'],
  'integration.kind': ['webhook', 'sap', 'netsuite', 'odoo', 'shopify', 'crm', 'custom'],
}

export function enumKey(model: string, field: string): string {
  const name = model.includes('.') ? model.slice(model.indexOf('.') + 1) : model
  return `${name}.${field}`
}
