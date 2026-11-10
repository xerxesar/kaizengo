import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { resolveFieldWidget } from '@/lib/field-widgets'
import { KEYMAP_ID_ATTR } from '@/lib/keymap/types'
import { fieldLabelClass } from '@/lib/styles/classes'
import { getI18n } from '@/lib/i18n-context'
import {
  createModelRecord,
  fetchModelViews,
  fetchViewSlots,
  formViewForCommand,
  formViewForModel,
  formatNamespace,
  getModelRecord,
  parseNamespace,
  resolveCqrsRef,
  updateModelRecord,
  type ModelField,
  type ModelView,
  type ModelViewGroup,
} from '@/lib/model-client'
import { KFormCtx } from './kform-context'
import type { KFormFieldContext } from './kform-types'

type Props = {
  command?: string
  model?: string
  id?: string
  view?: string
  submitLabel?: string
  successMessage?: string
  onsuccess?: (record: Record<string, unknown>) => void
  onerror?: (message: string) => void
  field?: (ctx: KFormFieldContext) => ReactNode
  children?: ReactNode
}

export function KForm(props: Props) {
  const formRef = useRef<HTMLFormElement>(null)

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [fields, setFields] = useState<ModelField[]>([])
  const [groups, setGroups] = useState<ModelViewGroup[]>([])
  const [draft, setDraft] = useState<Record<string, unknown>>({})
  const [boundApp, setBoundApp] = useState('')
  const [boundModel, setBoundModel] = useState('')
  const [modelRef, setModelRef] = useState('')
  const [createCommand, setCreateCommand] = useState<string | undefined>()
  const [updateCommand, setUpdateCommand] = useState<string | undefined>()

  const i18n = getI18n()
  const editing = Boolean(props.id?.trim())

  const buttonLabel =
    props.submitLabel ||
    (editing ? i18n.t(`${boundApp}.save`) : i18n.t(`${boundApp}.create`))

  function reportError(message: string) {
    setSuccess('')
    setError(message)
    props.onerror?.(message)
  }

  function defaultSuccessMessage(): string {
    const key = editing ? `${boundApp}.saved` : `${boundApp}.created`
    const message = i18n.t(key)
    return message !== key ? message : editing ? 'Saved.' : 'Created.'
  }

  function fieldLabel(field: ModelField, index: number): string {
    const keyed = i18n.t(`${boundApp}.field.${field.key}`)
    if (keyed !== `${boundApp}.field.${field.key}`) return keyed
    if (fields.length === 1) {
      const create = i18n.t(`${boundApp}.create`)
      if (create !== `${boundApp}.create`) return create
    }
    return field.label
  }

  function fieldPlaceholder(field: ModelField, index: number): string {
    const keyed = i18n.t(`${boundApp}.${field.key}_placeholder`)
    if (keyed !== `${boundApp}.${field.key}_placeholder`) return keyed
    if (index === 0) {
      const ph = i18n.t(`${boundApp}.new_placeholder`)
      if (ph !== `${boundApp}.new_placeholder`) return ph
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
    setDraft((prev) => ({ ...prev, [key]: value }))
  }

  function requestSubmit() {
    formRef.current?.requestSubmit()
  }

  const canSubmit = useMemo(
    () =>
      fields.length > 0 &&
      fields.filter((field) => field.required).every((field) => fieldHasValue(field)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [fields, draft],
  )

  const formContext = {
    get model() {
      return modelRef || props.command?.trim() || props.model?.trim() || ''
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
  }

  async function resolveFormView(): Promise<{ app: string; view: ModelView }> {
    const commandRef = props.command?.trim()
    if (commandRef) {
      const { app, field } = resolveCqrsRef(commandRef)
      const views = await fetchModelViews(app)
      const named = props.view?.trim()
        ? (views.find((item) => item.kind === 'form' && item.name === props.view!.trim()) ?? null)
        : null
      const view = named ?? formViewForCommand(views, field)
      if (!view?.fields?.length) {
        throw new Error(`no form view bound to command ${commandRef} (${field})`)
      }
      return { app, view }
    }

    const modelProp = props.model?.trim()
    if (modelProp) {
      const { app, name } = parseNamespace(modelProp)
      const views = await fetchModelViews(app)
      const formView = props.view?.trim()
        ? (views.find((item) => item.kind === 'form' && item.name === props.view!.trim()) ?? null)
        : formViewForModel(views, name)
      if (!formView?.fields?.length) {
        throw new Error(`no form view found for model ${modelProp}`)
      }
      return { app, view: formView }
    }

    throw new Error('KForm requires command (preferred) or model')
  }

  async function refresh() {
    setError('')
    setSuccess('')
    setLoading(true)
    try {
      const { app, view } = await resolveFormView()
      setBoundApp(app)
      setBoundModel(view.model)
      setModelRef(formatNamespace(app, view.model))
      setCreateCommand(view.createCommand?.trim() || undefined)
      setUpdateCommand(view.updateCommand?.trim() || undefined)
      setFields(view.fields ?? [])
      setGroups(view.groups ?? [])
      await fetchViewSlots(app, view.name)

      if (editing) {
        const record = await getModelRecord(
          app,
          view.model,
          props.id!.trim(),
          (view.fields ?? []).map((field) => field.key),
          view.getQuery?.trim() || undefined,
        )
        initDraft(view.fields ?? [], record)
      } else {
        initDraft(view.fields ?? [])
      }
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit || saving) return

    setSaving(true)
    setError('')
    setSuccess('')
    try {
      const record = editing
        ? await updateModelRecord(
            boundApp,
            boundModel,
            props.id!.trim(),
            fields,
            draft,
            updateCommand,
          )
        : await createModelRecord(boundApp, boundModel, fields, draft, createCommand)
      if (!editing) {
        initDraft(fields)
      } else {
        initDraft(fields, record)
      }
      setSuccess(props.successMessage?.trim() || defaultSuccessMessage())
      props.onsuccess?.(record)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.command, props.model, props.id, props.view])

  function renderDefaultField(fieldDef: ModelField, index: number) {
    const type = (fieldDef.type ?? 'string').toLowerCase()
    const widget = fieldDef.widget?.trim() ?? ''
    const value = draft[fieldDef.key]
    const placeholder = fieldPlaceholder(fieldDef, index)
    const Custom = resolveFieldWidget(widget)
    if (Custom) {
      return (
        <Custom
          value={value}
          record={draft}
          field={{
            key: fieldDef.key,
            label: fieldLabel(fieldDef, index),
            type: fieldDef.type,
            values: fieldDef.values,
          }}
          onChange={(next) => setFieldValue(fieldDef.key, next)}
        />
      )
    }

    if (widget === 'checkbox' || (!widget && type === 'bool')) {
      return (
        <label className="inline-flex cursor-pointer items-center gap-2">
          <Checkbox
            checked={Boolean(value)}
            onCheckedChange={(checked) => setFieldValue(fieldDef.key, checked === true)}
          />
          <span className="text-sm text-[var(--kg-text)]">{placeholder || fieldDef.label}</span>
        </label>
      )
    }

    if (widget === 'textarea' || (!widget && ['text', 'html', 'json'].includes(type))) {
      return (
        <Textarea
          value={String(value ?? '')}
          placeholder={placeholder}
          onChange={(e) => setFieldValue(fieldDef.key, e.target.value)}
        />
      )
    }

    if (widget === 'select' || (!widget && (fieldDef.values?.length ?? 0) > 0)) {
      const current = String(value ?? '')
      return (
        <Select value={current || undefined} onValueChange={(next) => setFieldValue(fieldDef.key, next)}>
          <SelectTrigger aria-label={fieldLabel(fieldDef, index)}>
            <SelectValue placeholder={placeholder} />
          </SelectTrigger>
          <SelectContent>
            {(fieldDef.values ?? []).map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    }

    const inputType = ['int', 'number', 'float', 'decimal'].includes(type)
      ? 'number'
      : type === 'date'
        ? 'date'
        : ['datetime', 'timestamp'].includes(type)
          ? 'datetime-local'
          : 'text'

    return (
      <Input
        type={inputType}
        placeholder={placeholder}
        value={String(value ?? '')}
        onChange={(e) => setFieldValue(fieldDef.key, e.target.value)}
      />
    )
  }

  function renderSpecFields() {
    const ordered = [...fields].sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    const indexOf = new Map(fields.map((field, index) => [field.key, index]))

    function box(fieldDef: ModelField, columns: number) {
      const index = indexOf.get(fieldDef.key) ?? 0
      const span = Math.max(1, Math.min(fieldDef.span ?? 1, columns))
      const ctx: KFormFieldContext = {
        field: fieldDef,
        index,
        draft,
        label: fieldLabel(fieldDef, index),
        placeholder: fieldPlaceholder(fieldDef, index),
        default: () => renderDefaultField(fieldDef, index),
        setValue: (value) => setFieldValue(fieldDef.key, value),
      }
      return (
        <div
          key={fieldDef.key}
          className="flex min-w-0 flex-col gap-1"
          style={{ gridColumn: `span ${span}`, width: fieldDef.width || undefined }}
        >
          {props.field ? (
            props.field(ctx)
          ) : (
            <>
              <Label className={fieldLabelClass}>
                {ctx.label}
                {fieldDef.required ? <span className="text-[var(--kg-danger)]"> *</span> : null}
              </Label>
              {ctx.default()}
            </>
          )}
        </div>
      )
    }

    function grid(items: ModelField[], columns: number) {
      const cols = Math.max(1, columns)
      return (
        <div
          className="grid w-full gap-5"
          style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
        >
          {items.map((fieldDef) => box(fieldDef, cols))}
        </div>
      )
    }

    const orderedGroups = [...groups].sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    if (!orderedGroups.length) return grid(ordered, 2)

    const used = new Set<string>()
    return (
      <div className="flex w-full flex-col gap-6">
        {orderedGroups.map((group) => {
          const items = ordered.filter((field) => (field.group ?? '') === group.id)
          items.forEach((field) => used.add(field.key))
          const titleKey = group.labelKey?.trim()
          const translated = titleKey ? i18n.t(titleKey) : ''
          const title = translated && translated !== titleKey ? translated : group.label || titleKey
          return (
            <section key={group.id} className="flex w-full flex-col gap-3">
              {title ? <h3 className="text-sm font-semibold text-[var(--kg-text)]">{title}</h3> : null}
              {grid(items, group.columns && group.columns > 0 ? group.columns : 1)}
            </section>
          )
        })}
        {ordered.some((field) => !used.has(field.key))
          ? grid(
              ordered.filter((field) => !used.has(field.key)),
              2,
            )
          : null}
      </div>
    )
  }

  return (
    <KFormCtx.Provider value={formContext}>
      <div className="flex flex-col gap-5">
        {success ? (
          <Alert variant="success">
            <div className="min-w-0 flex-1">{success}</div>
            <button
              type="button"
              className="shrink-0 text-current opacity-70 hover:opacity-100"
              aria-label="Dismiss"
              onClick={() => setSuccess('')}
            >
              ×
            </button>
          </Alert>
        ) : null}

        {error ? (
          <Alert variant="danger">
            <div className="min-w-0 flex-1">{error}</div>
            <button
              type="button"
              className="shrink-0 text-current opacity-70 hover:opacity-100"
              aria-label="Dismiss"
              onClick={() => setError('')}
            >
              ×
            </button>
          </Alert>
        ) : null}

        {loading ? (
          <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
            <Progress indeterminate className="w-48 max-w-full" />
          </div>
        ) : (
          <form ref={formRef} onSubmit={submit}>
            {props.children ? (
              <div className="flex flex-wrap items-start gap-5">{props.children}</div>
            ) : (
              renderSpecFields()
            )}
            <div className="mt-2 flex flex-wrap items-center">
              <Button
                type="submit"
                disabled={saving || !canSubmit}
                {...{ [KEYMAP_ID_ATTR]: 'form-save' }}
              >
                {saving ? '…' : buttonLabel}
              </Button>
            </div>
          </form>
        )}
      </div>
    </KFormCtx.Provider>
  )
}
