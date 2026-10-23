import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { Bookmark, ChevronRight, Filter, Layers, Search, SearchIcon, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Checkbox } from '@/components/ui/checkbox'
import { t } from '@/lib'
import { cn } from '@/lib/utils'
import { KEYMAP_ID_ATTR } from '@/lib/keymap/types'
import { KFilterCriteriaDialog } from './search/KFilterCriteriaDialog'
import { KSaveCriteriaDialog } from './search/KSaveCriteriaDialog'
import { useKSearchParams } from './search/params'
import {
  countFilterLeaves,
  domainContains,
  domainToFilterTree,
  filterTreeToDomain,
  andDomains,
  domainsEqual,
  peelSearchFacets,
  searchTermToDomain,
  toggleDomainConjunct,
  updateFilterNode,
  type FilterGroup,
  type FilterNode,
  type SearchFacet,
} from './search/domain-tree'
import { deleteSearchTemplate, saveSearchTemplate } from './search/templates'
import {
  type Domain,
  type DomainLeaf,
  type SearchField,
  type SearchFilterPreset,
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
  /** Spec-defined quick filters from `{app}Views.filterPresets`. */
  presets?: SearchFilterPreset[]
  /** Saved criteria (server + local). */
  savedTemplates?: SearchTemplate[]
  pageSize?: number
  onApplyPageSize?: (pageSize: number) => void
  onTemplatesChanged?: () => void | Promise<void>
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

type SearchSuggestion = {
  id: string
  label: string
  hint?: string
  searchIn: string[]
  q: string
}

function buildSearchSuggestions(fields: SearchField[], raw: string): SearchSuggestion[] {
  const trimmed = raw.trim()
  if (!trimmed) return []

  const colon = raw.indexOf(':')
  const prefix = colon >= 0 ? raw.slice(0, colon).trim().toLowerCase() : ''
  const queryForSearch = colon >= 0 ? raw.slice(colon + 1).trim() : trimmed

  const filtered = prefix
    ? fields.filter(
        (f) =>
          f.key.toLowerCase().includes(prefix) ||
          f.label.toLowerCase().includes(prefix),
      )
    : fields

  if (!filtered.length) return []

  const items: SearchSuggestion[] = []

  if (!prefix && queryForSearch) {
    items.push({
      id: '__all__',
      label: queryForSearch,
      hint: 'All fields',
      searchIn: [],
      q: queryForSearch,
    })
  }

  for (const f of filtered) {
    items.push({
      id: f.key,
      label: queryForSearch ? `${f.label}: ${queryForSearch}` : `${f.label}:`,
      searchIn: [f.key],
      q: queryForSearch,
    })
  }

  return items
}

function parsePresetDomain(raw?: string | null): Domain {
  if (!raw?.trim()) return []
  try {
    const parsed = JSON.parse(raw) as unknown
    return Array.isArray(parsed) ? (parsed as Domain) : []
  } catch {
    return []
  }
}

function presetLabel(p: SearchFilterPreset): string {
  if (p.labelKey) return t(p.labelKey)
  return p.label ?? p.id
}

type TagKind = 'search' | 'filter' | 'group' | 'preset'

const tagStyles: Record<TagKind, string> = {
  search:
    'border-[var(--kg-info-border)] bg-[var(--kg-info-bg)] text-[var(--kg-text)] [&_.kg-tag-label]:text-[var(--kg-info)]',
  filter:
    'border-[var(--kg-warning-border)] bg-[var(--kg-warning-bg)] text-[var(--kg-text)] [&_.kg-tag-label]:text-[var(--kg-warning)]',
  group:
    'border-[var(--kg-success-border)] bg-[var(--kg-success-bg)] text-[var(--kg-text)] [&_.kg-tag-label]:text-[var(--kg-success)]',
  preset:
    'border-[var(--kg-warning-border)] bg-[var(--kg-warning-bg)] text-[var(--kg-text)] [&_.kg-tag-label]:text-[var(--kg-warning)]',
}

const activeToolbarStyles = {
  filter:
    'border-[var(--kg-warning-border)] bg-[var(--kg-warning-bg)] text-[var(--kg-warning)] hover:bg-[var(--kg-warning-bg)] hover:opacity-90',
  group:
    'border-[var(--kg-success-border)] bg-[var(--kg-success-bg)] text-[var(--kg-success)] hover:bg-[var(--kg-success-bg)] hover:opacity-90',
} as const

function TagChip({
  kind,
  label,
  icon,
  children,
  onRemove,
  onClick,
  className,
}: {
  kind: TagKind
  label?: string
  icon?: ReactNode
  children: ReactNode
  onRemove?: () => void
  onClick?: () => void
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-flex max-w-full items-center gap-2 rounded-md border px-3 py-1.5 text-sm leading-snug',
        tagStyles[kind],
        onClick && 'cursor-pointer hover:opacity-90',
        className,
      )}
      onClick={onClick}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                onClick()
              }
            }
          : undefined
      }
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {icon}
      {label ? (
        <span className="kg-tag-label shrink-0 text-xs font-semibold uppercase tracking-wide">{label}</span>
      ) : null}
      <span className="min-w-0">{children}</span>
      {onRemove ? (
        <button
          type="button"
          className="shrink-0 rounded-sm p-0.5 opacity-70 transition hover:bg-black/10 hover:opacity-100"
          aria-label="Remove"
          onClick={(e) => {
            e.stopPropagation()
            onRemove()
          }}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      ) : null}
    </span>
  )
}

