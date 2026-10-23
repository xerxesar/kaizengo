import type { ReactNode } from 'react'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md'
export type AlertVariant = 'info' | 'success' | 'warning' | 'danger'
export type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'muted'

export type Column<T> = {
  key: string
  label: string
  width?: string
  align?: 'left' | 'center' | 'right'
  mono?: boolean
  render?: (row: T) => string
  cell?: (row: T) => ReactNode
}

export type TabItem = {
  id: string
  label: string
  icon?: string
  badge?: string | number
}

export type MenuItem = {
  id: string
  label: string
  labelKey?: string
  view?: string
  route?: string
  component?: string
  sourceApp?: string
  children?: MenuItem[]
}

export type ContainerSize = 'sm' | 'md' | 'lg' | 'xl' | 'fluid'
export type ContainerAlign = 'start' | 'center'

export type TreeNode<T = unknown> = {
  id: string
  label: string
  meta?: string
  data?: T
  children?: TreeNode<T>[]
}

export type FormFieldProps = {
  label: string
  hint?: string
  error?: string
  required?: boolean
  children: ReactNode
}
