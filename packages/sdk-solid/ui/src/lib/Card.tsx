import { Show, type JSX, type ParentProps } from 'solid-js'
import { cn } from './cn'

type Props = ParentProps & {
  title?: string
  actions?: JSX.Element
  class?: string
}

export function Card(props: Props): JSX.Element {
  return (
    <section class={cn('border border-[var(--kg-border)] bg-[var(--kg-surface)]', props.class)}>
      <Show when={props.title || props.actions}>
        <header class="flex items-center justify-between border-b border-[var(--kg-border)] px-5 py-4">
          <Show when={props.title}>
            <h3 class="text-sm font-semibold text-[var(--kg-text)]">{props.title}</h3>
          </Show>
          <Show when={props.actions}>
            <div class="flex gap-3">{props.actions}</div>
          </Show>
        </header>
      </Show>
      <div class="p-5">{props.children}</div>
    </section>
  )
}