function findActivePresets(
  presets: SearchFilterPreset[],
  templates: SearchTemplate[],
  domain: Domain,
): SearchFilterPreset[] {
  return presets.filter((preset) => {
    if (preset.id.startsWith('user:')) {
      const tpl = templates.find((t) => `user:${t.id}` === preset.id)
      return tpl ? domainContains(domain, tpl.domain) : false
    }
    const pd = parsePresetDomain(preset.domain)
    return domainContains(domain, pd)
  })
}

function FilterChipTree({
  node,
  fields,
  onRemove,
  depth = 0,
}: {
  node: FilterNode
  fields: SearchField[]
  onRemove: (id: string) => void
  depth?: number
}) {
  if (node.kind === 'leaf') {
    return (
      <TagChip
        kind="filter"
        label={depth === 0 ? 'Filter' : undefined}
        onRemove={() => onRemove(node.id)}
      >
        {leafLabel(fields, [node.field, node.op, node.value])}
      </TagChip>
    )
  }
  if (!node.children.length) return null
  if (node.children.length === 1) {
    return <FilterChipTree node={node.children[0]} fields={fields} onRemove={onRemove} depth={depth} />
  }
  return (
    <span
      className={cn(
        'inline-flex flex-wrap items-center gap-1.5 rounded-md border border-[var(--kg-warning-border)] bg-[var(--kg-warning-bg)] px-2.5 py-1.5',
        depth > 0 && 'ml-1 border-l-2 border-l-[var(--kg-warning)]',
      )}
    >
      <span className="rounded bg-[var(--kg-warning)]/15 px-1.5 py-0.5 text-xxs font-bold uppercase tracking-wide text-[var(--kg-warning)]">
        {node.junction}
      </span>
      {node.children.map((child, i) => (
        <span key={child.id} className="inline-flex items-center gap-1.5">
          {i > 0 ? <span className="text-xs text-[var(--kg-text-muted)]">·</span> : null}
          <FilterChipTree node={child} fields={fields} onRemove={onRemove} depth={depth + 1} />
        </span>
      ))}
    </span>
  )
}

