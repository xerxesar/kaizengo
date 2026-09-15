import type { MenuItem } from './types'

const menuSelection = `
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

function camelAppMenusField(app: string): string {
  const name = app.trim()
  if (!name) return 'menus'
  return name[0].toLowerCase() + name.slice(1) + 'Menus'
}

/** Fetch `{app}Menus` for any module that registers the standard menus query. */
export async function fetchAppMenus(app: string): Promise<MenuItem[]> {
  const field = camelAppMenusField(app)
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query: `query { ${field} { ${menuSelection} } }`,
    }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  const rows = body.data?.[field]
  return Array.isArray(rows) ? (rows as MenuItem[]) : []
}
