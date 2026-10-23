import type { ReactNode } from 'react'
import type { ModelField } from '@/lib/model-client'

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
  default: () => ReactNode
  setValue: (value: unknown) => void
}

export type KFormActionsContext = {
  saving: boolean
  canSubmit: boolean
  submit: () => void
  default: () => ReactNode
}
