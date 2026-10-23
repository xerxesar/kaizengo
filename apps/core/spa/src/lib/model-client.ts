export type ModelColumn = {
  key: string
  label: string
  width?: string
  align?: string
}

export type ModelField = {
  key: string
  label: string
  type?: string
  required?: boolean
  relation?: string
  inverse?: string
  values?: string[]
}

export type ModelFilterPreset = {
  id: string
  label?: string | null
  labelKey?: string | null
  domain?: string | null
  q?: string | null
  searchIn?: string[] | null
  groupBy?: string[] | null
}

export type ModelChartPreset = {
  id: string
  label?: string | null
  labelKey?: string | null
  type: string
  types?: string[] | null
  xField: string
  yField?: string | null
  seriesField?: string | null
  measure: string
}

export type ModelView = {
  name: string
  kind: string
  model: string
  columns?: ModelColumn[]
  fields?: ModelField[]
  filterPresets?: ModelFilterPreset[]
  chartPresets?: ModelChartPreset[]
  listQuery?: string | null
  getQuery?: string | null
  createCommand?: string | null
  updateCommand?: string | null
  deleteCommand?: string | null
}

export type ModelRecord = Record<string, unknown> & { id: string }

export {
  formatNamespace,
  isNamespaced,
  parseNamespace,
  resolveCqrsRef,
  type CqrsRef,
  type Namespace,
} from './namespace'

function pascal(s: string): string {
  return s
    .split('_')
    .map((p) => (p ? p[0].toUpperCase() + p.slice(1) : ''))
    .join('')
}

function camel(s: string): string {
  const p = pascal(s)
  return p ? p[0].toLowerCase() + p.slice(1) : ''
}

function listQueryName(app: string, model: string): string {
  return `${camel(app)}${pascal(model)}s`
}

function deleteMutationName(app: string, model: string): string {
  return `delete${pascal(app)}${pascal(model)}`
}

function createMutationName(app: string, model: string): string {
  return `create${pascal(app)}${pascal(model)}`
}

function updateMutationName(app: string, model: string): string {
  return `update${pascal(app)}${pascal(model)}`
}

function getQueryName(app: string, model: string): string {
  return `${camel(app)}${pascal(model)}`
}

function viewsQueryName(app: string): string {
  return `${camel(app)}Views`
}

function viewSlotsQueryName(app: string): string {
  return `${camel(app)}ViewSlots`
}

function pingQueryName(app: string): string {
  return `${camel(app)}Ping`
}

export type ViewSlot = {
  slot: string
  component: string
  module?: string
  sourceApp?: string
}

async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  return body.data as T
}

export type SecuredResource = {
  app: string
  kind: string
  name: string
  resource: string
  label: string
  description?: string | null
  actions: string[]
  fields: string[]
  surface?: string | null
}

export async function fetchResources(): Promise<SecuredResource[]> {
  const data = await gql<{ resources: SecuredResource[] }>(`query {
    resources { app kind name resource label description actions fields surface }
  }`)
  return data.resources ?? []
}

export async function fetchACLActions(): Promise<string[]> {
  const data = await gql<{ aclActions: string[] }>(`query { aclActions }`)
  return data.aclActions ?? []
}

const viewsCache = new Map<string, Promise<ModelView[]>>()

export async function fetchModelViews(app: string): Promise<ModelView[]> {
  let pending = viewsCache.get(app)
  if (!pending) {
    pending = (async () => {
      const field = viewsQueryName(app)
      const data = await gql<Record<string, ModelView[]>>(`query {
        ${field} {
          name kind model
          columns { key label width align }
          fields { key label type required relation inverse values }
          filterPresets { id label labelKey domain q searchIn groupBy }
          chartPresets { id label labelKey type types xField yField seriesField measure }
          listQuery getQuery createCommand updateCommand deleteCommand
        }
      }`)
      return data[field] ?? []
    })()
    viewsCache.set(app, pending)
  }
  try {
    return await pending
  } catch (e) {
    viewsCache.delete(app)
    throw e
  }
}

export function listViewForModel(views: ModelView[], model: string): ModelView | null {
  return views.find((v) => v.kind === 'list' && v.model === model) ?? null
}

