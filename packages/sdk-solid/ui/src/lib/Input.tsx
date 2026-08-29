import { Field } from '@ark-ui/solid/field'
import { splitProps, type JSX } from 'solid-js'
import { inputClass } from './ark/styles'
import { cn } from './cn'

type Props = {
  type?: string
  value?: string | number
  placeholder?: string
  disabled?: boolean
  class?: string
  onInput?: (value: string) => void
  onChange?: (value: string) => void
}

export function Input(props: Props): JSX.Element {
  const [local, rest] = splitProps(props, ['class', 'onInput', 'onChange', 'value', 'type', 'placeholder', 'disabled'])
  return (
    <Field.Input
      {...rest}
      type={local.type}
      placeholder={local.placeholder}
      disabled={local.disabled}
      class={cn(inputClass, local.class)}
      value={local.value ?? ''}
      onInput={(e) => {
        const value = e.currentTarget.value
        local.onInput?.(value)
        local.onChange?.(value)
      }}
    />
  )
}
