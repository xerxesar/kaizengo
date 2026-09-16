/** Odoo-style domain types for KSearch (server-backed). */

export type DomainOp =
  | '='
  | '!='
  | '>'
  | '<'
  | '>='
  | '<='
  | 'ilike'
  | 'like'
  | 'in'
  | 'not in'
  | 'is set'
  | 'is not set'

export type DomainLeaf = [field: string, op: DomainOp | string, value?: unknown]

/** Polish domain: leaves, or '|', '&', '!' operators. */
export type DomainNode = DomainLeaf | '|' | '&' | '!'

export type Domain = DomainNode[]

export type SearchField = {
  key: string
  label: string
  type?: string
  values?: string[]
  relation?: string
}

/** Spec-defined filter preset from `{app}Views.filterPresets`. */
export type SearchFilterPreset = {
  id: string
  label?: string
  labelKey?: string
  /** JSON-encoded polish domain. */
  domain?: string | null
  q?: string | null
  searchIn?: string[] | null
  groupBy?: string[] | null
}

export type SearchTemplate = {
  id: string
  name: string
  model: string
  q: string
  searchIn: string[]
  domain: Domain
  groupBy: string[]
  /** Nested group levels use the same groupBy array order. */
  shared?: boolean
  /** Auto-apply when opening this list (includes pageSize). */
  isDefault?: boolean
  /** Show in the Filters menu alongside appspec presets. */
  predefined?: boolean
  pageSize?: number
  ownerId?: string
  createdAt: string
  updatedAt: string
}

export type SearchState = {
  q: string
  searchIn: string[]
  domain: Domain
  groupBy: string[]
  /** Combinator used when adding sequential leaves in the UI. */
  junction: 'and' | 'or'
}

export const EMPTY_SEARCH: SearchState = {
  q: '',
  searchIn: [],
  domain: [],
  groupBy: [],
  junction: 'and',
}

export function opsForFieldType(type?: string): DomainOp[] {
  switch ((type ?? 'string').toLowerCase()) {
    case 'int':
    case 'number':
    case 'float':
    case 'decimal':
    case 'date':
    case 'datetime':
      return ['=', '!=', '>', '<', '>=', '<=', 'is set', 'is not set']
    case 'bool':
    case 'boolean':
      return ['=', '!=', 'is set', 'is not set']
    case 'enum':
    case 'many2one':
    case 'fk':
      return ['=', '!=', 'in', 'not in', 'is set', 'is not set']
    default:
      return ['ilike', '=', '!=', 'is set', 'is not set']
  }
}

export function encodeDomain(domain: Domain): string {
  if (!domain.length) return ''
  return JSON.stringify(domain)
}

export function decodeDomain(raw: string | null | undefined): Domain {
  if (!raw?.trim()) return []
  try {
    const parsed = JSON.parse(raw) as unknown
    return Array.isArray(parsed) ? (parsed as Domain) : []
  } catch {
    return []
  }
}

/** Build a polish domain from leaves using a junction between them. */
export function domainFromLeaves(leaves: DomainLeaf[], junction: 'and' | 'or' = 'and'): Domain {
  if (!leaves.length) return []
  if (leaves.length === 1) return [leaves[0]]
  const op: '|' | '&' = junction === 'or' ? '|' : '&'
  const out: Domain = []
  // Nested polish: & a & b c  →  [ '&', a, [ '&', b, c ] ] simplified as left-assoc chain
  let acc: DomainNode = leaves[leaves.length - 1]
  for (let i = leaves.length - 2; i >= 0; i--) {
    // Represent binary as prefix op + two nodes; flatten into array form Odoo expects:
    // ['&', leaf0, ['&', leaf1, leaf2]] can't nest arrays as DomainNode easily.
    // Use flat polish: ['&', leaf0, '&', leaf1, leaf2] — our parser takes one op then two nodes.
    // So left-assoc: start from right.
    acc = [op, leaves[i], acc] as unknown as DomainNode
  }
  // Flatten nested synthetic structure into polish list
  return flattenPolish(acc)
}

function flattenPolish(node: DomainNode | Domain): Domain {
  if (Array.isArray(node) && node.length === 3 && (node[0] === '&' || node[0] === '|')) {
    const op = node[0] as '&' | '|'
    const left = flattenPolish(node[1] as DomainNode)
    const right = flattenPolish(node[2] as DomainNode)
    // If left/right are single leaves, emit [op, leftLeaf, right...]
    return [op, ...expand(left), ...expand(right)]
  }
  if (Array.isArray(node) && typeof node[0] === 'string' && node[0] !== '&' && node[0] !== '|' && node[0] !== '!') {
    return [node as DomainLeaf]
  }
  if (Array.isArray(node)) return node as Domain
  return [node]
}

function expand(d: Domain): Domain {
  // When expanding a multi-node domain into an operand slot, wrap as nested domain array
  // Our parser expects parseOne to consume one node — a leaf array is one node.
  // For a sub-domain with multiple polish tokens, we need them as a nested list... 
  // Simpler approach: only support flat AND/OR lists for UI leaves.
  if (d.length === 1) return d
  // Nested domain as a single JSON array node isn't valid in our DomainNode union for polish.
  // Use chain of binary ops in flat polish: | a | b c  means |(| a b) c? 
  // Parser: | takes two parseOne. So ['|', a, '|', b, c] works: | a (| b c).
  return d
}

/** Simpler: UI stores leaves + junction; serialize to flat polish chain. */
export function serializeLeaves(leaves: DomainLeaf[], junction: 'and' | 'or'): Domain {
  if (!leaves.length) return []
  if (leaves.length === 1) return [...leaves]
  const op: '|' | '&' = junction === 'or' ? '|' : '&'
  const out: Domain = []
  // ['&', l0, '&', l1, l2] — right-assoc chain matching parseOne
  for (let i = 0; i < leaves.length - 1; i++) {
    out.push(op)
    out.push(leaves[i])
  }
  out.push(leaves[leaves.length - 1])
  return out
}

export function extractLeaves(domain: Domain): { leaves: DomainLeaf[]; junction: 'and' | 'or' } {
  if (!domain.length) return { leaves: [], junction: 'and' }
  let junction: 'and' | 'or' = 'and'
  const leaves: DomainLeaf[] = []
  for (const node of domain) {
    if (node === '|') {
      junction = 'or'
      continue
    }
    if (node === '&') {
      junction = 'and'
      continue
    }
    if (node === '!') continue
    if (Array.isArray(node) && typeof node[0] === 'string') {
      leaves.push(node as DomainLeaf)
    }
  }
  return { leaves, junction }
}

export function domainIsEmpty(domain: Domain): boolean {
  return extractLeaves(domain).leaves.length === 0
}
