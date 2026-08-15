export type CalendarOption = { id: string; name: string }

export type LocaleOption = { id: string; name: string; dir: 'ltr' | 'rtl' }

export type SettingsLabels = {
  title: string
  subtitle: string
  locale: string
  calendar: string
  shell: string
  save: string
  saved: string
}

export type PlatformSettings = {
  locale: string
  dir: 'ltr' | 'rtl'
  locales: LocaleOption[]
  defaultCalendar: string
  shellTitle: string
  calendars: CalendarOption[]
  labels: SettingsLabels
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

const settingsQuery = `{
  settings {
    locale dir locales { id name dir }
    defaultCalendar shellTitle
    calendars { id name }
    labels { title subtitle locale calendar shell save saved }
  }
}`

export function fetchSettings() {
  return gql<{ settings: PlatformSettings }>(settingsQuery)
}

export function updateSettings(input: {
  locale: string
  defaultCalendar: string
  shellTitle: string
}) {
  return gql<{ updateSettings: PlatformSettings }>(
    `mutation ($locale: String, $defaultCalendar: String, $shellTitle: String) {
      updateSettings(locale: $locale, defaultCalendar: $defaultCalendar, shellTitle: $shellTitle) {
        locale dir locales { id name dir }
        defaultCalendar shellTitle
        calendars { id name }
        labels { title subtitle locale calendar shell save saved }
      }
    }`,
    input,
  )
}

