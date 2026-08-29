import { Select as ArkSelect, createListCollection } from '@ark-ui/solid/select'
import { createMemo, type JSX } from 'solid-js'
import { selectContentClass, selectItemClass, selectTriggerClass } from './ark/styles'
import { cn } from './cn'

type Option = { value: string; label: string }

type Props = {
  value?: string
  options: Option[]
  placeholder?: string
  disabled?: boolean
  class?: string
  onChange?: (value: string, e?: Event) => void
}

export function Select(props: Props): JSX.Element {
  const collection = createMemo(() =>
    createListCollection({
      items: props.options,
      itemToString: (item) => item.label,
      itemToValue: (item) => item.value,
    }),
  )

  const value = () => (props.value ? [props.value] : ([] as string[]))

  return (
    <ArkSelect.Root
      collection={collection()}
      value={value()}
      disabled={props.disabled}
      onValueChange={(details) => props.onChange?.(details.value[0] ?? '')}
      class={props.class}
    >
      <ArkSelect.Control>
        <ArkSelect.Trigger class={cn(selectTriggerClass)}>
          <ArkSelect.ValueText placeholder={props.placeholder} />
          <ArkSelect.Indicator>▾</ArkSelect.Indicator>
        </ArkSelect.Trigger>
      </ArkSelect.Control>
      <ArkSelect.Positioner>
        <ArkSelect.Content class={selectContentClass}>
          <ArkSelect.ItemGroup>
            {props.options.map((item) => (
              <ArkSelect.Item item={item} class={selectItemClass}>
                <ArkSelect.ItemText>{item.label}</ArkSelect.ItemText>
                <ArkSelect.ItemIndicator>✓</ArkSelect.ItemIndicator>
              </ArkSelect.Item>
            ))}
          </ArkSelect.ItemGroup>
        </ArkSelect.Content>
      </ArkSelect.Positioner>
      <ArkSelect.HiddenSelect />
    </ArkSelect.Root>
  )
}
