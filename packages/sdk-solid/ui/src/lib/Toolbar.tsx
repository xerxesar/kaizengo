import { Show, type JSX, type ParentProps } from 'solid-js'

type Props = ParentProps & {
  start?: JSX.Element
  end?: JSX.Element
}

export function Toolbar(props: Props): JSX.Element {
  return (
    <div class="mb-3.5 flex min-h-10 flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2.5">
        <Show when={props.start} fallback={props.children} />
      </div>
      <Show when={props.end}>
        <div class="flex shrink-0 flex-wrap items-center gap-2.5">{props.end}</div>
      </Show>
    </div>
  )
}
