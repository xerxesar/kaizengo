import type { Snippet } from 'svelte'
import type { ModelField } from './model-client'

export type KFormContext = {
  model: string
  fields: ModelField[]
  draft: Record<string, unknown>
  saving: boolean
  canSubmit: boolean
  setValue: (key: string, value: unknown) => void
  submit: () => void
}

export type KFormFieldContext = {
  field: ModelField
  index: number
  draft: Record<string, unknown>
  label: string
  placeholder: string
  /** Built-in field renderer — call for Odoo-style view inheritance. */
  default: Snippet
  setValue: (value: unknown) => void
}

export type KFormActionsContext = {
  saving: boolean
  canSubmit: boolean
  submit: () => void
  /** Built-in submit button row. */
  default: Snippet
}
