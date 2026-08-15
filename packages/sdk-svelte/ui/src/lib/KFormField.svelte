<script lang="ts">
  import { setContext } from 'svelte'
  import type { Snippet } from 'svelte'
  import FormField from './FormField.svelte'
  import KInput from './KInput.svelte'
  import { getKFormContext } from './kform-context'
  import { KFORM_FIELD_CTX } from './kform-field-context'

  type Props = {
    /** Model field key bound to the form draft. */
    field: string
    label: string
    required?: boolean
    hint?: string
    error?: string
    placeholder?: string
    type?: string
    children?: Snippet
  }

  let {
    field,
    label,
    required,
    hint,
    error,
    placeholder = '',
    type = 'text',
    children,
  }: Props = $props()

  const form = getKFormContext()
  const fieldDef = $derived(form.fields.find((item) => item.key === field))
  const isRequired = $derived(required ?? fieldDef?.required ?? false)

  setContext(KFORM_FIELD_CTX, {
    get field() {
      return field
    },
  })
</script>

<FormField {label} required={isRequired} {hint} {error}>
  {#if children}
    {@render children()}
  {:else}
    <KInput {placeholder} {type} />
  {/if}
</FormField>
