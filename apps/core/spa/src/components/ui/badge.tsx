import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const badgeVariants = cva('inline-flex items-center px-2 py-0.5 text-xs font-medium', {
  variants: {
    variant: {
      default: 'bg-[var(--kg-surface-muted)] text-[var(--kg-text-secondary)]',
      muted: 'bg-[var(--kg-surface-muted)] text-[var(--kg-text-muted)]',
      success: 'bg-[var(--kg-success-bg)] text-[var(--kg-success)]',
      danger: 'bg-[var(--kg-danger-bg)] text-[var(--kg-danger)]',
      warning: 'bg-[var(--kg-warning-bg)] text-[var(--kg-warning)]',
      info: 'bg-[var(--kg-info-bg)] text-[var(--kg-info)]',
    },
  },
  defaultVariants: { variant: 'default' },
})

export function Badge({
  className,
  variant,
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />
}

export { badgeVariants }
