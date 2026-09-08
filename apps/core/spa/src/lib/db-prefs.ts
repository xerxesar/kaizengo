/** Client-only DB / debug preferences (URL → localStorage → cookie). */

const DB_KEY = 'kaizengo.db'
const DEBUG_KEY = 'kaizengo.debug'
const DB_COOKIE = 'kg_db'

function readURLParam(name: string): string | null {
  const v = new URLSearchParams(window.location.search).get(name)
  return v === null ? null : v.trim()
}

function writeCookie(name: string, value: string) {
  if (!value) {
    document.cookie = `${name}=; Path=/; Max-Age=0; SameSite=Lax`
    return
  }
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${365 * 24 * 60 * 60}; SameSite=Lax`
}

export function getStoredDB(): string {
  try {
    return (localStorage.getItem(DB_KEY) || '').trim()
  } catch {
    return ''
  }
}

export function getStoredDebug(): boolean {
  try {
    return localStorage.getItem(DEBUG_KEY) === '1'
  } catch {
    return false
  }
}

export function setStoredDB(name: string) {
  const next = name.trim()
  try {
    if (next) localStorage.setItem(DB_KEY, next)
    else localStorage.removeItem(DB_KEY)
  } catch {
    /* ignore */
  }
  writeCookie(DB_COOKIE, next)
}

export function setStoredDebug(on: boolean) {
  try {
    if (on) localStorage.setItem(DEBUG_KEY, '1')
    else localStorage.removeItem(DEBUG_KEY)
  } catch {
    /* ignore */
  }
}

/** Apply ?db= / ?debug= from the URL into localStorage (URL wins when present). */
export function syncPrefsFromURL(): { db: string; debug: boolean } {
  const dbParam = readURLParam('db')
  const debugParam = readURLParam('debug')
  if (dbParam !== null) setStoredDB(dbParam)
  if (debugParam !== null) setStoredDebug(debugParam === '1')
  return { db: getStoredDB(), debug: getStoredDebug() }
}

/** Keep the address bar in sync with stored prefs (does not reload). */
export function syncURLFromPrefs() {
  const p = new URLSearchParams(window.location.search)
  const db = getStoredDB()
  const debug = getStoredDebug()
  if (db) p.set('db', db)
  else p.delete('db')
  if (debug) p.set('debug', '1')
  else p.delete('debug')
  const q = p.toString()
  const url = window.location.pathname + (q ? `?${q}` : '') + window.location.hash
  history.replaceState(null, '', url)
}

/** Activate the stored DB on the Go server (client-driven, no catalog write). */
export async function activateStoredDB(): Promise<string> {
  const name = getStoredDB()
  if (!name) return ''
  const res = await fetch('/web/database/use', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) {
    const text = await res.text()
    let msg = text
    try {
      msg = JSON.parse(text).error || text
    } catch {
      /* ignore */
    }
    throw new Error(msg || `activate failed (${res.status})`)
  }
  writeCookie(DB_COOKIE, name)
  return name
}