export function formViewForModel(views: ModelView[], model: string): ModelView | null {
  return views.find((v) => v.kind === 'form' && v.model === model) ?? null
}

/** Find list view bound to a GraphQL list query field (`hellospecGreetings`). */
export function listViewForQuery(views: ModelView[], listQuery: string): ModelView | null {
  const field = listQuery.trim()
  if (!field) return null
  return views.find((v) => v.kind === 'list' && v.listQuery?.trim() === field) ?? null
}

/** Find form view bound to a create/update/delete GraphQL command field. */
export function formViewForCommand(views: ModelView[], command: string): ModelView | null {
  const field = command.trim()
  if (!field) return null
  return (
    views.find(
      (v) =>
        v.kind === 'form' &&
        (v.createCommand?.trim() === field ||
          v.updateCommand?.trim() === field ||
          v.deleteCommand?.trim() === field),
    ) ?? null
  )
}

/** Resolve CQRS GraphQL field from `{app}Views`, then fall back to legacy CRUD naming. */
async function resolveModelBinding(
  app: string,
  model: string,
  kind: 'listQuery' | 'getQuery' | 'createCommand' | 'updateCommand' | 'deleteCommand',
  override?: string,
): Promise<string> {
  const explicit = override?.trim()
  if (explicit) return explicit
  try {
    const views = await fetchModelViews(app)
    const view =
      kind === 'listQuery' || kind === 'getQuery'
        ? listViewForModel(views, model) ?? formViewForModel(views, model)
        : formViewForModel(views, model) ?? listViewForModel(views, model)
    const fromView = view?.[kind]?.trim()
    if (fromView) return fromView
  } catch {
    /* fall through to legacy names */
  }
  switch (kind) {
    case 'listQuery':
      return listQueryName(app, model)
    case 'getQuery':
      return getQueryName(app, model)
    case 'createCommand':
      return createMutationName(app, model)
    case 'updateCommand':
      return updateMutationName(app, model)
    case 'deleteCommand':
      return deleteMutationName(app, model)
  }
}

export async function fetchViewSlots(app: string, view: string): Promise<ViewSlot[]> {
  const field = viewSlotsQueryName(app)
  const data = await gql<Record<string, ViewSlot[]>>(`query($view: String!) {
    ${field}(view: $view) { slot component module sourceApp }
  }`, { view })
  return data[field] ?? []
}

export async function listModelRecords(
  app: string,
  model: string,
  fields: string[],
  queryField?: string,
): Promise<ModelRecord[]> {
  const page = await listModelRecordsPage(app, model, fields, undefined, queryField)
  return page.items
}

export type ModelListPage = {
  items: ModelRecord[]
  total: number
  page: number
  pageSize: number
}

export type ModelListPageOpts = {
  page: number
  pageSize: number
  domain?: string
  q?: string
  searchIn?: string[]
  groupBy?: string[]
}

