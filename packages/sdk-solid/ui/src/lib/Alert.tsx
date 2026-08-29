import { type JSX } from 'solid-js'
import { cn } from './cn'

type Props = {
  variant?: 'danger' | 'success' | 'info' | 'warning'
  dismissible?: boolean
  onDismiss?: () => void
  children: JSX.Element
  class?: string
}

const variantClass: Record<NonNullable<Props['variant']>, string> = {
  danger: 'border-red-200 bg-red-50 text-red-900',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  info: 'border-sky-200 bg-sky-50 text-sky-900',
  warning: 'border-amber-200 bg-amber-50 text-amber-900',
}

export function Alert(props: Props): JSX.Element {
  return (
    <div
      class={cn(
        'kg-alert flex items-start justify-between gap-3 border px-4 py-3 text-sm',
        variantClass[props.variant ?? 'info'],
        props.class,
      )}
      role="alert"
    >
      <div class="min-w-0 flex-1">{props.children}</div>
      {props.dismissible && (
        <button
          type="button"
          class="shrink-0 text-current opacity-70 hover:opacity-100"
          aria-label="Dismiss"
          onClick={() => props.onDismiss?.()}
        >
          ×
        </button>
      )}
    </div>
  )
}
