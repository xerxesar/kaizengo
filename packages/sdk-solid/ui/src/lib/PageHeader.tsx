import { Show, type JSX, type ParentProps } from 'solid-js'

type Props = ParentProps & {
  title: string
  subtitle?: string
  actions?: JSX.Element
}

export function PageHeader(props: Props): JSX.Element {
  return (
    <>
      <header class="flex items-center justify-between gap-6 py-6">
        <div>
          <h1 class="font-[var(--kg-font-display)] text-[2rem] font-normal leading-tight text-[var(--kg-text)]">
            {props.title}
          </h1>
          <Show when={props.subtitle}>
            <p class="mt-2 max-w-2xl text-sm text-[var(--kg-text-secondary)]">{props.subtitle}</p>
          </Show>
        </div>
        <Show when={props.actions}>
          <div class="flex shrink-0 items-center gap-3">{props.actions}</div>
        </Show>
      </header>
      {props.children}
    </>
  )
}
