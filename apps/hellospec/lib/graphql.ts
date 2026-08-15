export type HellospecGreeting = {
  id: string
  orgId: string
  authorId: string
  message: string
  deleted: boolean
  createdAt: string
  updatedAt: string
}

export type HellospecMenu = {
  id: string
  label: string
  labelKey?: string
  view?: string
  route?: string
  component?: string
  sourceApp?: string
  children?: HellospecMenu[]
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

export async function ping(): Promise<string> {
  const data = await gql<{ hellospecPing: string }>('query { hellospecPing }')
  return data.hellospecPing
}

export async function listGreetings(): Promise<HellospecGreeting[]> {
  const data = await gql<{ hellospecGreetings: HellospecGreeting[] }>(
    'query { hellospecGreetings { id message authorId updatedAt } }',
  )
  return data.hellospecGreetings
}

export async function createGreeting(message: string): Promise<HellospecGreeting> {
  const data = await gql<{ createHellospecGreeting: HellospecGreeting }>(
    'mutation($message: String!) { createHellospecGreeting(message: $message) { id message updatedAt } }',
    { message },
  )
  return data.createHellospecGreeting
}

export async function deleteGreeting(id: string): Promise<void> {
  await gql('mutation($id: ID!) { deleteHellospecGreeting(id: $id) }', { id })
}

/** Nested menu fields for `{app}Menus` (supports deep hierarchy). */
const menuFields = `
  id label labelKey view route component sourceApp
  children {
    id label labelKey view route component sourceApp
    children {
      id label labelKey view route component sourceApp
      children {
        id label labelKey view route component sourceApp
        children { id label labelKey view route component sourceApp }
      }
    }
  }
`

export async function listMenus(): Promise<HellospecMenu[]> {
  const data = await gql<{ hellospecMenus: HellospecMenu[] }>(
    `query { hellospecMenus { ${menuFields} } }`,
  )
  return data.hellospecMenus
}

export type HellospecViewSlot = {
  slot: string
  component: string
  module?: string
  sourceApp?: string
}

export async function listViewSlots(view: string): Promise<HellospecViewSlot[]> {
  const data = await gql<{ hellospecViewSlots: HellospecViewSlot[] }>(
    `query ($view: String!) {
      hellospecViewSlots(view: $view) { slot component module sourceApp }
    }`,
    { view },
  )
  return data.hellospecViewSlots
}
