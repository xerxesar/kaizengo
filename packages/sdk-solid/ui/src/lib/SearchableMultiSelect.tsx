import { Combobox as ArkCombobox, useListCollection } from '@ark-ui/solid/combobox'
import { createEffect, createMemo, createSignal, For, Show, type JSX } from 'solid-js'
import {
  comboboxControlClass,
  comboboxInputClass,
  comboboxTriggerClass,
  selectContentClass,
  selectItemClass,
} from './ark/styles'
import { cn } from './cn'

export type SearchableMultiSelectOption = {
  value: string
  label: string
}

type Props = {
  value?: string[]
  options: SearchableMultiSelectOption[]
  placeholder?: string
  disabled?: boolean
  /** Allow typing values that are not in options (Enter to add). Default true when options empty. */
  allowCustomValue?: boolean
  emptyMessage?: string
  class?: string
  onChange?: (value: string[]) => void
}

export function SearchableMultiSelect(props: Props): JSX.Element {
  const list = useListCollection<SearchableMultiSelectOption>({
    initialItems: props.options,
    itemToString: (item) => item.label,
    itemToValue: (item) => item.value,
    filter: (itemText, filterText) => itemText.toLowerCase().includes(filterText.toLowerCase()),
  })

  createEffect(() => {
    list.set([...props.options])
  })

  const allowCustom = () => props.allowCustomValue ?? props.options.length === 0
  const [inputValue, setInputValue] = createSignal('')
  const selected = createMemo(() => props.value ?? [])

  function emit(next: string[]) {
    const unique = [...new Set(next.map((v) => v.trim()).filter(Boolean))]
    props.onChange?.(unique)
  }

  function removeValue(value: string) {
    emit(selected().filter((v) => v !== value))
  }

  function addCustomFromInput() {
    const raw = inputValue().trim()
    if (!raw || !allowCustom()) return
    if (selected().includes(raw)) {
      setInputValue('')
      return
    }
    emit([...selected(), raw])
    setInputValue('')
    list.filter('')
  }

  return (
    <ArkCombobox.Root
      collection={list.collection()}
      value={selected()}
      inputValue={inputValue()}
      disabled={props.disabled}
      placeholder={selected().length ? undefined : props.placeholder}
      multiple
      allowCustomValue={allowCustom()}
      openOnClick
      closeOnSelect={false}
      onInputValueChange={(details) => {
        setInputValue(details.inputValue)
        list.filter(details.inputValue)
      }}
      onValueChange={(details) => {
        emit(details.value)
        setInputValue('')
        list.filter('')
      }}
      class={props.class}
    >
      <ArkCombobox.Control
        class={cn(
          comboboxControlClass,
          'h-auto min-h-10 flex-wrap items-center gap-1.5 py-1.5',
        )}
      >
        <For each={selected()}>
          {(value) => (
            <span class="inline-flex max-w-full items-center gap-1 rounded bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-700">
              <span class="min-w-0 truncate font-mono">{value}</span>
              <button
                type="button"
                class="shrink-0 border-0 bg-transparent p-0 text-sm leading-none text-zinc-500 hover:text-zinc-800"
                aria-label={`Remove ${value}`}
                disabled={props.disabled}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => removeValue(value)}
              >
                ×
              </button>
            </span>
          )}
        </For>
        <ArkCombobox.Input
          class={cn(comboboxInputClass, 'min-w-[6rem]')}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && allowCustom() && inputValue().trim()) {
              e.preventDefault()
              addCustomFromInput()
            }
            if (e.key === 'Backspace' && !inputValue() && selected().length > 0) {
              removeValue(selected()[selected().length - 1])
            }
          }}
        />
        <ArkCombobox.Trigger class={comboboxTriggerClass}>▾</ArkCombobox.Trigger>
      </ArkCombobox.Control>
      <ArkCombobox.Positioner>
        <ArkCombobox.Content class={selectContentClass}>
          <ArkCombobox.ItemGroup>
            <For each={list.collection().items}>
              {(item) => (
                <ArkCombobox.Item item={item} class={selectItemClass}>
                  <ArkCombobox.ItemText class="font-mono text-xs">{item.label}</ArkCombobox.ItemText>
                  <ArkCombobox.ItemIndicator>✓</ArkCombobox.ItemIndicator>
                </ArkCombobox.Item>
              )}
            </For>
            <Show when={list.collection().size === 0}>
              <div class={cn(selectItemClass, 'text-[var(--kg-text-muted)]')}>
                {props.emptyMessage ?? (allowCustom() ? 'Type and press Enter to add' : 'No matches')}
              </div>
            </Show>
          </ArkCombobox.ItemGroup>
        </ArkCombobox.Content>
      </ArkCombobox.Positioner>
    </ArkCombobox.Root>
  )
}
