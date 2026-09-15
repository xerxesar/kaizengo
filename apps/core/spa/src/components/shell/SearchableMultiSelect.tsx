import { useMemo, useState } from 'react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

export type MultiOption = {
  value: string
  label: string
}

type Props = {
  options: MultiOption[]
  value?: string[]
  onChange?: (value: string[]) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function SearchableMultiSelect({
  options,
  value = [],
  onChange,
  placeholder = 'Search…',
  className,
  disabled,
}: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const selected = new Set(value)

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return options
    return options.filter(
      (o) => o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q),
    )
  }, [options, query])

  function toggle(v: string) {
    const next = new Set(selected)
    if (next.has(v)) next.delete(v)
    else next.add(v)
    onChange?.([...next])
  }

  return (
    <div className={cn('relative', className)}>
      <Input
        disabled={disabled}
        value={open ? query : value.map((v) => options.find((o) => o.value === v)?.label ?? v).join(', ')}
        placeholder={placeholder}
        onFocus={() => {
          setOpen(true)
          setQuery('')
        }}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onBlur={() => window.setTimeout(() => setOpen(false), 150)}
      />
      {open ? (
        <div className="absolute z-50 mt-1 max-h-64 w-full overflow-auto border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] shadow-lg">
          {filtered.map((opt) => (
            <label
              key={opt.value}
              className="flex cursor-pointer items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--kg-surface-hover)]"
              onMouseDown={(e) => e.preventDefault()}
            >
              <input
                type="checkbox"
                checked={selected.has(opt.value)}
                onChange={() => toggle(opt.value)}
              />
              {opt.label}
            </label>
          ))}
        </div>
      ) : null}
    </div>
  )
}
