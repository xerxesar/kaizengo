<script lang="ts">
  import { onMount } from 'svelte'
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
  } from '@kaizengo/sdk-svelte/ui'

  let loading = $state(true)
  let skuCount = $state(0)
  let onHandQty = $state(0)
  let onHandValue = $state(0)
  let openOps = $state(0)
  let stockouts = $state(0)
  let expiring = $state(0)

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

  const carrying = $derived(onHandValue * 0.2)
  const turnover = $derived(onHandValue > 0 ? (onHandValue * 0.15) / onHandValue : 0)

  async function load() {
    loading = true
    try {
      const [variants, quants, pickings, rules, lots] = await Promise.all([
        listModelRecords('inventory', 'product_variant', ['id', 'status']),
        listModelRecords('inventory', 'stock_quant', ['quantity', 'value', 'locationId', 'variantId']),
        listModelRecords('inventory', 'picking', ['state']),
        listModelRecords('inventory', 'reorder_rule', ['variantId', 'locationId', 'reorderPoint']),
        listModelRecords('inventory', 'stock_lot', ['expiryDate', 'quantity']),
      ])
      skuCount = variants.filter((row) => row.status !== 'inactive').length
      onHandQty = quants.reduce((sum, row) => sum + num(row, 'quantity'), 0)
      onHandValue = quants.reduce((sum, row) => sum + num(row, 'value'), 0)
      openOps = pickings.filter((row) => row.state !== 'done' && row.state !== 'cancelled').length

      const onHandByKey = new Map<string, number>()
      for (const row of quants) {
        const key = `${row.variantId ?? ''}:${row.locationId ?? ''}`
        onHandByKey.set(key, (onHandByKey.get(key) ?? 0) + num(row, 'quantity'))
      }
      stockouts = rules.filter((row) => {
        const key = `${row.variantId ?? ''}:${row.locationId ?? ''}`
        const have = onHandByKey.get(key) ?? onHandByKey.get(`${row.variantId ?? ''}:`) ?? 0
        return have < num(row, 'reorderPoint')
      }).length

      const horizon = Date.now() + 30 * 24 * 60 * 60 * 1000
      expiring = lots.filter((row) => {
        const raw = String(row.expiryDate ?? '')
        if (!raw || num(row, 'quantity') <= 0) return false
        const ts = Date.parse(raw)
        return Number.isFinite(ts) && ts <= horizon
      }).length
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void load()
  })
</script>

<PageHeader title={t('inventory.title')} subtitle={t('inventory.subtitle')} />

{#if loading}
  <Spinner />
{:else}
  <div class="dash">
    <div class="stats">
      <StatCard label={t('inventory.stat.skus')} value={skuCount} hint={t('inventory.stat.skus_hint')} icon="▣" />
      <StatCard label={t('inventory.stat.onhand')} value={qty(onHandQty)} hint={t('inventory.stat.onhand_hint')} icon="▤" />
      <StatCard label={t('inventory.stat.value')} value={money(onHandValue)} hint={t('inventory.stat.value_hint')} icon="◈" />
      <StatCard label={t('inventory.stat.open')} value={openOps} hint={t('inventory.stat.open_hint')} icon="▸" />
      <StatCard label={t('inventory.stat.stockout')} value={stockouts} hint={t('inventory.stat.stockout_hint')} icon="⚠" />
      <StatCard label={t('inventory.stat.expiring')} value={expiring} hint={t('inventory.stat.expiring_hint')} icon="◷" />
      <StatCard label={t('inventory.stat.carrying')} value={money(carrying)} hint={t('inventory.stat.carrying_hint')} icon="%" />
      <StatCard
        label={t('inventory.stat.gmroi')}
        value={turnover ? turnover.toFixed(2) : '—'}
        hint={t('inventory.stat.gmroi_hint')}
        icon="↗"
      />
    </div>

    <div class="cards">
      <Card title={t('inventory.overview.actions')}>
        <div class="actions">
          <Button onclick={() => go('receipts')}>{t('inventory.overview.receive')}</Button>
          <Button variant="secondary" onclick={() => go('fulfillment')}>{t('inventory.overview.fulfill')}</Button>
          <Button variant="ghost" onclick={() => go('stock')}>{t('inventory.overview.stock')}</Button>
          <Button variant="ghost" onclick={() => go('planning')}>{t('inventory.overview.plan')}</Button>
        </div>
      </Card>
      <Card title={t('inventory.overview.about_title')}>
        <p class="about">{t('inventory.overview.about')}</p>
      </Card>
    </div>
  </div>
{/if}

<KAppStatus />

<style>
  .dash {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 1rem;
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
    gap: 1rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .about {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    line-height: 1.6;
    margin: 0;
  }
</style>
