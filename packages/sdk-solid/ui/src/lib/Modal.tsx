import { Dialog } from '@ark-ui/solid/dialog'
import { Show, type JSX, type ParentProps } from 'solid-js'
import { dialogBackdropClass, dialogContentClass } from './ark/styles'
import { cn } from './cn'

type Props = ParentProps & {
  open: boolean
  title: string
  size?: 'sm' | 'md' | 'lg'
  footer?: JSX.Element
  onClose?: () => void
  onOpenChange?: (open: boolean) => void
}

const sizeClass = {
  sm: 'max-w-sm',
  md: 'max-w-xl',
  lg: 'max-w-3xl',
}

export function Modal(props: Props): JSX.Element {
  const size = () => props.size ?? 'md'

  return (
    <Dialog.Root
      open={props.open}
      onOpenChange={(details) => {
        props.onOpenChange?.(details.open)
        if (!details.open) props.onClose?.()
      }}
    >
      <Dialog.Backdrop class={dialogBackdropClass} />
      <Dialog.Positioner>
        <Dialog.Content class={cn(dialogContentClass, sizeClass[size()])}>
          <header class="flex items-start justify-between gap-5 border-b border-[var(--kg-border)] px-6 py-4">
            <Dialog.Title class="text-xl font-normal">{props.title}</Dialog.Title>
            <Dialog.CloseTrigger class="border-0 bg-transparent text-2xl leading-none text-[var(--kg-text-secondary)] hover:text-[var(--kg-text)]">
              ×
            </Dialog.CloseTrigger>
          </header>
          <div class="overflow-y-auto px-6 py-6">{props.children}</div>
          <Show when={props.footer}>
            <footer class="flex justify-end gap-3 border-t border-[var(--kg-border)] bg-[var(--kg-surface-muted)] px-6 py-4">
              {props.footer}
            </footer>
          </Show>
        </Dialog.Content>
      </Dialog.Positioner>
    </Dialog.Root>
  )
}
