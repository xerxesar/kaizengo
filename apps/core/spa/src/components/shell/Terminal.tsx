import {
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { Maximize2, Minimize2, X } from 'lucide-react'
import { t } from '@/lib'
import { navigateApp } from '@/lib/menu-route'
import {
  TERMINAL_EVENT,
  TERMINAL_HEIGHT_DEFAULT,
  TERMINAL_HEIGHT_MIN,
  fetchPlatformCatalog,
  fetchShellCommands,
  inputPending,
  loadTerminalState,
  runShellLine,
  saveTerminalState,
  writeDomainsPending,
  type PlatformOperation,
  type ShellCommand,
  type TerminalRequest,
} from '@/lib/platform-shell'
import { TerminalTedEditor, type TedSession } from './TerminalTedEditor'

type Entry = { id: number; line: string; output: string; ok: boolean; pending?: boolean; data?: unknown }

const DOMAIN_SUBS = [
  'list',
  'show',
  'show-view',
  'show-menu',
  'show-query',
  'show-command',
  'show-model',
  'show-policy',
  'history',
  'conflicts',
  'rebase',
  'diff',
  'apply',
  'add-field',
  'set-field',
  'remove-field',
  'add-index',
  'set-index',
  'remove-index',
  'add-view',
  'set-view',
  'remove-view',
  'add-menu',
  'set-menu',
  'remove-menu',
  'add-query',
  'set-query',
  'remove-query',
  'add-command',
  'set-command',
  'remove-command',
  'revert',
  'delete',
  'export',
  'import',
]

const HISTORY_KEY = 'kaizengo:terminal.history'
const HISTORY_MAX = 200

function loadHistory(): string[] {
  try {
    const raw = JSON.parse(localStorage.getItem(HISTORY_KEY) ?? '[]')
    return Array.isArray(raw) ? raw.filter((x): x is string => typeof x === 'string') : []
  } catch {
    return []
  }
}

function commonPrefix(values: string[]): string {
  if (!values.length) return ''
  let prefix = values[0]
  for (const v of values.slice(1)) {
    while (!v.startsWith(prefix)) prefix = prefix.slice(0, -1)
  }
  return prefix
}

function appFromDomainLine(line: string): string | undefined {
  const m = line.match(/^domain\s+\S+\s+(\S+)/)
  if (!m) return undefined
  const tok = m[1]
  if (tok.includes('.')) return tok.split('.')[0]
  if (!tok.startsWith('-')) return tok
  return undefined
}

function maybeOpenDomains(line: string, result: Entry) {
  if (!line.startsWith('domain ')) return
  const out = result.output || ''
  const data = result.data as {
    migrateRequired?: boolean
    proposedSpec?: string
    migrateHints?: { path: string; hint: string; stub: string }[]
    domain?: { app?: string; spec?: string }
  } | null
  const migrate =
    data?.migrateRequired === true ||
    /migration required/i.test(out) ||
    /provide a data migration/i.test(out)
  const conflict = /conflict:\s*\d+\s*path/i.test(out) || /rebaseStatus.*conflict/i.test(out)
  if (!migrate && !conflict) return
  const app = data?.domain?.app || appFromDomainLine(line)
  const proposedSpec = (data?.proposedSpec || data?.domain?.spec || '').trim() || undefined
  writeDomainsPending({
    app,
    tab: migrate ? 'migrate' : 'conflicts',
    proposedSpec,
    migrateHints: data?.migrateHints,
    command: line,
  })
  const href = app
    ? `/app/settings/domains?app=${encodeURIComponent(app)}`
    : '/app/settings/domains'
  navigateApp(href)
}

function tedFromResult(result: Entry): TedSession | null {
  const data = result.data as { ted?: boolean; command?: string; seed?: string } | null
  if (!data?.ted || !data.command) return null
  return { command: data.command, seed: data.seed ?? '' }
}

function parseYamlNames(spec: string, listKey: string, nameKey: string): string[] {
  // Lightweight scan: `- name: Foo` / `id: bar` under a `listKey:` block (any indent).
  const lines = spec.split('\n')
  let inList = false
  let listIndent = 0
  const names: string[] = []
  const listRe = new RegExp(`^(\\s*)${listKey}:\\s*$`)
  const nameRe = new RegExp(`^\\s*-?\\s*${nameKey}:\\s*(\\S+)\\s*$`)
  for (const line of lines) {
    const lm = line.match(listRe)
    if (lm) {
      inList = true
      listIndent = lm[1].length
      continue
    }
    if (inList) {
      const indent = line.match(/^(\s*)/)?.[1].length ?? 0
      if (line.trim() && indent <= listIndent) {
        inList = false
      }
    }
    if (!inList) continue
    const m = line.match(nameRe)
    if (m) names.push(m[1].replace(/['"]/g, ''))
  }
  return names
}

/** Candidates for the last token of `text`, using the server command list and ACL-filtered catalog. */
function completions(
  text: string,
  commands: ShellCommand[],
  ops: PlatformOperation[],
  nameCache: Record<string, string[]>,
): string[] {
  const tokens = text.split(/\s+/).filter((t, i, a) => i < a.length - 1 || t !== '' || a.length === 1)
  // Preserve trailing empty token for completion of last word
  const rawTokens = text.split(/\s+/)
  const word = rawTokens[rawTokens.length - 1] ?? ''
  const parts = text.trimEnd().split(/\s+/)
  const cmd = parts[0] ?? ''

  if (parts.length <= 1 && !text.endsWith(' ')) {
    return [...commands.map((c) => c.name), 'clear'].filter((n) => n.startsWith(word))
  }

  if (cmd === 'help' || cmd === 'man') {
    if (parts.length === 1 || (parts.length === 2 && !text.endsWith(' '))) {
      return [...commands.map((c) => c.name), 'domain'].filter((n) => n.startsWith(word))
    }
    if (cmd === 'man' && parts[1] === 'domain') {
      return DOMAIN_SUBS.filter((s) => s.startsWith(word))
    }
  }

  if (cmd === 'domain') {
    if (parts.length === 1 || (parts.length === 2 && !text.endsWith(' '))) {
      return DOMAIN_SUBS.filter((s) => s.startsWith(word))
    }
    const sub = parts[1]
    const apps = [...new Set(ops.map((o) => o.app))]
    if (parts.length === 2 || (parts.length === 3 && !text.endsWith(' '))) {
      if (['list', 'ls'].includes(sub)) {
        return ['compiled', 'dynamic'].filter((s) => s.startsWith(word))
      }
      return apps.filter((a) => a.startsWith(word))
    }
    const app = parts[2]
    const named: Record<string, { list: string; key: string }> = {
      'show-view': { list: 'views', key: 'name' },
      'set-view': { list: 'views', key: 'name' },
      'remove-view': { list: 'views', key: 'name' },
      'show-menu': { list: 'menus', key: 'id' },
      'set-menu': { list: 'menus', key: 'id' },
      'remove-menu': { list: 'menus', key: 'id' },
      'show-query': { list: 'queries', key: 'name' },
      'set-query': { list: 'queries', key: 'name' },
      'remove-query': { list: 'queries', key: 'name' },
      'show-command': { list: 'commands', key: 'name' },
      'set-command': { list: 'commands', key: 'name' },
      'remove-command': { list: 'commands', key: 'name' },
      'show-model': { list: 'models', key: 'name' },
      'show-policy': { list: 'entries', key: 'name' },
    }
    const meta = named[sub]
    if (meta && (parts.length === 3 || (parts.length === 4 && !text.endsWith(' ')))) {
      const cacheKey = `${app}:${meta.list}`
      return (nameCache[cacheKey] ?? []).filter((n) => n.startsWith(word))
    }
    if (['set-view', 'set-menu', 'set-query', 'set-command', 'apply', 'diff', 'set-field', 'add-field'].includes(sub)) {
      const flags = ['-ted', '--ted', '--dry-run', '--no-migrate', '--migrate=', '--migrate-pre=', '--force']
      return flags.filter((f) => f.startsWith(word))
    }
    return []
  }

  if (tokens.length === 1 || (parts.length === 2 && !text.endsWith(' '))) {
    const apps = [...new Set(ops.map((o) => o.app))]
    switch (cmd) {
      case 'ls':
        return apps.filter((a) => a.startsWith(word))
      case 'query':
      case 'run':
      case 'describe': {
        const kind = cmd === 'query' ? 'query' : cmd === 'run' ? 'command' : null
        return ops
          .filter((o) => !kind || o.kind === kind)
          .map((o) => `${o.app}.${o.name}`)
          .filter((n) => n.startsWith(word))
      }
    }
  }
  if (cmd === 'query' || cmd === 'run') {
    const op = ops.find((o) => `${o.app}.${o.name}` === parts[1])
    if (!op) return []
    const used = new Set(parts.slice(2, -1).map((tok) => tok.split('=')[0]))
    return [...op.args.map((a) => `${a.name}=`), '--fields', '--all', '--json']
      .filter((n) => !used.has(n.replace(/=$/, '')) && n.startsWith(word))
  }
  return []
}

function clampHeight(px: number) {
  const max = typeof window !== 'undefined' ? Math.floor(window.innerHeight * 0.9) : 800
  return Math.max(TERMINAL_HEIGHT_MIN, Math.min(max, Math.round(px)))
}

export function Terminal() {
  const initial = loadTerminalState()
  const [open, setOpen] = useState(initial.open)
  const [fullscreen, setFullscreen] = useState(initial.fullscreen)
  const [heightPx, setHeightPx] = useState(initial.heightPx || TERMINAL_HEIGHT_DEFAULT)
  const [entries, setEntries] = useState<Entry[]>(initial.entries)
  const [input, setInput] = useState(initial.draft)
  const [hint, setHint] = useState('')
  const [commands, setCommands] = useState<ShellCommand[]>([])
  const [ops, setOps] = useState<PlatformOperation[]>([])
  const [ted, setTed] = useState<TedSession | null>(null)
  const [nameCache, setNameCache] = useState<Record<string, string[]>>({})
  const history = useRef<string[]>(loadHistory())
  const cursor = useRef(-1)
  const nextId = useRef(initial.nextId)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ startY: number; startH: number } | null>(null)
  const persistTimer = useRef<number | null>(null)

  function persist(partial?: Partial<{ open: boolean; heightPx: number; fullscreen: boolean; entries: Entry[]; draft: string }>) {
    const snapshot = {
      open: partial?.open ?? open,
      heightPx: partial?.heightPx ?? heightPx,
      fullscreen: partial?.fullscreen ?? fullscreen,
      entries: (partial?.entries ?? entries).filter((e) => !e.pending),
      draft: partial?.draft ?? input,
      nextId: nextId.current,
    }
    if (persistTimer.current) window.clearTimeout(persistTimer.current)
    persistTimer.current = window.setTimeout(() => saveTerminalState(snapshot), 120)
  }

  async function refreshCatalog() {
    try {
      const [c, o] = await Promise.all([fetchShellCommands(), fetchPlatformCatalog()])
      setCommands(c)
      setOps(o)
    } catch {
      // Completion is best-effort; commands still run server-side.
    }
  }

  async function warmNameCache(app: string) {
    if (!app || nameCache[`${app}:views`]) return
    try {
      const res = await fetch('/graphql', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: 'query($app:String!){ platformDomain(app:$app){ spec } }',
          variables: { app },
        }),
      })
      const body = await res.json()
      const spec = (body?.data?.platformDomain?.spec as string) || ''
      if (!spec) return
      setNameCache((prev) => ({
        ...prev,
        [`${app}:views`]: parseYamlNames(spec, 'views', 'name'),
        [`${app}:menus`]: parseYamlNames(spec, 'menus', 'id'),
        [`${app}:queries`]: parseYamlNames(spec, 'queries', 'name'),
        [`${app}:commands`]: parseYamlNames(spec, 'commands', 'name'),
        [`${app}:models`]: parseYamlNames(spec, 'models', 'name'),
        [`${app}:entries`]: parseYamlNames(spec, 'entries', 'name'),
      }))
    } catch {
      /* ignore */
    }
  }

  async function execute(line: string) {
    const trimmed = line.trim()
    if (!trimmed) return
    history.current = [trimmed, ...history.current.filter((h) => h !== trimmed)].slice(0, HISTORY_MAX)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(history.current))
    cursor.current = -1
    setInput('')
    setHint('')
    persist({ draft: '' })
    if (trimmed === 'clear') {
      setEntries([])
      persist({ entries: [], draft: '' })
      return
    }
    const id = nextId.current++
    setEntries((prev) => {
      const next = [...prev, { id, line: trimmed, output: '', ok: true, pending: true }]
      return next
    })
    let result: Entry
    try {
      const r = await runShellLine(trimmed)
      result = { id, line: trimmed, output: r.output, ok: r.ok, data: r.data }
    } catch (e) {
      result = { id, line: trimmed, output: e instanceof Error ? e.message : String(e), ok: false }
    }
    setEntries((prev) => {
      const next = prev.map((x) => (x.id === id ? result : x))
      persist({ entries: next, draft: '' })
      return next
    })
    if (/^domain\s+(apply|add-field|set-field|remove-field|set-view|set-menu|set-query|set-command|revert|delete|rm|reset|import|restore|rebase)\b/.test(trimmed)) {
      void refreshCatalog()
      window.dispatchEvent(new Event('kaizengo:apps'))
      const app = appFromDomainLine(trimmed)
      if (app) {
        setNameCache((prev) => {
          const next = { ...prev }
          for (const k of Object.keys(next)) {
            if (k.startsWith(`${app}:`)) delete next[k]
          }
          return next
        })
      }
    }
    const tedSession = tedFromResult(result)
    if (tedSession) {
      setTed(tedSession)
      return
    }
    maybeOpenDomains(trimmed, result)
  }

  async function saveTed(body: string) {
    if (!ted) return
    const session = ted
    setTed(null)
    const line = `${session.command} <<EOF\n${body.replace(/\n$/, '')}\nEOF`
    await execute(line)
  }

  useEffect(() => {
    const onRequest = (e: Event) => {
      const req = (e as CustomEvent<TerminalRequest>).detail ?? {}
      if (req.toggleFullscreen) {
        setOpen(true)
        setFullscreen((prev) => {
          const next = !prev
          persist({ open: true, fullscreen: next })
          return next
        })
        return
      }
      if (req.toggle) {
        setOpen((prev) => {
          const next = !prev
          persist({ open: next })
          return next
        })
        return
      }
      setOpen(true)
      persist({ open: true })
      if (req.line != null) {
        if (req.run) void execute(req.line)
        else setInput(req.line)
      }
    }
    window.addEventListener(TERMINAL_EVENT, onRequest)
    return () => window.removeEventListener(TERMINAL_EVENT, onRequest)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!open) return
    void refreshCatalog()
    requestAnimationFrame(() => inputRef.current?.focus())
  }, [open])

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight })
  }, [entries, ted])

  useEffect(() => {
    persist()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, heightPx, fullscreen, input])

  useEffect(() => {
    const m = input.match(/^domain\s+\S+\s+(\S+)/)
    if (m && !m[1].startsWith('-')) void warmNameCache(m[1].includes('.') ? m[1].split('.')[0] : m[1])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [input])

  function complete() {
    const candidates = completions(input, commands, ops, nameCache)
    if (!candidates.length) return
    const tokens = input.split(/\s+/)
    const word = tokens[tokens.length - 1]
    const prefix = commonPrefix(candidates)
    const single = candidates.length === 1
    const replacement = single && !prefix.endsWith('=') ? `${prefix} ` : prefix
    if (prefix.length > word.length || single) {
      setInput(input.slice(0, input.length - word.length) + replacement)
      setHint(single ? '' : candidates.join('  '))
    } else {
      setHint(candidates.join('  '))
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    const multiline = input.includes('\n')
    if (e.key === 'Enter' && !e.shiftKey) {
      if (inputPending(input)) return
      e.preventDefault()
      void execute(input)
      return
    }
    if (e.key === 'Tab' && !multiline) {
      e.preventDefault()
      complete()
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      if (fullscreen) {
        setFullscreen(false)
        persist({ fullscreen: false })
        return
      }
      setOpen(false)
      persist({ open: false })
      return
    }
    if (e.key === 'l' && e.ctrlKey) {
      e.preventDefault()
      setEntries([])
      persist({ entries: [] })
      return
    }
    if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') && !multiline) {
      const items = history.current
      if (!items.length) return
      e.preventDefault()
      const next =
        e.key === 'ArrowUp' ? Math.min(cursor.current + 1, items.length - 1) : Math.max(cursor.current - 1, -1)
      cursor.current = next
      setInput(next < 0 ? '' : items[next])
    }
  }

  function onResizePointerDown(e: ReactPointerEvent<HTMLDivElement>) {
    if (fullscreen) return
    e.preventDefault()
    dragRef.current = { startY: e.clientY, startH: heightPx }
    e.currentTarget.setPointerCapture(e.pointerId)
  }

  function onResizePointerMove(e: ReactPointerEvent<HTMLDivElement>) {
    if (!dragRef.current) return
    const delta = dragRef.current.startY - e.clientY
    setHeightPx(clampHeight(dragRef.current.startH + delta))
  }

  function onResizePointerUp(e: ReactPointerEvent<HTMLDivElement>) {
    if (!dragRef.current) return
    dragRef.current = null
    try {
      e.currentTarget.releasePointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
    persist({ heightPx })
  }

  return (
    <AnimatePresence>
      {open ? (
        <motion.section
          key="terminal"
          className={fullscreen ? 'kg-terminal kg-terminal-fullscreen' : 'kg-terminal'}
          style={fullscreen ? undefined : ({ ['--kg-terminal-h']: `${heightPx}px` } as CSSProperties)}
          aria-label={t('shell.terminal.title')}
          initial={{ y: '100%' }}
          animate={{ y: 0 }}
          exit={{ y: '100%' }}
          transition={{ type: 'spring', stiffness: 420, damping: 40 }}
        >
          {!fullscreen ? (
            <div
              className="kg-terminal-resize"
              role="separator"
              aria-orientation="horizontal"
              aria-label={t('shell.terminal.resize')}
              onPointerDown={onResizePointerDown}
              onPointerMove={onResizePointerMove}
              onPointerUp={onResizePointerUp}
              onPointerCancel={onResizePointerUp}
            />
          ) : null}
          <header className="kg-terminal-bar">
            <span className="kg-terminal-title">{t('shell.terminal.title')}</span>
            <span className="kg-terminal-hint">{t('shell.terminal.hint')}</span>
            <button
              type="button"
              className="kg-terminal-icon"
              aria-label={t('shell.terminal.fullscreen')}
              title={t('shell.terminal.fullscreen')}
              onClick={() => {
                setFullscreen((prev) => {
                  const next = !prev
                  persist({ fullscreen: next })
                  return next
                })
              }}
            >
              {fullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
            </button>
            <button
              type="button"
              className="kg-terminal-close"
              aria-label={t('shell.terminal.close')}
              onClick={() => {
                setOpen(false)
                persist({ open: false })
              }}
            >
              <X size={16} />
            </button>
          </header>
          {ted ? (
            <TerminalTedEditor session={ted} onSave={(body) => void saveTed(body)} onCancel={() => setTed(null)} />
          ) : (
            <div
              ref={scrollRef}
              className="kg-terminal-scroll"
              dir="ltr"
              onClick={() => {
                if (!window.getSelection()?.toString()) inputRef.current?.focus()
              }}
            >
              {entries.map((entry) => (
                <motion.div
                  key={entry.id}
                  className="kg-terminal-entry"
                  initial={{ opacity: 0, x: -4 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.12 }}
                >
                  <pre className="kg-terminal-line">
                    <span className="kg-terminal-prompt">›</span> {entry.line}
                  </pre>
                  {entry.pending ? (
                    <div className="kg-terminal-pending" />
                  ) : entry.output ? (
                    <pre className={entry.ok ? 'kg-terminal-output' : 'kg-terminal-output kg-terminal-error'}>
                      {entry.output}
                    </pre>
                  ) : null}
                </motion.div>
              ))}
              {hint ? <pre className="kg-terminal-candidates">{hint}</pre> : null}
              <div className="kg-terminal-input-row">
                <span className="kg-terminal-prompt">›</span>
                <textarea
                  ref={inputRef}
                  className="kg-terminal-input"
                  value={input}
                  rows={Math.min(Math.max(input.split('\n').length, 1), 16)}
                  spellCheck={false}
                  autoCapitalize="off"
                  autoComplete="off"
                  placeholder={t('shell.terminal.placeholder')}
                  onChange={(e) => {
                    setInput(e.target.value)
                    setHint('')
                  }}
                  onKeyDown={onKeyDown}
                />
              </div>
            </div>
          )}
        </motion.section>
      ) : null}
    </AnimatePresence>
  )
}
