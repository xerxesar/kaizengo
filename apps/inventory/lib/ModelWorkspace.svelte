<script lang="ts">
  import {
    Card,
    FormField,
    KAppStatus,
    KForm,
    KTable,
    PageHeader,
    Select,
    parseNamespace,
    t,
    type KFormFieldContext,
  } from '@kaizengo/sdk-svelte/ui'
  import { ENUMS, enumKey } from './enums'
  import RelationPicker from './RelationPicker.svelte'

  type Props = {
    model: string
    title: string
    subtitle?: string
    emptyKey?: string
    formTitle?: string
    deletable?: boolean
    showForm?: boolean
  }

  let {
    model,
    title,
    subtitle = '',
    emptyKey = 'inventory.empty',
    formTitle = '',
    deletable = true,
    showForm = true,
  }: Props = $props()

  let table = $state<{ refresh: () => Promise<void> }>()
  const ns = $derived(parseNamespace(model))

  function enumOptions(fieldKey: string) {
    const values = ENUMS[enumKey(model, fieldKey)] ?? []
    return values.map((value) => {
      const key = `inventory.enum.${enumKey(model, fieldKey)}.${value}`
      const label = t(key)
      return { value, label: label === key ? value.replaceAll('_', ' ') : label }
    })
  }
</script>

<PageHeader {title} {subtitle} />

<div class="workspace">
  {#if showForm}
    <Card title={formTitle || t('inventory.form.new')}>
    <KForm {model} onsuccess={() => void table?.refresh()}>
      {#snippet field(ctx: KFormFieldContext)}
        {@const enums = enumOptions(ctx.field.key)}
        {#if ctx.field.relation}
          <FormField label={ctx.label} required={ctx.field.required}>
            <RelationPicker
              relation={ctx.field.relation}
              fromApp={ns.app}
              value={String(ctx.draft[ctx.field.key] ?? '')}
              placeholder={ctx.placeholder}
              onchange={(id) => ctx.setValue(id)}
            />
          </FormField>
        {:else if enums.length}
          <FormField label={ctx.label} required={ctx.field.required}>
            <Select
              value={String(ctx.draft[ctx.field.key] ?? '')}
              options={enums}
              onchange={(e) => ctx.setValue((e.currentTarget as HTMLSelectElement).value)}
            />
          </FormField>
        {:else}
          {@render ctx.default()}
        {/if}
      {/snippet}
    </KForm>
    </Card>
  {/if}

  <KTable bind:this={table} {model} emptyMessage={t(emptyKey)} {deletable} />
</div>

<KAppStatus />

<style>
  .workspace {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .workspace :global(.composer) {
    flex-direction: column;
    align-items: stretch;
  }
</style>
