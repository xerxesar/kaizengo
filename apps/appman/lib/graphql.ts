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

export type AppRow = {
  id: string
  name: string
  title: string
  summary: string
  version: string
  installedVersion?: string | null
  installed: boolean
  loaded: boolean
  autoInstall: boolean
  upgrade: boolean
  status: string
  depends: string
}

const fields = `
  id name title summary version installedVersion
  installed loaded autoInstall upgrade status depends
`

export function installApp(name: string) {
  return gql<{ appmanInstallApp: AppRow }>(
    `mutation ($name: String!) { appmanInstallApp(name: $name) { ${fields} } }`,
    { name },
  ).then((d) => d.appmanInstallApp)
}

export function upgradeApp(name: string) {
  return gql<{ appmanUpgradeApp: AppRow }>(
    `mutation ($name: String!) { appmanUpgradeApp(name: $name) { ${fields} } }`,
    { name },
  ).then((d) => d.appmanUpgradeApp)
}
