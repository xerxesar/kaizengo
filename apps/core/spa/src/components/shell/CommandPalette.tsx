import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { motion } from 'motion/react'
import { Dialog } from '@/components/ui/dialog'
import { t } from '@/lib'
import type { NavEntry } from '@/lib/nav'
import {
  fetchPlatformCatalog,
  fetchShellCommands,
  openTerminal,
  operationLine,
  type PlatformOperation,
  type ShellCommand,
} from '@/lib/platform-shell'

type Item = {
  key: string
  group: 'apps' | 'queries' | 'commands' | 'shell'
  label: string
  detail: string
  run: () => void
}

const GROUP_LABEL: Record<Item['group'], string> = {
  apps: 'shell.palette.group_apps',
  queries: 'shell.palette.group_queries',
  commands: 'shell.palette.group_commands',
  shell: 'shell.palette.group_shell',
}

const GROUP_ORDER: Item['group'][] = ['apps', 'queries', 'commands', 'shell']

const MAX_ITEMS = 60

function score(item: Item, terms: string[]): number {
  const hay = `${item.label} ${item.detail}`.toLowerCase()
  let total = 0
  for (const term of terms) {
    const idx = hay.indexOf(term)
    if (idx < 0) return -1
    total += idx === 0 ? 3 : hay[idx - 1] === '.' || hay[idx - 1] === ' ' ? 2 : 1
  }
  return total
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  apps: NavEntry[]
  onNavigate: (route: string) => void
}

export function CommandPalette({ open, onOpenChange, apps, onNavigate }: Props) {
  const [filter, setFilter] = useState('')
  const [active, setActive] = useState(0)
  const [ops, setOps] = useState<PlatformOperation[]>([])
  const [commands, setCommands] = useState<ShellCommand[]>([])
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  useEffect(() => {
    if (!open) return
    setFilter('')
    setActive(0)
    requestAnimationFrame(() => inputRef.current?.focus())
    void Promise.all([fetchPlatformCatalog(), fetchShellCommands()])
      .then(([o, c]) => {
        setOps(o)
        setCommands(c)
      })
      .catch(() => {})
  }, [open])

  function close(then: () => void) {
    onOpenChange(false)
    then()
  }

  const all: Item[] = [
    ...apps.map<Item>((a) => ({
      key: `app:${a.route}`,
      group: 'apps',
      label: a.title,
      detail: a.route,
      run: () => close(() => onNavigate(a.route)),
    })),
    ...ops.map<Item>((op) => {
      const line = operationLine(op)
      const required = op.args.filter((a) => a.required)
      return {
        key: `op:${op.field}`,
        group: op.kind === 'command' ? 'commands' : 'queries',
        label: `${op.app}.${op.name}`,
        detail: op.args.map((a) => (a.required ? `${a.name}*` : a.name)).join(' '),
        run: () =>
          close(() =>
            op.kind === 'query' && !required.length
              ? openTerminal({ line, run: true })
              : openTerminal({ line: `${line} ` }),
          ),
      }
    }),
    ...commands.map<Item>((c) => ({
      key: `cmd:${c.name}`,
      group: 'shell',
      label: c.usage,
      detail: c.summary,
      run: () => close(() => openTerminal({ line: `${c.name} ` })),
    })),
  ]

  const terms = filter.toLowerCase().split(/\s+/).filter(Boolean)
  const items = all
    .map((item, i) => ({ item, i, s: terms.length ? score(item, terms) : 0 }))
    .filter((x) => x.s >= 0)
    .sort((a, b) => GROUP_ORDER.indexOf(a.item.group) - GROUP_ORDER.indexOf(b.item.group) || b.s - a.s || a.i - b.i)
    .map((x) => x.item)
    .slice(0, MAX_ITEMS)
  const current = Math.min(active, Math.max(items.length - 1, 0))

  useEffect(() => {
    listRef.current?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [current, filter])

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      const delta = e.key === 'ArrowDown' ? 1 : -1
      setActive((current + delta + items.length) % Math.max(items.length, 1))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (items[current]) items[current].run()
      else if (filter.trim()) close(() => openTerminal({ line: filter.trim(), run: true }))
    } else if (e.key === 'Escape') {
      e.preventDefault()
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange} className="kg-palette max-w-xl self-start mt-[12vh]">
      <motion.div
        initial={{ opacity: 0, scale: 0.98, y: -6 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.14 }}
        className="flex min-h-0 flex-col"
      >
        <input
          ref={inputRef}
          className="kg-palette-input"
          value={filter}
          placeholder={t('shell.palette.placeholder')}
          spellCheck={false}
          autoComplete="off"
          role="combobox"
          aria-expanded="true"
          aria-controls="kg-palette-list"
          onChange={(e) => {
            setFilter(e.target.value)
            setActive(0)
          }}
          onKeyDown={onKeyDown}
        />
        <ul ref={listRef} id="kg-palette-list" role="listbox" className="kg-palette-list">
          {items.map((item, i) => (
            <li key={item.key} role="presentation">
              {i === 0 || items[i - 1].group !== item.group ? (
                <div className="kg-palette-group">{t(GROUP_LABEL[item.group])}</div>
              ) : null}
              <button
                type="button"
                role="option"
                aria-selected={i === current}
                className="kg-palette-item"
                onMouseMove={() => setActive(i)}
                onClick={item.run}
              >
                <span className="kg-palette-label">{item.label}</span>
                <span className="kg-palette-detail">{item.detail}</span>
              </button>
            </li>
          ))}
          {!items.length ? (
            <li className="kg-palette-empty">{t('shell.palette.empty')}</li>
          ) : null}
        </ul>
      </motion.div>
    </Dialog>
  )
}
