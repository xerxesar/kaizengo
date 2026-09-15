import { useMemo, useState } from 'react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

export type ComboboxOption = {
  value: string
  label: string
}

type Props = {
  options: ComboboxOption[]
  value?: string
  onChange?: (value: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function SearchableCombobox({
  options,
  value = '',
  onChange,
  placeholder = 'Search…',
  className,
  disabled,
}: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)

  const selected = options.find((o) => o.value === value)
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return options
    return options.filter(
      (o) => o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q),
    )
  }, [options, query])

  return (
    <div className={cn('relative', className)}>
      <Input
        disabled={disabled}
        value={open ? query : (selected?.label ?? '')}
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
          {filtered.length === 0 ? (
            <div className="px-4 py-2 text-sm text-[var(--kg-text-muted)]">No matches</div>
          ) : (
            filtered.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className="flex w-full px-4 py-2 text-left text-sm hover:bg-[var(--kg-surface-hover)]"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  onChange?.(opt.value)
                  setOpen(false)
                }}
              >
                {opt.label}
              </button>
            ))
          )}
        </div>
      ) : null}
    </div>
  )
}
