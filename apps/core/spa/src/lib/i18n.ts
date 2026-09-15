/** Platform localization helpers. UI strings come from Vite-compiled `.po` catalogs; GraphQL `i18n` remains for server-side labels. */

export type TextDirection = 'ltr' | 'rtl'

export type I18nEntry = { key: string; value: string }

export type I18nBundle = {
  locale: string
  dir: TextDirection
  entries: I18nEntry[]
}

export type Translator = {
  locale: string
  dir: TextDirection
  /** Translate a catalog key; optional printf-style %s / %d args. */
  t: (key: string, ...args: Array<string | number>) => string
  /** All loaded messages keyed by catalog key. */
  messages: Record<string, string>
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

/** Apply document language + writing direction (RTL for fa, ar, …). */
export function applyLocale(locale: string, dir: TextDirection = 'ltr') {
  const root = document.documentElement
  root.lang = locale
  root.dir = dir
  root.dataset.kgLocale = locale
  root.dataset.kgDir = dir
}

/** Load messages by key list and/or key prefix(es) from the platform catalog. */
export async function fetchI18n(opts: {
  keys?: string[]
  prefix?: string
  prefixes?: string[]
}): Promise<Translator> {
  const prefixes = [...(opts.prefixes ?? []), ...(opts.prefix ? [opts.prefix] : [])]
  const data = await gql<{ i18n: I18nBundle }>(
    `query ($keys: [String!], $prefix: String, $prefixes: [String!]) {
      i18n(keys: $keys, prefix: $prefix, prefixes: $prefixes) {
        locale
        dir
        entries { key value }
      }
    }`,
    {
      keys: opts.keys ?? null,
      prefix: null,
      prefixes: prefixes.length ? prefixes : null,
    },
  )

  const messages: Record<string, string> = {}
  for (const e of data.i18n.entries) {
    messages[e.key] = e.value
  }

  const dir = (data.i18n.dir === 'rtl' ? 'rtl' : 'ltr') as TextDirection
  applyLocale(data.i18n.locale, dir)

  return {
    locale: data.i18n.locale,
    dir,
    messages,
    t(key: string, ...args: Array<string | number>) {
      let s = messages[key] ?? key
      for (const a of args) {
        s = s.replace(/%[sd]/, String(a))
      }
      return s
    },
  }
}

/** Sync document dir/lang from the active platform locale (shell startup). */
export async function syncDocumentLocale(): Promise<{ locale: string; dir: TextDirection }> {
  const data = await gql<{ settings: { locale: string; dir: string } }>(
    `query { settings { locale dir } }`,
  ).catch(() => null)
  if (!data) {
    applyLocale('en', 'ltr')
    return { locale: 'en', dir: 'ltr' }
  }
  const dir = (data.settings.dir === 'rtl' ? 'rtl' : 'ltr') as TextDirection
  applyLocale(data.settings.locale, dir)
  return { locale: data.settings.locale, dir }
}
