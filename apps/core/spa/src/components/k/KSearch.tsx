import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Bookmark, Filter, Layers, Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import { KEYMAP_ID_ATTR } from '@/lib/keymap/types'
import { useKSearchParams } from './search/params'
import {
  deleteSearchTemplate,
  listSearchTemplates,
  saveSearchTemplate,
} from './search/templates'
import {
  extractLeaves,
  opsForFieldType,
  serializeLeaves,
  type DomainLeaf,
  type DomainOp,
  type SearchField,
  type SearchState,
  type SearchTemplate,
} from './search/types'

export type KSearchProps = {
  /** Model key for templates, e.g. hellospec.greeting */
  model: string
  fields: SearchField[]
  /** Fields offered for quick search (default: string-ish fields). */
  searchable?: string[]
  /** Fields offered for filters (default: all fields). */
  filterable?: string[]
  /** Fields offered for groupBy (default: enum/bool/many2one + string). */
  groupable?: string[]
  placeholder?: string
  /** Controlled mode — parent owns URL via useKSearchParams. */
  q?: string
  searchIn?: string[]
  domain?: SearchState['domain']
  groupBy?: string[]
  junction?: 'and' | 'or'
  onChange?: (next: Partial<SearchState>) => void
  url?: boolean
  className?: string
}

function fieldLabel(fields: SearchField[], key: string) {
  return fields.find((f) => f.key === key)?.label ?? key
}

function leafLabel(fields: SearchField[], leaf: DomainLeaf): string {
  const [field, op, value] = leaf
  const name = fieldLabel(fields, field)
  if (op === 'is set' || op === 'is not set') return `${name} ${op}`
  return `${name} ${op} ${value == null ? '' : String(value)}`
}

/**
 * Server-backed search / filter / groupBy toolbar (Odoo-inspired).
 * Syncs to `?q=` `?searchIn=` `?domain=` `?groupBy=` by default.
 */
