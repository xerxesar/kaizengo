<script lang="ts">
  import { getContext } from 'svelte'
  import Input from './Input.svelte'
  import { getKFormContext } from './kform-context'
  import { KFORM_FIELD_CTX, type KFormFieldScope } from './kform-field-context'

  type Props = {
    /** Model field key; defaults to the enclosing `<KFormField field="…">`. */
    field?: string
    placeholder?: string
    type?: string
    disabled?: boolean
    readonly?: boolean
  }

  let {
    field: fieldProp,
    placeholder = '',
    type = 'text',
    disabled = false,
    readonly = false,
  }: Props = $props()

  const form = getKFormContext()
  const scope = getContext<KFormFieldScope | undefined>(KFORM_FIELD_CTX)
  const field = $derived(fieldProp ?? scope?.field ?? '')
  const draftValue = $derived((form.draft[field] as string) ?? '')

  function oninput(e: Event) {
    form.setValue(field, (e.currentTarget as HTMLInputElement).value)
  }
</script>

<Input {placeholder} {type} {disabled} {readonly} value={draftValue} {oninput} />
