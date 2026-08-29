import { createMemo, createSignal, For, onMount, Show, type JSX } from 'solid-js'
import {
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
  SearchableCombobox,
  FormActions,
  FormField,
  Input,
  KAppStatus,
  Select,
  Spinner,
  createModelRecord,
  deleteModelRecord,
  fetchACLActions,
  fetchResources,
  listModelRecords,
  t,
  updateModelRecord,
  type ModelField,
} from '@kaizengo/sdk-solid/ui'
import type { AclEntry, Role } from '../lib/types'

const SEED_AUTHOR = '00000000-0000-0000-0000-000000000001'

const aclCreateFields: ModelField[] = [
  { key: 'name', type: 'string', required: true, label: 'name' },
  { key: 'roleId', type: 'many2one', required: true, label: 'roleId' },
  { key: 'effect', type: 'string', required: true, label: 'effect' },
  { key: 'resource', type: 'string', required: true, label: 'resource' },
  { key: 'actions', type: 'string', required: true, label: 'actions' },
  { key: 'fields', type: 'string', required: true, label: 'fields' },
  { key: 'domain', type: 'string', required: true, label: 'domain' },
  { key: 'priority', type: 'int', required: true, label: 'priority' },
  { key: 'active', type: 'bool', required: true, label: 'active' },
]

type ResourceOption = { value: string; label: string }
type SourceFilter = 'all' | 'seed' | 'override'
type StatusFilter = 'all' | 'active' | 'inactive'
type EffectFilter = 'all' | 'allow' | 'deny'
type SortKey = 'resource' | 'name' | 'effect' | 'priority' | 'active' | 'source'
type SortDir = 'asc' | 'desc'

function formatJSON(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') {
    try {
      return JSON.stringify(JSON.parse(v))
    } catch {
      return v
    }
  }
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

function parseJSONString(raw: string, fallback: string): string {
  const s = raw.trim() || fallback
  JSON.parse(s)
  return s
}

function isSeedEntry(row: AclEntry): boolean {
  return String(row.authorId ?? '') === SEED_AUTHOR
}

function effectBadge(row: AclEntry): JSX.Element {
  return <Badge variant={String(row.effect) === 'deny' ? 'danger' : 'success'}>{String(row.effect ?? 'allow')}</Badge>
}

function sourceBadge(row: AclEntry): JSX.Element {
  return (
    <Badge variant={isSeedEntry(row) ? 'muted' : 'success'}>
      {isSeedEntry(row) ? t('permissions.source.seed') : t('permissions.source.override')}
    </Badge>
  )
}

function matchesSearch(row: AclEntry, query: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const hay = [
    row.name,
    row.resource,
    row.effect,
    row.actions,
    row.fields,
    row.domain,
    String(row.priority ?? ''),
  ]
    .map((v) => String(v ?? '').toLowerCase())
    .join(' ')
  return hay.includes(q)
}

function matchesResourcePrefix(row: AclEntry, prefix: string): boolean {
  const p = prefix.trim().toLowerCase()
  if (!p) return true
  return String(row.resource ?? '').toLowerCase().startsWith(p)
}

function matchesMinPriority(row: AclEntry, min: string): boolean {
  const raw = min.trim()
  if (!raw) return true
  const floor = parseInt(raw, 10)
  if (Number.isNaN(floor)) return true
  return Number(row.priority ?? 0) >= floor
}

function priorityCellStyle(priority: number, min: number, max: number): Record<string, string> {
  const span = Math.max(max - min, 1)
  const t = (priority - min) / span
  const lightness = 94 - t * 34
  const saturation = 35 + t * 45
  return {
    background: `hsl(275, ${saturation}%, ${lightness}%)`,
    color: t > 0.55 ? '#faf5ff' : 'var(--kg-text)',
  }
}

function sortIndicator(active: boolean, dir: SortDir): string {
  if (!active) return '↕'
  return dir === 'asc' ? '↑' : '↓'
}
async function loadResourceOptions(entryRows: AclEntry[]): Promise<ResourceOption[]> {
  const seen = new Set<string>()
  const options: ResourceOption[] = []

  function add(value: string) {
    const v = value.trim()
    if (!v || seen.has(v)) return
    seen.add(v)
    options.push({ value: v, label: v })
  }

  for (const row of entryRows) add(String(row.resource ?? ''))
  try {
    const resources = await fetchResources()
    for (const item of resources) add(item.resource)
  } catch {
    // entry-based options are enough when catalog lookup fails
  }

  options.sort((a, b) => a.value.localeCompare(b.value))
  return options
}

