import { type JSX, type ParentProps } from 'solid-js'

type Props = ParentProps & {
  title: string
  description?: string
}

export function FormSection(props: Props): JSX.Element {
  return (
    <fieldset class="m-0 flex flex-col gap-5 border-0 p-0">
      <legend class="p-0 text-sm font-semibold text-[var(--kg-text)]">{props.title}</legend>
      {props.description && (
        <p class="text-sm text-[var(--kg-text-secondary)]">{props.description}</p>
      )}
      <div class="flex flex-col gap-5">{props.children}</div>
    </fieldset>
  )
}
