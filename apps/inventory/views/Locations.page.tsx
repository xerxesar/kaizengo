import { createMemo, createSignal, onMount, Show, type JSX } from 'solid-js'
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
} from '@kaizengo/sdk-solid/ui'
import { ENUMS, enumKey } from '../lib/enums'
import { RelationPicker } from '../lib/RelationPicker'

type Loc = ModelRecord & { name?: string; code?: string; locationType?: string; parentId?: string }

export default function Locations(): JSX.Element {
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [rows, setRows] = createSignal<Loc[]>([])
  const [selectedId, setSelectedId] = createSignal<string | null>(null)

  const selected = createMemo(() => rows().find((row) => row.id === selectedId()) ?? null)

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

  const tree = createMemo(() => buildTree(rows()))

  async function load() {
    setLoading(true)
    setError('')
    try {
      setRows(
        (await listModelRecords('inventory', 'location', [
          'name',
          'code',
          'locationType',
          'parentId',
          'usage',
          'barcode',
          'active',
        ])) as Loc[],
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  function renderField(ctx: KFormFieldContext) {
    const enums = enumOptions(ctx.field.key)
    if (ctx.field.relation) {
      return (
        <FormField label={ctx.label} required={ctx.field.required}>
          <RelationPicker
            relation={ctx.field.relation}
            fromApp="inventory"
            value={String(ctx.draft[ctx.field.key] ?? '')}
            onChange={(id) => ctx.setValue(id)}
          />
        </FormField>
      )
    }
    if (enums.length) {
      return (
        <FormField label={ctx.label} required={ctx.field.required}>
          <Select
            value={String(ctx.draft[ctx.field.key] ?? '')}
            options={enums}
            onChange={(value) => ctx.setValue(value)}
          />
        </FormField>
      )
    }
    return ctx.default()
  }

  onMount(() => {
    void load()
  })

  return (
    <>
      <PageHeader title={t('inventory.locations.title')} subtitle={t('inventory.locations.subtitle')} />

      <Show when={error()}>
        <Alert variant="danger" dismissible onDismiss={() => setError('')}>
          {error()}
        </Alert>
      </Show>

      <Show when={!loading()} fallback={<Spinner />}>
        <div class="grid grid-cols-1 items-start gap-5 md:grid-cols-2">
          <Card title={t('inventory.locations.tree')}>
            <Toolbar>
              <span class="text-sm text-[var(--kg-text-secondary)]">{t('inventory.locations.count', rows().length)}</span>
            </Toolbar>
            <TreeView nodes={tree()} selectedId={selectedId()} onSelect={(node) => setSelectedId(node.id)} />
          </Card>

          <div class="flex flex-col gap-5">
            <Card title={t('inventory.locations.add')}>
              <KForm model="inventory.location" onsuccess={() => void load()} field={renderField} />
            </Card>

            <Show when={selected()}>
              {(loc) => (
                <Card title={t('inventory.locations.detail')}>
                  <dl class="m-0 grid grid-cols-[8rem_1fr] gap-x-3 gap-y-1.5 text-sm">
                    <dt class="text-[var(--kg-text-secondary)]">{t('inventory.field.name')}</dt>
                    <dd class="m-0">{loc().name}</dd>
                    <dt class="text-[var(--kg-text-secondary)]">{t('inventory.field.code')}</dt>
                    <dd class="m-0">{loc().code}</dd>
                    <dt class="text-[var(--kg-text-secondary)]">{t('inventory.field.locationType')}</dt>
                    <dd class="m-0">{typeLabel(String(loc().locationType ?? ''))}</dd>
                    <dt class="text-[var(--kg-text-secondary)]">{t('inventory.field.id')}</dt>
                    <dd class="m-0 break-all font-mono text-xs">{loc().id}</dd>
                  </dl>
                </Card>
              )}
            </Show>
          </div>
        </div>
      </Show>

      <KAppStatus />
    </>
  )
}
