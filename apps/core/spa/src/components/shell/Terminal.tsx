import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { X } from 'lucide-react'
import { t } from '@/lib'
import {
  TERMINAL_EVENT,
  fetchPlatformCatalog,
  fetchShellCommands,
  heredocPending,
  runShellLine,
  type PlatformOperation,
  type ShellCommand,
  type TerminalRequest,
} from '@/lib/platform-shell'

type Entry = { id: number; line: string; output: string; ok: boolean; pending?: boolean }

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

/** Candidates for the last token of `text`, using the server command list and ACL-filtered catalog. */
function completions(text: string, commands: ShellCommand[], ops: PlatformOperation[]): string[] {
  const tokens = text.split(/\s+/)
  const word = tokens[tokens.length - 1]
  if (tokens.length === 1) {
    return [...commands.map((c) => c.name), 'clear'].filter((n) => n.startsWith(word))
  }
  const cmd = tokens[0]
  if (tokens.length === 2) {
    const apps = [...new Set(ops.map((o) => o.app))]
    switch (cmd) {
      case 'ls':
        return apps.filter((a) => a.startsWith(word))
      case 'domain':
        return ['list', 'show', 'apply', 'diff', 'delete'].filter((s) => s.startsWith(word))
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
    return []
  }
  if (cmd === 'query' || cmd === 'run') {
    const op = ops.find((o) => `${o.app}.${o.name}` === tokens[1])
    if (!op) return []
    const used = new Set(tokens.slice(2, -1).map((tok) => tok.split('=')[0]))
    return [...op.args.map((a) => `${a.name}=`), '--fields', '--all', '--json']
      .filter((n) => !used.has(n.replace(/=$/, '')) && n.startsWith(word))
  }
  return []
}

export function Terminal() {
  const [open, setOpen] = useState(false)
  const [entries, setEntries] = useState<Entry[]>([])
  const [input, setInput] = useState('')
  const [hint, setHint] = useState('')
  const [commands, setCommands] = useState<ShellCommand[]>([])
  const [ops, setOps] = useState<PlatformOperation[]>([])
  const history = useRef<string[]>(loadHistory())
  const cursor = useRef(-1)
  const nextId = useRef(0)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  async function refreshCatalog() {
    try {
      const [c, o] = await Promise.all([fetchShellCommands(), fetchPlatformCatalog()])
      setCommands(c)
      setOps(o)
    } catch {
      // Completion is best-effort; commands still run server-side.
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
    if (trimmed === 'clear') {
      setEntries([])
      return
    }
    const id = nextId.current++
    setEntries((prev) => [...prev, { id, line: trimmed, output: '', ok: true, pending: true }])
    let result: Entry
    try {
      const r = await runShellLine(trimmed)
      result = { id, line: trimmed, output: r.output, ok: r.ok }
    } catch (e) {
      result = { id, line: trimmed, output: e instanceof Error ? e.message : String(e), ok: false }
    }
    setEntries((prev) => prev.map((x) => (x.id === id ? result : x)))
    if (/^domain\s+(apply|add-field|revert|delete|rm|reset|import|restore)\b/.test(trimmed)) {
      void refreshCatalog()
      window.dispatchEvent(new Event('kaizengo:apps'))
    }
  }

  useEffect(() => {
    const onRequest = (e: Event) => {
      const req = (e as CustomEvent<TerminalRequest>).detail ?? {}
      if (req.toggle) {
        setOpen((prev) => !prev)
        return
      }
      setOpen(true)
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
  }, [entries])

  function complete() {
    const candidates = completions(input, commands, ops)
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
      if (heredocPending(input)) return
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
      setOpen(false)
      return
    }
    if (e.key === 'l' && e.ctrlKey) {
      e.preventDefault()
      setEntries([])
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

  return (
    <AnimatePresence>
      {open ? (
        <motion.section
          key="terminal"
          className="kg-terminal"
          aria-label={t('shell.terminal.title')}
          initial={{ y: '100%' }}
          animate={{ y: 0 }}
          exit={{ y: '100%' }}
          transition={{ type: 'spring', stiffness: 420, damping: 40 }}
        >
          <header className="kg-terminal-bar">
            <span className="kg-terminal-title">{t('shell.terminal.title')}</span>
            <span className="kg-terminal-hint">{t('shell.terminal.hint')}</span>
            <button
              type="button"
              className="kg-terminal-close"
              aria-label={t('shell.terminal.close')}
              onClick={() => setOpen(false)}
            >
              <X size={16} />
            </button>
          </header>
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
        </motion.section>
      ) : null}
    </AnimatePresence>
  )
}
