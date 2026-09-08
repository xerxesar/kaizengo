import { createEffect, createMemo, createSignal, For, onMount, Show, type JSX } from 'solid-js'
import {
  Alert,
  Badge,
  Button,
  Card,
  FormActions,
  FormField,
  Input,
  KAppStatus,
  Modal,
  SearchableCombobox,
  SearchableMultiSelect,
  Select,
  Spinner,
  StatCard,
  Toolbar,
  createModelRecord,
  deleteModelRecord,
  fetchResources,
  listModelRecords,
  t,
  updateModelRecord,
  type ModelField,
} from '@kaizengo/sdk-solid/ui'
import type { AclEntry, Role } from '../lib/types'
import { inferResourceKind, isCallStyleKind, kindBadgeClass } from '../lib/types'

const SEED_AUTHOR = '00000000-0000-0000-0000-000000000001'

const KIND_FILTERS = ['', 'query', 'command', 'view', 'app', 'api', 'mutation'] as const

const aclCreateFields: ModelField[] = [
  { key: 'name', type: 'string', required: true, label: 'name' },
  { key: 'roleId', type: 'many2one', required: true, label: 'roleId' },
  { key: 'effect', type: 'string', required: true, label: 'effect' },
  { key: 'kind', type: 'string', required: true, label: 'kind' },
  { key: 'resource', type: 'string', required: true, label: 'resource' },
  { key: 'actions', type: 'string', required: true, label: 'actions' },
  { key: 'fields', type: 'string', required: true, label: 'fields' },
  { key: 'domain', type: 'string', required: true, label: 'domain' },
  { key: 'priority', type: 'int', required: true, label: 'priority' },
  { key: 'active', type: 'bool', required: true, label: 'active' },
]

type ResourceOption = { value: string; label: string; kind?: string; fields?: string[] }
type SourceFilter = 'all' | 'seed' | 'override'
type StatusFilter = 'all' | 'active' | 'inactive'
type EffectFilter = 'all' | 'allow' | 'deny'
type SortKey = 'resource' | 'name' | 'kind' | 'effect' | 'priority' | 'active' | 'source'
type SortDir = 'asc' | 'desc'

function entryKind(row: AclEntry): string {
  return String(row.kind || inferResourceKind(String(row.resource ?? '')))
}

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