export function KSearch(props: KSearchProps) {
  const controlled = props.onChange != null
  const fromUrl = useKSearchParams({ url: props.url !== false && !controlled })

  const q = props.q ?? fromUrl.q
  const searchIn = props.searchIn ?? fromUrl.searchIn
  const domain = props.domain ?? fromUrl.domain
  const groupBy = props.groupBy ?? fromUrl.groupBy
  const junction = props.junction ?? fromUrl.junction

  function patch(next: Partial<SearchState>) {
    if (props.onChange) props.onChange(next)
    else fromUrl.setState(next)
  }

  const searchable = useMemo(() => {
    if (props.searchable?.length) {
      return props.fields.filter((f) => props.searchable!.includes(f.key))
    }
    return props.fields.filter((f) => {
      const t = (f.type ?? 'string').toLowerCase()
      return t === 'string' || t === 'text' || t === 'enum' || !f.type
    })
  }, [props.fields, props.searchable])

  const filterable = useMemo(() => {
    if (props.filterable?.length) {
      return props.fields.filter((f) => props.filterable!.includes(f.key))
    }
    return props.fields
  }, [props.fields, props.filterable])

  const groupable = useMemo(() => {
    if (props.groupable?.length) {
      return props.fields.filter((f) => props.groupable!.includes(f.key))
    }
    return props.fields.filter((f) => {
      const t = (f.type ?? '').toLowerCase()
      return ['enum', 'bool', 'boolean', 'many2one', 'string'].includes(t) || !t
    })
  }, [props.fields, props.groupable])

  const { leaves } = extractLeaves(domain)

  const [draftQ, setDraftQ] = useState(q)
  useEffect(() => setDraftQ(q), [q])

  const [filterOpen, setFilterOpen] = useState(false)
  const [newField, setNewField] = useState(filterable[0]?.key ?? '')
  const [newOp, setNewOp] = useState<DomainOp>('ilike')
  const [newValue, setNewValue] = useState('')

  useEffect(() => {
    const f = filterable.find((x) => x.key === newField)
    const ops = opsForFieldType(f?.type)
    if (!ops.includes(newOp)) setNewOp(ops[0] ?? '=')
  }, [newField, filterable, newOp])

  const [templates, setTemplates] = useState<SearchTemplate[]>(() =>
    listSearchTemplates(props.model),
  )
  const [tplName, setTplName] = useState('')

  function refreshTemplates() {
    setTemplates(listSearchTemplates(props.model))
  }

  function submitSearch(e: FormEvent) {
    e.preventDefault()
    patch({ q: draftQ })
  }

  function toggleSearchField(key: string) {
    const set = new Set(searchIn)
    if (set.has(key)) set.delete(key)
    else set.add(key)
    patch({ searchIn: [...set] })
  }

  function addFilter() {
    if (!newField) return
    let value: unknown = newValue
    if (newOp === 'ilike' || newOp === 'like') {
      const s = String(newValue ?? '')
      value = s.includes('%') ? s : `%${s}%`
    }
    const leaf: DomainLeaf =
      newOp === 'is set' || newOp === 'is not set'
        ? [newField, newOp]
        : [newField, newOp, value]
    const nextLeaves = [...leaves, leaf]
    patch({ domain: serializeLeaves(nextLeaves, junction) })
    setNewValue('')
    setFilterOpen(false)
  }

  function removeLeaf(index: number) {
    const next = leaves.filter((_, i) => i !== index)
    patch({ domain: serializeLeaves(next, junction) })
  }

  function toggleGroup(key: string) {
    const idx = groupBy.indexOf(key)
    if (idx >= 0) patch({ groupBy: groupBy.filter((g) => g !== key) })
    else patch({ groupBy: [...groupBy, key] })
  }

  function applyTemplate(t: SearchTemplate) {
    patch({
      q: t.q,
      searchIn: t.searchIn,
      domain: t.domain,
      groupBy: t.groupBy,
    })
    setDraftQ(t.q)
  }

  function saveTemplate() {
    const name = tplName.trim()
    if (!name) return
    saveSearchTemplate({
      model: props.model,
      name,
      q,
      searchIn,
      domain,
      groupBy,
    })
    setTplName('')
    refreshTemplates()
  }

  const activeCount = leaves.length + (q ? 1 : 0) + groupBy.length

  return (
    <div
      className={cn(
        'flex flex-col gap-2 border border-[var(--kg-border)] bg-[var(--kg-surface)] p-3',
        props.className,
      )}
    >
      <div className="flex flex-wrap items-center gap-2">
        <form className="flex min-w-[14rem] flex-1 items-center gap-2" onSubmit={submitSearch}>
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--kg-text-muted)]" />
            <Input
              value={draftQ}
              onChange={(e) => setDraftQ(e.target.value)}
              onBlur={() => {
                if (draftQ !== q) patch({ q: draftQ })
              }}
              placeholder={props.placeholder ?? 'Search…'}
              className="pl-9"
              aria-label="Search"
              {...{ [KEYMAP_ID_ATTR]: 'k-search' }}
            />
          </div>
          <Button type="submit" size="sm" variant="secondary">
            Search
          </Button>
        </form>

        {/* Search fields */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" size="sm" variant="secondary">
              Fields{searchIn.length ? ` (${searchIn.length})` : ''}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="min-w-[14rem]">
            <DropdownMenuLabel>Search in</DropdownMenuLabel>
            {searchable.map((f) => (
              <DropdownMenuItem
                key={f.key}
                className="gap-2"
                onSelect={(e) => {
                  e.preventDefault()
                  toggleSearchField(f.key)
                }}
              >
                <Checkbox checked={searchIn.includes(f.key)} />
                {f.label}
              </DropdownMenuItem>
            ))}
            {!searchable.length ? (
              <div className="px-3 py-2 text-sm text-[var(--kg-text-muted)]">No fields</div>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Filters */}
        <div className="relative">
          <Button
            type="button"
            size="sm"
            variant={filterOpen || leaves.length ? 'secondary' : 'ghost'}
            onClick={() => setFilterOpen((o) => !o)}
            {...{ [KEYMAP_ID_ATTR]: 'k-search-filter' }}
          >
            <Filter className="h-4 w-4" />
            Filter{leaves.length ? ` (${leaves.length})` : ''}
          </Button>
          {filterOpen ? (
            <div className="absolute right-0 z-40 mt-1 w-[22rem] border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] p-3 shadow-lg">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-muted)]">
                  Add filter
                </span>
                <div className="inline-flex border border-[var(--kg-border)]">
                  <Button
                    type="button"
                    size="sm"
                    variant={junction === 'and' ? 'secondary' : 'ghost'}
                    onClick={() => {
                      patch({ junction: 'and', domain: serializeLeaves(leaves, 'and') })
                    }}
                  >
                    AND
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant={junction === 'or' ? 'secondary' : 'ghost'}
                    onClick={() => {
                      patch({ junction: 'or', domain: serializeLeaves(leaves, 'or') })
                    }}
                  >
                    OR
                  </Button>
                </div>
              </div>
              <div className="flex flex-col gap-2">
                <Select value={newField} onValueChange={setNewField}>
                  <SelectTrigger aria-label="Filter field">
                    <SelectValue placeholder="Field" />
                  </SelectTrigger>
                  <SelectContent>
                    {filterable.map((f) => (
                      <SelectItem key={f.key} value={f.key}>
                        {f.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select value={newOp} onValueChange={(v) => setNewOp(v as DomainOp)}>
                  <SelectTrigger aria-label="Operator">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {opsForFieldType(filterable.find((f) => f.key === newField)?.type).map((op) => (
                      <SelectItem key={op} value={op}>
                        {op}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {newOp !== 'is set' && newOp !== 'is not set' ? (
                  filterable.find((f) => f.key === newField)?.values?.length ? (
                    <Select value={newValue} onValueChange={setNewValue}>
                      <SelectTrigger aria-label="Value">
                        <SelectValue placeholder="Value" />
                      </SelectTrigger>
                      <SelectContent>
                        {filterable
                          .find((f) => f.key === newField)!
                          .values!.map((v) => (
                            <SelectItem key={v} value={v}>
                              {v}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  ) : (
                    <Input
                      value={newValue}
                      onChange={(e) => setNewValue(e.target.value)}
                      placeholder="Value"
                    />
                  )
                ) : null}
                <div className="flex justify-end gap-2">
                  <Button type="button" size="sm" variant="ghost" onClick={() => setFilterOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="button" size="sm" onClick={addFilter}>
                    Add
                  </Button>
                </div>
              </div>
            </div>
          ) : null}
        </div>

        {/* Group by */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              size="sm"
              variant={groupBy.length ? 'secondary' : 'ghost'}
              {...{ [KEYMAP_ID_ATTR]: 'k-search-group' }}
            >
              <Layers className="h-4 w-4" />
              Group{groupBy.length ? ` (${groupBy.length})` : ''}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="min-w-[14rem]">
            <DropdownMenuLabel>Group by (nested order)</DropdownMenuLabel>
            {groupable.map((f) => {
              const idx = groupBy.indexOf(f.key)
              return (
                <DropdownMenuItem
                  key={f.key}
                  className="gap-2"
                  onSelect={(e) => {
                    e.preventDefault()
                    toggleGroup(f.key)
                  }}
                >
                  <Checkbox checked={idx >= 0} />
                  <span className="flex-1">{f.label}</span>
                  {idx >= 0 ? (
                    <span className="text-xxs tabular-nums text-[var(--kg-text-muted)]">{idx + 1}</span>
                  ) : null}
                </DropdownMenuItem>
              )
            })}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Templates */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" size="sm" variant="ghost">
              <Bookmark className="h-4 w-4" />
              Saved
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="min-w-[16rem]">
            <DropdownMenuLabel>Filter templates</DropdownMenuLabel>
            {templates.map((t) => (
              <DropdownMenuItem
                key={t.id}
                className="flex items-center justify-between gap-2"
                onSelect={() => applyTemplate(t)}
              >
                <span className="truncate">{t.name}</span>
                <button
                  type="button"
                  className="text-xs text-[var(--kg-danger)]"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    deleteSearchTemplate(props.model, t.id)
                    refreshTemplates()
                  }}
                >
                  Delete
                </button>
              </DropdownMenuItem>
            ))}
            {!templates.length ? (
              <div className="px-3 py-2 text-sm text-[var(--kg-text-muted)]">No saved filters</div>
            ) : null}
            <div className="mt-1 flex gap-2 border-t border-[var(--kg-border)] p-2">
              <Input
                value={tplName}
                onChange={(e) => setTplName(e.target.value)}
                placeholder="Save as…"
                className="h-8"
              />
              <Button type="button" size="sm" onClick={saveTemplate}>
                Save
              </Button>
            </div>
          </DropdownMenuContent>
        </DropdownMenu>

        {activeCount > 0 ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => {
              patch({ q: '', searchIn: [], domain: [], groupBy: [] })
              setDraftQ('')
            }}
          >
            Clear
          </Button>
        ) : null}
      </div>

      {(leaves.length > 0 || groupBy.length > 0 || (q && searchIn.length)) && (
        <div className="flex flex-wrap items-center gap-2">
          {q && searchIn.length ? (
            <span className="inline-flex items-center gap-1 border border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))] px-2 py-1 text-xs">
              in: {searchIn.map((k) => fieldLabel(props.fields, k)).join(', ')}
            </span>
          ) : null}
          {leaves.map((leaf, i) => (
            <span
              key={`${leaf[0]}-${i}`}
              className="inline-flex items-center gap-1 border border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))] px-2 py-1 text-xs"
            >
              {i > 0 ? (
                <span className="mr-1 font-semibold uppercase text-[var(--kg-text-muted)]">
                  {junction}
                </span>
              ) : null}
              {leafLabel(props.fields, leaf)}
              <button type="button" aria-label="Remove filter" onClick={() => removeLeaf(i)}>
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
          {groupBy.map((g, i) => (
            <span
              key={g}
              className="inline-flex items-center gap-1 border border-[var(--kg-border)] px-2 py-1 text-xs"
            >
              group {i + 1}: {fieldLabel(props.fields, g)}
              <button type="button" aria-label="Remove group" onClick={() => toggleGroup(g)}>
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
