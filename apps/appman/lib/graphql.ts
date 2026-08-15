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

export type App = {
  name: string
  title: string
  summary: string
  version: string
  installedVersion?: string | null
  installed: boolean
  loaded: boolean
  autoInstall: boolean
  upgrade: boolean
  depends: string[]
}

const fields = `
  name title summary version installedVersion
  installed loaded autoInstall upgrade depends
`

export function fetchApps() {
  return gql<{ apps: App[] }>(`query { apps { ${fields} } }`).then((d) => d.apps)
}

export function installApp(name: string) {
  return gql<{ installApp: App }>(
    `mutation ($name: String!) { installApp(name: $name) { ${fields} } }`,
    { name },
  ).then((d) => d.installApp)
}

export function upgradeApp(name: string) {
  return gql<{ upgradeApp: App }>(
    `mutation ($name: String!) { upgradeApp(name: $name) { ${fields} } }`,
    { name },
  ).then((d) => d.upgradeApp)
}
