/** Runtime CLI client: everything goes through GraphQL (`platformShell*`, `platformCatalog`). */

export type ShellResult = {
  ok: boolean
  output: string
  data?: unknown
}

export type ShellCommand = {
  name: string
  usage: string
  summary: string
}

export type PlatformOperation = {
  app: string
  kind: 'query' | 'command'
  name: string
  field: string
  args: { name: string; type: string; required: boolean }[]
  dynamic: boolean
}

export const TERMINAL_EVENT = 'kaizengo:terminal'
export const TERMINAL_STATE_KEY = 'kaizengo:terminal.state'
export const TERMINAL_ENTRIES_MAX = 100
export const TERMINAL_HEIGHT_MIN = 128
export const TERMINAL_HEIGHT_DEFAULT = Math.round(typeof window !== 'undefined' ? Math.min(window.innerHeight * 0.42, 448) : 320)

/** Detail for {@link TERMINAL_EVENT}: open the terminal, optionally prefilling or running a line. */
export type TerminalRequest = {
  line?: string
  run?: boolean
  toggle?: boolean
  toggleFullscreen?: boolean
}

export type TerminalPersistedEntry = {
  id: number
  line: string
  output: string
  ok: boolean
}

export type TerminalPersistedState = {
  open: boolean
  heightPx: number
  fullscreen: boolean
  entries: TerminalPersistedEntry[]
  draft: string
  nextId: number
}

export function openTerminal(req: TerminalRequest = {}) {
  window.dispatchEvent(new CustomEvent<TerminalRequest>(TERMINAL_EVENT, { detail: req }))
}

export function loadTerminalState(): TerminalPersistedState {
  const fallback: TerminalPersistedState = {
    open: false,
    heightPx: TERMINAL_HEIGHT_DEFAULT,
    fullscreen: false,
    entries: [],
    draft: '',
    nextId: 0,
  }
  try {
    const raw = localStorage.getItem(TERMINAL_STATE_KEY)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as Partial<TerminalPersistedState>
    return {
      open: !!parsed.open,
      heightPx:
        typeof parsed.heightPx === 'number' && parsed.heightPx >= TERMINAL_HEIGHT_MIN
          ? parsed.heightPx
          : fallback.heightPx,
      fullscreen: !!parsed.fullscreen,
      entries: Array.isArray(parsed.entries)
        ? parsed.entries
            .filter(
              (e): e is TerminalPersistedEntry =>
                !!e &&
                typeof e.id === 'number' &&
                typeof e.line === 'string' &&
                typeof e.output === 'string' &&
                typeof e.ok === 'boolean',
            )
            .slice(-TERMINAL_ENTRIES_MAX)
        : [],
      draft: typeof parsed.draft === 'string' ? parsed.draft : '',
      nextId: typeof parsed.nextId === 'number' ? parsed.nextId : 0,
    }
  } catch {
    return fallback
  }
}

export function saveTerminalState(state: TerminalPersistedState) {
  try {
    const payload: TerminalPersistedState = {
      ...state,
      entries: state.entries.slice(-TERMINAL_ENTRIES_MAX).map(({ id, line, output, ok }) => ({
        id,
        line,
        output,
        ok,
      })),
    }
    localStorage.setItem(TERMINAL_STATE_KEY, JSON.stringify(payload))
  } catch {
    /* ignore quota */
  }
}

/** sessionStorage payload so Settings → Domains can open the right tab after a shell error. */
export const DOMAINS_PENDING_KEY = 'kaizengo:domains.pending'

export type DomainsPending = {
  app?: string
  tab?: 'conflicts' | 'migrate'
  proposedSpec?: string
  migrateHints?: { path: string; hint: string; stub: string }[]
  command?: string
  at: number
}

export const DOMAINS_PENDING_EVENT = 'kaizengo:domains-pending'

export function writeDomainsPending(p: Omit<DomainsPending, 'at'> & { at?: number }) {
  const payload = { ...p, at: p.at ?? Date.now() }
  sessionStorage.setItem(DOMAINS_PENDING_KEY, JSON.stringify(payload))
  window.dispatchEvent(new CustomEvent(DOMAINS_PENDING_EVENT, { detail: payload }))
}

export function readDomainsPending(): DomainsPending | null {
  try {
    const raw = sessionStorage.getItem(DOMAINS_PENDING_KEY)
    if (!raw) return null
    return JSON.parse(raw) as DomainsPending
  } catch {
    return null
  }
}

export function clearDomainsPending() {
  sessionStorage.removeItem(DOMAINS_PENDING_KEY)
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

export async function runShellLine(line: string): Promise<ShellResult> {
  const data = await gql<{ platformShell: { ok: boolean; output: string; data?: string | null } }>(
    'mutation($line: String!) { platformShell(line: $line) { ok output data } }',
    { line },
  )
  const r = data.platformShell
  let parsed: unknown
  if (r.data) {
    try {
      parsed = JSON.parse(r.data)
    } catch {
      parsed = r.data
    }
  }
  return { ok: r.ok, output: r.output, data: parsed }
}

export async function fetchShellCommands(): Promise<ShellCommand[]> {
  const data = await gql<{ platformShellCommands: ShellCommand[] }>(
    '{ platformShellCommands { name usage summary } }',
  )
  return data.platformShellCommands ?? []
}

export async function fetchPlatformCatalog(): Promise<PlatformOperation[]> {
  const data = await gql<{ platformCatalog: PlatformOperation[] }>(
    '{ platformCatalog { app kind name field dynamic args { name type required } } }',
  )
  return data.platformCatalog ?? []
}

/** The shell line that invokes an operation, e.g. `query crm.leads` / `run crm.addLead`. */
export function operationLine(op: PlatformOperation): string {
  return `${op.kind === 'command' ? 'run' : 'query'} ${op.app}.${op.name}`
}

/** True when a line opens a heredoc (`<<TAG`) that its last line has not closed yet. */
export function heredocPending(text: string): boolean {
  const lines = text.split('\n')
  const m = lines[0].match(/<<\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?\s*$/)
  if (!m) return false
  return lines.length < 2 || lines[lines.length - 1].trim() !== m[1]
}

/**
 * True when interactive clients should keep reading: open heredoc, unclosed
 * quote, or a trailing unescaped backslash (bash-like line continuation).
 * When the first line opens a heredoc, later newlines belong to that body.
 */
export function inputPending(text: string): boolean {
  if (heredocPending(text)) return true
  const first = text.split('\n')[0] ?? ''
  if (/<<\s*['"]?[A-Za-z_][A-Za-z0-9_]*['"]?\s*$/.test(first)) return false
  return tokenizeIncomplete(text)
}

function tokenizeIncomplete(s: string): boolean {
  let quote: '"' | "'" | 0 = 0
  let escaped = false
  for (const r of s) {
    if (escaped) {
      escaped = false
      continue
    }
    if (quote === "'") {
      if (r === "'") quote = 0
      continue
    }
    if (quote === '"') {
      if (r === '"') quote = 0
      else if (r === '\\') escaped = true
      continue
    }
    if (r === '\\') {
      escaped = true
      continue
    }
    if (r === "'" || r === '"') {
      quote = r
    }
  }
  return quote !== 0 || escaped
}
