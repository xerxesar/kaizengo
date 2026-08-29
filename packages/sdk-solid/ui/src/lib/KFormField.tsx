import { createSignal, onMount, Show, type JSX } from 'solid-js'
import { getKFormContext } from './kform-context'
import { FormField } from './FormField'
import { Input } from './Input'

type Props = {
  field: string
  label: string
  required?: boolean
  hint?: string
  error?: string
  placeholder?: string
  type?: string
  children?: JSX.Element
}

export function KFormField(props: Props): JSX.Element {
  const form = getKFormContext()
  const fieldDef = () => form.fields.find((item) => item.key === props.field)
  const isRequired = () => props.required ?? fieldDef()?.required ?? false

  return (
    <FormField label={props.label} required={isRequired()} hint={props.hint} error={props.error}>
      {props.children ?? (
        <KInput placeholder={props.placeholder} type={props.type} field={props.field} />
      )}
    </FormField>
  )
}

function KInput(props: { placeholder?: string; type?: string; field: string }) {
  const form = getKFormContext()
  const value = () => String(form.draft[props.field] ?? '')

  return (
    <Input
      type={props.type ?? 'text'}
      placeholder={props.placeholder}
      value={value()}
      onChange={(next) => form.setValue(props.field, next)}
    />
  )
}