function suggestRuleName(roleSlug: string, effect: string, kind: string, resource: string): string {
  const role = (roleSlug || 'role').trim().toLowerCase() || 'role'
  const eff = (effect || 'allow').trim().toLowerCase() || 'allow'
  const k = (kind || 'resource').trim().toLowerCase() || 'resource'
  const short = (resource.trim().split('.').pop() || resource.trim() || 'item')
    .replace(/[^a-zA-Z0-9_-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .toLowerCase()
  return `${role}-${eff}-${k}-${short || 'item'}`
}

function sortIndicator(active: boolean, dir: SortDir): string {
  if (!active) return '↕'
  return dir === 'asc' ? '↑' : '↓'
}

function kindBadge(kind: string): JSX.Element {
  return (
    <span
      class={`inline-flex items-center rounded px-2 py-0.5 text-xs font-medium ${kindBadgeClass(kind)}`}
    >
      {kind || '—'}
    </span>
  )
}

function effectBadge(row: AclEntry): JSX.Element {
  return (
    <Badge variant={String(row.effect) === 'deny' ? 'danger' : 'success'}>
      {String(row.effect ?? 'allow')}
    </Badge>
  )
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
  const hay = [row.name, entryKind(row), row.resource, row.effect, row.actions, row.fields, row.domain, String(row.priority ?? '')]
    .map((v) => String(v ?? '').toLowerCase())
    .join(' ')
  return hay.includes(q)
}

async function loadResourceOptions(entryRows: AclEntry[]): Promise<ResourceOption[]> {
  const options: ResourceOption[] = []
  const byValue = new Map<string, ResourceOption>()

  function add(value: string, kind?: string, fields?: string[]) {
    const v = value.trim()
    if (!v) return
    const k = kind || inferResourceKind(v)
    // Models/menus/nav are not client ACL catalog entries.
    if (k === 'model' || k === 'menu' || k === 'nav') return
    const existing = byValue.get(v)
    if (existing) {
      if (kind) existing.kind = k
      // Prefer catalog field lists when an ACL entry was added first without them.
      if (fields?.length) existing.fields = fields
      return
    }
    const opt: ResourceOption = {
      value: v,
      label: v,
      kind: k,
      fields: fields?.length ? fields : undefined,
    }
    byValue.set(v, opt)
    options.push(opt)
  }

  for (const row of entryRows) add(String(row.resource ?? ''), entryKind(row))
  try {
    const resources = await fetchResources()
    for (const item of resources) add(item.resource, item.kind, item.fields)
  } catch {
    /* entry-based options are enough when catalog lookup fails */
  }

  options.sort((a, b) => {
    const ka = a.kind || ''
    const kb = b.kind || ''
    if (ka !== kb) return ka.localeCompare(kb)
    return a.value.localeCompare(b.value)
  })
  return options
}

function RulesTable(props: {
  rows: AclEntry[]
  showRoleColumn?: boolean
  roleLabel?: (row: AclEntry) => string
  sortKey: SortKey
  sortDir: SortDir
  saving: boolean
  onSort: (key: SortKey) => void
  onDeactivate: (row: AclEntry) => void
  onActivate: (row: AclEntry) => void
  onDelete: (row: AclEntry) => void
}): JSX.Element {
  const header = (key: SortKey, label: string) => (
    <th class="px-3 py-2.5 text-left">
      <button
        type="button"
        class="inline-flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)] hover:text-[var(--kg-primary)]"
        aria-sort={props.sortKey === key ? (props.sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
        onClick={() => props.onSort(key)}
      >
        {label}
        <span class="text-[0.65rem] opacity-60">{sortIndicator(props.sortKey === key, props.sortDir)}</span>
      </button>
    </th>
  )

  return (
    <div class="overflow-x-auto border border-[var(--kg-border)] bg-[var(--kg-surface)]">
      <table class="kg-table w-full border-collapse text-sm">
        <thead>
          <tr class="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            {header('name', t('permissions.col.name'))}
            <Show when={props.showRoleColumn}>
              <th class="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
                {t('permissions.col.role')}
              </th>
            </Show>
            {header('kind', t('permissions.col.kind'))}
            {header('resource', t('permissions.col.resource'))}
            {header('effect', t('permissions.col.effect'))}
            {header('priority', t('permissions.col.priority'))}
            <th class="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
              {t('permissions.col.scope')}
            </th>
            {header('source', t('permissions.filter.source'))}
            {header('active', t('permissions.filter.status'))}
            <th class="kg-table-actions-col px-3 py-2.5 text-left"> </th>
          </tr>
        </thead>
        <tbody>
          <For each={props.rows}>
            {(row) => {
              const kind = () => entryKind(row)
              const call = () => isCallStyleKind(kind())
              return (
                <tr
                  class="border-b border-[var(--kg-border)] align-top"
                  classList={{
                    'bg-[var(--kg-surface-muted)]/40': !isSeedEntry(row),
                    'opacity-55': row.active === false,
                  }}
                >
                  <td class="px-3 py-2.5 font-medium text-[var(--kg-text)]">{String(row.name ?? '')}</td>
                  <Show when={props.showRoleColumn}>
                    <td class="px-3 py-2.5 text-sm text-[var(--kg-text-secondary)]">{props.roleLabel?.(row) ?? ''}</td>
                  </Show>
                  <td class="px-3 py-2.5">{kindBadge(kind())}</td>
                  <td class="max-w-[18rem] px-3 py-2.5 font-mono text-xs break-all text-[var(--kg-text)]">
                    {String(row.resource ?? '')}
                  </td>
                  <td class="px-3 py-2.5">{effectBadge(row)}</td>
                  <td class="px-3 py-2.5 text-center font-mono text-xs tabular-nums">{String(row.priority ?? 0)}</td>
                  <td class="px-3 py-2.5 font-mono text-xs text-[var(--kg-text-secondary)]">
                    <Show when={call()} fallback={
                      <div class="flex flex-col gap-0.5">
                        <span>{t('permissions.scope.actions')}: {formatJSON(row.actions)}</span>
                        <span>{t('permissions.scope.fields')}: {formatJSON(row.fields)}</span>
                        <span>{t('permissions.scope.domain')}: {formatJSON(row.domain)}</span>
                      </div>
                    }>
                      <span class="text-[var(--kg-text-muted)]">{t('permissions.scope.call')}</span>
                    </Show>
                  </td>
                  <td class="px-3 py-2.5">{sourceBadge(row)}</td>
                  <td class="px-3 py-2.5">
                    <Badge variant={row.active === false ? 'muted' : 'success'}>
                      {row.active === false ? t('permissions.filter.status_inactive') : t('permissions.filter.status_active')}
                    </Badge>
                  </td>
                  <td class="kg-table-actions px-3 py-2.5">
                    <div class="flex flex-wrap gap-1">
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
              )
            }}
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
  const [modalOpen, setModalOpen] = createSignal(false)

  const [roles, setRoles] = createSignal<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = createSignal('')
  const [entries, setEntries] = createSignal<AclEntry[]>([])
  const [resourceOptions, setResourceOptions] = createSignal<ResourceOption[]>([])

  const [searchQuery, setSearchQuery] = createSignal('')
  const [kindFilter, setKindFilter] = createSignal('')
  const [sourceFilter, setSourceFilter] = createSignal<SourceFilter>('all')
  const [statusFilter, setStatusFilter] = createSignal<StatusFilter>('active')
  const [effectFilter, setEffectFilter] = createSignal<EffectFilter>('all')
  const [resourcePrefix, setResourcePrefix] = createSignal('')
  const [minPriority, setMinPriority] = createSignal('')
  const [sortKey, setSortKey] = createSignal<SortKey>('priority')
  const [sortDir, setSortDir] = createSignal<SortDir>('desc')

  const [formName, setFormName] = createSignal('')
  const [formNameTouched, setFormNameTouched] = createSignal(false)
  const [formRoleId, setFormRoleId] = createSignal('')
  const [formEffect, setFormEffect] = createSignal('deny')
  const [formResource, setFormResource] = createSignal('')
  const [formPriority, setFormPriority] = createSignal('1000')
  const [formFieldsMode, setFormFieldsMode] = createSignal<'all' | 'selected'>('all')
  const [formSelectedFields, setFormSelectedFields] = createSignal<string[]>([])
  const [formDomainMode, setFormDomainMode] = createSignal<'all' | 'custom'>('all')
  const [formDomainCustom, setFormDomainCustom] = createSignal('[["authorId","=","$user.id"]]')

  const selectedRole = createMemo(() => roles().find((r) => r.id === selectedRoleId()) ?? null)
  const formRole = createMemo(() => roles().find((r) => r.id === formRoleId()) ?? null)
  const formKind = createMemo(() => inferResourceKind(formResource()))
  const formIsCallStyle = createMemo(() => isCallStyleKind(formKind()))
  const formSupportsFieldsDomain = createMemo(() => formKind() === 'query' || formKind() === 'command')
  const formResourceMeta = createMemo(
    () => resourceOptions().find((o) => o.value === formResource().trim()) ?? null,
  )
  const formAvailableFields = createMemo(() => formResourceMeta()?.fields ?? [])

  const suggestedName = createMemo(() =>
    suggestRuleName(String(formRole()?.name ?? ''), formEffect(), formKind(), formResource()),
  )

  createEffect(() => {
    if (formNameTouched()) return
    const next = suggestedName()
    if (next) setFormName(next)
  })

  const rolesById = createMemo(() => {
    const map: Record<string, Role> = {}
    for (const role of roles()) map[role.id] = role
    return map
  })

  const roleRuleCounts = createMemo(() => {
    const counts = new Map<string, number>()
    for (const e of entries()) {
      if (e.active === false) continue
      const rid = String(e.roleId ?? '')
      if (!rid) continue
      counts.set(rid, (counts.get(rid) ?? 0) + 1)
    }
    return counts
  })

  const summary = createMemo(() => {
    const rows = entries().filter((e) => !selectedRoleId() || String(e.roleId) === selectedRoleId())
    const active = rows.filter((e) => e.active !== false)
    return {
      total: rows.length,
      active: active.length,
      overrides: active.filter((e) => !isSeedEntry(e)).length,
      denies: active.filter((e) => String(e.effect) === 'deny').length,
      call: active.filter((e) => isCallStyleKind(entryKind(e))).length,
      app: active.filter((e) => {
        const k = entryKind(e)
        return k === 'app' || k === 'api'
      }).length,
    }
  })

  const filteredEntries = createMemo(() => {
    const rid = selectedRoleId()
    const kind = kindFilter()
    const prefix = resourcePrefix().trim().toLowerCase()
    const floorRaw = minPriority().trim()
    const floor = floorRaw ? parseInt(floorRaw, 10) : NaN

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
      .filter((e) => effectFilter() === 'all' || String(e.effect ?? '') === effectFilter())
      .filter((e) => !kind || entryKind(e) === kind)
      .filter((e) => !prefix || String(e.resource ?? '').toLowerCase().startsWith(prefix))
      .filter((e) => Number.isNaN(floor) || Number(e.priority ?? 0) >= floor)
      .filter((e) => matchesSearch(e, searchQuery()))
  })

  const sortedEntries = createMemo(() => {
    const rows = filteredEntries().slice()
    const key = sortKey()
    const mul = sortDir() === 'asc' ? 1 : -1
    rows.sort((a, b) => {
      switch (key) {
        case 'resource':
          return mul * String(a.resource ?? '').localeCompare(String(b.resource ?? ''))
        case 'name':
          return mul * String(a.name ?? '').localeCompare(String(b.name ?? ''))
        case 'kind':
          return mul * entryKind(a).localeCompare(entryKind(b))
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

  const hasActiveFilters = createMemo(
    () =>
      !!searchQuery().trim() ||
      !!kindFilter() ||
      sourceFilter() !== 'all' ||
      statusFilter() !== 'active' ||
      effectFilter() !== 'all' ||
      !!resourcePrefix().trim() ||
      !!minPriority().trim(),
  )

  const roleOptions = createMemo(() =>
    roles().map((r) => ({
      value: r.id,
      label: `${r.label || r.name} (${r.name})`,
    })),
  )

  function selectRole(roleId: string) {
    setSelectedRoleId(roleId)
    if (roleId) setFormRoleId(roleId)
    setSaved(false)
    setError('')
  }

  function clearFilters() {
    setSearchQuery('')
    setKindFilter('')
    setSourceFilter('all')
    setStatusFilter('active')
    setEffectFilter('all')
    setResourcePrefix('')
    setMinPriority('')
  }

  function toggleSort(key: SortKey) {
    if (sortKey() === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSortKey(key)
    setSortDir(key === 'priority' ? 'desc' : 'asc')
  }

  function openCreateModal() {
    resetForm()
    setError('')
    setSaved(false)
    setModalOpen(true)
  }

  function resetForm() {
    setFormNameTouched(false)
    setFormResource('')
    setFormPriority('1000')
    setFormEffect('deny')
    setFormFieldsMode('all')
    setFormSelectedFields([])
    setFormDomainMode('all')
    setFormDomainCustom('[["authorId","=","$user.id"]]')
    setFormRoleId(selectedRoleId() || roles()[0]?.id || '')
    setFormName(
      suggestRuleName(
        String((roles().find((r) => r.id === (selectedRoleId() || roles()[0]?.id)) ?? {}).name ?? ''),
        'deny',
        '',
        '',
      ),
    )
  }

  function selectFormResource(v: string) {
    setFormResource(v)
    setFormNameTouched(false)
    const meta = resourceOptions().find((o) => o.value === v.trim())
    const fields = meta?.fields ?? []
    setFormFieldsMode('all')
    setFormSelectedFields(fields.slice())
    setFormDomainMode('all')
  }

  onMount(() => {
    void loadAll()
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
          'kind',
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
      if (current && !nextRoles.some((r) => r.id === current)) setSelectedRoleId('')
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
    if (!window.confirm(t('permissions.confirm_delete'))) return
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

  async function createOverride() {
    const rid = formRoleId().trim() || selectedRoleId()
    if (!rid) {
      setError(t('permissions.error.role_required'))
      return
    }
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const name = formName().trim()
      if (!name) throw new Error(t('permissions.error.name_required'))
      const resource = formResource().trim()
      if (!resource) throw new Error(t('permissions.error.resource_required'))
      const kind = inferResourceKind(resource)
      const actions = isCallStyleKind(kind) ? '[]' : '["*"]'
      let fields = '"*"'
      let domain = '[]'
      if (kind === 'query' || kind === 'command') {
        if (formFieldsMode() === 'selected') {
          const selected = formSelectedFields()
          if (selected.length === 0) throw new Error(t('permissions.error.fields_required'))
          fields = JSON.stringify(selected)
        }
        if (formDomainMode() === 'custom') {
          domain = parseJSONString(formDomainCustom(), '[]')
        }
      }
      const priority = parseInt(formPriority(), 10)
      if (Number.isNaN(priority)) throw new Error(t('permissions.error.priority_invalid'))

      await createModelRecord('permissions', 'acl_entry', aclCreateFields, {
        name,
        roleId: rid,
        effect: formEffect(),
        kind,
        resource,
        actions,
        fields,
        domain,
        priority,
        active: true,
      })
      resetForm()
      setModalOpen(false)
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
        <div class="flex min-w-0 flex-col gap-[var(--kg-space-06)]">
          <Toolbar
            start={
              <div class="flex min-w-0 flex-col gap-1">
                <h1 class="m-0 text-2xl font-light tracking-tight text-[var(--kg-text)]">{t('permissions.access_title')}</h1>
                <p class="m-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.access_subtitle')}</p>
              </div>
            }
            end={
              <>
                <Button size="sm" variant="ghost" onClick={() => void loadAll()}>
                  {t('permissions.refresh')}
                </Button>
                <Button size="sm" onClick={openCreateModal} disabled={roles().length === 0}>
                  {t('permissions.override_new')}
                </Button>
              </>
            }
          />

          <Show when={error()}>
            <Alert variant="danger" dismissible onDismiss={() => setError('')}>
              {error()}
            </Alert>
          </Show>
          <Show when={saved()}>
            <Alert variant="success" dismissible onDismiss={() => setSaved(false)}>
              {t('permissions.saved')}
            </Alert>
          </Show>

          <Show when={roles().length > 0} fallback={<Alert variant="warning">{t('permissions.roles_empty')}</Alert>}>
            <div class="grid grid-cols-[repeat(auto-fit,minmax(9.5rem,1fr))] gap-3">
              <StatCard label={t('permissions.stat.total')} value={summary().total} hint={t('permissions.stat.total_hint')} />
              <StatCard label={t('permissions.stat.active')} value={summary().active} />
              <StatCard label={t('permissions.stat.overrides')} value={summary().overrides} hint={t('permissions.stat.overrides_hint')} />
              <StatCard label={t('permissions.stat.denies')} value={summary().denies} />
              <StatCard label={t('permissions.stat.call')} value={summary().call} hint={t('permissions.stat.call_hint')} />
              <StatCard label={t('permissions.stat.app')} value={summary().app} hint={t('permissions.stat.app_hint')} />
            </div>

            <div class="grid min-w-0 grid-cols-1 gap-[var(--kg-space-05)] xl:grid-cols-[16rem_minmax(0,1fr)]">
              <aside class="border border-[var(--kg-border)] bg-[var(--kg-surface)]">
                <div class="border-b border-[var(--kg-border)] px-4 py-3">
                  <h2 class="m-0 text-sm font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
                    {t('permissions.roles_title')}
                  </h2>
                </div>
                <nav class="flex max-h-[32rem] flex-col overflow-y-auto" aria-label={t('permissions.roles_title')}>
                  <button
                    type="button"
                    class="flex cursor-pointer items-center justify-between gap-2 border-0 border-b border-[var(--kg-border)] bg-transparent px-4 py-3 text-left text-sm hover:bg-[var(--kg-field-hover)]"
                    classList={{
                      'bg-[var(--kg-field-hover)] font-medium text-[var(--kg-text)]': !selectedRoleId(),
                      'text-[var(--kg-text-secondary)]': !!selectedRoleId(),
                    }}
                    onClick={() => selectRole('')}
                  >
                    <span>{t('permissions.filter.role_all')}</span>
                    <Badge variant="muted">{String(entries().filter((e) => e.active !== false).length)}</Badge>
                  </button>
                  <For each={roles()}>
                    {(role) => (
                      <button
                        type="button"
                        class="flex cursor-pointer items-center justify-between gap-2 border-0 border-b border-[var(--kg-border)] bg-transparent px-4 py-3 text-left text-sm hover:bg-[var(--kg-field-hover)]"
                        classList={{
                          'bg-[var(--kg-field-hover)] font-medium text-[var(--kg-text)]': selectedRoleId() === role.id,
                          'text-[var(--kg-text-secondary)]': selectedRoleId() !== role.id,
                        }}
                        onClick={() => selectRole(role.id)}
                      >
                        <span class="min-w-0 truncate">
                          <span class="block truncate">{role.label || role.name}</span>
                          <span class="block font-mono text-xs text-[var(--kg-text-muted)]">{String(role.name)}</span>
                        </span>
                        <Badge variant="muted">{String(roleRuleCounts().get(role.id) ?? 0)}</Badge>
                      </button>
                    )}
                  </For>
                </nav>
              </aside>

              <div class="flex min-w-0 flex-col gap-[var(--kg-space-05)]">
                <Card title={t('permissions.rules_title')}>
                  <Show when={selectedRole()}>
                    {(role) => (
                      <p class="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">
                        {t('permissions.rules_for_role')}{' '}
                        <span class="font-medium text-[var(--kg-text)]">{role().label || role().name}</span>
                        <Badge variant="muted" class="ms-2">
                          {String(role().name)}
                        </Badge>
                      </p>
                    )}
                  </Show>
                  <p class="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.rules_hint')}</p>

                  <div class="mb-4 flex flex-wrap gap-2">
                    <For each={[...KIND_FILTERS]}>
                      {(kind) => (
                        <button
                          type="button"
                          class="cursor-pointer rounded-sm border px-2.5 py-1 text-xs font-medium"
                          classList={{
                            'border-[var(--kg-text)] bg-[var(--kg-text)] text-[var(--kg-surface)]': kindFilter() === kind,
                            'border-[var(--kg-border)] bg-transparent text-[var(--kg-text-secondary)] hover:border-[var(--kg-text-muted)]':
                              kindFilter() !== kind,
                          }}
                          onClick={() => setKindFilter(kind)}
                        >
                          {kind ? kind : t('permissions.filter.kind_all')}
                        </button>
                      )}
                    </For>
                  </div>

                  <div class="mb-4 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                    <FormField label={t('permissions.filter.search')} class="mb-0">
                      <Input value={searchQuery()} onChange={setSearchQuery} placeholder={t('permissions.filter.search_placeholder')} />
                    </FormField>
                    <FormField label={t('permissions.filter.resource_prefix')} class="mb-0">
                      <Input value={resourcePrefix()} onChange={setResourcePrefix} placeholder="hellospec." />
                    </FormField>
                    <FormField label={t('permissions.filter.min_priority')} class="mb-0">
                      <Input value={minPriority()} onChange={setMinPriority} placeholder="0" />
                    </FormField>
                    <FormField label={t('permissions.filter.source')} class="mb-0">
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
                    <FormField label={t('permissions.filter.status')} class="mb-0">
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
                    <FormField label={t('permissions.filter.effect')} class="mb-0">
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
                  </div>

                  <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <p class="m-0 text-sm text-[var(--kg-text-muted)]">
                      {t('permissions.rules_count').replace('%d', String(sortedEntries().length))}
                    </p>
                    <Show when={hasActiveFilters()}>
                      <Button size="sm" variant="ghost" onClick={clearFilters}>
                        {t('permissions.filter.clear')}
                      </Button>
                    </Show>
                  </div>

                  <Show
                    when={sortedEntries().length > 0}
                    fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.rules_empty')}</p>}
                  >
                    <RulesTable
                      rows={sortedEntries()}
                      showRoleColumn={!selectedRoleId()}
                      roleLabel={(row) => {
                        const role = rolesById()[String(row.roleId ?? '')]
                        return role ? String(role.label || role.name) : String(row.roleId ?? '')
                      }}
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
              </div>
            </div>
          </Show>
        </div>
      </Show>

      <Modal
        open={modalOpen()}
        title={t('permissions.override_title')}
        size="lg"
        onClose={() => setModalOpen(false)}
        footer={
          <FormActions>
            <Button variant="ghost" onClick={() => setModalOpen(false)}>
              {t('permissions.cancel')}
            </Button>
            <Button loading={saving()} onClick={() => void createOverride()}>
              {t('permissions.override_submit')}
            </Button>
          </FormActions>
        }
      >
        <p class="mb-5 mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.override_hint')}</p>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField label={t('permissions.field.role')} required>
            <Select
              value={formRoleId()}
              options={roleOptions()}
              onChange={(v) => {
                setFormRoleId(v)
                setFormNameTouched(false)
              }}
            />
          </FormField>
          <FormField label={t('permissions.field.effect')} required>
            <Select
              value={formEffect()}
              options={[
                { value: 'allow', label: t('permissions.effect.allow') },
                { value: 'deny', label: t('permissions.effect.deny') },
              ]}
              onChange={(v) => {
                setFormEffect(v)
                setFormNameTouched(false)
              }}
            />
          </FormField>
          <FormField label={t('permissions.field.resource')} required class="md:col-span-2">
            <SearchableCombobox
              value={formResource()}
              options={resourceOptions()}
              placeholder="hellospec.command.hellospecPostGreeting"
              onChange={selectFormResource}
            />
          </FormField>
          <Show when={formResource().trim()}>
            <FormField label={t('permissions.field.kind')} hint={t('permissions.field.kind_hint')} class="md:col-span-2">
              <div class="flex items-center gap-2">
                {kindBadge(formKind())}
                <Show when={formIsCallStyle() && !formSupportsFieldsDomain()}>
                  <span class="text-sm text-[var(--kg-text-muted)]">{t('permissions.field.call_style_hint')}</span>
                </Show>
              </div>
            </FormField>
          </Show>
          <Show when={formSupportsFieldsDomain()}>
            <FormField label={t('permissions.field.fields')} hint={t('permissions.field.fields_hint')} class="md:col-span-2">
              <Select
                value={formFieldsMode()}
                options={[
                  { value: 'all', label: t('permissions.fields.all') },
                  { value: 'selected', label: t('permissions.fields.selected') },
                ]}
                onChange={(v) => {
                  const mode = v as 'all' | 'selected'
                  setFormFieldsMode(mode)
                  if (mode === 'all') setFormSelectedFields(formAvailableFields().slice())
                }}
              />
            </FormField>
            <Show when={formFieldsMode() === 'selected'}>
              <FormField label={t('permissions.field.fields_pick')} hint={t('permissions.field.fields_pick_hint')} class="md:col-span-2">
                <SearchableMultiSelect
                  value={formSelectedFields()}
                  options={formAvailableFields().map((field) => ({ value: field, label: field }))}
                  placeholder={t('permissions.field.fields_pick_placeholder')}
                  allowCustomValue={formAvailableFields().length === 0}
                  emptyMessage={t('permissions.field.fields_pick_empty')}
                  onChange={(next) => {
                    setFormSelectedFields(next)
                    setFormFieldsMode('selected')
                  }}
                />
              </FormField>
            </Show>
            <FormField label={t('permissions.field.domain')} hint={t('permissions.field.domain_hint')} class="md:col-span-2">
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
              <FormField label={t('permissions.field.domain')} class="md:col-span-2">
                <Input value={formDomainCustom()} onChange={setFormDomainCustom} placeholder='[["authorId","=","$user.id"]]' />
              </FormField>
            </Show>
          </Show>
          <FormField label={t('permissions.field.priority')} hint={t('permissions.field.priority_hint')} required class="md:col-span-2">
            <Input value={formPriority()} onChange={setFormPriority} placeholder="1000" />
          </FormField>
          <FormField label={t('permissions.field.name')} hint={t('permissions.field.name_hint')} required class="md:col-span-2">
            <Input
              value={formName()}
              onChange={(v) => {
                setFormNameTouched(true)
                setFormName(v)
              }}
              placeholder={suggestedName() || 'member-deny-view-Users'}
            />
          </FormField>
        </div>
      </Modal>

      <KAppStatus />
    </>
  )
}
