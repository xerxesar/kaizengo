import type { Domain, DomainLeaf, DomainNode, DomainOp } from './types'

export type FilterLeaf = {
  kind: 'leaf'
  id: string
  field: string
  op: DomainOp | string
  value?: unknown
}

export type FilterGroup = {
  kind: 'group'
  id: string
  junction: 'and' | 'or'
  children: FilterNode[]
}

export type FilterNode = FilterLeaf | FilterGroup

let filterIdSeq = 0

export function newFilterId(): string {
  filterIdSeq += 1
  return `f${filterIdSeq}`
}

export function newFilterLeaf(
  field: string,
  op: DomainOp | string = 'ilike',
  value?: unknown,
): FilterLeaf {
  return { kind: 'leaf', id: newFilterId(), field, op, value }
}

export function newFilterGroup(
  junction: 'and' | 'or' = 'and',
  children: FilterNode[] = [],
): FilterGroup {
  return { kind: 'group', id: newFilterId(), junction, children }
}

export function emptyFilterTree(): FilterGroup {
  return newFilterGroup('and', [])
}

function isLeafTuple(item: unknown): item is DomainLeaf {
  return (
    Array.isArray(item) &&
    item.length >= 2 &&
    typeof item[0] === 'string' &&
    item[0] !== '&' &&
    item[0] !== '|' &&
    item[0] !== '!'
  )
}

function leafFromTuple(tuple: DomainLeaf): FilterLeaf {
  const [field, op, value] = tuple
  if (op === 'is set' || op === 'is not set') {
    return { kind: 'leaf', id: newFilterId(), field, op }
  }
  return { kind: 'leaf', id: newFilterId(), field, op, value }
}

function parseDomainOne(arr: Domain, i: number): { node: FilterNode; next: number } {
  if (i >= arr.length) {
    throw new Error('unexpected end of domain')
  }
  const item = arr[i]
  if (item === '|' || item === '&') {
    const junction = item === '|' ? 'or' : 'and'
    const left = parseDomainOne(arr, i + 1)
    const right = parseDomainOne(arr, left.next)
    return {
      node: newFilterGroup(junction, [left.node, right.node]),
      next: right.next,
    }
  }
  if (item === '!') {
    const child = parseDomainOne(arr, i + 1)
    return child
  }
  if (isLeafTuple(item)) {
    return { node: leafFromTuple(item), next: i + 1 }
  }
  throw new Error('invalid domain node')
}

function normalizeGroup(node: FilterNode): FilterNode {
  if (node.kind === 'leaf') return node
  const children = node.children.map(normalizeGroup).flatMap((child) => {
    if (child.kind === 'group' && child.junction === node.junction) return child.children
    return [child]
  })
  if (children.length === 1) return children[0]
  return { ...node, children }
}

/** Parse a polish / flat domain into an editable filter tree. */
export function domainToFilterTree(domain: Domain): FilterGroup {
  if (!domain.length) return emptyFilterTree()

  const hasOps = domain.some((n) => n === '|' || n === '&' || n === '!')
  if (!hasOps) {
    const leaves = domain.filter(isLeafTuple).map(leafFromTuple)
    return newFilterGroup('and', leaves)
  }

  const { node } = parseDomainOne(domain, 0)
  const normalized = normalizeGroup(node)
  if (normalized.kind === 'group') return normalized
  return newFilterGroup('and', [normalized])
}

function leafToDomain(leaf: FilterLeaf): Domain {
  if (leaf.op === 'is set' || leaf.op === 'is not set') {
    return [[leaf.field, leaf.op]]
  }
  return [[leaf.field, leaf.op, leaf.value]]
}

function foldGroupDomains(domains: Domain[], junction: 'and' | 'or'): Domain {
  if (!domains.length) return []
  if (domains.length === 1) return domains[0]
  const op: DomainNode = junction === 'or' ? '|' : '&'
  let acc = domains[domains.length - 1]
  for (let i = domains.length - 2; i >= 0; i--) {
    acc = [op, ...domains[i], ...acc]
  }
  return acc
}

/** Serialize a filter tree to a backend-compatible polish domain. */
export function filterTreeToDomain(root: FilterNode | null | undefined): Domain {
  if (!root) return []
  if (root.kind === 'leaf') return leafToDomain(root)
  if (!root.children.length) return []
  const childDomains = root.children.map((child) => filterTreeToDomain(child))
  return foldGroupDomains(childDomains, root.junction)
}

export function filterTreeIsEmpty(root: FilterNode | null | undefined): boolean {
  if (!root) return true
  if (root.kind === 'leaf') return false
  return root.children.length === 0 || root.children.every(filterTreeIsEmpty)
}

export function countFilterLeaves(root: FilterNode | null | undefined): number {
  if (!root) return 0
  if (root.kind === 'leaf') return 1
  return root.children.reduce((n, c) => n + countFilterLeaves(c), 0)
}

export function cloneFilterTree(root: FilterNode): FilterNode {
  if (root.kind === 'leaf') {
    return { ...root, id: newFilterId() }
  }
  return {
    ...root,
    id: newFilterId(),
    children: root.children.map(cloneFilterTree),
  }
}

export function updateFilterNode(
  root: FilterGroup,
  id: string,
  updater: (node: FilterNode) => FilterNode | null,
): FilterGroup {
  function walk(node: FilterNode): FilterNode | null {
    if (node.id === id) return updater(node)
    if (node.kind === 'leaf') return node
    const children = node.children
      .map(walk)
      .filter((child): child is FilterNode => child != null)
    return { ...node, children }
  }
  const next = walk(root)
  if (next == null) return emptyFilterTree()
  return next.kind === 'group' ? next : newFilterGroup('and', [next])
}