/** Paginated + filtered list + companion `{listField}Count`. Omit opts for the full list. */
export async function listModelRecordsPage(
  app: string,
  model: string,
  fields: string[],
  opts?: ModelListPageOpts,
  queryField?: string,
): Promise<ModelListPage> {
  const field = await resolveModelBinding(app, model, 'listQuery', queryField)
  const unique = [...new Set(['id', ...fields])]
  const selection = unique.join(' ')
  const page = Math.max(1, opts?.page ?? 1)
  const pageSize = opts?.pageSize ?? 0
  const paginate = pageSize > 0
  const domain = opts?.domain?.trim() || ''
  const q = opts?.q?.trim() || ''
  const searchIn = opts?.searchIn?.filter(Boolean) ?? []
  const groupBy = opts?.groupBy?.filter(Boolean) ?? []
  const hasFilter = Boolean(domain || q || searchIn.length || groupBy.length)

  if (!paginate && !hasFilter) {
    const data = await gql<Record<string, ModelRecord[]>>(`query {
      ${field} { ${selection} }
    }`)
    const items = data[field] ?? []
    return { items, total: items.length, page: 1, pageSize: items.length }
  }

  const countField = `${field}Count`
  const varDefs = ['$page: Int!', '$pageSize: Int!']
  const listArgs = ['page: $page', 'pageSize: $pageSize']
  const countArgs: string[] = []
  const variables: Record<string, unknown> = { page, pageSize: paginate ? pageSize : 1 }

  if (domain) {
    varDefs.push('$domain: String')
    listArgs.push('domain: $domain')
    countArgs.push('domain: $domain')
    variables.domain = domain
  }
  if (q) {
    varDefs.push('$q: String')
    listArgs.push('q: $q')
    countArgs.push('q: $q')
    variables.q = q
  }
  if (searchIn.length) {
    varDefs.push('$searchIn: [String!]')
    listArgs.push('searchIn: $searchIn')
    countArgs.push('searchIn: $searchIn')
    variables.searchIn = searchIn
  }
  if (groupBy.length) {
    varDefs.push('$groupBy: [String!]')
    listArgs.push('groupBy: $groupBy')
    variables.groupBy = groupBy
  }

  // When not paginating but filtering, fetch a large page.
  if (!paginate) {
    variables.page = 1
    variables.pageSize = 100000
  }

  const countCall = countArgs.length
    ? `${countField}(${countArgs.join(', ')})`
    : `${countField}`

  const data = await gql<Record<string, ModelRecord[] | number>>(
    `query(${varDefs.join(', ')}) {
      ${field}(${listArgs.join(', ')}) { ${selection} }
      ${countCall}
    }`,
    variables,
  )
  const items = (data[field] as ModelRecord[] | undefined) ?? []
  const total = typeof data[countField] === 'number' ? (data[countField] as number) : items.length
  return {
    items,
    total,
    page: paginate ? page : 1,
    pageSize: paginate ? pageSize : items.length,
  }
}

export type GroupBucket = { values: string[]; count: number }

export async function listModelGroups(
  app: string,
  model: string,
  opts: {
    domain?: string
    q?: string
    searchIn?: string[]
    groupBy: string[]
  },
  queryField?: string,
): Promise<GroupBucket[]> {
  if (!opts.groupBy.length) return []
  const field = await resolveModelBinding(app, model, 'listQuery', queryField)
  const groupsField = `${field}Groups`
  const varDefs = ['$groupBy: [String!]!']
  const args = ['groupBy: $groupBy']
  const variables: Record<string, unknown> = { groupBy: opts.groupBy }
  if (opts.domain?.trim()) {
    varDefs.push('$domain: String')
    args.push('domain: $domain')
    variables.domain = opts.domain.trim()
  }
  if (opts.q?.trim()) {
    varDefs.push('$q: String')
    args.push('q: $q')
    variables.q = opts.q.trim()
  }
  if (opts.searchIn?.length) {
    varDefs.push('$searchIn: [String!]')
    args.push('searchIn: $searchIn')
    variables.searchIn = opts.searchIn
  }
  const data = await gql<Record<string, GroupBucket[]>>(
    `query(${varDefs.join(', ')}) {
      ${groupsField}(${args.join(', ')}) { values count }
    }`,
    variables,
  )
  return data[groupsField] ?? []
}

export async function deleteModelRecord(
  app: string,
  model: string,
  id: string,
  mutationField?: string,
): Promise<void> {
  const mutation = await resolveModelBinding(app, model, 'deleteCommand', mutationField)
  await gql(`mutation($id: ID!) { ${mutation}(id: $id) }`, { id })
}

function gqlInputType(type: string | undefined, required: boolean): string {
  let base = 'String'
  switch ((type ?? 'string').toLowerCase()) {
    case 'int':
      base = 'Int'
      break
    case 'number':
    case 'float':
    case 'decimal':
      base = 'Float'
      break
    case 'bool':
    case 'boolean':
      base = 'Boolean'
      break
    case 'many2one':
    case 'fk':
    case 'relation':
      base = 'ID'
      break
    case 'many2many':
    case 'one2many':
      base = '[ID!]'
      break
    default:
      base = 'String'
  }
  return required ? `${base}!` : base
}

