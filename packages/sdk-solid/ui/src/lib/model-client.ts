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
}

export type ModelView = {
  name: string
  kind: string
  model: string
  columns?: ModelColumn[]
  fields?: ModelField[]
}

export type ModelRecord = Record<string, unknown> & { id: string }

export { formatNamespace, isNamespaced, parseNamespace, type Namespace } from './namespace'

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
  surface?: string | null
}

export async function fetchResources(): Promise<SecuredResource[]> {
  const data = await gql<{ resources: SecuredResource[] }>(`query {
    resources { app kind name resource label description actions surface }
  }`)
  return data.resources ?? []
}

export async function fetchACLActions(): Promise<string[]> {
  const data = await gql<{ aclActions: string[] }>(`query { aclActions }`)
  return data.aclActions ?? []
}

export async function fetchModelViews(app: string): Promise<ModelView[]> {
  const field = viewsQueryName(app)
  const data = await gql<Record<string, ModelView[]>>(`query {
    ${field} {
      name kind model
      columns { key label width align }
      fields { key label type required relation inverse }
    }
  }`)
  return data[field] ?? []
}

export function listViewForModel(views: ModelView[], model: string): ModelView | null {
  return views.find((v) => v.kind === 'list' && v.model === model) ?? null
}

export function formViewForModel(views: ModelView[], model: string): ModelView | null {
  return views.find((v) => v.kind === 'form' && v.model === model) ?? null
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
): Promise<ModelRecord[]> {
  const queryField = listQueryName(app, model)
  const unique = [...new Set(['id', ...fields])]
  const selection = unique.join(' ')
  const data = await gql<Record<string, ModelRecord[]>>(`query {
    ${queryField} { ${selection} }
  }`)
  return data[queryField] ?? []
}

export async function deleteModelRecord(app: string, model: string, id: string): Promise<void> {
  const mutation = deleteMutationName(app, model)
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
): { query: string; variables: Record<string, unknown> } {
  const mutation = kind === 'create' ? createMutationName(app, model) : updateMutationName(app, model)
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
  return { query, variables }
}

export async function getModelRecord(
  app: string,
  model: string,
  id: string,
  fields: string[],
): Promise<ModelRecord> {
  const queryField = getQueryName(app, model)
  const unique = [...new Set(['id', ...fields])]
  const selection = unique.join(' ')
  const data = await gql<Record<string, ModelRecord>>(`query($id: ID!) {
    ${queryField}(id: $id) { ${selection} }
  }`, { id })
  const record = data[queryField]
  if (!record) throw new Error(`record not found: ${model}/${id}`)
  return record
}

export async function createModelRecord(
  app: string,
  model: string,
  fields: ModelField[],
  values: Record<string, unknown>,
): Promise<ModelRecord> {
  const { query, variables } = buildWriteMutation('create', app, model, fields, values)
  const mutation = createMutationName(app, model)
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
): Promise<ModelRecord> {
  const { query, variables } = buildWriteMutation('update', app, model, fields, values, id)
  const mutation = updateMutationName(app, model)
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
