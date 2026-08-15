<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Card,
    FormField,
    KAppStatus,
    KForm,
    PageHeader,
    Select,
    Spinner,
    Toolbar,
    TreeView,
    listModelRecords,
    t,
    type KFormFieldContext,
    type ModelRecord,
    type TreeNode,
  } from '@kaizengo/sdk-svelte/ui'
  import { ENUMS, enumKey } from '../lib/enums'
  import RelationPicker from '../lib/RelationPicker.svelte'

  type Loc = ModelRecord & { name?: string; code?: string; locationType?: string; parentId?: string }

  let loading = $state(true)
  let error = $state('')
  let rows = $state<Loc[]>([])
  let selectedId = $state<string | null>(null)

  const selected = $derived(rows.find((row) => row.id === selectedId) ?? null)

  function enumOptions(fieldKey: string) {
    return (ENUMS[enumKey('location', fieldKey)] ?? []).map((value) => {
      const key = `inventory.enum.location.${fieldKey}.${value}`
      const label = t(key)
      return { value, label: label === key ? value.replaceAll('_', ' ') : label }
    })
  }

  function typeLabel(value: string): string {
    const key = `inventory.enum.location.locationType.${value}`
    const label = t(key)
    return label === key ? value : label
  }

  function buildTree(items: Loc[]): TreeNode<Loc>[] {
    const byParent = new Map<string, Loc[]>()
    for (const item of items) {
      const parent = String(item.parentId ?? '')
      const list = byParent.get(parent) ?? []
      list.push(item)
      byParent.set(parent, list)
    }
    const walk = (parent: string): TreeNode<Loc>[] =>
      (byParent.get(parent) ?? []).map((item) => ({
        id: String(item.id),
        label: `${item.name ?? item.code ?? item.id}`,
        meta: typeLabel(String(item.locationType ?? '')),
        data: item,
        children: walk(String(item.id)),
      }))
    return walk('')
  }

  const tree = $derived(buildTree(rows))

  async function load() {
    loading = true
    error = ''
    try {
      rows = (await listModelRecords('inventory', 'location', [
        'name',
        'code',
        'locationType',
        'parentId',
        'usage',
        'barcode',
        'active',
      ])) as Loc[]
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void load()
  })
</script>

<PageHeader title={t('inventory.locations.title')} subtitle={t('inventory.locations.subtitle')} />

{#if error}
  <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
{/if}

{#if loading}
  <Spinner />
{:else}
  <div class="layout">
    <Card title={t('inventory.locations.tree')}>
      <Toolbar>
        <span class="count">{t('inventory.locations.count', rows.length)}</span>
      </Toolbar>
      <TreeView nodes={tree} {selectedId} onselect={(node) => (selectedId = node.id)} />
    </Card>

    <div class="side">
      <Card title={t('inventory.locations.add')}>
        <KForm model="inventory.location" onsuccess={() => void load()}>
          {#snippet field(ctx: KFormFieldContext)}
            {@const enums = enumOptions(ctx.field.key)}
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
      </Card>

      {#if selected}
        <Card title={t('inventory.locations.detail')}>
          <dl class="meta">
            <dt>{t('inventory.field.name')}</dt>
            <dd>{selected.name}</dd>
            <dt>{t('inventory.field.code')}</dt>
            <dd>{selected.code}</dd>
            <dt>{t('inventory.field.locationType')}</dt>
            <dd>{typeLabel(String(selected.locationType ?? ''))}</dd>
            <dt>{t('inventory.field.id')}</dt>
            <dd class="mono">{selected.id}</dd>
          </dl>
        </Card>
      {/if}
    </div>
  </div>
{/if}

<KAppStatus />

<style>
  .layout {
    display: grid;
    grid-template-columns: minmax(16rem, 1fr) minmax(18rem, 1fr);
    gap: 1.25rem;
    align-items: start;
  }
  @media (max-width: 48rem) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
  .side {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .count {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
  }
  .meta {
    display: grid;
    grid-template-columns: 8rem 1fr;
    gap: 0.35rem 0.75rem;
    margin: 0;
    font-size: 0.875rem;
  }
  .meta dt {
    color: var(--kg-text-secondary);
  }
  .meta dd {
    margin: 0;
  }
  .mono {
    font-family: var(--kg-font-mono, ui-monospace, monospace);
    font-size: 0.75rem;
    word-break: break-all;
  }
  .side :global(.composer) {
    flex-direction: column;
    align-items: stretch;
  }
</style>
