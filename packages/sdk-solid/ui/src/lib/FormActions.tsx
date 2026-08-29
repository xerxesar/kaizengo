import { type JSX, type ParentProps } from 'solid-js'

export function FormActions(props: ParentProps): JSX.Element {
  return <div class="kg-form-actions flex justify-end gap-2 pt-2">{props.children}</div>
}