function serializeActions(actions: string[]): string {
  return JSON.stringify(actions.length ? actions : ['read'])
}

function RulesTable(props: {
  rows: AclEntry[]
  showRoleColumn?: boolean
  roleLabel?: (row: AclEntry) => string
  priorityMin: number
  priorityMax: number
  sortKey: SortKey
  sortDir: SortDir
  saving: boolean
  onSort: (key: SortKey) => void
  onDeactivate: (row: AclEntry) => void
  onActivate: (row: AclEntry) => void
  onDelete: (row: AclEntry) => void
}): JSX.Element {
  const header = (key: SortKey, label: string) => (
    <th class="px-4 py-3 text-left">
      <button
        type="button"
        class="inline-flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-sm font-semibold text-[var(--kg-text)] hover:text-[var(--kg-primary)]"
        aria-sort={props.sortKey === key ? (props.sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
        onClick={() => props.onSort(key)}
      >
        {label}
        <span class="text-xs opacity-60">{sortIndicator(props.sortKey === key, props.sortDir)}</span>
      </button>
    </th>
  )

  return (
    <div class="overflow-x-auto border-y border-[var(--kg-border)] bg-[var(--kg-surface)]">
      <table class="kg-table w-full border-collapse text-sm">
        <thead>
          <tr class="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            {header('name', t('permissions.col.name'))}
            <Show when={props.showRoleColumn}>
              <th class="px-4 py-3 text-left font-semibold text-[var(--kg-text)]">{t('permissions.col.role')}</th>
            </Show>
            {header('resource', t('permissions.col.resource'))}
            {header('effect', t('permissions.col.effect'))}
            {header('priority', t('permissions.col.priority'))}
            <th class="px-4 py-3 text-left font-semibold text-[var(--kg-text)]">{t('permissions.col.actions')}</th>
            <th class="px-4 py-3 text-left font-semibold text-[var(--kg-text)]">{t('permissions.col.fields')}</th>
            <th class="px-4 py-3 text-left font-semibold text-[var(--kg-text)]">{t('permissions.col.domain')}</th>
            {header('source', t('permissions.filter.source'))}
            {header('active', t('permissions.filter.status'))}
            <th class="kg-table-actions-col px-4 py-3 text-left font-semibold text-[var(--kg-text)]"> </th>
          </tr>
        </thead>
        <tbody>
          <For each={props.rows}>
            {(row) => (
              <tr
                class="border-b border-[var(--kg-border)]"
                classList={{
                  'bg-amber-50/80': !isSeedEntry(row),
                  'opacity-60': row.active === false,
                }}
              >
                <td class="px-4 py-3 font-medium text-[var(--kg-text)]">{String(row.name ?? '')}</td>
                <Show when={props.showRoleColumn}>
                  <td class="px-4 py-3 text-sm">{props.roleLabel?.(row) ?? ''}</td>
                </Show>
                <td class="px-4 py-3 font-mono text-xs">{String(row.resource ?? '')}</td>
                <td class="px-4 py-3">{effectBadge(row)}</td>
                <td
                  class="px-4 py-3 text-center font-mono text-xs font-semibold"
                  style={priorityCellStyle(Number(row.priority ?? 0), props.priorityMin, props.priorityMax)}
                >
                  {String(row.priority ?? 0)}
                </td>
                <td class="px-4 py-3 font-mono text-xs">{formatJSON(row.actions)}</td>
                <td class="px-4 py-3 font-mono text-xs">{formatJSON(row.fields)}</td>
                <td class="px-4 py-3 font-mono text-xs">{formatJSON(row.domain)}</td>
                <td class="px-4 py-3">{sourceBadge(row)}</td>
                <td class="px-4 py-3">
                  <Badge variant={row.active === false ? 'muted' : 'success'}>
                    {row.active === false ? t('permissions.filter.status_inactive') : t('permissions.filter.status_active')}
                  </Badge>
                </td>
                <td class="kg-table-actions px-4 py-3">
                  <div class="flex flex-wrap gap-[var(--kg-space-02)]">
                    <Show when={row.active !== false}>
                      <Button size="sm" variant="ghost" loading={props.saving} onClick={() => props.onDeactivate(row)}>
                        {t('permissions.deactivate')}
                      </Button>
                    </Show>
                    <Show when={row.active === false}>
                      <Button size="sm" variant="ghost" loading={props.saving} onClick={() => props.onActivate(row)}>
                        {t('permissions.activate')}
                      </Button>
                    </Show>
                    <Show when={!isSeedEntry(row)}>
                      <Button size="sm" variant="ghost" loading={props.saving} onClick={() => props.onDelete(row)}>
                        {t('permissions.delete')}
                      </Button>
                    </Show>
                  </div>
                </td>
              </tr>
            )}
          </For>
        </tbody>
      </table>
    </div>
  )
}

export default function Access(): JSX.Element {
  const [loading, setLoading] = createSignal(true)
  const [saving, setSaving] = createSignal(false)
  const [error, setError] = createSignal('')
  const [saved, setSaved] = createSignal(false)

  const [roles, setRoles] = createSignal<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = createSignal('')
  const [entries, setEntries] = createSignal<AclEntry[]>([])
  const [resourceOptions, setResourceOptions] = createSignal<ResourceOption[]>([])
  const [aclActions, setAclActions] = createSignal<string[]>([])

  const [searchQuery, setSearchQuery] = createSignal('')
  const [sourceFilter, setSourceFilter] = createSignal<SourceFilter>('all')
  const [statusFilter, setStatusFilter] = createSignal<StatusFilter>('active')
  const [effectFilter, setEffectFilter] = createSignal<EffectFilter>('all')
  const [resourcePrefix, setResourcePrefix] = createSignal('')
  const [minPriority, setMinPriority] = createSignal('')
  const [sortKey, setSortKey] = createSignal<SortKey>('resource')
  const [sortDir, setSortDir] = createSignal<SortDir>('asc')

  const [formName, setFormName] = createSignal('')
  const [formRoleId, setFormRoleId] = createSignal('')
  const [formEffect, setFormEffect] = createSignal('deny')
  const [formResource, setFormResource] = createSignal('')
  const [formActions, setFormActions] = createSignal<string[]>(['read'])
  const [formFieldsMode, setFormFieldsMode] = createSignal<'all' | 'custom'>('all')
  const [formFieldsCustom, setFormFieldsCustom] = createSignal('["message"]')
  const [formDomainMode, setFormDomainMode] = createSignal<'all' | 'custom'>('all')
  const [formDomainCustom, setFormDomainCustom] = createSignal('[["authorId","=","$user.id"]]')
  const [formPriority, setFormPriority] = createSignal('1000')

  const selectedRole = createMemo(() => roles().find((r) => r.id === selectedRoleId()) ?? null)

  const roleFilterOptions = createMemo(() => [
    { value: '', label: t('permissions.filter.role_all') },
    ...roles().map((r) => ({
      value: r.id,
      label: `${r.label || r.name} (${r.name})`,
    })),
  ])

  const roleOptions = createMemo(() =>
    roles().map((r) => ({
      value: r.id,
      label: `${r.label || r.name} (${r.name})`,
    })),
  )

  const rolesById = createMemo(() => {
    const map: Record<string, Role> = {}
    for (const role of roles()) {
      map[role.id] = role
    }
    return map
  })

  function selectRoleFilter(roleId: string) {
    setSelectedRoleId(roleId)
    if (roleId) setFormRoleId(roleId)
    setSaved(false)
    setError('')
  }

  const filteredRoleEntries = createMemo(() => {
    const rid = selectedRoleId()
    return entries()
      .filter((e) => !rid || String(e.roleId) === rid)
      .filter((e) => {
        if (sourceFilter() === 'seed') return isSeedEntry(e)
        if (sourceFilter() === 'override') return !isSeedEntry(e)
        return true
      })
      .filter((e) => {
        if (statusFilter() === 'active') return e.active !== false
        if (statusFilter() === 'inactive') return e.active === false
        return true
      })
      .filter((e) => {
        if (effectFilter() === 'all') return true
        return String(e.effect ?? '') === effectFilter()
      })
      .filter((e) => matchesResourcePrefix(e, resourcePrefix()))
      .filter((e) => matchesMinPriority(e, minPriority()))
      .filter((e) => matchesSearch(e, searchQuery()))
  })

  const sortedRoleEntries = createMemo(() => {
    const rows = filteredRoleEntries().slice()
    const key = sortKey()
    const dir = sortDir()
    const mul = dir === 'asc' ? 1 : -1
    rows.sort((a, b) => {
      switch (key) {
        case 'resource':
          return mul * String(a.resource ?? '').localeCompare(String(b.resource ?? ''))
        case 'name':
          return mul * String(a.name ?? '').localeCompare(String(b.name ?? ''))
        case 'effect':
          return mul * String(a.effect ?? '').localeCompare(String(b.effect ?? ''))
        case 'priority':
          return mul * (Number(a.priority ?? 0) - Number(b.priority ?? 0))
        case 'active':
          return mul * (Number(a.active !== false) - Number(b.active !== false))
        case 'source':
          return mul * (Number(isSeedEntry(a)) - Number(isSeedEntry(b)))
        default:
          return 0
      }
    })
    return rows
  })

  const priorityRange = createMemo(() => {
    let min = Infinity
    let max = -Infinity
    for (const row of sortedRoleEntries()) {
      const p = Number(row.priority ?? 0)
      min = Math.min(min, p)
      max = Math.max(max, p)
    }
    if (!Number.isFinite(min)) return { min: 0, max: 0 }
    return { min, max }
  })

  function toggleSort(key: SortKey) {
    if (sortKey() === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSortKey(key)
    setSortDir('asc')
  }

  onMount(() => {
    void loadAll()
    void fetchACLActions().then(setAclActions).catch(() => setAclActions(['read', 'create', 'update', 'delete', 'execute', '*']))
  })

  let loadingAll = false

  async function loadAll() {
    if (loadingAll) return
    loadingAll = true
    setLoading(true)
    setError('')
    try {
      const [roleRows, entryRows] = await Promise.all([
        listModelRecords('permissions', 'role', ['name', 'label', 'active']),
        listModelRecords('permissions', 'acl_entry', [
          'name',
          'roleId',
          'authorId',
          'effect',
          'resource',
          'actions',
          'fields',
          'domain',
          'priority',
          'active',
        ]),
      ])
      const nextRoles = (roleRows as Role[])
        .filter((r) => r.active !== false)
        .sort((a, b) => String(a.name).localeCompare(String(b.name)))
      setRoles(nextRoles)
      setEntries(entryRows as AclEntry[])
      void loadResourceOptions(entryRows as AclEntry[]).then(setResourceOptions)
      const current = selectedRoleId()
      if (current && !nextRoles.some((r) => r.id === current)) {
        setSelectedRoleId('')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
      loadingAll = false
    }
  }

  async function activateEntry(row: AclEntry) {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await updateModelRecord('permissions', 'acl_entry', row.id, aclCreateFields, { active: true })
      setSaved(true)
      await loadAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  async function deactivateEntry(row: AclEntry) {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await updateModelRecord('permissions', 'acl_entry', row.id, aclCreateFields, { active: false })
      setSaved(true)
      await loadAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  async function deleteEntry(row: AclEntry) {
    if (isSeedEntry(row)) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await deleteModelRecord('permissions', 'acl_entry', row.id)
      setSaved(true)
      await loadAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  function toggleAction(action: string, checked: boolean) {
    const current = new Set(formActions())
    if (action === '*') {
      setFormActions(checked ? ['*'] : ['read'])
      return
    }
    current.delete('*')
    if (checked) current.add(action)
    else current.delete(action)
    const next = [...current]
    setFormActions(next.length ? next : ['read'])
  }

  function resetForm() {
    setFormName('')
    setFormResource('')
    setFormActions(['read'])
    setFormFieldsMode('all')
    setFormFieldsCustom('["message"]')
    setFormDomainMode('all')
    setFormDomainCustom('[["authorId","=","$user.id"]]')
    setFormPriority('1000')
    setFormEffect('deny')
    setFormRoleId(selectedRoleId())
  }

  async function createOverride() {
    const rid = formRoleId().trim() || selectedRoleId()
    if (!rid) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const name = formName().trim()
      if (!name) throw new Error(t('permissions.error.name_required'))
      const resource = formResource().trim()
      if (!resource) throw new Error(t('permissions.error.resource_required'))
      const actions = serializeActions(formActions())
      const fields =
        formFieldsMode() === 'all' ? '"*"' : parseJSONString(formFieldsCustom(), '"*"')
      const domain = formDomainMode() === 'all' ? '[]' : parseJSONString(formDomainCustom(), '[]')
      const priority = parseInt(formPriority(), 10)
      if (Number.isNaN(priority)) throw new Error(t('permissions.error.priority_invalid'))

      await createModelRecord('permissions', 'acl_entry', aclCreateFields, {
        name,
        roleId: rid,
        effect: formEffect(),
        resource,
        actions,
        fields,
        domain,
        priority,
        active: true,
      })
      resetForm()
      setSaved(true)
      await loadAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <Show
        when={!loading()}
        fallback={
          <div class="flex items-center gap-[var(--kg-space-05)]">
            <Spinner />
            <p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.loading')}</p>
          </div>
        }
      >
        <Show when={error()}>
          <Alert variant="danger">{error()}</Alert>
        </Show>
        <Show when={saved()}>
          <Alert variant="success">{t('permissions.saved')}</Alert>
        </Show>

        <div class="flex min-w-0 flex-col gap-[var(--kg-space-07)]">
          <Show when={roles().length > 0} fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.roles_empty')}</p>}>
            <header class="flex flex-wrap items-end gap-[var(--kg-space-05)]">
              <FormField label={t('permissions.field.role')} class="mb-0 min-w-[16rem]">
                <Select value={selectedRoleId()} options={roleFilterOptions()} onChange={selectRoleFilter} />
              </FormField>
              <Show when={selectedRole()}>
                {(role) => (
                  <div class="flex flex-wrap items-center gap-[var(--kg-space-03)] pb-1">
                    <h2 class="m-0 text-xl">{role().label || role().name}</h2>
                    <Badge variant="muted">{String(role().name)}</Badge>
                  </div>
                )}
              </Show>
              <Button size="sm" variant="ghost" class="ms-auto" onClick={() => void loadAll()}>
                {t('permissions.refresh')}
              </Button>
            </header>

            <Card title={t('permissions.rules_title')}>
                    <p class="mb-[var(--kg-space-05)] mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.rules_hint')}</p>
                    <div class="mb-[var(--kg-space-05)] grid grid-cols-1 gap-[var(--kg-space-04)] md:grid-cols-2 lg:grid-cols-3">
                      <FormField label={t('permissions.filter.search')}>
                        <Input value={searchQuery()} onChange={setSearchQuery} placeholder={t('permissions.filter.search_placeholder')} />
                      </FormField>
                      <FormField label={t('permissions.filter.resource_prefix')}>
                        <Input value={resourcePrefix()} onChange={setResourcePrefix} placeholder="identity." />
                      </FormField>
                      <FormField label={t('permissions.filter.min_priority')}>
                        <Input type="number" value={minPriority()} onChange={setMinPriority} placeholder="1000" />
                      </FormField>
                      <FormField label={t('permissions.filter.source')}>
                        <Select
                          value={sourceFilter()}
                          options={[
                            { value: 'all', label: t('permissions.filter.source_all') },
                            { value: 'seed', label: t('permissions.source.seed') },
                            { value: 'override', label: t('permissions.source.override') },
                          ]}
                          onChange={(v) => setSourceFilter(v as SourceFilter)}
                        />
                      </FormField>
                      <FormField label={t('permissions.filter.effect')}>
                        <Select
                          value={effectFilter()}
                          options={[
                            { value: 'all', label: t('permissions.filter.effect_all') },
                            { value: 'allow', label: t('permissions.effect.allow') },
                            { value: 'deny', label: t('permissions.effect.deny') },
                          ]}
                          onChange={(v) => setEffectFilter(v as EffectFilter)}
                        />
                      </FormField>
                      <FormField label={t('permissions.filter.status')}>
                        <Select
                          value={statusFilter()}
                          options={[
                            { value: 'all', label: t('permissions.filter.status_all') },
                            { value: 'active', label: t('permissions.filter.status_active') },
                            { value: 'inactive', label: t('permissions.filter.status_inactive') },
                          ]}
                          onChange={(v) => setStatusFilter(v as StatusFilter)}
                        />
                      </FormField>
                    </div>

                    <p class="mb-[var(--kg-space-04)] mt-0 text-xs text-[var(--kg-text-muted)]">
                      {t('permissions.rules_count', sortedRoleEntries().length)}
                    </p>

                    <Show
                      when={sortedRoleEntries().length > 0}
                      fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.rules_empty')}</p>}
                    >
                      <RulesTable
                        rows={sortedRoleEntries()}
                        showRoleColumn={!selectedRoleId()}
                        roleLabel={(row) => {
                          const role = rolesById()[String(row.roleId ?? '')]
                          return role ? String(role.label || role.name) : String(row.roleId ?? '')
                        }}
                        priorityMin={priorityRange().min}
                        priorityMax={priorityRange().max}
                        sortKey={sortKey()}
                        sortDir={sortDir()}
                        saving={saving()}
                        onSort={toggleSort}
                        onDeactivate={(row) => void deactivateEntry(row)}
                        onActivate={(row) => void activateEntry(row)}
                        onDelete={(row) => void deleteEntry(row)}
                      />
                    </Show>
                  </Card>

                  <Card title={t('permissions.override_title')}>
                    <p class="mb-[var(--kg-space-05)] mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.override_hint')}</p>
                    <div class="grid grid-cols-1 gap-[var(--kg-space-05)] md:grid-cols-2">
                      <FormField label={t('permissions.field.role')} required>
                        <Select value={formRoleId()} options={roleOptions()} onChange={setFormRoleId} />
                      </FormField>
                      <FormField label={t('permissions.field.name')} required>
                        <Input value={formName()} onChange={setFormName} placeholder="role-resource-action" />
                      </FormField>
                      <FormField label={t('permissions.field.effect')} required>
                        <Select
                          value={formEffect()}
                          options={[
                            { value: 'allow', label: t('permissions.effect.allow') },
                            { value: 'deny', label: t('permissions.effect.deny') },
                          ]}
                          onChange={setFormEffect}
                        />
                      </FormField>
                      <FormField label={t('permissions.field.priority')} hint={t('permissions.field.priority_hint')} required>
                        <Select
                          value={formPriority()}
                          options={[
                            { value: '0', label: '0' },
                            { value: '100', label: '100' },
                            { value: '1000', label: '1000 (override)' },
                            { value: '2000', label: '2000 (strong override)' },
                          ]}
                          onChange={setFormPriority}
                        />
                      </FormField>
                      <FormField label={t('permissions.field.resource')} required class="md:col-span-2">
                        <SearchableCombobox
                          value={formResource()}
                          options={resourceOptions()}
                          placeholder="identity.menu.users"
                          onChange={setFormResource}
                        />
                      </FormField>
                      <FormField label={t('permissions.field.actions')} hint={t('permissions.field.actions_hint')} required class="md:col-span-2">
                        <div class="flex flex-wrap gap-[var(--kg-space-04)]">
                          <For each={aclActions()}>
                            {(action) => (
                              <Checkbox
                                label={action}
                                checked={formActions().includes(action)}
                                onChange={(checked) => toggleAction(action, checked)}
                              />
                            )}
                          </For>
                        </div>
                      </FormField>
                      <FormField label={t('permissions.field.fields')} hint={t('permissions.field.fields_hint')} required>
                        <Select
                          value={formFieldsMode()}
                          options={[
                            { value: 'all', label: t('permissions.fields.all') },
                            { value: 'custom', label: t('permissions.fields.custom') },
                          ]}
                          onChange={(v) => setFormFieldsMode(v as 'all' | 'custom')}
                        />
                      </FormField>
                      <Show when={formFieldsMode() === 'custom'}>
                        <FormField label={t('permissions.field.fields')} required>
                          <Input value={formFieldsCustom()} onChange={setFormFieldsCustom} placeholder={'["message"]'} />
                        </FormField>
                      </Show>
                      <FormField label={t('permissions.field.domain')} hint={t('permissions.field.domain_hint')} required>
                        <Select
                          value={formDomainMode()}
                          options={[
                            { value: 'all', label: t('permissions.domain.all') },
                            { value: 'custom', label: t('permissions.domain.custom') },
                          ]}
                          onChange={(v) => setFormDomainMode(v as 'all' | 'custom')}
                        />
                      </FormField>
                      <Show when={formDomainMode() === 'custom'}>
                        <FormField label={t('permissions.field.domain')} required>
                          <Input value={formDomainCustom()} onChange={setFormDomainCustom} placeholder={'[["authorId","=","$user.id"]]'} />
                        </FormField>
                      </Show>
                      <div class="md:col-span-2">
                        <FormActions>
                          <Button loading={saving()} onClick={() => void createOverride()}>
                            {t('permissions.override_submit')}
                          </Button>
                        </FormActions>
                      </div>
                    </div>
                  </Card>
          </Show>
        </div>
      </Show>

      <KAppStatus />
    </>
  )
}
