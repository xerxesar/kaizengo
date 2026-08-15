<script lang="ts">
  import {
    Card,
    FormField,
    KAppStatus,
    KForm,
    KTable,
    PageHeader,
    Select,
    t,
    type KFormFieldContext,
  } from '@kaizengo/sdk-svelte/ui'
  import { ENUMS, enumKey } from './enums'
  import RelationPicker from './RelationPicker.svelte'
  import SyncValue from './SyncValue.svelte'

  type Props = {
    pickingType?: string
    title: string
    subtitle?: string
  }

  let { pickingType = '', title, subtitle = '' }: Props = $props()

  let pickingTable = $state<{ refresh: () => Promise<void> }>()
  let moveTable = $state<{ refresh: () => Promise<void> }>()

  function enumOptions(model: string, fieldKey: string) {
    const values = ENUMS[enumKey(model, fieldKey)] ?? []
    return values.map((value) => {
      const key = `inventory.enum.${enumKey(model, fieldKey)}.${value}`
      const label = t(key)
      return { value, label: label === key ? value.replaceAll('_', ' ') : label }
    })
  }
</script>

<PageHeader {title} {subtitle} />

<div class="board">
  <Card title={t('inventory.operations.header')}>
    <p class="hint">{t('inventory.operations.hint')}</p>
    <KForm model="inventory.picking" onsuccess={() => void pickingTable?.refresh()}>
      {#snippet field(ctx: KFormFieldContext)}
        {@const enums = enumOptions('picking', ctx.field.key)}
        {#if ctx.field.key === 'pickingType' && pickingType}
          <SyncValue value={pickingType} current={ctx.draft[ctx.field.key]} setValue={ctx.setValue} />
          <FormField label={ctx.label} required={ctx.field.required}>
            <Select value={pickingType} options={enumOptions('picking', 'pickingType')} disabled />
          </FormField>
        {:else if ctx.field.relation}
          <FormField label={ctx.label} required={ctx.field.required}>
            <RelationPicker
              relation={ctx.field.relation}
              fromApp="inventory"
              value={String(ctx.draft[ctx.field.key] ?? '')}
              onchange={(id) => ctx.setValue(id)}
            />
          </FormField>
        {:else if enums.length}
          <FormField label={ctx.label} required={ctx.field.required}>
            <Select
              value={String(ctx.draft[ctx.field.key] ?? '')}
              options={enums}
              onchange={(e) => ctx.setValue((e.currentTarget as HTMLSelectElement).value)}
            />
          </FormField>
        {:else}
          {@render ctx.default()}
        {/if}
      {/snippet}
    </KForm>
    <KTable bind:this={pickingTable} model="inventory.picking" emptyMessage={t('inventory.empty.operations')} />
  </Card>

  <Card title={t('inventory.moves.title')}>
    <p class="hint">{t('inventory.moves.hint')}</p>
    <KForm model="inventory.stock_move" onsuccess={() => void moveTable?.refresh()}>
      {#snippet field(ctx: KFormFieldContext)}
        {@const enums = enumOptions('stock_move', ctx.field.key)}
        {#if ctx.field.relation}
          <FormField label={ctx.label} required={ctx.field.required}>
            <RelationPicker
              relation={ctx.field.relation}
              fromApp="inventory"
              value={String(ctx.draft[ctx.field.key] ?? '')}
              onchange={(id) => ctx.setValue(id)}
            />
          </FormField>
        {:else if enums.length}
          <FormField label={ctx.label} required={ctx.field.required}>
            <Select
              value={String(ctx.draft[ctx.field.key] ?? '')}
              options={enums}
              onchange={(e) => ctx.setValue((e.currentTarget as HTMLSelectElement).value)}
            />
          </FormField>
        {:else}
          {@render ctx.default()}
        {/if}
      {/snippet}
    </KForm>
    <KTable bind:this={moveTable} model="inventory.stock_move" emptyMessage={t('inventory.empty.moves')} />
  </Card>
</div>

<KAppStatus />

<style>
  .board {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .hint {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    margin: 0 0 1rem;
    line-height: 1.5;
  }
  .board :global(.composer) {
    flex-direction: column;
    align-items: stretch;
  }
</style>
