import { createMemo, createSignal, onMount, Show, type JSX } from 'solid-js'
import {
  Button,
  Card,
  KAppStatus,
  PageHeader,
  Spinner,
  StatCard,
  listModelRecords,
  menuPagePath,
  navigateApp,
  t,
  type ModelRecord,
} from '@kaizengo/sdk-solid/ui'

export default function Dashboard(): JSX.Element {
  const [loading, setLoading] = createSignal(true)
  const [skuCount, setSkuCount] = createSignal(0)
  const [onHandQty, setOnHandQty] = createSignal(0)
  const [onHandValue, setOnHandValue] = createSignal(0)
  const [openOps, setOpenOps] = createSignal(0)
  const [stockouts, setStockouts] = createSignal(0)
  const [expiring, setExpiring] = createSignal(0)

  function go(page: string) {
    navigateApp(menuPagePath('inventory', page))
  }

  function num(row: ModelRecord, key: string): number {
    const v = row[key]
    const n = typeof v === 'number' ? v : Number(v)
    return Number.isFinite(n) ? n : 0
  }

  function money(n: number): string {
    return n.toLocaleString(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
  }

  function qty(n: number): string {
    return n.toLocaleString(undefined, { maximumFractionDigits: 2 })
  }

  const carrying = createMemo(() => onHandValue() * 0.2)
  const turnover = createMemo(() => (onHandValue() > 0 ? (onHandValue() * 0.15) / onHandValue() : 0))

  async function load() {
    setLoading(true)
    try {
      const [variants, quants, pickings, rules, lots] = await Promise.all([
        listModelRecords('inventory', 'product_variant', ['id', 'status']),
        listModelRecords('inventory', 'stock_quant', ['quantity', 'value', 'locationId', 'variantId']),
        listModelRecords('inventory', 'picking', ['state']),
        listModelRecords('inventory', 'reorder_rule', ['variantId', 'locationId', 'reorderPoint']),
        listModelRecords('inventory', 'stock_lot', ['expiryDate', 'quantity']),
      ])
      setSkuCount(variants.filter((row) => row.status !== 'inactive').length)
      setOnHandQty(quants.reduce((sum, row) => sum + num(row, 'quantity'), 0))
      setOnHandValue(quants.reduce((sum, row) => sum + num(row, 'value'), 0))
      setOpenOps(pickings.filter((row) => row.state !== 'done' && row.state !== 'cancelled').length)

      const onHandByKey = new Map<string, number>()
      for (const row of quants) {
        const key = `${row.variantId ?? ''}:${row.locationId ?? ''}`
        onHandByKey.set(key, (onHandByKey.get(key) ?? 0) + num(row, 'quantity'))
      }
      setStockouts(
        rules.filter((row) => {
          const key = `${row.variantId ?? ''}:${row.locationId ?? ''}`
          const have = onHandByKey.get(key) ?? onHandByKey.get(`${row.variantId ?? ''}:`) ?? 0
          return have < num(row, 'reorderPoint')
        }).length,
      )

      const horizon = Date.now() + 30 * 24 * 60 * 60 * 1000
      setExpiring(
        lots.filter((row) => {
          const raw = String(row.expiryDate ?? '')
          if (!raw || num(row, 'quantity') <= 0) return false
          const ts = Date.parse(raw)
          return Number.isFinite(ts) && ts <= horizon
        }).length,
      )
    } finally {
      setLoading(false)
    }
  }

  onMount(() => {
    void load()
  })

  return (
    <>
      <PageHeader title={t('inventory.title')} subtitle={t('inventory.subtitle')} />

      <Show when={!loading()} fallback={<Spinner />}>
        <div class="flex flex-col gap-5">
          <div class="grid grid-cols-[repeat(auto-fit,minmax(14rem,1fr))] gap-4">
            <StatCard label={t('inventory.stat.skus')} value={skuCount()} hint={t('inventory.stat.skus_hint')} icon="▣" />
            <StatCard label={t('inventory.stat.onhand')} value={qty(onHandQty())} hint={t('inventory.stat.onhand_hint')} icon="▤" />
            <StatCard label={t('inventory.stat.value')} value={money(onHandValue())} hint={t('inventory.stat.value_hint')} icon="◈" />
            <StatCard label={t('inventory.stat.open')} value={openOps()} hint={t('inventory.stat.open_hint')} icon="▸" />
            <StatCard label={t('inventory.stat.stockout')} value={stockouts()} hint={t('inventory.stat.stockout_hint')} icon="⚠" />
            <StatCard label={t('inventory.stat.expiring')} value={expiring()} hint={t('inventory.stat.expiring_hint')} icon="◷" />
            <StatCard label={t('inventory.stat.carrying')} value={money(carrying())} hint={t('inventory.stat.carrying_hint')} icon="%" />
            <StatCard
              label={t('inventory.stat.gmroi')}
              value={turnover() ? turnover().toFixed(2) : '—'}
              hint={t('inventory.stat.gmroi_hint')}
              icon="↗"
            />
          </div>

          <div class="grid grid-cols-[repeat(auto-fit,minmax(18rem,1fr))] gap-4">
            <Card title={t('inventory.overview.actions')}>
              <div class="flex flex-wrap gap-2">
                <Button onClick={() => go('receipts')}>{t('inventory.overview.receive')}</Button>
                <Button variant="secondary" onClick={() => go('fulfillment')}>
                  {t('inventory.overview.fulfill')}
                </Button>
                <Button variant="ghost" onClick={() => go('stock')}>
                  {t('inventory.overview.stock')}
                </Button>
                <Button variant="ghost" onClick={() => go('planning')}>
                  {t('inventory.overview.plan')}
                </Button>
              </div>
            </Card>
            <Card title={t('inventory.overview.about_title')}>
              <p class="m-0 text-sm leading-relaxed text-[var(--kg-text-secondary)]">{t('inventory.overview.about')}</p>
            </Card>
          </div>
        </div>
      </Show>

      <KAppStatus />
    </>
  )
}
