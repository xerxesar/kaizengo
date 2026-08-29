import { type JSX } from 'solid-js'
import { cn } from './cn'
import type { BadgeVariant } from './types'

const variants: Record<BadgeVariant, string> = {
  success: 'bg-emerald-100 text-emerald-800',
  danger: 'bg-red-100 text-red-800',
  warning: 'bg-amber-100 text-amber-900',
  muted: 'bg-zinc-100 text-zinc-600',
  info: 'bg-sky-100 text-sky-800',
}

type Props = {
  variant?: BadgeVariant
  children: JSX.Element
  class?: string
}

export function Badge(props: Props): JSX.Element {
  return (
    <span
      class={cn(
        'inline-flex items-center rounded px-2 py-0.5 text-xs font-medium',
        variants[props.variant ?? 'muted'],
        props.class,
      )}
    >
      {props.children}
    </span>
  )
}
