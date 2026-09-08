import { Combobox as ArkCombobox, useListCollection } from '@ark-ui/solid/combobox'
import { createEffect, createSignal, For, Show, type JSX } from 'solid-js'
import {
  comboboxControlClass,
  comboboxInputClass,
  comboboxTriggerClass,
  selectContentClass,
  selectItemClass,
} from './ark/styles'
import { cn } from './cn'

export type SearchableComboboxOption = {
  value: string
  label: string
  /** Optional kind tag shown as a colored badge before the label. */
  kind?: string
}

type Props = {
  value?: string
  options: SearchableComboboxOption[]
  placeholder?: string
  disabled?: boolean
  allowCustomValue?: boolean
  emptyMessage?: string
  class?: string
  onChange?: (value: string) => void
}

function kindTagClass(kind: string): string {
  switch (kind) {
    case 'query':
      return 'bg-sky-100 text-sky-800'
    case 'command':
      return 'bg-emerald-100 text-emerald-800'
    case 'view':
      return 'bg-zinc-200 text-zinc-700'
    default:
      return 'bg-zinc-100 text-zinc-600'
  }
}

export function SearchableCombobox(props: Props): JSX.Element {
  const list = useListCollection<SearchableComboboxOption>({
    initialItems: props.options,
    itemToString: (item) => (item.kind ? `${item.kind} ${item.label}` : item.label),
    itemToValue: (item) => item.value,
    filter: (itemText, filterText) => itemText.toLowerCase().includes(filterText.toLowerCase()),
  })

  createEffect(() => {
    list.set([...props.options])
  })

  const allowCustom = () => props.allowCustomValue ?? true
  const [inputValue, setInputValue] = createSignal(props.value ?? '')
  let lastEmitted = props.value ?? ''

  createEffect(() => {
    const external = props.value ?? ''
    if (external !== lastEmitted) {
      lastEmitted = external
      setInputValue(external)
    }
  })

  const selectedValue = () => {
    if (allowCustom()) return [] as string[]
    const current = props.value
    if (!current) return [] as string[]
    return props.options.some((option) => option.value === current) ? [current] : []
  }

  return (
    <ArkCombobox.Root
      collection={list.collection()}
      value={selectedValue()}
      inputValue={inputValue()}
      disabled={props.disabled}
      placeholder={props.placeholder}
      allowCustomValue={allowCustom()}
      selectionBehavior="preserve"
      inputBehavior="none"
      openOnClick
      closeOnSelect={!allowCustom()}
      onInputValueChange={(details) => {
        setInputValue(details.inputValue)
        list.filter(details.inputValue)
        if (details.reason === 'input-change' || details.reason === 'clear-trigger') {
          lastEmitted = details.inputValue
          props.onChange?.(details.inputValue)
        }
      }}
      onValueChange={(details) => {
        const next = details.value[0]
        if (next == null) return
        lastEmitted = next
        setInputValue(next)
        props.onChange?.(next)
      }}
      class={props.class}
    >
      <ArkCombobox.Control class={comboboxControlClass}>
        <ArkCombobox.Input class={comboboxInputClass} />
        <ArkCombobox.Trigger class={comboboxTriggerClass}>▾</ArkCombobox.Trigger>
      </ArkCombobox.Control>
      <ArkCombobox.Positioner>
        <ArkCombobox.Content class={selectContentClass}>
          <ArkCombobox.ItemGroup>
            <For each={list.collection().items}>
              {(item) => (
                <ArkCombobox.Item item={item} class={selectItemClass}>
                  <ArkCombobox.ItemText class="flex min-w-0 items-center gap-2">
                    <Show when={item.kind}>
                      {(kind) => (
                        <span
                          class={cn(
                            'shrink-0 rounded px-1.5 py-0.5 text-[0.65rem] font-semibold uppercase tracking-wide',
                            kindTagClass(kind()),
                          )}
                        >
                          {kind()}
                        </span>
                      )}
                    </Show>
                    <span class="min-w-0 truncate font-mono text-xs">{item.label}</span>
                  </ArkCombobox.ItemText>
                  <ArkCombobox.ItemIndicator>✓</ArkCombobox.ItemIndicator>
                </ArkCombobox.Item>
              )}
            </For>
            <Show when={list.collection().size === 0}>
              <div class={cn(selectItemClass, 'text-[var(--kg-text-muted)]')}>
                {props.emptyMessage ?? 'No matches'}
              </div>
            </Show>
          </ArkCombobox.ItemGroup>
        </ArkCombobox.Content>
      </ArkCombobox.Positioner>
    </ArkCombobox.Root>
  )
}
