import { useEffect, useMemo, useState, type ReactNode } from 'react'
import {
  Alert,
  Badge,
  Button,
  Input,
  Label,
  Progress,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Card,
  FormActions,
  SearchableCombobox,
  SearchableMultiSelect,
  StatCard,
  Toolbar,
  cn,
  createModelRecord,
  deleteModelRecord,
  fetchResources,
  listModelRecords,
  t,
  updateModelRecord,
  type ModelField,
} from '@/lib'
import { KAppStatus } from '@/k'
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
type Option = { value: string; label: string }

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

function kindBadge(kind: string): ReactNode {
  return (
    <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-medium ${kindBadgeClass(kind)}`}>
      {kind || '—'}
    </span>
  )
}

function OptionSelect({
  value,
  options,
  onChange,
  'aria-label': ariaLabel,
}: {
  value: string
  options: Option[]
  onChange: (value: string) => void
  'aria-label'?: string
}) {
  return (
    <Select value={value || undefined} onValueChange={onChange}>
      <SelectTrigger aria-label={ariaLabel}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

async function loadResourceOptions(entryRows: AclEntry[]): Promise<ResourceOption[]> {
  const options: ResourceOption[] = []
  const byValue = new Map<string, ResourceOption>()

  function add(value: string, kind?: string, fields?: string[]) {
    const v = value.trim()
    if (!v) return
    const k = kind || inferResourceKind(v)
    if (k === 'model' || k === 'menu' || k === 'nav') return
    const existing = byValue.get(v)
    if (existing) {
      if (kind) existing.kind = k
      if (fields?.length) existing.fields = fields
      return
    }
    const opt: ResourceOption = { value: v, label: v, kind: k, fields: fields?.length ? fields : undefined }
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
}) {
  const header = (key: SortKey, label: string) => (
    <th className="px-3 py-2.5 text-left">
      <button
        type="button"
        className="inline-flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)] hover:text-[var(--kg-primary)]"
        aria-sort={props.sortKey === key ? (props.sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
        onClick={() => props.onSort(key)}
      >
        {label}
        <span className="text-[0.65rem] opacity-60">{sortIndicator(props.sortKey === key, props.sortDir)}</span>
      </button>
    </th>
  )

  return (
    <div className="overflow-x-auto border border-[var(--kg-border)] bg-[var(--kg-surface)]">
      <table className="kg-table w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            {header('name', t('permissions.col.name'))}
            {props.showRoleColumn ? (
              <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
                {t('permissions.col.role')}
              </th>
            ) : null}
            {header('kind', t('permissions.col.kind'))}
            {header('resource', t('permissions.col.resource'))}
            {header('effect', t('permissions.col.effect'))}
            {header('priority', t('permissions.col.priority'))}
            <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
              {t('permissions.col.scope')}
            </th>
            {header('source', t('permissions.filter.source'))}
            {header('active', t('permissions.filter.status'))}
            <th className="kg-table-actions-col px-3 py-2.5 text-left"> </th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row) => {
            const kind = entryKind(row)
            const call = isCallStyleKind(kind)
            return (
              <tr
                key={row.id}
                className={cn(
                  'border-b border-[var(--kg-border)] align-top',
                  !isSeedEntry(row) && 'bg-[var(--kg-surface-muted)]/40',
                  row.active === false && 'opacity-55',
                )}
              >
                <td className="px-3 py-2.5 font-medium text-[var(--kg-text)]">{String(row.name ?? '')}</td>
                {props.showRoleColumn ? (
                  <td className="px-3 py-2.5 text-sm text-[var(--kg-text-secondary)]">
                    {props.roleLabel?.(row) ?? ''}
                  </td>
                ) : null}
                <td className="px-3 py-2.5">{kindBadge(kind)}</td>
                <td className="max-w-[18rem] break-all px-3 py-2.5 font-mono text-xs text-[var(--kg-text)]">
                  {String(row.resource ?? '')}
                </td>
                <td className="px-3 py-2.5">
                  <Badge variant={String(row.effect) === 'deny' ? 'danger' : 'success'}>
                    {String(row.effect ?? 'allow')}
                  </Badge>
                </td>
                <td className="px-3 py-2.5 text-center font-mono text-xs tabular-nums">
                  {String(row.priority ?? 0)}
                </td>
                <td className="px-3 py-2.5 font-mono text-xs text-[var(--kg-text-secondary)]">
                  {call ? (
                    <span className="text-[var(--kg-text-muted)]">{t('permissions.scope.call')}</span>
                  ) : (
                    <div className="flex flex-col gap-0.5">
                      <span>
                        {t('permissions.scope.actions')}: {formatJSON(row.actions)}
                      </span>
                      <span>
                        {t('permissions.scope.fields')}: {formatJSON(row.fields)}
                      </span>
                      <span>
                        {t('permissions.scope.domain')}: {formatJSON(row.domain)}
                      </span>
                    </div>
                  )}
                </td>
                <td className="px-3 py-2.5">
                  <Badge variant={isSeedEntry(row) ? 'muted' : 'success'}>
                    {isSeedEntry(row) ? t('permissions.source.seed') : t('permissions.source.override')}
                  </Badge>
                </td>
                <td className="px-3 py-2.5">
                  <Badge variant={row.active === false ? 'muted' : 'success'}>
                    {row.active === false
                      ? t('permissions.filter.status_inactive')
                      : t('permissions.filter.status_active')}
                  </Badge>
                </td>
                <td className="kg-table-actions px-3 py-2.5">
                  <div className="flex flex-wrap gap-1">
                    {row.active !== false ? (
                      <Button size="sm" variant="ghost" disabled={props.saving} onClick={() => props.onDeactivate(row)}>
                        {t('permissions.deactivate')}
                      </Button>
                    ) : (
                      <Button size="sm" variant="ghost" disabled={props.saving} onClick={() => props.onActivate(row)}>
                        {t('permissions.activate')}
                      </Button>
                    )}
                    {!isSeedEntry(row) ? (
                      <Button size="sm" variant="ghost" disabled={props.saving} onClick={() => props.onDelete(row)}>
                        {t('permissions.delete')}
                      </Button>
                    ) : null}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export default function Access() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)

  const [roles, setRoles] = useState<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = useState('')
  const [entries, setEntries] = useState<AclEntry[]>([])
  const [resourceOptions, setResourceOptions] = useState<ResourceOption[]>([])

  const [searchQuery, setSearchQuery] = useState('')
  const [kindFilter, setKindFilter] = useState('')
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('all')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('active')
  const [effectFilter, setEffectFilter] = useState<EffectFilter>('all')
  const [resourcePrefix, setResourcePrefix] = useState('')
  const [minPriority, setMinPriority] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('priority')
  const [sortDir, setSortDir] = useState<SortDir>('desc')

  const [formName, setFormName] = useState('')
  const [formNameTouched, setFormNameTouched] = useState(false)
  const [formRoleId, setFormRoleId] = useState('')
  const [formEffect, setFormEffect] = useState('deny')
  const [formResource, setFormResource] = useState('')
  const [formPriority, setFormPriority] = useState('1000')
  const [formFieldsMode, setFormFieldsMode] = useState<'all' | 'selected'>('all')
  const [formSelectedFields, setFormSelectedFields] = useState<string[]>([])
  const [formDomainMode, setFormDomainMode] = useState<'all' | 'custom'>('all')
  const [formDomainCustom, setFormDomainCustom] = useState('[["authorId","=","$user.id"]]')

  const selectedRole = roles.find((r) => r.id === selectedRoleId) ?? null
  const formRole = roles.find((r) => r.id === formRoleId) ?? null
  const formKind = inferResourceKind(formResource)
  const formIsCallStyle = isCallStyleKind(formKind)
  const formSupportsFieldsDomain = formKind === 'query' || formKind === 'command'
  const formResourceMeta = resourceOptions.find((o) => o.value === formResource.trim()) ?? null
  const formAvailableFields = formResourceMeta?.fields ?? []

  const suggestedName = useMemo(
    () => suggestRuleName(String(formRole?.name ?? ''), formEffect, formKind, formResource),
    [formRole?.name, formEffect, formKind, formResource],
  )

  useEffect(() => {
    if (formNameTouched) return
    if (suggestedName) setFormName(suggestedName)
  }, [formNameTouched, suggestedName])

  const rolesById = useMemo(() => {
    const map: Record<string, Role> = {}
    for (const role of roles) map[role.id] = role
    return map
  }, [roles])

  const roleRuleCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const e of entries) {
      if (e.active === false) continue
      const rid = String(e.roleId ?? '')
      if (!rid) continue
      counts.set(rid, (counts.get(rid) ?? 0) + 1)
    }
    return counts
  }, [entries])

  const summary = useMemo(() => {
    const rows = entries.filter((e) => !selectedRoleId || String(e.roleId) === selectedRoleId)
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
  }, [entries, selectedRoleId])

  const filteredEntries = useMemo(() => {
    const prefix = resourcePrefix.trim().toLowerCase()
    const floorRaw = minPriority.trim()
    const floor = floorRaw ? parseInt(floorRaw, 10) : NaN
    return entries
      .filter((e) => !selectedRoleId || String(e.roleId) === selectedRoleId)
      .filter((e) => {
        if (sourceFilter === 'seed') return isSeedEntry(e)
        if (sourceFilter === 'override') return !isSeedEntry(e)
        return true
      })
      .filter((e) => {
        if (statusFilter === 'active') return e.active !== false
        if (statusFilter === 'inactive') return e.active === false
        return true
      })
      .filter((e) => effectFilter === 'all' || String(e.effect ?? '') === effectFilter)
      .filter((e) => !kindFilter || entryKind(e) === kindFilter)
      .filter((e) => !prefix || String(e.resource ?? '').toLowerCase().startsWith(prefix))
      .filter((e) => Number.isNaN(floor) || Number(e.priority ?? 0) >= floor)
      .filter((e) => {
        const q = searchQuery.trim().toLowerCase()
        if (!q) return true
        const hay = [e.name, entryKind(e), e.resource, e.effect, e.actions, e.fields, e.domain, String(e.priority ?? '')]
          .map((v) => String(v ?? '').toLowerCase())
          .join(' ')
        return hay.includes(q)
      })
  }, [
    entries,
    selectedRoleId,
    sourceFilter,
    statusFilter,
    effectFilter,
    kindFilter,
    resourcePrefix,
    minPriority,
    searchQuery,
  ])

  const sortedEntries = useMemo(() => {
    const rows = filteredEntries.slice()
    const mul = sortDir === 'asc' ? 1 : -1
    rows.sort((a, b) => {
      switch (sortKey) {
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
  }, [filteredEntries, sortKey, sortDir])

  const hasActiveFilters =
    !!searchQuery.trim() ||
    !!kindFilter ||
    sourceFilter !== 'all' ||
    statusFilter !== 'active' ||
    effectFilter !== 'all' ||
    !!resourcePrefix.trim() ||
    !!minPriority.trim()

  const roleOptions = useMemo(
    () => roles.map((r) => ({ value: r.id, label: `${r.label || r.name} (${r.name})` })),
    [roles],
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
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSortKey(key)
    setSortDir(key === 'priority' ? 'desc' : 'asc')
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
    const rid = selectedRoleId || roles[0]?.id || ''
    setFormRoleId(rid)
    setFormName(
      suggestRuleName(String((roles.find((r) => r.id === rid) ?? {}).name ?? ''), 'deny', '', ''),
    )
  }

  function openCreateModal() {
    resetForm()
    setError('')
    setSaved(false)
    setModalOpen(true)
  }

  function selectFormResource(v: string) {
    setFormResource(v)
    setFormNameTouched(false)
    const meta = resourceOptions.find((o) => o.value === v.trim())
    const fields = meta?.fields ?? []
    setFormFieldsMode('all')
    setFormSelectedFields(fields.slice())
    setFormDomainMode('all')
  }

  async function loadAll() {
    setLoading(true)
    setError('')
    try {
      const [roleRows, entryRows] = await Promise.all([
        listModelRecords('permissions', 'role', ['name', 'label', 'active'], 'permissionsRoles'),
        listModelRecords(
          'permissions',
          'acl_entry',
          ['name', 'roleId', 'authorId', 'effect', 'kind', 'resource', 'actions', 'fields', 'domain', 'priority', 'active'],
          'permissionsAclEntries',
        ),
      ])
      const nextRoles = (roleRows as Role[])
        .filter((r) => r.active !== false)
        .sort((a, b) => String(a.name).localeCompare(String(b.name)))
      setRoles(nextRoles)
      setEntries(entryRows as AclEntry[])
      void loadResourceOptions(entryRows as AclEntry[]).then(setResourceOptions)
      setSelectedRoleId((current) => (current && !nextRoles.some((r) => r.id === current) ? '' : current))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadAll()
  }, [])

  async function activateEntry(row: AclEntry) {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await updateModelRecord('permissions', 'acl_entry', row.id, aclCreateFields, { active: true }, 'permissionsReviseAclEntry')
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
      await updateModelRecord('permissions', 'acl_entry', row.id, aclCreateFields, { active: false }, 'permissionsReviseAclEntry')
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
      await deleteModelRecord('permissions', 'acl_entry', row.id, 'permissionsDiscardAclEntry')
      setSaved(true)
      await loadAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  async function createOverride() {
    const rid = formRoleId.trim() || selectedRoleId
    if (!rid) {
      setError(t('permissions.error.role_required'))
      return
    }
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const name = formName.trim()
      if (!name) throw new Error(t('permissions.error.name_required'))
      const resource = formResource.trim()
      if (!resource) throw new Error(t('permissions.error.resource_required'))
      const kind = inferResourceKind(resource)
      const actions = isCallStyleKind(kind) ? '[]' : '["*"]'
      let fields = '"*"'
      let domain = '[]'
      if (kind === 'query' || kind === 'command') {
        if (formFieldsMode === 'selected') {
          if (formSelectedFields.length === 0) throw new Error(t('permissions.error.fields_required'))
          fields = JSON.stringify(formSelectedFields)
        }
        if (formDomainMode === 'custom') {
          domain = parseJSONString(formDomainCustom, '[]')
        }
      }
      const priority = parseInt(formPriority, 10)
      if (Number.isNaN(priority)) throw new Error(t('permissions.error.priority_invalid'))

      await createModelRecord(
        'permissions',
        'acl_entry',
        aclCreateFields,
        { name, roleId: rid, effect: formEffect, kind, resource, actions, fields, domain, priority, active: true },
        'permissionsPostAclEntry',
      )
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

  if (loading) {
    return (
      <div className="flex items-center gap-[var(--kg-space-05)]">
        <Progress indeterminate className="w-32" />
        <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.loading')}</p>
      </div>
    )
  }

  return (
    <>
      <div className="flex min-w-0 flex-col gap-[var(--kg-space-06)]">
        <Toolbar
          start={
            <div className="flex min-w-0 flex-col gap-1">
              <h1 className="m-0 text-2xl font-light tracking-tight text-[var(--kg-text)]">
                {t('permissions.access_title')}
              </h1>
              <p className="m-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.access_subtitle')}</p>
            </div>
          }
          end={
            <>
              <Button size="sm" variant="ghost" onClick={() => void loadAll()}>
                {t('permissions.refresh')}
              </Button>
              <Button size="sm" onClick={openCreateModal} disabled={roles.length === 0}>
                {t('permissions.override_new')}
              </Button>
            </>
          }
        />

        {error ? (
          <Alert variant="danger">
            <div className="min-w-0 flex-1">{error}</div>
            <button type="button" className="shrink-0" onClick={() => setError('')}>
              ×
            </button>
          </Alert>
        ) : null}
        {saved ? (
          <Alert variant="success">
            <div className="min-w-0 flex-1">{t('permissions.saved')}</div>
            <button type="button" className="shrink-0" onClick={() => setSaved(false)}>
              ×
            </button>
          </Alert>
        ) : null}

        {roles.length === 0 ? (
          <Alert variant="warning">{t('permissions.roles_empty')}</Alert>
        ) : (
          <>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(9.5rem,1fr))] gap-3">
              <StatCard label={t('permissions.stat.total')} value={summary.total} hint={t('permissions.stat.total_hint')} />
              <StatCard label={t('permissions.stat.active')} value={summary.active} />
              <StatCard label={t('permissions.stat.overrides')} value={summary.overrides} hint={t('permissions.stat.overrides_hint')} />
              <StatCard label={t('permissions.stat.denies')} value={summary.denies} />
              <StatCard label={t('permissions.stat.call')} value={summary.call} hint={t('permissions.stat.call_hint')} />
              <StatCard label={t('permissions.stat.app')} value={summary.app} hint={t('permissions.stat.app_hint')} />
            </div>

            <div className="grid min-w-0 grid-cols-1 gap-[var(--kg-space-05)] xl:grid-cols-[16rem_minmax(0,1fr)]">
              <aside className="border border-[var(--kg-border)] bg-[var(--kg-surface)]">
                <div className="border-b border-[var(--kg-border)] px-4 py-3">
                  <h2 className="m-0 text-sm font-semibold uppercase tracking-wide text-[var(--kg-text-secondary)]">
                    {t('permissions.roles_title')}
                  </h2>
                </div>
                <nav className="flex max-h-[32rem] flex-col overflow-y-auto" aria-label={t('permissions.roles_title')}>
                  <button
                    type="button"
                    className={cn(
                      'flex cursor-pointer items-center justify-between gap-2 border-0 border-b border-[var(--kg-border)] bg-transparent px-4 py-3 text-left text-sm hover:bg-[var(--kg-field-hover)]',
                      !selectedRoleId
                        ? 'bg-[var(--kg-field-hover)] font-medium text-[var(--kg-text)]'
                        : 'text-[var(--kg-text-secondary)]',
                    )}
                    onClick={() => selectRole('')}
                  >
                    <span>{t('permissions.filter.role_all')}</span>
                    <Badge variant="muted">{String(entries.filter((e) => e.active !== false).length)}</Badge>
                  </button>
                  {roles.map((role) => (
                    <button
                      key={role.id}
                      type="button"
                      className={cn(
                        'flex cursor-pointer items-center justify-between gap-2 border-0 border-b border-[var(--kg-border)] bg-transparent px-4 py-3 text-left text-sm hover:bg-[var(--kg-field-hover)]',
                        selectedRoleId === role.id
                          ? 'bg-[var(--kg-field-hover)] font-medium text-[var(--kg-text)]'
                          : 'text-[var(--kg-text-secondary)]',
                      )}
                      onClick={() => selectRole(role.id)}
                    >
                      <span className="min-w-0 truncate">
                        <span className="block truncate">{role.label || role.name}</span>
                        <span className="block font-mono text-xs text-[var(--kg-text-muted)]">{String(role.name)}</span>
                      </span>
                      <Badge variant="muted">{String(roleRuleCounts.get(role.id) ?? 0)}</Badge>
                    </button>
                  ))}
                </nav>
              </aside>

              <div className="flex min-w-0 flex-col gap-[var(--kg-space-05)]">
                <Card title={t('permissions.rules_title')}>
                  {selectedRole ? (
                    <p className="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">
                      {t('permissions.rules_for_role')}{' '}
                      <span className="font-medium text-[var(--kg-text)]">{selectedRole.label || selectedRole.name}</span>
                      <Badge variant="muted" className="ms-2">
                        {String(selectedRole.name)}
                      </Badge>
                    </p>
                  ) : null}
                  <p className="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.rules_hint')}</p>

                  <div className="mb-4 flex flex-wrap gap-2">
                    {[...KIND_FILTERS].map((kind) => (
                      <button
                        key={kind || 'all'}
                        type="button"
                        className={cn(
                          'cursor-pointer rounded-sm border px-2.5 py-1 text-xs font-medium',
                          kindFilter === kind
                            ? 'border-[var(--kg-text)] bg-[var(--kg-text)] text-[var(--kg-surface)]'
                            : 'border-[var(--kg-border)] bg-transparent text-[var(--kg-text-secondary)] hover:border-[var(--kg-text-muted)]',
                        )}
                        onClick={() => setKindFilter(kind)}
                      >
                        {kind ? kind : t('permissions.filter.kind_all')}
                      </button>
                    ))}
                  </div>

                  <div className="mb-4 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.search')}</Label>
                      <Input value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} placeholder={t('permissions.filter.search_placeholder')} />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.resource_prefix')}</Label>
                      <Input value={resourcePrefix} onChange={(e) => setResourcePrefix(e.target.value)} placeholder="hellospec." />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.min_priority')}</Label>
                      <Input value={minPriority} onChange={(e) => setMinPriority(e.target.value)} placeholder="0" />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.source')}</Label>
                      <OptionSelect
                        value={sourceFilter}
                        options={[
                          { value: 'all', label: t('permissions.filter.source_all') },
                          { value: 'seed', label: t('permissions.source.seed') },
                          { value: 'override', label: t('permissions.source.override') },
                        ]}
                        onChange={(v) => setSourceFilter(v as SourceFilter)}
                      />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.status')}</Label>
                      <OptionSelect
                        value={statusFilter}
                        options={[
                          { value: 'all', label: t('permissions.filter.status_all') },
                          { value: 'active', label: t('permissions.filter.status_active') },
                          { value: 'inactive', label: t('permissions.filter.status_inactive') },
                        ]}
                        onChange={(v) => setStatusFilter(v as StatusFilter)}
                      />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('permissions.filter.effect')}</Label>
                      <OptionSelect
                        value={effectFilter}
                        options={[
                          { value: 'all', label: t('permissions.filter.effect_all') },
                          { value: 'allow', label: t('permissions.effect.allow') },
                          { value: 'deny', label: t('permissions.effect.deny') },
                        ]}
                        onChange={(v) => setEffectFilter(v as EffectFilter)}
                      />
                    </div>
                  </div>

                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <p className="m-0 text-sm text-[var(--kg-text-muted)]">
                      {t('permissions.rules_count').replace('%d', String(sortedEntries.length))}
                    </p>
                    {hasActiveFilters ? (
                      <Button size="sm" variant="ghost" onClick={clearFilters}>
                        {t('permissions.filter.clear')}
                      </Button>
                    ) : null}
                  </div>

                  {sortedEntries.length > 0 ? (
                    <RulesTable
                      rows={sortedEntries}
                      showRoleColumn={!selectedRoleId}
                      roleLabel={(row) => {
                        const role = rolesById[String(row.roleId ?? '')]
                        return role ? String(role.label || role.name) : String(row.roleId ?? '')
                      }}
                      sortKey={sortKey}
                      sortDir={sortDir}
                      saving={saving}
                      onSort={toggleSort}
                      onDeactivate={(row) => void deactivateEntry(row)}
                      onActivate={(row) => void activateEntry(row)}
                      onDelete={(row) => void deleteEntry(row)}
                    />
                  ) : (
                    <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.rules_empty')}</p>
                  )}
                </Card>
              </div>
            </div>
          </>
        )}
      </div>

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader className="flex items-center justify-between">
            <DialogTitle>{t('permissions.override_title')}</DialogTitle>
            <button
              type="button"
              className="border-0 bg-transparent text-2xl"
              onClick={() => setModalOpen(false)}
              aria-label="Close"
            >
              ×
            </button>
          </DialogHeader>
          <div className="overflow-auto p-5">
            <p className="mb-5 mt-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.override_hint')}</p>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="flex flex-col gap-1">
                <Label>{t('permissions.field.role')}</Label>
                <OptionSelect
                  value={formRoleId}
                  options={roleOptions}
                  onChange={(v) => {
                    setFormRoleId(v)
                    setFormNameTouched(false)
                  }}
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label>{t('permissions.field.effect')}</Label>
                <OptionSelect
                  value={formEffect}
                  options={[
                    { value: 'allow', label: t('permissions.effect.allow') },
                    { value: 'deny', label: t('permissions.effect.deny') },
                  ]}
                  onChange={(v) => {
                    setFormEffect(v)
                    setFormNameTouched(false)
                  }}
                />
              </div>
              <div className="flex flex-col gap-1 md:col-span-2">
                <Label>{t('permissions.field.resource')}</Label>
                <SearchableCombobox
                  value={formResource}
                  options={resourceOptions}
                  placeholder="hellospec.command.hellospecPostGreeting"
                  onChange={selectFormResource}
                />
              </div>
              {formResource.trim() ? (
                <div className="flex flex-col gap-1 md:col-span-2">
                  <Label>{t('permissions.field.kind')}</Label>
                  <div className="flex items-center gap-2">
                    {kindBadge(formKind)}
                    {formIsCallStyle && !formSupportsFieldsDomain ? (
                      <span className="text-sm text-[var(--kg-text-muted)]">
                        {t('permissions.field.call_style_hint')}
                      </span>
                    ) : null}
                  </div>
                </div>
              ) : null}
              {formSupportsFieldsDomain ? (
                <>
                  <div className="flex flex-col gap-1 md:col-span-2">
                    <Label>{t('permissions.field.fields')}</Label>
                    <OptionSelect
                      value={formFieldsMode}
                      options={[
                        { value: 'all', label: t('permissions.fields.all') },
                        { value: 'selected', label: t('permissions.fields.selected') },
                      ]}
                      onChange={(v) => {
                        const mode = v as 'all' | 'selected'
                        setFormFieldsMode(mode)
                        if (mode === 'all') setFormSelectedFields(formAvailableFields.slice())
                      }}
                    />
                  </div>
                  {formFieldsMode === 'selected' ? (
                    <div className="flex flex-col gap-1 md:col-span-2">
                      <Label>{t('permissions.field.fields_pick')}</Label>
                      <SearchableMultiSelect
                        value={formSelectedFields}
                        options={formAvailableFields.map((field) => ({ value: field, label: field }))}
                        placeholder={t('permissions.field.fields_pick_placeholder')}
                        onChange={(next) => {
                          setFormSelectedFields(next)
                          setFormFieldsMode('selected')
                        }}
                      />
                    </div>
                  ) : null}
                  <div className="flex flex-col gap-1 md:col-span-2">
                    <Label>{t('permissions.field.domain')}</Label>
                    <OptionSelect
                      value={formDomainMode}
                      options={[
                        { value: 'all', label: t('permissions.domain.all') },
                        { value: 'custom', label: t('permissions.domain.custom') },
                      ]}
                      onChange={(v) => setFormDomainMode(v as 'all' | 'custom')}
                    />
                  </div>
                  {formDomainMode === 'custom' ? (
                    <div className="flex flex-col gap-1 md:col-span-2">
                      <Label>{t('permissions.field.domain')}</Label>
                      <Input
                        value={formDomainCustom}
                        onChange={(e) => setFormDomainCustom(e.target.value)}
                        placeholder='[["authorId","=","$user.id"]]'
                      />
                    </div>
                  ) : null}
                </>
              ) : null}
              <div className="flex flex-col gap-1 md:col-span-2">
                <Label>{t('permissions.field.priority')}</Label>
                <Input
                  value={formPriority}
                  onChange={(e) => setFormPriority(e.target.value)}
                  placeholder="1000"
                />
              </div>
              <div className="flex flex-col gap-1 md:col-span-2">
                <Label>{t('permissions.field.name')}</Label>
                <Input
                  value={formName}
                  onChange={(e) => {
                    setFormNameTouched(true)
                    setFormName(e.target.value)
                  }}
                  placeholder={suggestedName || 'member-deny-view-Users'}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <FormActions>
              <Button variant="ghost" onClick={() => setModalOpen(false)}>
                {t('permissions.cancel')}
              </Button>
              <Button disabled={saving} onClick={() => void createOverride()}>
                {saving ? '…' : t('permissions.override_submit')}
              </Button>
            </FormActions>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <KAppStatus />
    </>
  )
}