function coerceFieldValue(type: string | undefined, value: unknown): unknown {
  switch ((type ?? 'string').toLowerCase()) {
    case 'int':
      return typeof value === 'number' ? value : parseInt(String(value), 10)
    case 'number':
    case 'float':
    case 'decimal':
      return typeof value === 'number' ? value : parseFloat(String(value))
    case 'bool':
    case 'boolean':
      return Boolean(value)
    case 'many2many':
    case 'one2many':
      if (Array.isArray(value)) return value.map(String)
      return String(value ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    default:
      return String(value ?? '')
  }
}

function fieldHasValue(field: ModelField, value: unknown): boolean {
  if (value == null) return false
  switch ((field.type ?? 'string').toLowerCase()) {
    case 'bool':
    case 'boolean':
      return true
    case 'int':
    case 'number':
    case 'float':
    case 'decimal':
      return value !== '' && !Number.isNaN(Number(value))
    case 'many2many':
    case 'one2many':
      return Array.isArray(value) ? value.length > 0 : String(value).trim() !== ''
    default:
      return String(value).trim() !== ''
  }
}

function buildWriteMutation(
  kind: 'create' | 'update',
  app: string,
  model: string,
  fields: ModelField[],
  values: Record<string, unknown>,
  id?: string,
  mutationField?: string,
): { query: string; variables: Record<string, unknown>; mutation: string } {
  const mutation =
    mutationField?.trim() ||
    (kind === 'create' ? createMutationName(app, model) : updateMutationName(app, model))
  const used = fields.filter((field) => {
    if (kind === 'create') {
      return field.required || fieldHasValue(field, values[field.key])
    }
    return fieldHasValue(field, values[field.key])
  })

  const varDefs: string[] = []
  const args: string[] = []
  const variables: Record<string, unknown> = {}

  if (kind === 'update') {
    varDefs.push('$id: ID!')
    args.push('id: $id')
    variables.id = id
  }

  for (const field of used) {
    const required = kind === 'create' && !!field.required
    varDefs.push(`$${field.key}: ${gqlInputType(field.type, required)}`)
    args.push(`${field.key}: $${field.key}`)
    variables[field.key] = coerceFieldValue(field.type, values[field.key])
  }

  const selection = [...new Set(['id', ...fields.map((field) => field.key)])].join(' ')
  const query = `mutation(${varDefs.join(', ')}) {
    ${mutation}(${args.join(', ')}) { ${selection} }
  }`
  return { query, variables, mutation }
}

export async function getModelRecord(
  app: string,
  model: string,
  id: string,
  fields: string[],
  queryField?: string,
): Promise<ModelRecord> {
  const field = await resolveModelBinding(app, model, 'getQuery', queryField)
  const unique = [...new Set(['id', ...fields])]
  const selection = unique.join(' ')
  const data = await gql<Record<string, ModelRecord>>(`query($id: ID!) {
    ${field}(id: $id) { ${selection} }
  }`, { id })
  const record = data[field]
  if (!record) throw new Error(`record not found: ${model}/${id}`)
  return record
}

export async function createModelRecord(
  app: string,
  model: string,
  fields: ModelField[],
  values: Record<string, unknown>,
  mutationField?: string,
): Promise<ModelRecord> {
  const resolved = await resolveModelBinding(app, model, 'createCommand', mutationField)
  const { query, variables, mutation } = buildWriteMutation(
    'create',
    app,
    model,
    fields,
    values,
    undefined,
    resolved,
  )
  const data = await gql<Record<string, ModelRecord>>(query, variables)
  const record = data[mutation]
  if (!record) throw new Error(`create failed for ${model}`)
  return record
}

export async function updateModelRecord(
  app: string,
  model: string,
  id: string,
  fields: ModelField[],
  values: Record<string, unknown>,
  mutationField?: string,
): Promise<ModelRecord> {
  const resolved = await resolveModelBinding(app, model, 'updateCommand', mutationField)
  const { query, variables, mutation } = buildWriteMutation(
    'update',
    app,
    model,
    fields,
    values,
    id,
    resolved,
  )
  const data = await gql<Record<string, ModelRecord>>(query, variables)
  const record = data[mutation]
  if (!record) throw new Error(`update failed for ${model}/${id}`)
  return record
}

/** Engine `{app}Ping` health query (`hellospecPing` → `"hellospec ok"`). */
export async function fetchAppPing(app: string): Promise<string> {
  const field = pingQueryName(app)
  const data = await gql<Record<string, string>>(`query { ${field} }`)
  return data[field] ?? ''
}
