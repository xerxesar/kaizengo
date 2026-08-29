import { type JSX } from 'solid-js'
import { buttonSizes, buttonVariants } from './ark/styles'
import { cn } from './cn'
import { KEYMAP_ID_ATTR } from './keymap/types'

type Props = {
  variant?: keyof typeof buttonVariants
  size?: keyof typeof buttonSizes
  type?: 'button' | 'submit' | 'reset'
  loading?: boolean
  disabled?: boolean
  class?: string
  keymapId?: string
  onClick?: (e: MouseEvent) => void
  children: JSX.Element
}

export function Button(props: Props): JSX.Element {
  const size = () => props.size ?? 'md'
  const variant = () => props.variant ?? 'primary'
  return (
    <button
      type={props.type ?? 'button'}
      class={cn(
        'inline-flex items-center justify-center gap-2 font-medium transition',
        buttonSizes[size()],
        buttonVariants[variant()],
        props.class,
      )}
      disabled={props.disabled || props.loading}
      onClick={props.onClick}
      {...(props.keymapId ? { [KEYMAP_ID_ATTR]: props.keymapId } : {})}
    >
      {props.loading ? '…' : props.children}
    </button>
  )
}
