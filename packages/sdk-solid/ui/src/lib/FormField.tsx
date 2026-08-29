import { Field } from '@ark-ui/solid/field'
import { Show, type JSX, type ParentProps } from 'solid-js'
import { fieldErrorClass, fieldHintClass, fieldLabelClass } from './ark/styles'
import { cn } from './cn'

type Props = ParentProps & {
  label: string
  required?: boolean
  hint?: string
  error?: string
  class?: string
}

export function FormField(props: Props): JSX.Element {
  return (
    <Field.Root required={props.required} invalid={Boolean(props.error)} class={cn('flex min-w-[12rem] flex-1 flex-col gap-1', props.class)}>
      <Field.Label class={fieldLabelClass}>
        {props.label}
        <Field.RequiredIndicator fallback={<span class="text-red-600"> *</span>} />
      </Field.Label>
      {props.children}
      <Show when={props.hint && !props.error}>
        <Field.HelperText class={fieldHintClass}>{props.hint}</Field.HelperText>
      </Show>
      <Show when={props.error}>
        <Field.ErrorText class={fieldErrorClass}>{props.error}</Field.ErrorText>
      </Show>
    </Field.Root>
  )
}