function GroupNestChip({
  groupBy,
  fields,
  onRemove,
}: {
  groupBy: string[]
  fields: SearchField[]
  onRemove: (key: string) => void
}) {
  return (
    <TagChip
      kind="group"
      label="Group"
      icon={<Layers className="h-3.5 w-3.5 shrink-0 opacity-80" />}
    >
      <span className="inline-flex flex-wrap items-center gap-1">
        {groupBy.map((key, i) => (
          <span key={key} className="inline-flex items-center gap-1">
            {i > 0 ? (
              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-[var(--kg-success)]" aria-hidden />
            ) : null}
            <span className="inline-flex items-center gap-1.5 rounded bg-black/10 px-2 py-0.5">
              <span
                className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-[var(--kg-success)]/20 text-xxs font-bold tabular-nums text-[var(--kg-success)]"
                aria-hidden
              >
                {i + 1}
              </span>
              <span>{fieldLabel(fields, key)}</span>
              <button
                type="button"
                className="rounded-sm p-0.5 opacity-70 transition hover:bg-black/10 hover:opacity-100"
                aria-label={`Remove group ${fieldLabel(fields, key)}`}
                onClick={() => onRemove(key)}
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          </span>
        ))}
      </span>
    </TagChip>
  )
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

  function patch(next: Partial<SearchState>) {
    if (props.onChange) props.onChange(next)
    else fromUrl.setState(next)
  }

  const filterTree = useMemo(() => domainToFilterTree(domain), [domain])
  const { facets: searchFacets, remaining: filterOnlyTree } = useMemo(
    () => peelSearchFacets(filterTree),
    [filterTree],
  )

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

  const [draftQ, setDraftQ] = useState('')
  const [draftSearchIn, setDraftSearchIn] = useState<string[]>([])
  const searchInputRef = useRef<HTMLInputElement>(null)

  const [suggestOpen, setSuggestOpen] = useState(false)
  const [highlightIndex, setHighlightIndex] = useState(-1)

  const suggestions = useMemo(
    () => buildSearchSuggestions(searchable, draftQ),
    [searchable, draftQ],
  )
  const showSuggestions = suggestOpen && suggestions.length > 0

  useEffect(() => {
    if (!suggestions.length) {
      setHighlightIndex(-1)
      return
    }
    setHighlightIndex((i) => (i < 0 || i >= suggestions.length ? 0 : i))
  }, [suggestions])

  const [criteriaOpen, setCriteriaOpen] = useState(false)

  const templates = props.savedTemplates ?? []
  const [saveOpen, setSaveOpen] = useState(false)
  const [saveSeed, setSaveSeed] = useState<Partial<SearchTemplate>>({})

  async function refreshTemplates() {
    await props.onTemplatesChanged?.()
  }

  function openSaveDialog(seed?: Partial<SearchTemplate>) {
    setSaveSeed(seed ?? {})
    setSaveOpen(true)
  }

  function applySuggestion(item: SearchSuggestion) {
    const nextQ = item.q.trim()
    const nextSearchIn = item.searchIn
    if (!nextQ) {
      setDraftSearchIn(nextSearchIn)
      setSuggestOpen(false)
      setHighlightIndex(-1)
      return
    }
    commitSearchTerm(nextQ, nextSearchIn)
    setSuggestOpen(false)
    setHighlightIndex(-1)
  }

  function commitSearchTerm(term: string, fields: string[]) {
    const trimmed = term.trim()
    if (!trimmed) return
    const keys = fields.length ? fields : searchable.map((f) => f.key)
    if (!keys.length) return

    let nextDomain = domain
    // Flush any leftover free-text q into the domain first.
    if (q.trim()) {
      const flushKeys = searchIn.length ? searchIn : searchable.map((f) => f.key)
      if (flushKeys.length) {
        nextDomain = andDomains(nextDomain, searchTermToDomain(q.trim(), flushKeys))
      }
    }
    nextDomain = andDomains(nextDomain, searchTermToDomain(trimmed, keys))
    patch({ q: '', searchIn: [], domain: nextDomain })
    setDraftQ('')
    setDraftSearchIn([])
  }

  function submitSearch(e: FormEvent) {
    e.preventDefault()
    if (showSuggestions && highlightIndex >= 0 && suggestions[highlightIndex]) {
      applySuggestion(suggestions[highlightIndex])
      return
    }
    commitSearchTerm(draftQ, draftSearchIn)
    setSuggestOpen(false)
  }

  function editSearchFacet(facet: SearchFacet) {
    const next = updateFilterNode(filterTree, facet.id, () => null)
    patch({ domain: filterTreeToDomain(next), q: '', searchIn: [] })
    setDraftQ(facet.q)
    setDraftSearchIn(facet.searchIn)
    window.setTimeout(() => searchInputRef.current?.focus(), 0)
  }

  function editPendingQ() {
    setDraftQ(q)
    setDraftSearchIn(searchIn)
    patch({ q: '', searchIn: [] })
    window.setTimeout(() => searchInputRef.current?.focus(), 0)
  }

  function removeSearchFacet(facet: SearchFacet) {
    const next = updateFilterNode(filterTree, facet.id, () => null)
    patch({ domain: filterTreeToDomain(next) })
  }

  function handleSearchInputKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown') {
      if (!suggestions.length) return
      e.preventDefault()
      setSuggestOpen(true)
      setHighlightIndex((i) => Math.min(i < 0 ? 0 : i + 1, suggestions.length - 1))
      return
    }
    if (e.key === 'ArrowUp') {
      if (!suggestions.length) return
      e.preventDefault()
      setSuggestOpen(true)
      setHighlightIndex((i) => Math.max(i < 0 ? 0 : i - 1, 0))
      return
    }
    if (e.key === 'Escape') {
      if (!showSuggestions) return
      e.preventDefault()
      setSuggestOpen(false)
      setHighlightIndex(-1)
      return
    }
    if (e.key === 'Enter' && showSuggestions && highlightIndex >= 0) {
      e.preventDefault()
      applySuggestion(suggestions[highlightIndex])
    }
  }

  function toggleSearchField(key: string) {
    const set = new Set(searchIn)
    if (set.has(key)) set.delete(key)
    else set.add(key)
    patch({ searchIn: [...set] })
  }

  function normalizeFilterTree(node: FilterNode): FilterNode {
    if (node.kind === 'leaf') {
      if (node.op === 'ilike' || node.op === 'like') {
        const s = String(node.value ?? '')
        if (s && !s.includes('%')) return { ...node, value: `%${s}%` }
      }
      return node
    }
    return { ...node, children: node.children.map(normalizeFilterTree) }
  }

  function applyFilterTree(next: FilterGroup) {
    const normalized = normalizeFilterTree(next) as FilterGroup
    const searchDomain =
      searchFacets.length === 0
        ? []
        : filterTreeToDomain(
            searchFacets.length === 1
              ? searchFacets[0].node
              : {
                  kind: 'group',
                  id: 'search',
                  junction: 'and',
                  children: searchFacets.map((f) => f.node),
                },
          )
    patch({
      domain: andDomains(searchDomain, filterTreeToDomain(normalized)),
    })
  }

  function removeFilterNode(id: string) {
    const next = updateFilterNode(filterOnlyTree, id, () => null)
    applyFilterTree(next)
  }

  function clearFilters() {
    const searchDomain =
      searchFacets.length === 0
        ? []
        : filterTreeToDomain(
            searchFacets.length === 1
              ? searchFacets[0].node
              : {
                  kind: 'group',
                  id: 'search',
                  junction: 'and',
                  children: searchFacets.map((f) => f.node),
                },
          )
    patch({ domain: searchDomain })
  }

  function applyTemplateState(t: Pick<SearchTemplate, 'q' | 'searchIn' | 'domain' | 'groupBy' | 'pageSize'>) {
    let nextDomain = t.domain
    if (t.q.trim()) {
      const keys = t.searchIn.length ? t.searchIn : searchable.map((f) => f.key)
      if (keys.length) nextDomain = andDomains(nextDomain, searchTermToDomain(t.q.trim(), keys))
    }
    patch({
      q: '',
      searchIn: [],
      domain: nextDomain,
      groupBy: t.groupBy,
    })
    setDraftQ('')
    setDraftSearchIn([])
    if (t.pageSize && props.onApplyPageSize) props.onApplyPageSize(t.pageSize)
  }

  function presetDomain(preset: SearchFilterPreset): Domain {
    if (preset.id.startsWith('user:')) {
      const tpl = templates.find((t) => `user:${t.id}` === preset.id)
      if (!tpl) return []
      let d = tpl.domain
      if (tpl.q.trim()) {
        const keys = tpl.searchIn.length ? tpl.searchIn : searchable.map((f) => f.key)
        if (keys.length) d = andDomains(d, searchTermToDomain(tpl.q.trim(), keys))
      }
      return d
    }
    let d = parsePresetDomain(preset.domain)
    const pq = preset.q?.trim() ?? ''
    if (pq) {
      const keys = preset.searchIn?.length ? preset.searchIn : searchable.map((f) => f.key)
      if (keys.length) d = andDomains(d, searchTermToDomain(pq, keys))
    }
    return d
  }

  function togglePreset(preset: SearchFilterPreset) {
    const conjunct = presetDomain(preset)
    if (!conjunct.length) return
    patch({ domain: toggleDomainConjunct(domain, conjunct) })
  }

  function toggleGroup(key: string) {
    const idx = groupBy.indexOf(key)
    if (idx >= 0) patch({ groupBy: groupBy.filter((g) => g !== key) })
    else patch({ groupBy: [...groupBy, key] })
  }

  function applyTemplate(t: SearchTemplate) {
    applyTemplateState(t)
  }

  async function saveTemplate(opts: {
    name: string
    shared: boolean
    isDefault: boolean
    predefined: boolean
  }) {
    await saveSearchTemplate({
      model: props.model,
      name: opts.name,
      q,
      searchIn,
      domain,
      groupBy,
      id: saveSeed.id,
      shared: opts.shared,
      isDefault: opts.isDefault,
      predefined: opts.predefined,
      pageSize: props.pageSize,
    })
    await refreshTemplates()
  }

  const presets = props.presets ?? []
  const activePresets = useMemo(
    () => findActivePresets(presets, templates, domain),
    [presets, templates, domain],
  )
  const activePresetIds = useMemo(() => new Set(activePresets.map((p) => p.id)), [activePresets])
  const customFilterNodes = useMemo(() => {
    return filterOnlyTree.children.filter((child) => {
      const childDomain = filterTreeToDomain(child)
      return !activePresets.some((p) => {
        if (p.id.startsWith('user:')) {
          const tpl = templates.find((t) => `user:${t.id}` === p.id)
          return tpl ? domainsEqual(tpl.domain, childDomain) : false
        }
        return domainsEqual(parsePresetDomain(p.domain), childDomain)
      })
    })
  }, [filterOnlyTree, activePresets, templates])
  const customFilterCount = useMemo(
    () => customFilterNodes.reduce((n, child) => n + countFilterLeaves(child), 0),
    [customFilterNodes],
  )
  const hasPendingQ = q.trim().length > 0
  const hasSearch = hasPendingQ || searchFacets.length > 0
  const hasFilters = customFilterCount > 0 || activePresets.length > 0
  const hasGroups = groupBy.length > 0
  const activeCount =
    customFilterCount + searchFacets.length + (hasPendingQ ? 1 : 0) + groupBy.length + activePresets.length

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
            <Search className="pointer-events-none absolute left-3 top-1/2 z-10 h-4 w-4 -translate-y-1/2 text-[var(--kg-text-muted)]" />
            <Input
              ref={searchInputRef}
              value={draftQ}
              onChange={(e) => {
                const next = e.target.value
                setDraftQ(next)
                if (!next.trim()) setDraftSearchIn([])
                setSuggestOpen(next.trim().length > 0)
              }}
              onFocus={() => {
                if (draftQ.trim()) setSuggestOpen(true)
              }}
              onBlur={() => {
                window.setTimeout(() => setSuggestOpen(false), 150)
              }}
              onKeyDown={handleSearchInputKeyDown}
              placeholder={props.placeholder ?? 'Search…'}
              className="pl-9"
              aria-label="Search"
              role="combobox"
              aria-expanded={showSuggestions}
              aria-controls="k-search-suggestions"
              aria-activedescendant={
                showSuggestions && highlightIndex >= 0
                  ? `k-search-suggestion-${suggestions[highlightIndex]?.id}`
                  : undefined
              }
              aria-autocomplete="list"
              {...{ [KEYMAP_ID_ATTR]: 'k-search' }}
            />
            {showSuggestions ? (
              <div
                id="k-search-suggestions"
                role="listbox"
                className="absolute z-50 mt-1 max-h-64 w-full overflow-auto border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] shadow-lg"
              >
                {suggestions.map((item, i) => (
                  <button
                    key={item.id}
                    id={`k-search-suggestion-${item.id}`}
                    type="button"
                    role="option"
                    aria-selected={i === highlightIndex}
                    className={cn(
                      'flex w-full items-baseline gap-2 px-3 py-2 text-left text-sm',
                      i === highlightIndex
                        ? 'bg-[var(--kg-surface-hover)]'
                        : 'hover:bg-[var(--kg-surface-hover)]',
                    )}
                    onMouseDown={(e) => e.preventDefault()}
                    onMouseEnter={() => setHighlightIndex(i)}
                    onClick={() => applySuggestion(item)}
                  >
                    <span className="min-w-0 flex-1 truncate">{item.label}</span>
                    {item.hint ? (
                      <span className="shrink-0 text-xs text-[var(--kg-text-muted)]">{item.hint}</span>
                    ) : null}
                  </button>
                ))}
              </div>
            ) : null}
          </div>
          <Button type="submit" size="sm" variant="secondary">
            <SearchIcon className="h-4 w-4" />
          </Button>
        </form>

        {/* Filters */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              size="sm"
              variant={hasFilters ? 'secondary' : 'ghost'}
              className={cn(hasFilters && activeToolbarStyles.filter)}
              {...{ [KEYMAP_ID_ATTR]: 'k-search-filter' }}
            >
              <Filter className="h-4 w-4" />
              Filter{hasFilters ? ` (${activePresets.length + customFilterCount})` : ''}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="min-w-[14rem]">
            <DropdownMenuLabel>Filters</DropdownMenuLabel>
            {presets.map((preset) => {
              const active = activePresetIds.has(preset.id)
              return (
                <DropdownMenuItem
                  key={preset.id}
                  className="gap-2"
                  onSelect={(e) => {
                    e.preventDefault()
                    togglePreset(preset)
                  }}
                >
                  <Checkbox checked={active} />
                  <span className="flex-1">{presetLabel(preset)}</span>
                </DropdownMenuItem>
              )
            })}
            {presets.length ? (
              <div className="my-1 border-t border-[var(--kg-border)]" role="separator" />
            ) : null}
            <DropdownMenuItem
              onSelect={(e) => {
                e.preventDefault()
                setCriteriaOpen(true)
              }}
            >
              Custom criteria…
            </DropdownMenuItem>
            {customFilterCount > 0 || activePresets.length > 0 ? (
              <DropdownMenuItem
                className="text-[var(--kg-danger)]"
                onSelect={(e) => {
                  e.preventDefault()
                  clearFilters()
                }}
              >
                Clear filters
              </DropdownMenuItem>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>

        <KFilterCriteriaDialog
          open={criteriaOpen}
          onOpenChange={setCriteriaOpen}
          fields={filterable}
          root={filterOnlyTree}
          onApply={applyFilterTree}
          onSaveCriteria={() => openSaveDialog()}
        />

        <KSaveCriteriaDialog
          open={saveOpen}
          onOpenChange={setSaveOpen}
          initialName={saveSeed.name ?? ''}
          initialShared={saveSeed.shared}
          initialDefault={saveSeed.isDefault}
          initialPredefined={saveSeed.predefined}
          pageSize={props.pageSize}
          onSave={saveTemplate}
        />

        {/* Group by */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              size="sm"
              variant={hasGroups ? 'secondary' : 'ghost'}
              className={cn(hasGroups && activeToolbarStyles.group)}
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
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate">{t.name}</span>
                  <span className="text-xxs text-[var(--kg-text-muted)]">
                    {[t.shared && 'Shared', t.isDefault && 'Default', t.predefined && 'Filter menu']
                      .filter(Boolean)
                      .join(' · ') || 'Private'}
                  </span>
                </span>
                <button
                  type="button"
                  className="text-xs text-[var(--kg-text-muted)]"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    openSaveDialog(t)
                  }}
                >
                  Edit
                </button>
                <button
                  type="button"
                  className="text-xs text-[var(--kg-danger)]"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    void deleteSearchTemplate(props.model, t.id).then(refreshTemplates)
                  }}
                >
                  Delete
                </button>
              </DropdownMenuItem>
            ))}
            {!templates.length ? (
              <div className="px-3 py-2 text-sm text-[var(--kg-text-muted)]">No saved criteria</div>
            ) : null}
            <div className="mt-1 border-t border-[var(--kg-border)] p-2">
              <Button type="button" size="sm" className="w-full" onClick={() => openSaveDialog()}>
                Save current criteria…
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
              setDraftSearchIn([])
            }}
          >
            Clear
          </Button>
        ) : null}
      </div>

      {(hasFilters || hasGroups || hasSearch) ? (
        <div className="flex flex-wrap items-center gap-2 pt-0.5">
          {searchFacets.map((facet) => (
            <TagChip
              key={facet.id}
              kind="search"
              label="Search"
              icon={<Search className="h-3.5 w-3.5 shrink-0 opacity-80" />}
              onClick={() => editSearchFacet(facet)}
              onRemove={() => removeSearchFacet(facet)}
            >
              {facet.searchIn.length === 1 ? (
                <>
                  <span className="text-[var(--kg-text-muted)]">in</span>{' '}
                  {fieldLabel(props.fields, facet.searchIn[0])}
                  <span className="text-[var(--kg-text-muted)]">:</span>{' '}
                  <span className="font-medium">{facet.q}</span>
                </>
              ) : facet.searchIn.length > 1 &&
                facet.searchIn.length < searchable.length ? (
                <>
                  <span className="text-[var(--kg-text-muted)]">in</span>{' '}
                  {facet.searchIn.map((k) => fieldLabel(props.fields, k)).join(', ')}
                  <span className="text-[var(--kg-text-muted)]">:</span>{' '}
                  <span className="font-medium">{facet.q}</span>
                </>
              ) : (
                <>
                  <span className="text-[var(--kg-text-muted)]">All fields</span>
                  <span className="text-[var(--kg-text-muted)]">:</span>{' '}
                  <span className="font-medium">{facet.q}</span>
                </>
              )}
            </TagChip>
          ))}
          {hasPendingQ ? (
            <TagChip
              kind="search"
              label="Search"
              icon={<Search className="h-3.5 w-3.5 shrink-0 opacity-80" />}
              onClick={editPendingQ}
              onRemove={() => patch({ q: '', searchIn: [] })}
            >
              {searchIn.length ? (
                <>
                  <span className="text-[var(--kg-text-muted)]">in</span>{' '}
                  {searchIn.map((k) => fieldLabel(props.fields, k)).join(', ')}
                  <span className="text-[var(--kg-text-muted)]">:</span>{' '}
                  <span className="font-medium">{q.trim()}</span>
                </>
              ) : (
                <>
                  <span className="text-[var(--kg-text-muted)]">All fields</span>
                  <span className="text-[var(--kg-text-muted)]">:</span>{' '}
                  <span className="font-medium">{q.trim()}</span>
                </>
              )}
            </TagChip>
          ) : null}
          {activePresets.map((preset) => (
            <TagChip
              key={preset.id}
              kind="preset"
              label="Filter"
              onRemove={() => togglePreset(preset)}
            >
              {presetLabel(preset)}
            </TagChip>
          ))}
          {customFilterNodes.map((child) => (
            <FilterChipTree
              key={child.id}
              node={child}
              fields={props.fields}
              onRemove={removeFilterNode}
            />
          ))}
          {hasGroups ? (
            <GroupNestChip groupBy={groupBy} fields={props.fields} onRemove={toggleGroup} />
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
