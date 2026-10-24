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

/** Detail for {@link TERMINAL_EVENT}: open the terminal, optionally prefilling or running a line. */
export type TerminalRequest = {
  line?: string
  run?: boolean
  toggle?: boolean
}

export function openTerminal(req: TerminalRequest = {}) {
  window.dispatchEvent(new CustomEvent<TerminalRequest>(TERMINAL_EVENT, { detail: req }))
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
