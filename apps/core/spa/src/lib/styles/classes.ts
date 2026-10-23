/** Shared Tailwind classes for UI primitives — kaizen theme tokens (--kg-*). */

export const inputClass =
  'h-10 w-full border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 text-sm text-[var(--kg-text)] outline-none focus:border-[var(--kg-primary)] disabled:cursor-not-allowed disabled:bg-[var(--kg-surface-muted)] disabled:text-[var(--kg-text-muted)]'

export const textareaClass =
  'min-h-24 w-full resize-y border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 py-3 text-sm text-[var(--kg-text)] outline-none focus:border-[var(--kg-primary)]'

export const buttonVariants = {
  primary:
    'bg-[var(--kg-primary)] text-white hover:opacity-90 disabled:opacity-50',
  secondary:
    'border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] text-[var(--kg-text)] hover:bg-[var(--kg-surface-hover)] disabled:opacity-50',
  ghost:
    'bg-transparent text-[var(--kg-text)] hover:bg-[var(--kg-surface-hover)] disabled:opacity-50',
  danger: 'bg-[var(--kg-danger)] text-[var(--kg-text-on-color)] hover:opacity-90 disabled:opacity-50',
} as const

export const buttonSizes = {
  sm: 'h-8 px-3 text-sm',
  md: 'h-10 px-4 text-sm',
} as const

export const selectTriggerClass =
  'inline-flex h-10 w-full items-center justify-between gap-2 border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 text-sm text-[var(--kg-text)] outline-none focus:border-[var(--kg-primary)] disabled:cursor-not-allowed disabled:opacity-50'

export const selectContentClass =
  'z-50 max-h-64 min-w-[8rem] overflow-auto border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] py-1 shadow-lg'

export const selectItemClass =
  'flex cursor-pointer items-center px-4 py-2 text-sm text-[var(--kg-text)] outline-none data-highlighted:bg-[var(--kg-surface-hover)] data-selected:font-semibold data-selected:text-[var(--kg-primary)]'

export const comboboxControlClass =
  'flex h-10 w-full items-center gap-2 border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 focus-within:border-[var(--kg-primary)] disabled:cursor-not-allowed disabled:opacity-50'

export const comboboxInputClass =
  'min-w-0 flex-1 border-0 bg-transparent p-0 text-sm text-[var(--kg-text)] outline-none placeholder:text-[var(--kg-text-muted)] disabled:cursor-not-allowed'

export const comboboxTriggerClass =
  'inline-flex shrink-0 cursor-pointer items-center border-0 bg-transparent p-0 text-sm text-[var(--kg-text-muted)] outline-none disabled:cursor-not-allowed'

export const fieldLabelClass = 'text-sm font-medium text-[var(--kg-text)]'
export const fieldHintClass = 'text-xxs text-[var(--kg-text-muted)] font-light'
export const fieldErrorClass = 'text-xs text-[var(--kg-danger)]'

export const dialogBackdropClass = 'fixed inset-0 z-[9000] bg-black/50'
export const dialogPositionerClass =
  'fixed inset-0 z-[9001] flex items-center justify-center p-4'
export const dialogContentClass =
  'flex max-h-[90vh] w-full flex-col bg-[var(--kg-surface)] shadow-lg outline-none'

export const menuContentClass =
  'z-500000 min-w-48 overflow-auto border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] py-1 shadow-lg outline-none'

export const menuItemClass =
  'flex cursor-pointer items-center px-4 py-2 text-sm text-[var(--kg-text)] outline-none data-highlighted:bg-[var(--kg-surface-hover)]'

export const checkboxControlClass =
  'flex h-4 w-4 items-center justify-center border border-[var(--kg-border-strong)] bg-[var(--kg-field)] data-checked:border-[var(--kg-primary)] data-checked:bg-[var(--kg-primary)] data-checked:text-[var(--kg-text-on-color)]'

export const progressTrackClass = 'h-1 w-full overflow-hidden bg-[var(--kg-border)]'
export const progressRangeClass =
  'h-full bg-[var(--kg-primary)] transition-all data-[progress=loading]:animate-pulse'

export const menubarLinkClass =
  'gap-1.5 inline-flex h-10 items-center border-b-[3px] border-transparent px-5 text-sm text-[var(--kg-text)] no-underline hover:bg-[var(--kg-surface-muted,var(--kg-field-hover))] data-current:border-[var(--kg-primary)] data-current:font-semibold data-current:text-[var(--kg-primary)]'

export const treeItemClass =
  'flex w-full items-center gap-2 border-b border-[var(--kg-border)] px-5 py-3 text-left text-sm text-[var(--kg-text)] outline-none hover:bg-[var(--kg-surface-hover)] data-selected:border-l-[3px] data-selected:border-l-[var(--kg-primary)] data-selected:bg-[var(--kg-primary-subtle)] data-selected:text-[var(--kg-primary)]'

/** Semantic status surfaces — prefer these over raw Tailwind palette colors. */
export const badgeVariants = {
  default: 'bg-[var(--kg-surface-muted)] text-[var(--kg-text-secondary)]',
  muted: 'bg-[var(--kg-surface-muted)] text-[var(--kg-text-muted)]',
  success: 'bg-[var(--kg-success-bg)] text-[var(--kg-success)]',
  danger: 'bg-[var(--kg-danger-bg)] text-[var(--kg-danger)]',
  warning: 'bg-[var(--kg-warning-bg)] text-[var(--kg-warning)]',
  info: 'bg-[var(--kg-info-bg)] text-[var(--kg-info)]',
} as const

export const alertVariants = {
  info: 'border-[var(--kg-info-border)] bg-[var(--kg-info-bg)] text-[var(--kg-info)]',
  success: 'border-[var(--kg-success-border)] bg-[var(--kg-success-bg)] text-[var(--kg-success)]',
  warning: 'border-[var(--kg-warning-border)] bg-[var(--kg-warning-bg)] text-[var(--kg-warning)]',
  danger: 'border-[var(--kg-danger-border)] bg-[var(--kg-danger-bg)] text-[var(--kg-danger)]',
} as const

export function kindTagClass(kind: string): string {
  switch (kind) {
    case 'query':
      return badgeVariants.info
    case 'command':
      return badgeVariants.success
    case 'view':
      return badgeVariants.default
    default:
      return badgeVariants.muted
  }
}
