import { type JSX } from 'solid-js'

type Props = {
  title: string
  description?: string
}

export function EmptyState(props: Props): JSX.Element {
  return (
    <div class="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
      <p class="text-base font-medium text-[var(--kg-text)]">{props.title}</p>
      {props.description && (
        <p class="mt-2 text-sm text-[var(--kg-text-secondary)]">{props.description}</p>
      )}
    </div>
  )
}
