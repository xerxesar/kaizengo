import { Checkbox } from '@ark-ui/solid/checkbox'
import { type JSX } from 'solid-js'
import { checkboxControlClass } from './ark/styles'
import { cn } from './cn'

type Props = {
  checked?: boolean
  disabled?: boolean
  label?: string
  class?: string
  onChange?: (checked: boolean) => void
}

export function CheckboxInput(props: Props): JSX.Element {
  return (
    <Checkbox.Root
      checked={props.checked}
      disabled={props.disabled}
      ids={{}}
      class={cn('inline-flex cursor-pointer items-center gap-2', props.class)}
      onCheckedChange={(details) => props.onChange?.(details.checked === true)}
    >
      <Checkbox.Control class={checkboxControlClass}>
        <Checkbox.Indicator>✓</Checkbox.Indicator>
      </Checkbox.Control>
      {props.label && (
        <Checkbox.Label class="text-sm text-[var(--kg-text)]">{props.label}</Checkbox.Label>
      )}
      <Checkbox.HiddenInput />
    </Checkbox.Root>
  )
}

export { CheckboxInput as Checkbox }
