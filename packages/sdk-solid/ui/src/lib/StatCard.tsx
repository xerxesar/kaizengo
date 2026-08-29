import { type JSX } from 'solid-js'

type Props = {
  label: string
  value: string | number
  hint?: string
  icon?: string
}

export function StatCard(props: Props): JSX.Element {
  return (
    <div class="flex items-start gap-5 border border-[var(--kg-border)] bg-[var(--kg-surface)] p-5">
      {props.icon && (
        <span class="text-xl leading-none text-[var(--kg-text-secondary)]" aria-hidden="true">
          {props.icon}
        </span>
      )}
      <div class="flex min-w-0 flex-col gap-1">
        <span class="text-xs tracking-wide text-[var(--kg-text-secondary)]">{props.label}</span>
        <span class="text-[1.75rem] font-light leading-tight text-[var(--kg-text)]">{props.value}</span>
        {props.hint && <span class="text-sm text-[var(--kg-text-muted)]">{props.hint}</span>}
      </div>
    </div>
  )
}
