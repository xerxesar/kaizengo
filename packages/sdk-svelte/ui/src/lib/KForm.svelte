<script lang="ts">
  import { onMount, setContext } from 'svelte'
  import type { Snippet } from 'svelte'
  import Alert from './Alert.svelte'
  import Button from './Button.svelte'
  import FormField from './FormField.svelte'
  import Input from './Input.svelte'
  import KViewSlots from './KViewSlots.svelte'
  import Spinner from './Spinner.svelte'
  import { KFORM_CTX } from './kform-context'
  import type { KFormActionsContext, KFormContext, KFormFieldContext } from './kform-types'
  import { getI18n } from './i18n-context'
  import {
    createModelRecord,
    fetchModelViews,
    fetchViewSlots,
    formViewForModel,
    getModelRecord,
    parseNamespace,
    updateModelRecord,
    type ModelField,
    type ViewSlot,
  } from './model-client'

  type Props = {
    /** Namespaced model ref (`hellospec.greeting`, `notes.note`, …). */
    model: string
    /** When set, loads and updates an existing record. */
    id?: string
    /** Override the form view name when multiple form views exist. */
    view?: string
    submitLabel?: string
    /** Override the post-save success banner; defaults to `{app}.saved` / `{app}.created`. */
    successMessage?: string
    onsuccess?: (record: Record<string, unknown>) => void
    onerror?: (message: string) => void
    /** Snippet — content above the form (overrides view slots). */
    before?: Snippet
    /** Snippet — content below the form (overrides view slots). */
    after?: Snippet
    /** Snippet — submit row; call `{@render ctx.default()}` to keep the stock button. */
    actions?: Snippet<[KFormActionsContext]>
    /** Snippet — one field in the auto loop; call `{@render ctx.default()}` for stock widgets. */
    field?: Snippet<[KFormFieldContext]>
    /** Implicit child markup or `{#snippet children(ctx)}` — replaces the auto field loop. */
    children?: Snippet | Snippet<[KFormContext]>
  }

  let {
    model,
    id = '',
    view = '',
    submitLabel = '',
    successMessage = '',
    onsuccess,
    onerror,
    before,
    after,
    actions,
    field,
    children,
  }: Props = $props()

  let formEl = $state<HTMLFormElement | null>(null)
  let loading = $state(true)
  let saving = $state(false)
  let error = $state('')
  let success = $state('')
  let viewName = $state('')
  let viewSlots = $state<ViewSlot[]>([])
  let fields = $state<ModelField[]>([])
  let draft = $state<Record<string, unknown>>({})

  const i18n = getI18n()
  const ns = $derived(parseNamespace(model))
  const editing = $derived(Boolean(id.trim()))
  const buttonLabel = $derived(
    submitLabel ||
      (editing ? i18n.t(`${ns.app}.save`) : i18n.t(`${ns.app}.create`)),
  )

  function reportError(message: string) {
    success = ''
    error = message
    onerror?.(message)
  }

  function defaultSuccessMessage(): string {
    const key = editing ? `${ns.app}.saved` : `${ns.app}.created`
    const message = i18n.t(key)
    return message !== key ? message : editing ? 'Saved.' : 'Created.'
  }

  function fieldLabel(field: ModelField, index: number): string {
    const keyed = i18n.t(`${ns.app}.field.${field.key}`)
    if (keyed !== `${ns.app}.field.${field.key}`) return keyed
    if (fields.length === 1) {
      const create = i18n.t(`${ns.app}.create`)
      if (create !== `${ns.app}.create`) return create
    }
    return field.label
  }

  function fieldPlaceholder(field: ModelField, index: number): string {
    const keyed = i18n.t(`${ns.app}.${field.key}_placeholder`)
    if (keyed !== `${ns.app}.${field.key}_placeholder`) return keyed
    if (index === 0) {
      const ph = i18n.t(`${ns.app}.new_placeholder`)
      if (ph !== `${ns.app}.new_placeholder`) return ph
    }
    return ''
  }

  function defaultValue(field: ModelField): unknown {
    switch ((field.type ?? 'string').toLowerCase()) {
      case 'bool':
        return false
      case 'int':
        return ''
      default:
        return ''
    }
  }

  function initDraft(formFields: ModelField[], record?: Record<string, unknown>) {
    const next: Record<string, unknown> = {}
    for (const field of formFields) {
      next[field.key] = record?.[field.key] ?? defaultValue(field)
    }
    draft = next
  }

  function fieldHasValue(field: ModelField): boolean {
    const value = draft[field.key]
    if (value == null) return false
    switch ((field.type ?? 'string').toLowerCase()) {
      case 'bool':
        return true
      case 'int':
        return value !== '' && !Number.isNaN(Number(value))
      default:
        return String(value).trim() !== ''
    }
  }

  function setFieldValue(key: string, value: unknown) {
    draft[key] = value
  }

  function requestSubmit() {
    formEl?.requestSubmit()
  }

  const canSubmit = $derived(
    fields.length > 0 && fields.filter((field) => field.required).every((field) => fieldHasValue(field)),
  )

  const formContext = $derived({
    model,
    fields,
    draft,
    saving,
    canSubmit,
    setValue: setFieldValue,
    submit: requestSubmit,
  } satisfies KFormContext)

  setContext(KFORM_CTX, {
    get model() {
      return model
    },
    get fields() {
      return fields
    },
    get draft() {
      return draft
    },
    get saving() {
      return saving
    },
    get canSubmit() {
      return canSubmit
    },
    setValue: setFieldValue,
    submit: requestSubmit,
  })

  async function refresh() {
    error = ''
    success = ''
    loading = true
    try {
      const { app, name } = ns
      const views = await fetchModelViews(app)
      const formView = view.trim()
        ? views.find((item) => item.kind === 'form' && item.name === view.trim()) ?? null
        : formViewForModel(views, name)
      if (!formView?.fields?.length) {
        throw new Error(`no form view found for model ${model}`)
      }

      viewName = formView.name
      fields = formView.fields
      viewSlots = await fetchViewSlots(app, formView.name)

      if (editing) {
        const record = await getModelRecord(
          app,
          name,
          id.trim(),
          fields.map((field) => field.key),
        )
        initDraft(fields, record)
      } else {
        initDraft(fields)
      }
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      loading = false
    }
  }

  async function submit(e: Event) {
    e.preventDefault()
    if (!canSubmit || saving) return

    saving = true
    error = ''
    success = ''
    try {
      const { app, name } = ns
      const record = editing
        ? await updateModelRecord(app, name, id.trim(), fields, draft)
        : await createModelRecord(app, name, fields, draft)
      if (!editing) {
        initDraft(fields)
      } else {
        initDraft(fields, record)
      }
      success = successMessage.trim() || defaultSuccessMessage()
      onsuccess?.(record)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      saving = false
    }
  }

  onMount(() => {
    void refresh()
  })