export function formatFilterNode(
  node: FilterNode,
  fields: { key: string; label: string }[],
): string {
  const name = (key: string) => fields.find((f) => f.key === key)?.label ?? key
  if (node.kind === 'leaf') {
    const label = name(node.field)
    if (node.op === 'is set' || node.op === 'is not set') return `${label} ${node.op}`
    return `${label} ${node.op} ${node.value == null ? '' : String(node.value)}`
  }
  const inner = node.children.map((c) => formatFilterNode(c, fields)).join(` ${node.junction.toUpperCase()} `)
  return node.children.length > 1 ? `(${inner})` : inner
}

/** OR two polish domains together. */
export function orDomains(a: Domain, b: Domain): Domain {
  if (!a.length) return [...b]
  if (!b.length) return [...a]
  return foldGroupDomains([a, b], 'or')
}

/** AND two polish domains together. */
export function andDomains(a: Domain, b: Domain): Domain {
  if (!a.length) return [...b]
  if (!b.length) return [...a]
  return foldGroupDomains([a, b], 'and')
}

export function domainsEqual(a: Domain, b: Domain): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

/** Top-level siblings of a domain (AND or OR root children, else the domain itself). */
function topLevelSiblings(domain: Domain): FilterNode[] {
  if (!domain.length) return []
  const tree = domainToFilterTree(domain)
  if (filterTreeIsEmpty(tree)) return []
  if (tree.children.length > 0) return [...tree.children]
  return [tree]
}

/** True when `needle` is the whole domain or a top-level sibling of it. */
export function domainContains(haystack: Domain, needle: Domain): boolean {
  if (!needle.length) return false
  if (domainsEqual(haystack, needle)) return true
  return topLevelSiblings(haystack).some((child) =>
    domainsEqual(filterTreeToDomain(child), needle),
  )
}

/** Toggle a filter domain: add with OR if missing, remove if present. */
export function toggleDomainConjunct(current: Domain, conjunct: Domain): Domain {
  if (!conjunct.length) return current
  if (!current.length) return [...conjunct]

  if (domainsEqual(current, conjunct)) return []

  const siblings = topLevelSiblings(current)
  const idx = siblings.findIndex((child) =>
    domainsEqual(filterTreeToDomain(child), conjunct),
  )

  let next: FilterNode[]
  if (idx >= 0) {
    next = siblings.filter((_, i) => i !== idx)
  } else {
    const added = domainToFilterTree(conjunct)
    const node =
      added.children.length === 1 ? added.children[0] : added
    next = [...siblings, node]
  }

  if (!next.length) return []
  if (next.length === 1) return filterTreeToDomain(next[0])
  return filterTreeToDomain(newFilterGroup('or', next))
}

function stripLikeValue(value: unknown): string {
  return String(value ?? '').replace(/^%|%$/g, '').trim()
}

function isSearchLeaf(node: FilterNode): node is FilterLeaf {
  return (
    node.kind === 'leaf' &&
    (node.op === 'ilike' || node.op === 'like') &&
    stripLikeValue(node.value).length > 0
  )
}

/** A search facet is a single ilike leaf, or an OR of ilike leaves sharing one value. */
export type SearchFacet = {
  id: string
  q: string
  searchIn: string[]
  node: FilterNode
}

export function isSearchFacetNode(node: FilterNode): boolean {
  if (isSearchLeaf(node)) return true
  if (node.kind !== 'group' || node.junction !== 'or' || !node.children.length) return false
  if (!node.children.every(isSearchLeaf)) return false
  const first = stripLikeValue(node.children[0].value)
  return node.children.every((c) => c.kind === 'leaf' && stripLikeValue(c.value) === first)
}

export function searchFacetFromNode(node: FilterNode): SearchFacet | null {
  if (!isSearchFacetNode(node)) return null
  if (node.kind === 'leaf') {
    return {
      id: node.id,
      q: stripLikeValue(node.value),
      searchIn: [node.field],
      node,
    }
  }
  const leaves = node.children.filter(isSearchLeaf)
  return {
    id: node.id,
    q: stripLikeValue(leaves[0]?.value),
    searchIn: leaves.map((l) => l.field),
    node,
  }
}

/** Split AND-root into search facets vs remaining filter criteria. */
export function peelSearchFacets(root: FilterGroup): {
  facets: SearchFacet[]
  remaining: FilterGroup
} {
  if (filterTreeIsEmpty(root)) return { facets: [], remaining: emptyFilterTree() }

  if (isSearchFacetNode(root)) {
    const facet = searchFacetFromNode(root)
    return { facets: facet ? [facet] : [], remaining: emptyFilterTree() }
  }

  if (root.junction === 'and') {
    const facets: SearchFacet[] = []
    const rest: FilterNode[] = []
    for (const child of root.children) {
      const facet = searchFacetFromNode(child)
      if (facet) facets.push(facet)
      else rest.push(child)
    }
    return { facets, remaining: newFilterGroup('and', rest) }
  }

  const single = root.children.length === 1 ? searchFacetFromNode(root.children[0]) : null
  if (single) return { facets: [single], remaining: emptyFilterTree() }

  return { facets: [], remaining: root }
}

/** Build a domain for one search submission (field-scoped or OR across fields). */
export function searchTermToDomain(q: string, searchIn: string[]): Domain {
  const term = q.trim()
  if (!term || !searchIn.length) return []
  const pattern = term.includes('%') ? term : `%${term}%`
  const leaves = searchIn.map((field) => newFilterLeaf(field, 'ilike', pattern))
  if (leaves.length === 1) return filterTreeToDomain(leaves[0])
  return filterTreeToDomain(newFilterGroup('or', leaves))
}
