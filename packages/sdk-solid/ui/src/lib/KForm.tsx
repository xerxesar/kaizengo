import { Field } from '@ark-ui/solid/field'
import { createMemo, createSignal, For, onMount, Show, type JSX, type ParentProps } from 'solid-js'
import { Alert } from './Alert'
import { Button } from './Button'
import { CheckboxInput } from './Checkbox'
import { FormField } from './FormField'
import { Input } from './Input'
import { Spinner } from './Spinner'
import { textareaClass } from './ark/styles'
import { KFormCtx } from './kform-context'
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
} from './model-client'

import type { KFormFieldContext } from './kform-types'

type Props = ParentProps & {
  model: string
  id?: string
  view?: string
  submitLabel?: string
  successMessage?: string
  onsuccess?: (record: Record<string, unknown>) => void
  onerror?: (message: string) => void
  field?: (ctx: KFormFieldContext) => JSX.Element
}

export function KForm(props: Props): JSX.Element {
  let formEl: HTMLFormElement | undefined

  const [loading, setLoading] = createSignal(true)
  const [saving, setSaving] = createSignal(false)
  const [error, setError] = createSignal('')
  const [success, setSuccess] = createSignal('')
  const [fields, setFields] = createSignal<ModelField[]>([])
  const [draft, setDraft] = createSignal<Record<string, unknown>>({})

  const i18n = getI18n()
  const ns = () => parseNamespace(props.model)
  const editing = () => Boolean(props.id?.trim())

  const buttonLabel = createMemo(
    () =>
      props.submitLabel ||
      (editing() ? i18n.t(`${ns().app}.save`) : i18n.t(`${ns().app}.create`)),
  )

  function reportError(message: string) {
    setSuccess('')
    setError(message)
    props.onerror?.(message)
  }

  function defaultSuccessMessage(): string {
    const key = editing() ? `${ns().app}.saved` : `${ns().app}.created`
    const message = i18n.t(key)
    return message !== key ? message : editing() ? 'Saved.' : 'Created.'
  }

  function fieldLabel(field: ModelField, index: number): string {
    const keyed = i18n.t(`${ns().app}.field.${field.key}`)
    if (keyed !== `${ns().app}.field.${field.key}`) return keyed
    if (fields().length === 1) {
      const create = i18n.t(`${ns().app}.create`)
      if (create !== `${ns().app}.create`) return create
    }
    return field.label
  }

  function fieldPlaceholder(field: ModelField, index: number): string {
    const keyed = i18n.t(`${ns().app}.${field.key}_placeholder`)
    if (keyed !== `${ns().app}.${field.key}_placeholder`) return keyed
    if (index === 0) {
      const ph = i18n.t(`${ns().app}.new_placeholder`)
      if (ph !== `${ns().app}.new_placeholder`) return ph
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
    setDraft(next)
  }

  function fieldHasValue(field: ModelField): boolean {
    const value = draft()[field.key]
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
    setDraft((prev) => ({ ...prev, [key]: value }))
  }

  function requestSubmit() {
    formEl?.requestSubmit()
  }

  const canSubmit = createMemo(
    () =>
      fields().length > 0 &&
      fields()
        .filter((field) => field.required)
        .every((field) => fieldHasValue(field)),
  )

  const formContext = {
    get model() {
      return props.model
    },
    get fields() {
      return fields()
    },
    get draft() {
      return draft()
    },
    get saving() {
      return saving()
    },
    get canSubmit() {
      return canSubmit()
    },
    setValue: setFieldValue,
    submit: requestSubmit,
  }

  async function refresh() {
    setError('')
    setSuccess('')
    setLoading(true)
    try {
      const { app, name } = ns()
      const views = await fetchModelViews(app)
      const formView = props.view?.trim()
        ? views.find((item) => item.kind === 'form' && item.name === props.view!.trim()) ?? null
        : formViewForModel(views, name)
      if (!formView?.fields?.length) {
        throw new Error(`no form view found for model ${props.model}`)
      }

      setFields(formView.fields)
      await fetchViewSlots(app, formView.name)

      if (editing()) {
        const record = await getModelRecord(
          app,
          name,
          props.id!.trim(),
          formView.fields.map((field) => field.key),
        )
        initDraft(formView.fields, record)
      } else {
        initDraft(formView.fields)
      }
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function submit(e: Event) {
    e.preventDefault()
    if (!canSubmit() || saving()) return

    setSaving(true)
    setError('')
    setSuccess('')
    try {
      const { app, name } = ns()
      const record = editing()
        ? await updateModelRecord(app, name, props.id!.trim(), fields(), draft())
        : await createModelRecord(app, name, fields(), draft())
      if (!editing()) {
        initDraft(fields())
      } else {
        initDraft(fields(), record)
      }
      setSuccess(props.successMessage?.trim() || defaultSuccessMessage())
      props.onsuccess?.(record)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  onMount(() => {
    void refresh()
  })

  function renderDefaultField(fieldDef: ModelField, index: number) {
    const type = (fieldDef.type ?? 'string').toLowerCase()
    const value = () => draft()[fieldDef.key]
    const placeholder = fieldPlaceholder(fieldDef, index)

    if (type === 'bool') {
      return (
        <CheckboxInput
          checked={Boolean(value())}
          label={placeholder || fieldDef.label}
          onChange={(checked) => setFieldValue(fieldDef.key, checked)}
        />
      )
    }

    if (['int', 'number', 'float', 'decimal'].includes(type)) {
      return (
        <Input
          type="number"
          placeholder={placeholder}
          value={String(value() ?? '')}
          onChange={(next) => setFieldValue(fieldDef.key, next)}
        />
      )
    }

    if (type === 'date') {
      return (
        <Input
          type="date"
          value={String(value() ?? '')}
          onChange={(next) => setFieldValue(fieldDef.key, next)}
        />
      )
    }

    if (['datetime', 'timestamp'].includes(type)) {
      return (
        <Input
          type="datetime-local"
          value={String(value() ?? '')}
          onChange={(next) => setFieldValue(fieldDef.key, next)}
        />
      )
    }

    if (['text', 'html', 'json'].includes(type)) {
      return (
        <Field.Textarea
          class={textareaClass}
          value={String(value() ?? '')}
          placeholder={placeholder}
          onInput={(e) => setFieldValue(fieldDef.key, e.currentTarget.value)}
        />
      )
    }

    return (
      <Input
        placeholder={placeholder}
        value={String(value() ?? '')}
        onChange={(next) => setFieldValue(fieldDef.key, next)}
      />
    )
  }

  return (
    <KFormCtx.Provider value={formContext}>
      <div class="flex flex-col gap-5">
        <Show when={success()}>
          <Alert variant="success" dismissible onDismiss={() => setSuccess('')}>
            {success()}
          </Alert>
        </Show>

        <Show when={error()}>
          <Alert variant="danger" dismissible onDismiss={() => setError('')}>
            {error()}
          </Alert>
        </Show>

        <Show when={!loading()} fallback={<Spinner />}>
          <form ref={formEl} class="" onSubmit={submit}>
            <div class="flex flex-wrap items-start gap-5">
            <Show
              when={props.children}
              fallback={
                <For each={fields()}>
                  {(fieldDef, index) => {
                    const ctx: KFormFieldContext = {
                      field: fieldDef,
                      index: index(),
                      draft: draft(),
                      label: fieldLabel(fieldDef, index()),
                      placeholder: fieldPlaceholder(fieldDef, index()),
                      default: () => renderDefaultField(fieldDef, index()),
                      setValue: (value) => setFieldValue(fieldDef.key, value),
                    }
                    return props.field ? (
                      props.field(ctx)
                    ) : (
                      <FormField label={ctx.label} required={fieldDef.required}>
                        {ctx.default()}
                      </FormField>
                    )
                  }}
                </For>
              }
            >
              {props.children}
            </Show>
            </div>
            <div class="flex flex-wrap items-center mt-2">
              <Button
                type="submit"
                loading={saving()}
                disabled={!canSubmit()}
                keymapId="form-save"
              >
                {buttonLabel()}
              </Button>
            </div>
          </form>
        </Show>
      </div>
    </KFormCtx.Provider>
  )
}