</script>

<div class="kg-kform">
  {#if success}
    <Alert variant="success" dismissible ondismiss={() => (success = '')}>{success}</Alert>
  {/if}

  {#if error}
    <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
  {/if}

  {#if loading}
    <Spinner />
  {:else}
    {#if before}
      {@render before()}
    {:else}
      <KViewSlots slots={viewSlots} slotName="before" />
    {/if}

    <form bind:this={formEl} class="composer" onsubmit={submit}>
      <KViewSlots slots={viewSlots} slotName="toolbar" />

      {#if children}
        {@render children(formContext)}
      {:else}
        {#each fields as fieldDef, index (fieldDef.key)}
          <KViewSlots slots={viewSlots} slotName={`field.${fieldDef.key}.before`} />

          {#snippet defaultField()}
            <FormField label={fieldLabel(fieldDef, index)} required={fieldDef.required}>
              {#snippet children()}
                {#if (fieldDef.type ?? 'string').toLowerCase() === 'bool'}
                  <label class="kg-checkbox">
                    <input
                      type="checkbox"
                      checked={Boolean(draft[fieldDef.key])}
                      onchange={(e) => {
                        draft[fieldDef.key] = (e.currentTarget as HTMLInputElement).checked
                      }}
                    />
                    <span>{fieldPlaceholder(fieldDef, index) || fieldDef.label}</span>
                  </label>
                {:else if ['int', 'number', 'float', 'decimal'].includes((fieldDef.type ?? '').toLowerCase())}
                  <Input
                    type="number"
                    bind:value={draft[fieldDef.key] as string}
                    placeholder={fieldPlaceholder(fieldDef, index)}
                  />
                {:else if (fieldDef.type ?? '').toLowerCase() === 'date'}
                  <Input
                    type="date"
                    bind:value={draft[fieldDef.key] as string}
                  />
                {:else if ['datetime', 'timestamp'].includes((fieldDef.type ?? '').toLowerCase())}
                  <Input
                    type="datetime-local"
                    bind:value={draft[fieldDef.key] as string}
                  />
                {:else if ['text', 'html', 'json'].includes((fieldDef.type ?? '').toLowerCase())}
                  <textarea
                    class="kg-textarea"
                    bind:value={draft[fieldDef.key] as string}
                    placeholder={fieldPlaceholder(fieldDef, index)}
                  ></textarea>
                {:else}
                  <Input
                    bind:value={draft[fieldDef.key] as string}
                    placeholder={fieldPlaceholder(fieldDef, index)}
                  />
                {/if}
              {/snippet}
            </FormField>
          {/snippet}

          {#if field}
            {@render field({
              field: fieldDef,
              index,
              draft,
              label: fieldLabel(fieldDef, index),
              placeholder: fieldPlaceholder(fieldDef, index),
              default: defaultField,
              setValue: (value) => setFieldValue(fieldDef.key, value),
            })}
          {:else}
            <KViewSlots slots={viewSlots} slotName={`field.${fieldDef.key}`} />
            {@render defaultField()}
          {/if}

          <KViewSlots slots={viewSlots} slotName={`field.${fieldDef.key}.after`} />
        {/each}
      {/if}

      {#snippet defaultActions()}
        <Button type="submit" loading={saving} disabled={!canSubmit}>{buttonLabel}</Button>
      {/snippet}

      <div class="actions">
        <KViewSlots slots={viewSlots} slotName="actions" />
        {#if actions}
          {@render actions({
            saving,
            canSubmit,
            submit: requestSubmit,
            default: defaultActions,
          })}
        {:else}
          {@render defaultActions()}
        {/if}
      </div>
    </form>

    {#if after}
      {@render after()}
    {:else}
      <KViewSlots slots={viewSlots} slotName="after" />
    {/if}
  {/if}
</div>

<style>
  .kg-kform {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
  }

  .composer {
    display: flex;
    flex-wrap: wrap;
    gap: var(--kg-space-05);
    align-items: flex-end;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--kg-space-04);
    align-items: center;
  }

  .kg-textarea {
    width: 100%;
    min-height: 6rem;
    padding: var(--kg-space-04) var(--kg-space-05);
    border: none;
    border-bottom: 1px solid var(--kg-border-strong);
    border-radius: var(--kg-radius);
    background: var(--kg-field);
    color: var(--kg-text);
    font: inherit;
    resize: vertical;
  }

  .kg-checkbox {
    display: inline-flex;
    align-items: center;
    gap: var(--kg-space-03);
    font-size: 0.875rem;
    color: var(--kg-text);
  }
</style>
