import type { ReactNode } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getKFormContext } from './kform-context'
import { fieldErrorClass, fieldHintClass, fieldLabelClass } from '@/lib/styles/classes'

type Props = {
  field: string
  label: string
  required?: boolean
  hint?: string
  error?: string
  placeholder?: string
  type?: string
  children?: ReactNode
}

export function KFormField(props: Props) {
  const form = getKFormContext()
  const fieldDef = form.fields.find((item) => item.key === props.field)
  const isRequired = props.required ?? fieldDef?.required ?? false

  return (
    <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
      <Label className={fieldLabelClass}>
        {props.label}
        {isRequired ? <span className="text-[var(--kg-danger)]"> *</span> : null}
      </Label>
      {props.children ?? (
        <KInput placeholder={props.placeholder} type={props.type} field={props.field} />
      )}
      {props.hint && !props.error ? <p className={fieldHintClass}>{props.hint}</p> : null}
      {props.error ? <p className={fieldErrorClass}>{props.error}</p> : null}
    </div>
  )
}

function KInput(props: { placeholder?: string; type?: string; field: string }) {
  const form = getKFormContext()
  const value = String(form.draft[props.field] ?? '')

  return (
    <Input
      type={props.type ?? 'text'}
      placeholder={props.placeholder}
      value={value}
      onChange={(e) => form.setValue(props.field, e.target.value)}
    />
  )
}
